<?php
require_once __DIR__ . '/../config/config.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/User.php';
require_once SRC_PATH . '/Auth.php';

$auth = new Auth();

// Redirect to dashboard if logged in, otherwise to login
if ($auth->isLoggedIn()) {
    header('Location: /dashboard.php');
} else {
    header('Location: /login.php');
}
exit;
