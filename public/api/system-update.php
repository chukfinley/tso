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

// Initialize logger
$logger = Logger::getInstance();
$db = Database::getInstance();

// Log the update action
$logger->logUpdate('initiated', [
    'source' => 'web_ui',
    'user' => $currentUser['username'],
], $currentUser['id']);

$db->execute(
    "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
    [
        $currentUser['id'],
        'system_update',
        'System update initiated from web UI',
        $_SERVER['REMOTE_ADDR']
    ]
);

// Path to update script
$updateScript = '/opt/serveros/update.sh';

// Check if update script exists
if (!file_exists($updateScript)) {
    $logger->error("Update script not found", [
        'script_path' => $updateScript,
        'type' => 'system_update'
    ], $currentUser['id']);
    
    echo json_encode([
        'success' => false,
        'error' => 'Update script not found at ' . $updateScript
    ]);
    exit;
}

// Execute the update script
$command = "sudo bash $updateScript 2>&1";
$output = [];
$returnCode = 0;

$logger->logCommand($command, null, 0, $currentUser['id']);

exec($command, $output, $returnCode);

// Join output lines
$outputText = implode("\n", $output);

// Log the update result
if ($returnCode === 0) {
    $logger->logUpdate('completed', [
        'source' => 'web_ui',
        'user' => $currentUser['username'],
        'output_length' => strlen($outputText)
    ], $currentUser['id']);
    
    $logger->logCommand($command, $outputText, $returnCode, $currentUser['id']);
    
    echo json_encode([
        'success' => true,
        'output' => $outputText,
        'message' => 'System updated successfully'
    ]);
} else {
    $logger->error("System update failed", [
        'source' => 'web_ui',
        'user' => $currentUser['username'],
        'return_code' => $returnCode,
        'output' => substr($outputText, 0, 1000)
    ], $currentUser['id']);
    
    $logger->logCommand($command, $outputText, $returnCode, $currentUser['id']);
    
    echo json_encode([
        'success' => false,
        'error' => 'Update script failed with exit code ' . $returnCode,
        'output' => $outputText
    ]);
}

