<?php
/**
 * Centralized Logging System
 * Handles all application logs including errors, warnings, info, and debug messages
 */
class Logger {
    private static $instance = null;
    private $db;
    private $logFile;
    private $logDir;

    private function __construct() {
        $this->db = Database::getInstance();
        $this->logDir = ROOT_PATH . '/logs';
        $this->logFile = $this->logDir . '/app-' . date('Y-m-d') . '.log';
        
        // Ensure logs directory exists
        if (!is_dir($this->logDir)) {
            mkdir($this->logDir, 0755, true);
        }
    }

    public static function getInstance() {
        if (self::$instance === null) {
            self::$instance = new self();
        }
        return self::$instance;
    }

    /**
     * Log an error message
     */
    public function error($message, $context = [], $userId = null) {
        return $this->log('error', $message, $context, $userId);
    }

    /**
     * Log a warning message
     */
    public function warning($message, $context = [], $userId = null) {
        return $this->log('warning', $message, $context, $userId);
    }

    /**
     * Log an info message
     */
    public function info($message, $context = [], $userId = null) {
        return $this->log('info', $message, $context, $userId);
    }

    /**
     * Log a debug message
     */
    public function debug($message, $context = [], $userId = null) {
        return $this->log('debug', $message, $context, $userId);
    }

    /**
     * Log an exception
     */
    public function exception(Exception $e, $userId = null, $context = []) {
        $message = $e->getMessage();
        $context['exception'] = [
            'class' => get_class($e),
            'file' => $e->getFile(),
            'line' => $e->getLine(),
            'trace' => $e->getTraceAsString()
        ];
        
        return $this->error($message, $context, $userId);
    }

    /**
     * Main logging method
     * Public so ErrorHandler can access it
     */
    public function log($level, $message, $context = [], $userId = null) {
        $timestamp = date('Y-m-d H:i:s');
        $ipAddress = $_SERVER['REMOTE_ADDR'] ?? 'unknown';
        
        // Get user ID from session if not provided
        if ($userId === null && isset($_SESSION['user_id'])) {
            $userId = $_SESSION['user_id'];
        }

        // Prepare context data
        $contextJson = !empty($context) ? json_encode($context) : null;

        // Log to database
        try {
            $this->db->query(
                "INSERT INTO system_logs (level, message, context, user_id, ip_address, created_at) 
                 VALUES (?, ?, ?, ?, ?, NOW())",
                [$level, $message, $contextJson, $userId, $ipAddress]
            );
        } catch (Exception $e) {
            // If database logging fails, try file logging
            error_log("Failed to log to database: " . $e->getMessage());
        }

        // Log to file
        $logEntry = sprintf(
            "[%s] [%s] %s | IP: %s | User: %s | Context: %s\n",
            $timestamp,
            strtoupper($level),
            $message,
            $ipAddress,
            $userId ?? 'system',
            $contextJson ?? '{}'
        );

        file_put_contents($this->logFile, $logEntry, FILE_APPEND | LOCK_EX);

        // Also log to PHP error log for critical errors
        if ($level === 'error') {
            error_log("[$level] $message" . ($contextJson ? " | Context: $contextJson" : ""));
        }

        return true;
    }

    /**
     * Get logs from database
     */
    public function getLogs($filters = [], $limit = 100, $offset = 0) {
        $where = [];
        $params = [];

        if (!empty($filters['level'])) {
            $where[] = "level = ?";
            $params[] = $filters['level'];
        }

        if (!empty($filters['user_id'])) {
            $where[] = "user_id = ?";
            $params[] = $filters['user_id'];
        }

        if (!empty($filters['start_date'])) {
            $where[] = "created_at >= ?";
            $params[] = $filters['start_date'];
        }

        if (!empty($filters['end_date'])) {
            $where[] = "created_at <= ?";
            $params[] = $filters['end_date'];
        }

        if (!empty($filters['search'])) {
            $where[] = "(message LIKE ? OR context LIKE ?)";
            $searchTerm = '%' . $filters['search'] . '%';
            $params[] = $searchTerm;
            $params[] = $searchTerm;
        }

        $whereClause = !empty($where) ? "WHERE " . implode(" AND ", $where) : "";

        $sql = "SELECT sl.*, u.username 
                FROM system_logs sl
                LEFT JOIN users u ON sl.user_id = u.id
                $whereClause
                ORDER BY sl.created_at DESC
                LIMIT ? OFFSET ?";

        $params[] = $limit;
        $params[] = $offset;

        return $this->db->fetchAll($sql, $params);
    }

