<?php
require_once __DIR__ . '/../../config/config.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/Auth.php';
require_once SRC_PATH . '/Logger.php';

header('Content-Type: application/json');

$auth = new Auth();
if (!$auth->isLoggedIn()) {
    http_response_code(401);
    echo json_encode(['success' => false, 'error' => 'Unauthorized']);
    exit;
}

$logger = Logger::getInstance();
$action = $_GET['action'] ?? $_POST['action'] ?? '';

try {
    switch ($action) {
        case 'get_logs':
            $filters = [];
            $limit = intval($_GET['limit'] ?? 100);
            $offset = intval($_GET['offset'] ?? 0);

            if (!empty($_GET['level'])) {
                $filters['level'] = $_GET['level'];
            }
            if (!empty($_GET['user_id'])) {
                $filters['user_id'] = intval($_GET['user_id']);
            }
            if (!empty($_GET['start_date'])) {
                $filters['start_date'] = $_GET['start_date'];
            }
            if (!empty($_GET['end_date'])) {
                $filters['end_date'] = $_GET['end_date'];
            }
            if (!empty($_GET['search'])) {
                $filters['search'] = $_GET['search'];
            }

            $logs = $logger->getLogs($filters, $limit, $offset);
            $total = $logger->getLogCount($filters);

            echo json_encode([
                'success' => true,
                'logs' => $logs,
                'total' => $total,
                'limit' => $limit,
                'offset' => $offset
            ]);
            break;

        case 'clear_old':
            // Only admin can clear logs
            if (!isset($_SESSION['role']) || $_SESSION['role'] !== 'admin') {
                http_response_code(403);
                echo json_encode(['success' => false, 'error' => 'Forbidden']);
                exit;
            }

            $days = intval($_GET['days'] ?? $_POST['days'] ?? 30);
            $logger->clearOldLogs($days);

            echo json_encode([
                'success' => true,
                'message' => "Cleared logs older than $days days"
            ]);
            break;

        case 'get_stats':
            $db = Database::getInstance();
            $stats = [
                'total' => $db->fetchOne("SELECT COUNT(*) as count FROM system_logs")['count'] ?? 0,
                'errors' => $db->fetchOne("SELECT COUNT(*) as count FROM system_logs WHERE level = 'error'")['count'] ?? 0,
                'warnings' => $db->fetchOne("SELECT COUNT(*) as count FROM system_logs WHERE level = 'warning'")['count'] ?? 0,
                'info' => $db->fetchOne("SELECT COUNT(*) as count FROM system_logs WHERE level = 'info'")['count'] ?? 0,
                'debug' => $db->fetchOne("SELECT COUNT(*) as count FROM system_logs WHERE level = 'debug'")['count'] ?? 0,
                'recent_errors' => $db->fetchOne("SELECT COUNT(*) as count FROM system_logs WHERE level = 'error' AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)")['count'] ?? 0,
            ];

            echo json_encode([
                'success' => true,
                'stats' => $stats
            ]);
            break;

        case 'log_error':
            // Allow client-side to log errors
            $input = json_decode(file_get_contents('php://input'), true);
            if (!$input || !isset($input['message'])) {
                http_response_code(400);
                echo json_encode(['success' => false, 'error' => 'Invalid input']);
                exit;
            }
            
            $level = $input['level'] ?? 'error';
            $message = $input['message'];
            $context = $input['context'] ?? [];
            
            // Add client-side specific context
            $context['source'] = 'client-side';
            $context['user_agent'] = $_SERVER['HTTP_USER_AGENT'] ?? 'unknown';
            $context['referer'] = $_SERVER['HTTP_REFERER'] ?? 'unknown';
            
            $logger->log($level, $message, $context);
            
            echo json_encode([
                'success' => true,
                'message' => 'Error logged successfully'
            ]);
            break;

        default:
            http_response_code(400);
            echo json_encode(['success' => false, 'error' => 'Invalid action']);
            break;
    }
} catch (Exception $e) {
    http_response_code(500);
    echo json_encode([
        'success' => false,
        'error' => $e->getMessage()
    ]);
}

