<?php
/**
 * Server Management System - Configuration
 */

// Database Configuration
define('DB_HOST', 'localhost');
define('DB_NAME', 'servermanager');
define('DB_USER', 'root');
define('DB_PASS', '');

// Application Settings
define('APP_NAME', 'ServerOS');
define('APP_VERSION', '1.0.0');
define('BASE_URL', 'https://localhost');

// Security Settings
define('SESSION_LIFETIME', 3600); // 1 hour
define('PASSWORD_MIN_LENGTH', 8);

// Paths
define('ROOT_PATH', dirname(__DIR__));
define('PUBLIC_PATH', ROOT_PATH . '/public');
define('SRC_PATH', ROOT_PATH . '/src');
define('VIEWS_PATH', ROOT_PATH . '/views');

// Timezone
date_default_timezone_set('Europe/Berlin');

// Error Reporting (disable in production)
error_reporting(E_ALL);
ini_set('display_errors', 0); // Changed to 0 for production - errors will be logged
ini_set('log_errors', 1);

// Start session
session_start();

// Force HTTPS for all web requests (allow CLI scripts to run without redirect)
if (PHP_SAPI !== 'cli') {
    $isSecure = (!empty($_SERVER['HTTPS']) && strtolower($_SERVER['HTTPS']) !== 'off')
        || (isset($_SERVER['SERVER_PORT']) && (int)$_SERVER['SERVER_PORT'] === 443)
        || (!empty($_SERVER['HTTP_X_FORWARDED_PROTO']) && strtolower($_SERVER['HTTP_X_FORWARDED_PROTO']) === 'https')
        || (!empty($_SERVER['HTTP_X_FORWARDED_SSL']) && strtolower($_SERVER['HTTP_X_FORWARDED_SSL']) === 'on');

    if (!$isSecure) {
        $host = $_SERVER['HTTP_HOST'] ?? $_SERVER['SERVER_NAME'] ?? 'localhost';
        $requestUri = $_SERVER['REQUEST_URI'] ?? '/';
        $httpsUrl = 'https://' . $host . $requestUri;

        header('Strict-Transport-Security: max-age=63072000; includeSubDomains; preload');
        header('Location: ' . $httpsUrl, true, 301);
        exit;
    }
}

// Initialize error handler after paths are defined
// This must be after session_start() because Logger may need session info
try {
    require_once SRC_PATH . '/Database.php';
    require_once SRC_PATH . '/ErrorHandler.php';
    ErrorHandler::init();
} catch (Exception $e) {
    // If error handler can't be initialized, fall back to default PHP error handling
    error_log("Failed to initialize error handler: " . $e->getMessage());
}
