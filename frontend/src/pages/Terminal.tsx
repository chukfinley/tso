import { useState, useRef, useEffect, KeyboardEvent } from 'react';
import { terminalAPI } from '../api';
import './Terminal.css';

interface HistoryEntry {
  command: string;
  output: string;
  success: boolean;
  timestamp: Date;
  cwd: string;
}

function Terminal() {
  const [input, setInput] = useState('');
  const [history, setHistory] = useState<HistoryEntry[]>([]);
  const [commandHistory, setCommandHistory] = useState<string[]>([]);
  const [historyIndex, setHistoryIndex] = useState(-1);
  const [isExecuting, setIsExecuting] = useState(false);
  const [cwd, setCwd] = useState('/root');
  const inputRef = useRef<HTMLInputElement>(null);
  const outputRef = useRef<HTMLDivElement>(null);

  // Focus input on mount and keep focus
  useEffect(() => {
    const focusInput = () => {
      if (inputRef.current && document.activeElement !== inputRef.current) {
        inputRef.current.focus();
      }
    };
    focusInput();
    // Keep checking focus
    const interval = setInterval(focusInput, 100);
    return () => clearInterval(interval);
  }, []);

  // Auto-scroll after history changes
  useEffect(() => {
    if (outputRef.current) {
      outputRef.current.scrollTop = outputRef.current.scrollHeight;
    }
  }, [history]);

  // Refocus when clicking anywhere in terminal
  const handleTerminalClick = () => {
    inputRef.current?.focus();
  };

  const executeCommand = async (cmd: string) => {
    const trimmedCmd = cmd.trim();
    if (!trimmedCmd) return;

    setIsExecuting(true);
    setCommandHistory((prev) => [...prev, trimmedCmd]);
    setHistoryIndex(-1);

    const currentCwd = cwd;

    try {
      const result = await terminalAPI.execute(trimmedCmd, currentCwd);

      setHistory((prev) => [
        ...prev,
        {
          command: trimmedCmd,
          output: result.output,
          success: result.success,
          timestamp: new Date(),
          cwd: currentCwd,
        },
      ]);

      // Update cwd from response
      if (result.cwd) {
        setCwd(result.cwd);
      }
    } catch (error) {
      setHistory((prev) => [
        ...prev,
        {
          command: trimmedCmd,
          output: 'Error: Failed to execute command',
          success: false,
          timestamp: new Date(),
          cwd: currentCwd,
        },
      ]);
    }

    setIsExecuting(false);
    setInput('');
  };

  const handleKeyDown = (e: KeyboardEvent<HTMLInputElement>) => {
    if (e.key === 'Enter' && !isExecuting) {
      executeCommand(input);
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      if (commandHistory.length > 0) {
        const newIndex = historyIndex < commandHistory.length - 1 ? historyIndex + 1 : historyIndex;
        setHistoryIndex(newIndex);
        setInput(commandHistory[commandHistory.length - 1 - newIndex] || '');
      }
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      if (historyIndex > 0) {
        const newIndex = historyIndex - 1;
        setHistoryIndex(newIndex);
        setInput(commandHistory[commandHistory.length - 1 - newIndex] || '');
      } else if (historyIndex === 0) {
        setHistoryIndex(-1);
        setInput('');
      }
    } else if (e.key === 'c' && e.ctrlKey) {
      setInput('');
      setHistory((prev) => [
        ...prev,
        {
          command: input + '^C',
          output: '',
          success: true,
          timestamp: new Date(),
          cwd: cwd,
        },
      ]);
    } else if (e.key === 'l' && e.ctrlKey) {
      e.preventDefault();
      setHistory([]);
    }
  };

  const clearTerminal = () => {
    setHistory([]);
    inputRef.current?.focus();
  };

  const formatPath = (path: string) => {
    if (path === '/root') return '~';
    if (path.startsWith('/root/')) return '~' + path.slice(5);
    return path;
  };

  const getHostname = () => {
    try {
      return window.location.hostname || 'tso';
    } catch {
      return 'tso';
    }
  };

  return (
    <div className="terminal-page">
      <div className="terminal-header">
        <h1>Terminal</h1>
        <div className="terminal-actions">
          <button onClick={clearTerminal} className="btn-clear">
            Clear
          </button>
        </div>
      </div>

      <div className="terminal-container" onClick={handleTerminalClick}>
        <div className="terminal-output" ref={outputRef}>
          <div className="terminal-welcome">
            Welcome to TSO Terminal. Type commands below.
          </div>
          {history.map((entry, index) => (
            <div key={index} className="terminal-entry">
              <div className="terminal-command-line">
                <span className="terminal-prompt">
                  <span className="prompt-user">root</span>
                  <span className="prompt-at">@</span>
                  <span className="prompt-host">{getHostname()}</span>
                  <span className="prompt-colon">:</span>
                  <span className="prompt-path">{formatPath(entry.cwd)}</span>
                  <span className="prompt-symbol">#</span>
                </span>
                <span className="terminal-command">{entry.command}</span>
              </div>
              {entry.output && (
                <pre className={`terminal-output-text ${!entry.success ? 'error' : ''}`}>
                  {entry.output}
                </pre>
              )}
            </div>
          ))}
          <div className="terminal-input-line">
            <span className="terminal-prompt">
              <span className="prompt-user">root</span>
              <span className="prompt-at">@</span>
              <span className="prompt-host">{getHostname()}</span>
              <span className="prompt-colon">:</span>
              <span className="prompt-path">{formatPath(cwd)}</span>
              <span className="prompt-symbol">#</span>
            </span>
            <input
              ref={inputRef}
              type="text"
              value={input}
              onChange={(e) => setInput(e.target.value)}
              onKeyDown={handleKeyDown}
              className="terminal-input"
              disabled={isExecuting}
              autoComplete="off"
              autoCorrect="off"
              autoCapitalize="off"
              spellCheck={false}
            />
            {isExecuting && <span className="terminal-spinner"></span>}
          </div>
        </div>
      </div>

      <div className="terminal-hints">
        <span>Enter: Execute</span>
        <span>Up/Down: History</span>
        <span>Ctrl+L: Clear</span>
        <span>Ctrl+C: Cancel</span>
      </div>
    </div>
  );
}

export default Terminal;
