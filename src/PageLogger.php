<?php
/**
 * Page Access Logger
 * Automatically logs page accesses when included
 */

// Only log if Logger is available
if (class_exists('Logger')) {
    $logger = Logger::getInstance();
    
    // Get current page
    $page = $_SERVER['REQUEST_URI'] ?? '/';
    $method = $_SERVER['REQUEST_METHOD'] ?? 'GET';
    
    // Get user ID from session if available
    $userId = null;
    if (isset($_SESSION['user_id'])) {
        $userId = $_SESSION['user_id'];
    }
    
    // Log page access (use debug level to avoid spam)
    $logger->logPageAccess($page, $method, $userId);
}

