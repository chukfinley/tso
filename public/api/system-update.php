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

// Log the update action
$db = Database::getInstance();
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

exec($command, $output, $returnCode);

// Join output lines
$outputText = implode("\n", $output);

if ($returnCode === 0) {
    echo json_encode([
        'success' => true,
        'output' => $outputText,
        'message' => 'System updated successfully'
    ]);
} else {
    echo json_encode([
        'success' => false,
        'error' => 'Update script failed with exit code ' . $returnCode,
        'output' => $outputText
    ]);
}

