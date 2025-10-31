<?php
/**
 * Global Error Handler
 * Automatically logs all errors and exceptions
 */

class ErrorHandler {
    private static $logger = null;

    public static function init() {
        // Only initialize once
        if (self::$logger !== null) {
            return;
        }

        // Get logger instance
        require_once __DIR__ . '/Logger.php';
        self::$logger = Logger::getInstance();

        // Set error handler
        set_error_handler([self::class, 'handleError']);

        // Set exception handler
        set_exception_handler([self::class, 'handleException']);

        // Set shutdown handler for fatal errors
        register_shutdown_function([self::class, 'handleShutdown']);

        // Configure error reporting
        error_reporting(E_ALL);
        ini_set('display_errors', 0);
        ini_set('log_errors', 1);
    }

    /**
     * Handle PHP errors
     */
    public static function handleError($errno, $errstr, $errfile, $errline, $errcontext = null) {
        // Don't log errors that are suppressed with @
        if (error_reporting() === 0) {
            return false;
        }

        $level = 'error';
        switch ($errno) {
            case E_WARNING:
            case E_USER_WARNING:
                $level = 'warning';
                break;
            case E_NOTICE:
            case E_USER_NOTICE:
            case E_STRICT:
            case E_DEPRECATED:
            case E_USER_DEPRECATED:
                $level = 'info';
                break;
            default:
                $level = 'error';
        }

        $message = sprintf(
            "[%s] %s in %s on line %d",
            self::getErrorType($errno),
            $errstr,
            $errfile,
            $errline
        );

        $context = [
            'error_type' => self::getErrorType($errno),
            'error_code' => $errno,
            'file' => $errfile,
            'line' => $errline,
        ];

        if ($errcontext) {
            // Don't include full context to avoid memory issues
            $context['has_context'] = true;
        }

        // Try to log, but don't fail if logger isn't available yet
        try {
            if (self::$logger) {
                self::$logger->log($level, $message, $context);
            } else {
                // Fallback to file logging if logger isn't initialized
                error_log("[$level] $message");
            }
        } catch (Exception $e) {
            // Fallback to standard error log
            error_log("[$level] $message");
        }

        // Return false to use PHP's default error handler as well
        return false;
    }

    /**
     * Handle uncaught exceptions
     */
    public static function handleException($exception) {
        try {
            if (self::$logger) {
                self::$logger->exception($exception);
            } else {
                // Fallback to file logging
                error_log("Uncaught exception: " . $exception->getMessage() . " in " . $exception->getFile() . " on line " . $exception->getLine());
            }
        } catch (Exception $e) {
            // Even if logging fails, try to log to standard error log
            error_log("Exception: " . $exception->getMessage());
        }

        // Don't output the exception if we're in an API context
        if (!empty($_SERVER['HTTP_X_REQUESTED_WITH']) || 
            strpos($_SERVER['REQUEST_URI'] ?? '', '/api/') !== false) {
            http_response_code(500);
            header('Content-Type: application/json');
            echo json_encode([
                'success' => false,
                'error' => 'An internal error occurred. Please check the logs.',
                'message' => $exception->getMessage()
            ]);
            exit;
        }

        // For regular pages, let PHP handle it (or custom error page)
    }

    /**
     * Handle fatal errors
     */
    public static function handleShutdown() {
        $error = error_get_last();
        if ($error !== null && in_array($error['type'], [E_ERROR, E_CORE_ERROR, E_COMPILE_ERROR, E_PARSE, E_RECOVERABLE_ERROR])) {
            $message = sprintf(
                "Fatal error: %s in %s on line %d",
                $error['message'],
                $error['file'],
                $error['line']
            );

            $context = [
                'error_type' => self::getErrorType($error['type']),
                'error_code' => $error['type'],
                'file' => $error['file'],
                'line' => $error['line'],
            ];

            try {
                if (self::$logger) {
                    self::$logger->error($message, $context);
                } else {
                    // Fallback to file logging
                    error_log($message);
                }
            } catch (Exception $e) {
                // Even if logging fails, try standard error log
                error_log($message);
            }
        }
    }

    /**
     * Get error type name
     */
    private static function getErrorType($errno) {
        switch ($errno) {
            case E_ERROR: return 'E_ERROR';
            case E_WARNING: return 'E_WARNING';
            case E_PARSE: return 'E_PARSE';
            case E_NOTICE: return 'E_NOTICE';
            case E_CORE_ERROR: return 'E_CORE_ERROR';
            case E_CORE_WARNING: return 'E_CORE_WARNING';
            case E_COMPILE_ERROR: return 'E_COMPILE_ERROR';
            case E_COMPILE_WARNING: return 'E_COMPILE_WARNING';
            case E_USER_ERROR: return 'E_USER_ERROR';
            case E_USER_WARNING: return 'E_USER_WARNING';
            case E_USER_NOTICE: return 'E_USER_NOTICE';
            case E_STRICT: return 'E_STRICT';
            case E_RECOVERABLE_ERROR: return 'E_RECOVERABLE_ERROR';
            case E_DEPRECATED: return 'E_DEPRECATED';
            case E_USER_DEPRECATED: return 'E_USER_DEPRECATED';
            default: return 'UNKNOWN';
        }
    }
}

