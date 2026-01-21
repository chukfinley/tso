import { useEffect, useRef, useState, useCallback } from 'react';
import { Terminal as XTerm } from '@xterm/xterm';
import { FitAddon } from '@xterm/addon-fit';
import { WebLinksAddon } from '@xterm/addon-web-links';
import { Unicode11Addon } from '@xterm/addon-unicode11';
import '@xterm/xterm/css/xterm.css';
import './Terminal.css';

type ConnectionStatus = 'connecting' | 'connected' | 'disconnected' | 'error';

function Terminal() {
  const terminalRef = useRef<HTMLDivElement>(null);
  const xtermRef = useRef<XTerm | null>(null);
  const fitAddonRef = useRef<FitAddon | null>(null);
  const wsRef = useRef<WebSocket | null>(null);
  const [status, setStatus] = useState<ConnectionStatus>('disconnected');
  const reconnectTimeoutRef = useRef<number | null>(null);
  const pingIntervalRef = useRef<number | null>(null);

  // Get WebSocket URL based on current location
  const getWsUrl = useCallback(() => {
    const protocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    // API is on port 8080
    const apiHost = window.location.hostname;
    const apiPort = '8080';
    return `${protocol}//${apiHost}:${apiPort}/api/terminal/ws`;
  }, []);

  // Send resize event to backend
  const sendResize = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN && xtermRef.current) {
      const dimensions = fitAddonRef.current?.proposeDimensions();
      if (dimensions) {
        wsRef.current.send(JSON.stringify({
          type: 'resize',
          data: { cols: dimensions.cols, rows: dimensions.rows }
        }));
      }
    }
  }, []);

  // Connect to WebSocket
  const connect = useCallback(() => {
    if (wsRef.current?.readyState === WebSocket.OPEN) {
      return;
    }

    setStatus('connecting');

    const ws = new WebSocket(getWsUrl());
    wsRef.current = ws;

    ws.binaryType = 'arraybuffer';

    ws.onopen = () => {
      setStatus('connected');
      xtermRef.current?.writeln('\x1b[32mConnected to terminal.\x1b[0m\r\n');

      // Send initial resize
      setTimeout(() => {
        fitAddonRef.current?.fit();
        sendResize();
      }, 100);

      // Start ping interval to keep connection alive
      if (pingIntervalRef.current) {
        clearInterval(pingIntervalRef.current);
      }
      pingIntervalRef.current = window.setInterval(() => {
        if (ws.readyState === WebSocket.OPEN) {
          ws.send(JSON.stringify({ type: 'ping' }));
        }
      }, 30000);
    };

    ws.onmessage = (event) => {
      if (xtermRef.current) {
        if (event.data instanceof ArrayBuffer) {
          // Binary data from PTY
          const text = new TextDecoder().decode(event.data);
          xtermRef.current.write(text);
        } else if (typeof event.data === 'string') {
          // JSON message
          try {
            const msg = JSON.parse(event.data);
            if (msg.type === 'error') {
              xtermRef.current.writeln(`\x1b[31mError: ${msg.data}\x1b[0m`);
            }
            // Ignore pong messages
          } catch {
            // Not JSON, write as text
            xtermRef.current.write(event.data);
          }
        }
      }
    };

    ws.onerror = () => {
      setStatus('error');
      xtermRef.current?.writeln('\x1b[31mConnection error.\x1b[0m');
    };

    ws.onclose = () => {
      setStatus('disconnected');
      if (pingIntervalRef.current) {
        clearInterval(pingIntervalRef.current);
        pingIntervalRef.current = null;
      }
      xtermRef.current?.writeln('\r\n\x1b[33mDisconnected from terminal.\x1b[0m');

      // Auto-reconnect after 3 seconds
      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      reconnectTimeoutRef.current = window.setTimeout(() => {
        xtermRef.current?.writeln('\x1b[33mReconnecting...\x1b[0m');
        connect();
      }, 3000);
    };
  }, [getWsUrl, sendResize]);

  // Initialize terminal
  useEffect(() => {
    if (!terminalRef.current || xtermRef.current) return;

    const term = new XTerm({
      cursorBlink: true,
      cursorStyle: 'block',
      fontFamily: "'JetBrains Mono', 'Fira Code', 'Consolas', 'Monaco', monospace",
      fontSize: 14,
      lineHeight: 1.2,
      theme: {
        background: '#1a1a1a',
        foreground: '#e0e0e0',
        cursor: '#4ec9b0',
        cursorAccent: '#1a1a1a',
        selectionBackground: '#3a3a3a',
        selectionForeground: '#ffffff',
        black: '#1a1a1a',
        red: '#f44747',
        green: '#4ec9b0',
        yellow: '#dcdcaa',
        blue: '#569cd6',
        magenta: '#c586c0',
        cyan: '#4fc1ff',
        white: '#e0e0e0',
        brightBlack: '#808080',
        brightRed: '#f44747',
        brightGreen: '#4ec9b0',
        brightYellow: '#dcdcaa',
        brightBlue: '#569cd6',
        brightMagenta: '#c586c0',
        brightCyan: '#4fc1ff',
        brightWhite: '#ffffff',
      },
      allowProposedApi: true,
      scrollback: 10000,
      convertEol: true,
    });

    // Load addons
    const fitAddon = new FitAddon();
    const webLinksAddon = new WebLinksAddon();
    const unicode11Addon = new Unicode11Addon();

    term.loadAddon(fitAddon);
    term.loadAddon(webLinksAddon);
    term.loadAddon(unicode11Addon);
    term.unicode.activeVersion = '11';

    term.open(terminalRef.current);
    fitAddon.fit();

    xtermRef.current = term;
    fitAddonRef.current = fitAddon;

    // Handle terminal input
    term.onData((data) => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type: 'input', data }));
      }
    });

    // Handle binary input (for special keys)
    term.onBinary((data) => {
      if (wsRef.current?.readyState === WebSocket.OPEN) {
        wsRef.current.send(JSON.stringify({ type: 'input', data }));
      }
    });

    // Welcome message
    term.writeln('TSO Terminal - Full PTY Support');
    term.writeln('Supports interactive commands: vim, htop, btop, etc.');
    term.writeln('');

    // Connect to WebSocket
    connect();

    // Handle resize
    const handleResize = () => {
      if (fitAddonRef.current) {
        fitAddonRef.current.fit();
        sendResize();
      }
    };

    window.addEventListener('resize', handleResize);

    // Cleanup
    return () => {
      window.removeEventListener('resize', handleResize);

      if (reconnectTimeoutRef.current) {
        clearTimeout(reconnectTimeoutRef.current);
      }
      if (pingIntervalRef.current) {
        clearInterval(pingIntervalRef.current);
      }
      if (wsRef.current) {
        wsRef.current.close();
      }
      if (xtermRef.current) {
        xtermRef.current.dispose();
        xtermRef.current = null;
      }
    };
  }, [connect, sendResize]);

  // Handle container resize with ResizeObserver
  useEffect(() => {
    if (!terminalRef.current) return;

    const resizeObserver = new ResizeObserver(() => {
      if (fitAddonRef.current && xtermRef.current) {
        fitAddonRef.current.fit();
        sendResize();
      }
    });

    resizeObserver.observe(terminalRef.current);

    return () => {
      resizeObserver.disconnect();
    };
  }, [sendResize]);

  const handleReconnect = () => {
    if (reconnectTimeoutRef.current) {
      clearTimeout(reconnectTimeoutRef.current);
      reconnectTimeoutRef.current = null;
    }
    if (wsRef.current) {
      wsRef.current.close();
    }
    xtermRef.current?.clear();
    xtermRef.current?.writeln('\x1b[33mReconnecting...\x1b[0m');
    connect();
  };

  const handleClear = () => {
    xtermRef.current?.clear();
  };

  const getStatusColor = () => {
    switch (status) {
      case 'connected': return '#4ec9b0';
      case 'connecting': return '#dcdcaa';
      case 'disconnected': return '#808080';
      case 'error': return '#f44747';
      default: return '#808080';
    }
  };

  const getStatusText = () => {
    switch (status) {
      case 'connected': return 'Connected';
      case 'connecting': return 'Connecting...';
      case 'disconnected': return 'Disconnected';
      case 'error': return 'Error';
      default: return 'Unknown';
    }
  };

  return (
    <div className="terminal-page">
      <div className="terminal-header">
        <h1>Terminal</h1>
        <div className="terminal-status">
          <span
            className="status-indicator"
            style={{ backgroundColor: getStatusColor() }}
          />
          <span className="status-text">{getStatusText()}</span>
        </div>
        <div className="terminal-actions">
          <button onClick={handleClear} className="btn-terminal">
            Clear
          </button>
          <button
            onClick={handleReconnect}
            className="btn-terminal"
            disabled={status === 'connecting'}
          >
            Reconnect
          </button>
        </div>
      </div>

      <div className="terminal-container">
        <div ref={terminalRef} className="terminal-xterm" />
      </div>

      <div className="terminal-hints">
        <span>Full PTY support</span>
        <span>Interactive commands work (vim, htop, etc.)</span>
        <span>Tab for autocomplete</span>
      </div>
    </div>
  );
}

export default Terminal;
