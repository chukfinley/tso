package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
)

var vmConsoleUpgrader = websocket.Upgrader{
	ReadBufferSize:  4096,
	WriteBufferSize: 4096,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

// GetVMConsoleInfoHandler returns console connection info for a VM
func GetVMConsoleInfoHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	vm, err := scanVM(db.QueryRow("SELECT "+vmFields+" FROM virtual_machines WHERE id = ?", id))
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	if vm.Status != "running" {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   "VM is not running",
		})
		return
	}

	// Generate a temporary token for WebSocket authentication
	token := generatePassword()

	json.NewEncoder(w).Encode(map[string]any{
		"success": true,
		"console": map[string]any{
			"type":           vm.DisplayType,
			"vnc_port":       vm.VNCPort,
			"vnc_host":       "localhost",
			"vnc_password":   vm.VNCPassword,
			"spice_port":     vm.SpicePort,
			"spice_host":     "localhost",
			"spice_password": vm.SpicePassword,
			"websocket_url":  fmt.Sprintf("/api/vms/%d/console/ws?token=%s", id, token),
			"token":          token,
		},
	})
}

// VMConsoleWebSocketHandler handles WebSocket connections for VM console (noVNC proxy)
func VMConsoleWebSocketHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	vm, err := scanVM(db.QueryRow("SELECT "+vmFields+" FROM virtual_machines WHERE id = ?", id))
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	if vm.Status != "running" {
		http.Error(w, "VM is not running", http.StatusBadRequest)
		return
	}

	// Determine VNC port to connect to
	vncPort := vm.VNCPort
	if vncPort == 0 {
		vncPort = 5900 // Default VNC port
	}

	// Connect to VNC server
	vncAddr := fmt.Sprintf("localhost:%d", vncPort)
	vncConn, err := net.DialTimeout("tcp", vncAddr, 5*time.Second)
	if err != nil {
		log.Printf("Failed to connect to VNC server at %s: %v", vncAddr, err)
		http.Error(w, "Failed to connect to VM console", http.StatusInternalServerError)
		return
	}
	defer vncConn.Close()

	// Upgrade HTTP connection to WebSocket
	wsConn, err := vmConsoleUpgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade failed: %v", err)
		return
	}
	defer wsConn.Close()

	log.Printf("VM console WebSocket connected for VM %d (VNC port %d)", id, vncPort)

	// Create proxy between WebSocket and VNC
	proxy := &VNCWebSocketProxy{
		ws:    wsConn,
		vnc:   vncConn,
		vmID:  id,
		done:  make(chan struct{}),
	}

	proxy.Start()

	log.Printf("VM console WebSocket disconnected for VM %d", id)
}

// VNCWebSocketProxy handles bidirectional communication between WebSocket and VNC
type VNCWebSocketProxy struct {
	ws    *websocket.Conn
	vnc   net.Conn
	vmID  int
	done  chan struct{}
	once  sync.Once
}

func (p *VNCWebSocketProxy) Start() {
	var wg sync.WaitGroup
	wg.Add(2)

	// WebSocket -> VNC
	go func() {
		defer wg.Done()
		p.wsToVNC()
	}()

	// VNC -> WebSocket
	go func() {
		defer wg.Done()
		p.vncToWS()
	}()

	wg.Wait()
}

func (p *VNCWebSocketProxy) wsToVNC() {
	for {
		select {
		case <-p.done:
			return
		default:
		}

		messageType, data, err := p.ws.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				log.Printf("WebSocket read error: %v", err)
			}
			p.close()
			return
		}

		if messageType == websocket.BinaryMessage {
			_, err = p.vnc.Write(data)
			if err != nil {
				log.Printf("VNC write error: %v", err)
				p.close()
				return
			}
		}
	}
}

func (p *VNCWebSocketProxy) vncToWS() {
	buf := make([]byte, 32*1024) // 32KB buffer

	for {
		select {
		case <-p.done:
			return
		default:
		}

		// Set read deadline to avoid blocking forever
		p.vnc.SetReadDeadline(time.Now().Add(30 * time.Second))

		n, err := p.vnc.Read(buf)
		if err != nil {
			if err == io.EOF || isTimeout(err) {
				continue
			}
			log.Printf("VNC read error: %v", err)
			p.close()
			return
		}

		if n > 0 {
			err = p.ws.WriteMessage(websocket.BinaryMessage, buf[:n])
			if err != nil {
				log.Printf("WebSocket write error: %v", err)
				p.close()
				return
			}
		}
	}
}

func (p *VNCWebSocketProxy) close() {
	p.once.Do(func() {
		close(p.done)
		p.ws.Close()
		p.vnc.Close()
	})
}

func isTimeout(err error) bool {
	if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
		return true
	}
	return false
}

// SendVMConsoleKeyHandler sends special key combinations to the VM
func SendVMConsoleKeyHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, _ := strconv.Atoi(vars["id"])

	var req struct {
		Key string `json:"key"` // "ctrl-alt-del", "ctrl-alt-f1", etc.
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request", http.StatusBadRequest)
		return
	}

	db, err := NewDatabase()
	if err != nil {
		http.Error(w, "Database error", http.StatusInternalServerError)
		return
	}
	defer db.Close()

	vm, err := scanVM(db.QueryRow("SELECT "+vmFields+" FROM virtual_machines WHERE id = ?", id))
	if err != nil {
		http.Error(w, "VM not found", http.StatusNotFound)
		return
	}

	if vm.Status != "running" {
		http.Error(w, "VM is not running", http.StatusBadRequest)
		return
	}

	// Send key via QMP
	var qmpKey string
	switch req.Key {
	case "ctrl-alt-del":
		qmpKey = `{"execute": "send-key", "arguments": {"keys": [{"type": "qcode", "data": "ctrl"}, {"type": "qcode", "data": "alt"}, {"type": "qcode", "data": "delete"}]}}`
	case "ctrl-alt-f1":
		qmpKey = `{"execute": "send-key", "arguments": {"keys": [{"type": "qcode", "data": "ctrl"}, {"type": "qcode", "data": "alt"}, {"type": "qcode", "data": "f1"}]}}`
	case "ctrl-alt-f2":
		qmpKey = `{"execute": "send-key", "arguments": {"keys": [{"type": "qcode", "data": "ctrl"}, {"type": "qcode", "data": "alt"}, {"type": "qcode", "data": "f2"}]}}`
	case "ctrl-alt-f7":
		qmpKey = `{"execute": "send-key", "arguments": {"keys": [{"type": "qcode", "data": "ctrl"}, {"type": "qcode", "data": "alt"}, {"type": "qcode", "data": "f7"}]}}`
	default:
		http.Error(w, "Unknown key combination", http.StatusBadRequest)
		return
	}

	err = sendQMPCommand(vm.QMPSocketPath, qmpKey)
	if err != nil {
		json.NewEncoder(w).Encode(map[string]any{
			"success": false,
			"error":   err.Error(),
		})
		return
	}

	json.NewEncoder(w).Encode(map[string]bool{"success": true})
}