    /**
     * Get log count with filters
     */
    public function getLogCount($filters = []) {
        $where = [];
        $params = [];

        if (!empty($filters['level'])) {
            $where[] = "level = ?";
            $params[] = $filters['level'];
        }

        if (!empty($filters['user_id'])) {
            $where[] = "user_id = ?";
            $params[] = $filters['user_id'];
        }

        if (!empty($filters['start_date'])) {
            $where[] = "created_at >= ?";
            $params[] = $filters['start_date'];
        }

        if (!empty($filters['end_date'])) {
            $where[] = "created_at <= ?";
            $params[] = $filters['end_date'];
        }

        if (!empty($filters['search'])) {
            $where[] = "(message LIKE ? OR context LIKE ?)";
            $searchTerm = '%' . $filters['search'] . '%';
            $params[] = $searchTerm;
            $params[] = $searchTerm;
        }

        $whereClause = !empty($where) ? "WHERE " . implode(" AND ", $where) : "";

        $sql = "SELECT COUNT(*) as count FROM system_logs $whereClause";
        $result = $this->db->fetchOne($sql, $params);
        return $result['count'] ?? 0;
    }

    /**
     * Clear old logs (older than specified days)
     */
    public function clearOldLogs($days = 30) {
        $cutoffDate = date('Y-m-d H:i:s', strtotime("-$days days"));
        
        // Clear from database
        $this->db->query(
            "DELETE FROM system_logs WHERE created_at < ?",
            [$cutoffDate]
        );

        // Clear old log files (keep last 7 days)
        $files = glob($this->logDir . '/app-*.log');
        foreach ($files as $file) {
            if (filemtime($file) < strtotime("-7 days")) {
                unlink($file);
            }
        }

        return true;
    }

    /**
     * Log a command execution
     */
    public function logCommand($command, $output = null, $returnCode = 0, $userId = null) {
        $context = [
            'type' => 'command_execution',
            'command' => substr($command, 0, 500),
            'return_code' => $returnCode,
        ];
        
        if ($output !== null) {
            $context['output_length'] = strlen($output);
            // Only include output for errors
            if ($returnCode !== 0) {
                $context['output'] = substr($output, 0, 1000);
            }
        }
        
        $level = ($returnCode === 0) ? 'info' : 'warning';
        $message = "Command executed: " . substr($command, 0, 200);
        
        return $this->log($level, $message, $context, $userId);
    }

    /**
     * Log a service state change
     */
    public function logService($service, $action, $status = null, $userId = null) {
        $context = [
            'type' => 'service_action',
            'service' => $service,
            'action' => $action,
        ];
        
        if ($status !== null) {
            $context['status'] = $status;
        }
        
        $message = "Service $action: $service";
        
        return $this->log('info', $message, $context, $userId);
    }

    /**
     * Log a page access
     */
    public function logPageAccess($page, $method = 'GET', $userId = null) {
        $context = [
            'type' => 'page_access',
            'page' => $page,
            'method' => $method,
        ];
        
        $message = "Page accessed: $method $page";
        
        return $this->log('debug', $message, $context, $userId);
    }

    /**
     * Log a system update
     */
    public function logUpdate($updateType, $details = [], $userId = null) {
        $context = array_merge([
            'type' => 'system_update',
            'update_type' => $updateType,
        ], $details);
        
        $message = "System update: $updateType";
        
        return $this->log('info', $message, $context, $userId);
    }

    // Prevent cloning
    private function __clone() {}

    // Prevent unserialization
    public function __wakeup() {
        throw new Exception("Cannot unserialize singleton");
    }
}

