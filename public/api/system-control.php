<?php
require_once __DIR__ . '/../../config/config.php';
require_once SRC_PATH . '/Auth.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/Logger.php';

header('Content-Type: application/json');

$auth = new Auth();
if (!$auth->isLoggedIn()) {
    http_response_code(401);
    echo json_encode(['success' => false, 'error' => 'Unauthorized']);
    exit;
}

// Check if user is admin
$currentUser = $auth->getCurrentUser();
if ($currentUser['role'] !== 'admin') {
    http_response_code(403);
    echo json_encode(['success' => false, 'error' => 'Admin access required']);
    exit;
}

// Only allow POST requests
if ($_SERVER['REQUEST_METHOD'] !== 'POST') {
    http_response_code(405);
    echo json_encode(['success' => false, 'error' => 'Method not allowed']);
    exit;
}

// Get input data
$input = json_decode(file_get_contents('php://input'), true);
$action = $input['action'] ?? '';

// Initialize logger and database
$logger = Logger::getInstance();
$db = Database::getInstance();

// Validate action
$allowedActions = ['reboot', 'shutdown', 'sleep', 'hibernate', 'suspend'];
if (!in_array($action, $allowedActions)) {
    http_response_code(400);
    echo json_encode(['success' => false, 'error' => 'Invalid action']);
    exit;
}

// Prepare commands and descriptions
// Try systemctl first (modern Linux), fallback to traditional commands
$commands = [
    'reboot' => 'sudo systemctl reboot',
    'shutdown' => 'sudo systemctl poweroff',
    'sleep' => 'sudo systemctl suspend',
    'hibernate' => 'sudo systemctl hibernate',
    'suspend' => 'sudo systemctl suspend'
];

$descriptions = [
    'reboot' => 'System reboot initiated',
    'shutdown' => 'System shutdown initiated',
    'sleep' => 'System suspend initiated',
    'hibernate' => 'System hibernate initiated',
    'suspend' => 'System suspend initiated'
];

$actionNames = [
    'reboot' => 'Reboot',
    'shutdown' => 'Shutdown',
    'sleep' => 'Suspend',
    'hibernate' => 'Hibernate',
    'suspend' => 'Suspend'
];

$command = $commands[$action];
$description = $descriptions[$action];
$actionName = $actionNames[$action];

// Log the action before execution
$logger->warning("System control action initiated", [
    'type' => 'system_control',
    'action' => $action,
    'user' => $currentUser['username'],
    'command' => $command
], $currentUser['id']);

$db->execute(
    "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
    [
        $currentUser['id'],
        'system_control',
        $actionName . ' initiated from web UI by ' . $currentUser['username'],
        $_SERVER['REMOTE_ADDR']
    ]
);

// Execute the command
// Note: For shutdown/reboot, the command will execute and we won't get a response
// For suspend/hibernate, we may also not get a response depending on system state
$output = [];
$returnCode = 0;

// Log the command
$logger->logCommand($command, null, 0, $currentUser['id']);

// Execute command (non-blocking for immediate response)
// We use nohup to detach the process
$fullCommand = sprintf(
    'nohup %s > /dev/null 2>&1 &',
    escapeshellcmd($command)
);

exec($fullCommand, $output, $returnCode);

// Return success response immediately
// The actual system action will happen in the background
echo json_encode([
    'success' => true,
    'message' => $actionName . ' command executed successfully',
    'action' => $action,
    'note' => 'The system will ' . strtolower($actionName) . ' shortly. This page may become unavailable.'
]);

