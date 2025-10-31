<?php
// Suppress output buffering and ensure clean JSON output
ob_start();
error_reporting(E_ALL);
ini_set('display_errors', 0); // Suppress error display for API responses
ini_set('log_errors', 1);

require_once __DIR__ . '/../../config/config.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/Auth.php';
require_once SRC_PATH . '/VM.php';

// Clear any output that might have been generated
ob_clean();
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
        case 'upload_iso':
            // Handle file upload
            if (!isset($_FILES['iso_file']) || $_FILES['iso_file']['error'] !== UPLOAD_ERR_OK) {
                throw new Exception('No file uploaded or upload error occurred');
            }

            $uploadedFile = $_FILES['iso_file'];
            $fileName = $uploadedFile['name'];
            $fileTmpPath = $uploadedFile['tmp_name'];
            $fileSize = $uploadedFile['size'];

            // Validate file extension
            $fileExtension = strtolower(pathinfo($fileName, PATHINFO_EXTENSION));
            if ($fileExtension !== 'iso') {
                throw new Exception('Invalid file type. Only .iso files are allowed.');
            }

            // Validate file size (max 20GB)
            $maxSize = 20 * 1024 * 1024 * 1024; // 20GB in bytes
            if ($fileSize > $maxSize) {
                throw new Exception('File size too large. Maximum size is 20GB.');
            }

            // Sanitize filename (remove special characters, keep alphanumeric, dash, underscore, dot)
            $safeFileName = preg_replace('/[^a-zA-Z0-9._-]/', '_', basename($fileName));
            $isoDir = '/opt/serveros/storage/isos';

            // Ensure ISO directory exists
            if (!is_dir($isoDir)) {
                if (!mkdir($isoDir, 0775, true)) {
                    throw new Exception('Failed to create ISO directory');
                }
                // Set ownership to www-data if possible (suppress errors if not permitted)
                if (function_exists('posix_getpwuid') && function_exists('posix_geteuid')) {
                    $wwwDataUid = posix_getpwnam('www-data')['uid'] ?? null;
                    if ($wwwDataUid !== null) {
                        @chown($isoDir, $wwwDataUid);
                    }
                }
            }
            
            // Check if directory is writable
            if (!is_writable($isoDir)) {
                throw new Exception('ISO directory is not writable. Please check permissions on: ' . $isoDir);
            }

            // Check if file already exists
            $destinationPath = $isoDir . '/' . $safeFileName;
            if (file_exists($destinationPath)) {
                // Add timestamp to make filename unique
                $baseName = pathinfo($safeFileName, PATHINFO_FILENAME);
                $extension = pathinfo($safeFileName, PATHINFO_EXTENSION);
                $safeFileName = $baseName . '_' . time() . '.' . $extension;
                $destinationPath = $isoDir . '/' . $safeFileName;
            }

            // Move uploaded file to ISO directory
            if (!move_uploaded_file($fileTmpPath, $destinationPath)) {
                // Get more detailed error information
                $error = error_get_last();
                $errorMsg = 'Failed to save uploaded file';
                if ($error && $error['message']) {
                    $errorMsg .= ': ' . $error['message'];
                }
                if (!is_writable(dirname($destinationPath))) {
                    $errorMsg .= ' - Directory is not writable';
                }
                throw new Exception($errorMsg);
            }

            // Set proper permissions (readable by all, writable by owner)
            chmod($destinationPath, 0644);
            
            // Try to set ownership to www-data if possible (suppress errors if not permitted)
            if (function_exists('posix_getpwuid') && function_exists('posix_geteuid')) {
                $wwwDataUid = posix_getpwnam('www-data')['uid'] ?? null;
                if ($wwwDataUid !== null) {
                    @chown($destinationPath, $wwwDataUid);
                }
            }

            // Log activity
            $db->query(
                "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                [$_SESSION['user_id'], 'iso_upload', "Uploaded ISO file: {$safeFileName} (" . number_format($fileSize / 1024 / 1024, 2) . " MB)", $_SERVER['REMOTE_ADDR']]
            );

            // Format file size for display
            $formatBytes = function($bytes, $precision = 2) {
                $units = ['B', 'KB', 'MB', 'GB', 'TB'];
                $bytes = max($bytes, 0);
                $pow = floor(($bytes ? log($bytes) : 0) / log(1024));
                $pow = min($pow, count($units) - 1);
                $bytes /= pow(1024, $pow);
                return round($bytes, $precision) . ' ' . $units[$pow];
            };

            // Clear output buffer to ensure clean JSON
            ob_clean();
            echo json_encode([
                'success' => true,
                'message' => 'ISO file uploaded successfully',
                'filename' => $safeFileName,
                'path' => $destinationPath,
                'size' => $fileSize,
                'size_formatted' => $formatBytes($fileSize)
            ]);
            exit;
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
            ob_clean();
            echo json_encode(['success' => false, 'error' => 'Invalid action']);
            exit;
    }
} catch (Exception $e) {
    ob_clean();
    http_response_code(500);
    echo json_encode(['success' => false, 'error' => $e->getMessage()]);
    exit;
} catch (Error $e) {
    ob_clean();
    http_response_code(500);
    echo json_encode(['success' => false, 'error' => 'Server error: ' . $e->getMessage()]);
    exit;
}
