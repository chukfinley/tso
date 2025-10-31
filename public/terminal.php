<?php
require_once __DIR__ . '/../config/config.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/User.php';
require_once SRC_PATH . '/Auth.php';

$auth = new Auth();
$auth->requireLogin();

$currentUser = $auth->getCurrentUser();
$pageTitle = 'Web Terminal';

// Check if user is admin
if ($currentUser['role'] !== 'admin') {
    header('Location: /dashboard.php');
    exit;
}

// Log terminal access
$db = Database::getInstance();
$db->execute(
    "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
    [
        $currentUser['id'],
        'terminal_access',
        'Accessed web terminal',
        $_SERVER['REMOTE_ADDR']
    ]
);
?>

<?php include VIEWS_PATH . '/layout/header.php'; ?>
<?php include VIEWS_PATH . '/layout/navbar.php'; ?>

<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/xterm@5.3.0/css/xterm.min.css" />

<style>
.terminal-container {
    margin-top: 80px;
    padding: 20px;
    max-width: 100%;
}

.terminal-card {
    background: #1a1a1a;
    border: 1px solid #333;
    border-radius: 8px;
    overflow: hidden;
}

.terminal-header {
    background: #242424;
    padding: 15px 20px;
    border-bottom: 1px solid #333;
    display: flex;
    justify-content: space-between;
    align-items: center;
}

.terminal-title {
    color: #fff;
    font-weight: 600;
    display: flex;
    align-items: center;
    gap: 10px;
}

.terminal-warning {
    background: rgba(255, 152, 0, 0.1);
    border: 1px solid #ff9800;
    color: #ff9800;
    padding: 15px;
    margin-bottom: 20px;
    border-radius: 5px;
}

.terminal-controls {
    display: flex;
    gap: 10px;
}

.terminal-body {
    padding: 0;
    background: #000;
}

#terminal {
    height: 600px;
    padding: 10px;
}

.terminal-btn {
    padding: 8px 15px;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    font-size: 13px;
    font-weight: 600;
    transition: all 0.3s ease;
}

.terminal-btn-clear {
    background: #333;
    color: #fff;
}

.terminal-btn-clear:hover {
    background: #444;
}

.terminal-btn-reconnect {
    background: #ff8c00;
    color: #fff;
}

.terminal-btn-reconnect:hover {
    background: #ff7700;
}

.connection-status {
    display: inline-block;
    padding: 4px 10px;
    border-radius: 12px;
    font-size: 12px;
    font-weight: 600;
}

.connection-status.connected {
    background: rgba(76, 175, 80, 0.2);
    color: #4caf50;
    border: 1px solid #4caf50;
}

.connection-status.disconnected {
    background: rgba(244, 67, 54, 0.2);
    color: #f44336;
    border: 1px solid #f44336;
}
</style>

<div class="terminal-container">
    <div class="terminal-warning">
        <strong>⚠️ Security Warning:</strong> You are accessing a root shell with full system access. 
        All commands are logged and can be audited. Be careful with destructive commands.
    </div>

    <div class="terminal-card">
        <div class="terminal-header">
            <div class="terminal-title">
                <span>Web Terminal</span>
                <span class="connection-status connected" id="connection-status">Ready</span>
            </div>
            <div class="terminal-controls">
                <button class="terminal-btn terminal-btn-clear" onclick="clearTerminal()">Clear</button>
                <button class="terminal-btn terminal-btn-reconnect" onclick="reconnectTerminal()">Reconnect</button>
            </div>
        </div>
        <div class="terminal-body">
            <div id="terminal"></div>
        </div>
    </div>
</div>

<script src="https://cdn.jsdelivr.net/npm/xterm@5.3.0/lib/xterm.min.js"></script>
<script src="https://cdn.jsdelivr.net/npm/xterm-addon-fit@0.8.0/lib/xterm-addon-fit.min.js"></script>

<script>
let term;
let fitAddon;
let sessionId = null;
let commandBuffer = '';

function initTerminal() {
    // Create terminal
    term = new Terminal({
        cursorBlink: true,
        fontSize: 14,
        fontFamily: 'Menlo, Monaco, "Courier New", monospace',
        theme: {
            background: '#000000',
            foreground: '#ffffff',
            cursor: '#ffffff',
            selection: 'rgba(255, 255, 255, 0.3)',
            black: '#000000',
            red: '#e06c75',
            green: '#98c379',
            yellow: '#d19a66',
            blue: '#61afef',
            magenta: '#c678dd',
            cyan: '#56b6c2',
            white: '#abb2bf',
            brightBlack: '#5c6370',
            brightRed: '#e06c75',
            brightGreen: '#98c379',
            brightYellow: '#d19a66',
            brightBlue: '#61afef',
            brightMagenta: '#c678dd',
            brightCyan: '#56b6c2',
            brightWhite: '#ffffff'
        }
    });

    // Add fit addon
    fitAddon = new FitAddon.FitAddon();
    term.loadAddon(fitAddon);

    // Open terminal
    term.open(document.getElementById('terminal'));
    fitAddon.fit();

    // Handle window resize
    window.addEventListener('resize', () => {
        fitAddon.fit();
    });

    // Initialize session
    initSession();

    // Handle terminal input
    term.onData(data => {
        if (data === '\r') {
            // Enter pressed - execute command
            executeCommand(commandBuffer);
            commandBuffer = '';
        } else if (data === '\x7F') {
            // Backspace
            if (commandBuffer.length > 0) {
                commandBuffer = commandBuffer.slice(0, -1);
                term.write('\b \b');
            }
        } else if (data === '\x03') {
            // Ctrl+C
            term.write('^C\r\n$ ');
            commandBuffer = '';
        } else {
            // Regular character
            commandBuffer += data;
            term.write(data);
        }
    });
}

function initSession() {
    fetch('/api/terminal-exec.php', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'init' })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            sessionId = data.session_id;
            term.write('\x1b[32mServerOS Web Terminal\x1b[0m\r\n');
            term.write('Type commands and press Enter. Type "exit" to close.\r\n\r\n');
            term.write('$ ');
        } else {
            term.write('\x1b[31mFailed to initialize terminal session\x1b[0m\r\n');
        }
    })
    .catch(error => {
        term.write('\x1b[31mError: ' + error.message + '\x1b[0m\r\n');
    });
}

function executeCommand(command) {
    if (!command.trim()) {
        term.write('\r\n$ ');
        return;
    }

    term.write('\r\n');

    if (command.toLowerCase() === 'exit') {
        term.write('\x1b[33mBye!\x1b[0m\r\n');
        commandBuffer = '';
        return;
    }

    if (command.toLowerCase() === 'clear') {
        term.clear();
        term.write('$ ');
        return;
    }

    fetch('/api/terminal-exec.php', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            action: 'exec',
            command: command,
            session_id: sessionId
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            if (data.output) {
                term.write(data.output.replace(/\n/g, '\r\n'));
            }
        } else {
            term.write('\x1b[31mError: ' + (data.error || 'Command failed') + '\x1b[0m\r\n');
        }
        term.write('$ ');
    })
    .catch(error => {
        term.write('\x1b[31mError: ' + error.message + '\x1b[0m\r\n');
        term.write('$ ');
    });
}

function clearTerminal() {
    term.clear();
    term.write('$ ');
}

function reconnectTerminal() {
    term.clear();
    term.write('\x1b[33mReconnecting...\x1b[0m\r\n');
    commandBuffer = '';
    initSession();
}

// Initialize on page load
document.addEventListener('DOMContentLoaded', initTerminal);
</script>

<?php include VIEWS_PATH . '/layout/footer.php'; ?>

