package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"syscall"
	"unsafe"

	"github.com/creack/pty"
	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow all origins for terminal connections
		// The auth middleware already handles access control
		return true
	},
}

// TerminalMessage represents a message from the client
type TerminalMessage struct {
	Type string          `json:"type"` // "input", "resize", "ping"
	Data json.RawMessage `json:"data,omitempty"`
}

// ResizeMessage represents terminal resize data
type ResizeMessage struct {
	Cols uint16 `json:"cols"`
	Rows uint16 `json:"rows"`
}

// TerminalSession manages a PTY session
type TerminalSession struct {
	cmd    *exec.Cmd
	ptmx   *os.File
	ws     *websocket.Conn
	mu     sync.Mutex
	closed bool
}

// NewTerminalSession creates a new terminal session
func NewTerminalSession(ws *websocket.Conn) (*TerminalSession, error) {
	// Start bash with login shell for proper environment
	cmd := exec.Command("/bin/bash", "-l")
	cmd.Env = append(os.Environ(),
		"TERM=xterm-256color",
		"COLORTERM=truecolor",
		"LC_ALL=en_US.UTF-8",
		"LANG=en_US.UTF-8",
	)

	// Start the command with a PTY
	ptmx, err := pty.Start(cmd)
	if err != nil {
		return nil, err
	}

	session := &TerminalSession{
		cmd:  cmd,
		ptmx: ptmx,
		ws:   ws,
	}

	return session, nil
}

// Resize changes the terminal size
func (s *TerminalSession) Resize(cols, rows uint16) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return nil
	}

	window := struct {
		row uint16
		col uint16
		x   uint16
		y   uint16
	}{
		row: rows,
		col: cols,
	}

	_, _, errno := syscall.Syscall(
		syscall.SYS_IOCTL,
		s.ptmx.Fd(),
		syscall.TIOCSWINSZ,
		uintptr(unsafe.Pointer(&window)),
	)

	if errno != 0 {
		return errno
	}

	return nil
}

// Close terminates the session
func (s *TerminalSession) Close() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.closed {
		return
	}
	s.closed = true

	// Close PTY first
	if s.ptmx != nil {
		s.ptmx.Close()
	}

	// Kill the process if still running
	if s.cmd != nil && s.cmd.Process != nil {
		s.cmd.Process.Kill()
		s.cmd.Wait()
	}

	// Close WebSocket
	if s.ws != nil {
		s.ws.Close()
	}
}

// ReadFromPTY reads from PTY and writes to WebSocket
func (s *TerminalSession) ReadFromPTY() {
	buf := make([]byte, 4096)
	for {
		n, err := s.ptmx.Read(buf)
		if err != nil {
			if err != io.EOF {
				log.Printf("PTY read error: %v", err)
			}
			break
		}

		s.mu.Lock()
		closed := s.closed
		s.mu.Unlock()

		if closed {
			break
		}

		// Send binary data to WebSocket
		err = s.ws.WriteMessage(websocket.BinaryMessage, buf[:n])
		if err != nil {
			log.Printf("WebSocket write error: %v", err)
			break
		}
	}
	s.Close()
}

// WriteFromWebSocket reads from WebSocket and writes to PTY
func (s *TerminalSession) WriteFromWebSocket() {
	for {
		messageType, message, err := s.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			break
		}

		s.mu.Lock()
		closed := s.closed
		s.mu.Unlock()

		if closed {
			break
		}

		if messageType == websocket.TextMessage {
			// Parse as JSON message
			var msg TerminalMessage
			if err := json.Unmarshal(message, &msg); err != nil {
				// Treat as raw input if not valid JSON
				s.ptmx.Write(message)
				continue
			}

			switch msg.Type {
			case "input":
				var input string
				if err := json.Unmarshal(msg.Data, &input); err == nil {
					s.ptmx.Write([]byte(input))
				}
			case "resize":
				var resize ResizeMessage
				if err := json.Unmarshal(msg.Data, &resize); err == nil {
					s.Resize(resize.Cols, resize.Rows)
				}
			case "ping":
				s.ws.WriteMessage(websocket.TextMessage, []byte(`{"type":"pong"}`))
			}
		} else if messageType == websocket.BinaryMessage {
			// Raw binary input
			s.ptmx.Write(message)
		}
	}
	s.Close()
}

// TerminalWebSocketHandler handles WebSocket terminal connections
func TerminalWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	// Upgrade to WebSocket
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}

	// Create terminal session
	session, err := NewTerminalSession(conn)
	if err != nil {
		log.Printf("Failed to create terminal session: %v", err)
		conn.WriteMessage(websocket.TextMessage, []byte(`{"type":"error","data":"Failed to create terminal session"}`))
		conn.Close()
		return
	}

	log.Printf("New terminal session started")

	// Set initial size (default 80x24)
	session.Resize(80, 24)

	// Start bidirectional copy
	go session.ReadFromPTY()
	session.WriteFromWebSocket()

	log.Printf("Terminal session ended")
}
