<?php
require_once __DIR__ . '/../../config/config.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/Auth.php';
require_once SRC_PATH . '/VM.php';

header('Content-Type: application/json');

$auth = new Auth();
if (!$auth->isLoggedIn()) {
    http_response_code(401);
    echo json_encode(['success' => false, 'error' => 'Unauthorized']);
    exit;
}

$vm = new VM();
$db = Database::getInstance();

// Get JSON input
$input = json_decode(file_get_contents('php://input'), true);
$action = $input['action'] ?? $_GET['action'] ?? '';
$vmId = $input['vm_id'] ?? $_GET['vm_id'] ?? 0;

try {
    switch ($action) {
        case 'start':
            $vm->start($vmId);

            // Log activity
            $vmData = $vm->getById($vmId);
            $db->query(
                "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                [$_SESSION['user_id'], 'vm_start', "Started VM: {$vmData['name']}", $_SERVER['REMOTE_ADDR']]
            );

            echo json_encode(['success' => true, 'message' => 'VM started successfully']);
            break;

        case 'stop':
            $force = $input['force'] ?? false;
            $vm->stop($vmId, $force);

            // Log activity
            $vmData = $vm->getById($vmId);
            $db->query(
                "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                [$_SESSION['user_id'], 'vm_stop', "Stopped VM: {$vmData['name']}" . ($force ? ' (forced)' : ''), $_SERVER['REMOTE_ADDR']]
            );

            echo json_encode(['success' => true, 'message' => 'VM stopped successfully']);
            break;

        case 'restart':
            $vm->restart($vmId);

            // Log activity
            $vmData = $vm->getById($vmId);
            $db->query(
                "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                [$_SESSION['user_id'], 'vm_restart', "Restarted VM: {$vmData['name']}", $_SERVER['REMOTE_ADDR']]
            );

            echo json_encode(['success' => true, 'message' => 'VM restarted successfully']);
            break;

        case 'status':
            $status = $vm->getStatus($vmId);
            echo json_encode(['success' => true, 'status' => $status]);
            break;

        case 'logs':
            $lines = $input['lines'] ?? 100;
            $logs = $vm->getLogs($vmId, $lines);
            echo json_encode(['success' => true, 'logs' => $logs]);
            break;

        case 'list_isos':
            $isos = $vm->listIsos();
            echo json_encode(['success' => true, 'isos' => $isos]);
            break;

        case 'list_disks':
            $disks = $vm->listPhysicalDisks();
            echo json_encode(['success' => true, 'disks' => $disks]);
            break;

        case 'list_bridges':
            $bridges = $vm->listNetworkBridges();
            echo json_encode(['success' => true, 'bridges' => $bridges]);
            break;

        case 'create_backup':
            $notes = $input['notes'] ?? '';
            $result = $vm->createBackup($vmId, $notes);

            // Log activity
            $vmData = $vm->getById($vmId);
            $db->query(
                "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                [$_SESSION['user_id'], 'vm_backup_create', "Created backup for VM: {$vmData['name']}", $_SERVER['REMOTE_ADDR']]
            );

            echo json_encode(['success' => true, 'backup_id' => $result['backup_id'], 'message' => 'Backup started']);
            break;

        case 'list_backups':
            $backups = $vm->getBackups($vmId);
            echo json_encode(['success' => true, 'backups' => $backups]);
            break;

        case 'list_all_backups':
            $backups = $vm->getAllBackups();
            echo json_encode(['success' => true, 'backups' => $backups]);
            break;

        case 'check_backup_status':
            $backupId = $input['backup_id'] ?? 0;
            $status = $vm->checkBackupStatus($backupId);
            echo json_encode(['success' => true, 'status' => $status]);
            break;

        case 'restore_backup':
            $backupId = $input['backup_id'] ?? 0;
            $vm->restoreBackup($backupId);

            // Log activity
            $backup = $vm->getBackupById($backupId);
            $db->query(
                "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                [$_SESSION['user_id'], 'vm_backup_restore', "Restored VM from backup: {$backup['backup_name']}", $_SERVER['REMOTE_ADDR']]
            );

            echo json_encode(['success' => true, 'message' => 'Backup restored successfully']);
            break;

        case 'delete_backup':
            $backupId = $input['backup_id'] ?? 0;
            $backup = $vm->getBackupById($backupId);
            $vm->deleteBackup($backupId);

            // Log activity
            $db->query(
                "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                [$_SESSION['user_id'], 'vm_backup_delete', "Deleted backup: {$backup['backup_name']}", $_SERVER['REMOTE_ADDR']]
            );

            echo json_encode(['success' => true, 'message' => 'Backup deleted successfully']);
            break;

        default:
            echo json_encode(['success' => false, 'error' => 'Invalid action']);
    }
} catch (Exception $e) {
    echo json_encode(['success' => false, 'error' => $e->getMessage()]);
}
