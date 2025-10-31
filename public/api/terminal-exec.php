<?php
require_once __DIR__ . '/../../config/config.php';
require_once SRC_PATH . '/Auth.php';
require_once SRC_PATH . '/Database.php';

header('Content-Type: application/json');

$auth = new Auth();
if (!$auth->isLoggedIn()) {
    http_response_code(401);
    echo json_encode(['success' => false, 'error' => 'Unauthorized']);
    exit;
}

$currentUser = $auth->getCurrentUser();
if ($currentUser['role'] !== 'admin') {
    http_response_code(403);
    echo json_encode(['success' => false, 'error' => 'Admin access required']);
    exit;
}

// Get JSON input
$input = json_decode(file_get_contents('php://input'), true);
$action = $input['action'] ?? '';

if ($action === 'init') {
    // Initialize terminal session
    $sessionId = uniqid('term_', true);

    // Store session in PHP session for validation
    $_SESSION['terminal_session_id'] = $sessionId;
    $_SESSION['terminal_session_time'] = time();

    // Log terminal access
    $db = Database::getInstance();
    $db->query(
        "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
        [
            $currentUser['id'],
            'terminal_session_start',
            'Started terminal session: ' . $sessionId,
            $_SERVER['REMOTE_ADDR']
        ]
    );

    echo json_encode([
        'success' => true,
        'session_id' => $sessionId
    ]);
    exit;
}

if ($action === 'exec') {
    $command = $input['command'] ?? '';
    $sessionId = $input['session_id'] ?? '';

    if (empty($command)) {
        echo json_encode(['success' => false, 'error' => 'No command provided']);
        exit;
    }

    // Validate session ID
    if (empty($sessionId) || !isset($_SESSION['terminal_session_id']) || $_SESSION['terminal_session_id'] !== $sessionId) {
        echo json_encode(['success' => false, 'error' => 'Invalid session']);
        exit;
    }

    // Check session timeout (30 minutes)
    if (isset($_SESSION['terminal_session_time']) && (time() - $_SESSION['terminal_session_time']) > 1800) {
        echo json_encode(['success' => false, 'error' => 'Session expired. Please reconnect.']);
        exit;
    }

    // Update session time
    $_SESSION['terminal_session_time'] = time();

    // Trim command
    $command = trim($command);

    // Security: Enhanced command blocking
    $blockedPatterns = [
        '/rm\s+(-[rf]+\s+)?\//i',           // rm -rf / or variations
        '/mkfs/i',                           // mkfs commands
        '/dd\s+if=\/dev\/(zero|random)/i',  // dd with /dev/zero or /dev/random
        '/:\(\)\{\s*:\|:&\s*\};:/i',        // Fork bomb
        '/>\s*\/dev\/sd[a-z]/i',            // Writing to disk devices
        '/(systemctl|service)\s+(stop|disable|mask)\s+sshd/i',  // Disabling SSH
        '/iptables\s+-[A-Z]\s+INPUT\s+-j\s+DROP/i',  // Blocking all incoming
        '/shutdown/i',                       // Shutdown commands
        '/reboot/i',                         // Reboot commands
        '/halt/i',                           // Halt commands
        '/init\s+[06]/i',                    // Init runlevel changes
        '/>\s*\/etc\/(passwd|shadow|sudoers)/i',  // Overwriting critical files
    ];

    foreach ($blockedPatterns as $pattern) {
        if (preg_match($pattern, $command)) {
            // Log blocked command attempt
            require_once SRC_PATH . '/Logger.php';
            $logger = Logger::getInstance();
            $logger->warning("Blocked command attempt", [
                'type' => 'command_blocked',
                'command' => substr($command, 0, 500),
                'user' => $currentUser['username']
            ], $currentUser['id']);
            
            $db = Database::getInstance();
            $db->query(
                "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                [
                    $currentUser['id'],
                    'terminal_blocked',
                    'BLOCKED command: ' . substr($command, 0, 200),
                    $_SERVER['REMOTE_ADDR']
                ]
            );

            echo json_encode([
                'success' => false,
                'error' => 'Command blocked for security reasons. This action has been logged.'
            ]);
            exit;
        }
    }

    // Initialize logger
    require_once SRC_PATH . '/Logger.php';
    $logger = Logger::getInstance();
    
    // Log command execution
    $db = Database::getInstance();
    $db->query(
        "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
        [
            $currentUser['id'],
            'terminal_command',
            'Executed: ' . substr($command, 0, 200),
            $_SERVER['REMOTE_ADDR']
        ]
    );

    // Execute command with timeout and proper escaping
    $output = '';
    $returnCode = 0;

    // Use sudo to execute as root, with a timeout of 30 seconds
    $fullCommand = sprintf(
        'timeout 30 bash -c %s 2>&1',
        escapeshellarg('cd /opt/serveros && ' . $command)
    );

    exec($fullCommand, $outputLines, $returnCode);

    $output = implode("\n", $outputLines);
    
    // Log command execution with Logger
    $logger->logCommand($fullCommand, $output, $returnCode, $currentUser['id']);

    // Handle timeout
    if ($returnCode === 124) {
        $output = "Command timed out after 30 seconds";
    } elseif ($returnCode !== 0 && empty($output)) {
        $output = "Command exited with code: $returnCode";
    }

    echo json_encode([
        'success' => true,
        'output' => $output,
        'return_code' => $returnCode
    ]);
    exit;
}

echo json_encode(['success' => false, 'error' => 'Invalid action']);

