<?php
/**
 * Share Management API
 */
// Start output buffering to catch any accidental output
ob_start();

header('Content-Type: application/json');

try {
    require_once __DIR__ . '/../../config/config.php';
    require_once SRC_PATH . '/Database.php';
    require_once SRC_PATH . '/User.php';
    require_once SRC_PATH . '/Auth.php';
    require_once SRC_PATH . '/Share.php';
    
    // Clear any output that might have been generated during requires
    ob_clean();

    $auth = new Auth();

    // Check if user is logged in
    if (!$auth->check()) {
        http_response_code(401);
        echo json_encode(['success' => false, 'error' => 'Unauthorized']);
        exit;
    }

    $share = new Share();
    $action = $_GET['action'] ?? $_POST['action'] ?? '';

    switch ($action) {
        // ============ Share Operations ============
        
        case 'list':
            $shares = $share->getAll();
            echo json_encode(['success' => true, 'shares' => $shares ?: []]);
            break;

        case 'get':
            $id = $_GET['id'] ?? null;
            if (!$id) {
                throw new Exception("Share ID is required");
            }
            
            $shareData = $share->getById($id);
            if (!$shareData) {
                throw new Exception("Share not found");
            }
            
            echo json_encode(['success' => true, 'share' => $shareData]);
            break;

        case 'create':
            $data = [
                'share_name' => $_POST['share_name'] ?? '',
                'display_name' => $_POST['display_name'] ?? '',
                'path' => $_POST['path'] ?? '',
                'comment' => $_POST['comment'] ?? '',
                'browseable' => isset($_POST['browseable']) ? (bool)$_POST['browseable'] : true,
                'readonly' => isset($_POST['readonly']) ? (bool)$_POST['readonly'] : false,
                'guest_ok' => isset($_POST['guest_ok']) ? (bool)$_POST['guest_ok'] : false,
                'case_sensitive' => $_POST['case_sensitive'] ?? 'auto',
                'preserve_case' => isset($_POST['preserve_case']) ? (bool)$_POST['preserve_case'] : true,
                'short_preserve_case' => isset($_POST['short_preserve_case']) ? (bool)$_POST['short_preserve_case'] : true,
                'create_mask' => $_POST['create_mask'] ?? '0664',
                'directory_mask' => $_POST['directory_mask'] ?? '0775',
                'force_user' => $_POST['force_user'] ?? null,
                'force_group' => $_POST['force_group'] ?? null,
            ];

            $shareId = $share->create($data);
            echo json_encode(['success' => true, 'share_id' => $shareId, 'message' => 'Share created successfully']);
            break;

        case 'update':
            $id = $_POST['id'] ?? null;
            if (!$id) {
                throw new Exception("Share ID is required");
            }

            $data = [];
            $fields = [
                'display_name', 'path', 'comment', 'browseable', 'readonly', 'guest_ok',
                'case_sensitive', 'preserve_case', 'short_preserve_case',
                'create_mask', 'directory_mask', 'force_user', 'force_group', 'is_active'
            ];

            foreach ($fields as $field) {
                if (isset($_POST[$field])) {
                    if (in_array($field, ['browseable', 'readonly', 'guest_ok', 'preserve_case', 'short_preserve_case', 'is_active'])) {
                        $data[$field] = (bool)$_POST[$field];
                    } else {
                        $data[$field] = $_POST[$field];
                    }
                }
            }

            $share->update($id, $data);
            echo json_encode(['success' => true, 'message' => 'Share updated successfully']);
            break;

        case 'delete':
            $id = $_POST['id'] ?? null;
            $deleteDirectory = isset($_POST['delete_directory']) ? (bool)$_POST['delete_directory'] : false;
            
            if (!$id) {
                throw new Exception("Share ID is required");
            }

            $share->delete($id, $deleteDirectory);
            echo json_encode(['success' => true, 'message' => 'Share deleted successfully']);
            break;

        case 'toggle':
            $id = $_POST['id'] ?? null;
            if (!$id) {
                throw new Exception("Share ID is required");
            }

            $newStatus = $share->toggleActive($id);
            echo json_encode([
                'success' => true, 
                'is_active' => $newStatus,
                'message' => $newStatus ? 'Share enabled' : 'Share disabled'
            ]);
            break;

        // ============ User Operations ============

        case 'list_users':
            $users = $share->getAllUsers();
            echo json_encode(['success' => true, 'users' => $users]);
            break;

        case 'get_user':
            $id = $_GET['id'] ?? null;
            if (!$id) {
                throw new Exception("User ID is required");
            }
            
            $userData = $share->getUserById($id);
            if (!$userData) {
                throw new Exception("User not found");
            }
            
            echo json_encode(['success' => true, 'user' => $userData]);
            break;

        case 'create_user':
            $username = $_POST['username'] ?? '';
            $password = $_POST['password'] ?? '';
            $fullName = $_POST['full_name'] ?? null;

            if (empty($username) || empty($password)) {
                throw new Exception("Username and password are required");
            }

            $userId = $share->createUser($username, $password, $fullName);
            echo json_encode(['success' => true, 'user_id' => $userId, 'message' => 'User created successfully']);
            break;

        case 'update_user_password':
            $id = $_POST['id'] ?? null;
            $password = $_POST['password'] ?? '';

            if (!$id || empty($password)) {
                throw new Exception("User ID and password are required");
            }

            $share->updateUserPassword($id, $password);
            echo json_encode(['success' => true, 'message' => 'Password updated successfully']);
            break;

        case 'delete_user':
            $id = $_POST['id'] ?? null;
            if (!$id) {
                throw new Exception("User ID is required");
            }

            $share->deleteUser($id);
            echo json_encode(['success' => true, 'message' => 'User deleted successfully']);
            break;

        case 'toggle_user':
            $id = $_POST['id'] ?? null;
            if (!$id) {
                throw new Exception("User ID is required");
            }

            $newStatus = $share->toggleUserActive($id);
            echo json_encode([
                'success' => true, 
                'is_active' => $newStatus,
                'message' => $newStatus ? 'User enabled' : 'User disabled'
            ]);
            break;

        // ============ Permission Operations ============

        case 'get_permissions':
            $shareId = $_GET['share_id'] ?? null;
            if (!$shareId) {
                throw new Exception("Share ID is required");
            }

            $permissions = $share->getSharePermissions($shareId);
            echo json_encode(['success' => true, 'permissions' => $permissions]);
            break;

        case 'get_user_permissions':
            $userId = $_GET['user_id'] ?? null;
            if (!$userId) {
                throw new Exception("User ID is required");
            }

            $permissions = $share->getUserPermissions($userId);
            echo json_encode(['success' => true, 'permissions' => $permissions]);
            break;

        case 'set_permission':
            $shareId = $_POST['share_id'] ?? null;
            $userId = $_POST['user_id'] ?? null;
            $permissionLevel = $_POST['permission_level'] ?? null;

            if (!$shareId || !$userId || !$permissionLevel) {
                throw new Exception("Share ID, User ID, and permission level are required");
            }

            $share->setPermission($shareId, $userId, $permissionLevel);
            echo json_encode(['success' => true, 'message' => 'Permission set successfully']);
            break;

        case 'remove_permission':
            $shareId = $_POST['share_id'] ?? null;
            $userId = $_POST['user_id'] ?? null;

            if (!$shareId || !$userId) {
                throw new Exception("Share ID and User ID are required");
            }

            $share->removePermission($shareId, $userId);
            echo json_encode(['success' => true, 'message' => 'Permission removed successfully']);
            break;

        // ============ System Operations ============

        case 'test_config':
            $result = $share->testSambaConfig();
            echo json_encode(['success' => $result['valid'], 'output' => $result['output']]);
            break;

        case 'status':
            $status = $share->getSambaStatus();
            echo json_encode(['success' => true, 'status' => $status]);
            break;

        case 'connected_clients':
            $clients = $share->getConnectedClients();
            echo json_encode(['success' => true, 'clients' => $clients]);
            break;

        case 'list_directories':
            $directories = $share->listAvailableDirectories();
            echo json_encode(['success' => true, 'directories' => $directories]);
            break;

        case 'create_directory':
            $path = $_POST['path'] ?? '';
            if (empty($path)) {
                throw new Exception("Path is required");
            }

            $share->createDirectory($path);
            echo json_encode(['success' => true, 'message' => 'Directory created successfully']);
            break;

        case 'get_logs':
            $shareId = $_GET['share_id'] ?? null;
            $limit = $_GET['limit'] ?? 100;

            $logs = $share->getAccessLogs($shareId, $limit);
            echo json_encode(['success' => true, 'logs' => $logs]);
            break;

        default:
            throw new Exception("Invalid action: $action");
    }
    
    // Clear output buffer before sending response
    ob_end_clean();

} catch (Exception $e) {
    // Clear output buffer on error
    ob_end_clean();
    http_response_code(400);
    echo json_encode([
        'success' => false,
        'error' => $e->getMessage()
    ]);
} catch (Error $e) {
    // Catch fatal errors (PHP 7+)
    ob_end_clean();
    http_response_code(500);
    echo json_encode([
        'success' => false,
        'error' => 'Internal server error: ' . $e->getMessage()
    ]);
} catch (Throwable $e) {
    // Catch any other throwable (PHP 7+)
    ob_end_clean();
    http_response_code(500);
    echo json_encode([
        'success' => false,
        'error' => 'Unexpected error: ' . $e->getMessage()
    ]);
}

