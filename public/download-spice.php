<?php
require_once __DIR__ . '/../config/config.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/Auth.php';
require_once SRC_PATH . '/VM.php';

$auth = new Auth();
$auth->requireLogin();

$vmId = $_GET['vm_id'] ?? 0;

if (empty($vmId)) {
    die('VM ID required');
}

$vmManager = new VM();
$vm = $vmManager->getById($vmId);

if (!$vm) {
    die('VM not found');
}

try {
    $spiceContent = $vmManager->generateSpiceFile($vmId);

    // Log activity
    $db = Database::getInstance();
    $db->query(
        "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
        [$_SESSION['user_id'], 'vm_spice_download', "Downloaded SPICE file for VM: {$vm['name']}", $_SERVER['REMOTE_ADDR']]
    );

    // Send file
    header('Content-Type: application/x-virt-viewer');
    header('Content-Disposition: attachment; filename="' . $vm['name'] . '.vv"');
    header('Content-Length: ' . strlen($spiceContent));

    echo $spiceContent;
} catch (Exception $e) {
    die('Error: ' . htmlspecialchars($e->getMessage()));
}
