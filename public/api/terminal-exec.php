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
    
    // Log terminal access
    $db = Database::getInstance();
    $db->execute(
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
    
    // Security: Prevent some dangerous commands
    $blockedCommands = ['rm -rf /', 'mkfs', 'dd if=/dev/zero'];
    foreach ($blockedCommands as $blocked) {
        if (stripos($command, $blocked) !== false) {
            echo json_encode([
                'success' => false,
                'error' => 'Command blocked for security reasons'
            ]);
            exit;
        }
    }
    
    // Log command execution
    $db = Database::getInstance();
    $db->execute(
        "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
        [
            $currentUser['id'],
            'terminal_command',
            'Executed: ' . substr($command, 0, 200),
            $_SERVER['REMOTE_ADDR']
        ]
    );
    
    // Execute command
    $output = '';
    $returnCode = 0;
    
    // Change to root directory for execution
    $fullCommand = "cd /root && " . $command . " 2>&1";
    exec($fullCommand, $outputLines, $returnCode);
    
    $output = implode("\n", $outputLines);
    
    // Add return code if non-zero
    if ($returnCode !== 0 && empty($output)) {
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

