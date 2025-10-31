<?php
/**
 * Authentication Class
 */
class Auth {
    private $db;
    private $user;

    public function __construct() {
        $this->db = Database::getInstance();
        $this->user = new User();
    }

    /**
     * Login user
     */
    public function login($username, $password) {
        $user = $this->user->getByUsername($username);

        if (!$user) {
            return false;
        }

        if (!$user['is_active']) {
            return false;
        }

        if (!$this->user->verifyPassword($password, $user['password'])) {
            return false;
        }

        // Create session
        $this->createSession($user);

        // Update last login
        $this->user->updateLastLogin($user['id']);

        // Log activity
        $this->logActivity($user['id'], 'login', 'User logged in');

        return true;
    }

    /**
     * Logout user
     */
    public function logout() {
        if (isset($_SESSION['user_id'])) {
            $this->logActivity($_SESSION['user_id'], 'logout', 'User logged out');
            $this->destroySession();
        }

        session_destroy();
        return true;
    }

    /**
     * Check if user is logged in
     */
    public function isLoggedIn() {
        return isset($_SESSION['user_id']) && isset($_SESSION['username']);
    }

    /**
     * Get current user
     */
    public function getCurrentUser() {
        if (!$this->isLoggedIn()) {
            return null;
        }

        return $this->user->getById($_SESSION['user_id']);
    }

    /**
     * Check if user is admin
     */
    public function isAdmin() {
        if (!$this->isLoggedIn()) {
            return false;
        }

        $user = $this->getCurrentUser();
        return $user && $user['role'] === 'admin';
    }

    /**
     * Require login (redirect if not logged in)
     */
    public function requireLogin() {
        if (!$this->isLoggedIn()) {
            header('Location: /login.php');
            exit;
        }
    }

    /**
     * Require admin (redirect if not admin)
     */
    public function requireAdmin() {
        $this->requireLogin();

        if (!$this->isAdmin()) {
            header('Location: /dashboard.php?error=unauthorized');
            exit;
        }
    }

    /**
     * Create session
     */
    private function createSession($user) {
        $_SESSION['user_id'] = $user['id'];
        $_SESSION['username'] = $user['username'];
        $_SESSION['role'] = $user['role'];
        $_SESSION['full_name'] = $user['full_name'];
        $_SESSION['login_time'] = time();

        // Store session in database
        $sessionId = session_id();
        $ipAddress = $_SERVER['REMOTE_ADDR'] ?? 'unknown';
        $userAgent = $_SERVER['HTTP_USER_AGENT'] ?? 'unknown';
        $expiresAt = date('Y-m-d H:i:s', time() + SESSION_LIFETIME);

        $sql = "INSERT INTO sessions (id, user_id, ip_address, user_agent, expires_at)
                VALUES (?, ?, ?, ?, ?)
                ON DUPLICATE KEY UPDATE
                    ip_address = VALUES(ip_address),
                    user_agent = VALUES(user_agent),
                    expires_at = VALUES(expires_at)";

        $this->db->query($sql, [$sessionId, $user['id'], $ipAddress, $userAgent, $expiresAt]);
    }

    /**
     * Destroy session
     */
    private function destroySession() {
        $sessionId = session_id();
        $sql = "DELETE FROM sessions WHERE id = ?";
        $this->db->query($sql, [$sessionId]);
    }

    /**
     * Log user activity
     */
    private function logActivity($userId, $action, $description) {
        $ipAddress = $_SERVER['REMOTE_ADDR'] ?? 'unknown';
        $sql = "INSERT INTO activity_log (user_id, action, description, ip_address)
                VALUES (?, ?, ?, ?)";

        $this->db->query($sql, [$userId, $action, $description, $ipAddress]);
    }

    /**
     * Clean expired sessions
     */
    public function cleanExpiredSessions() {
        $sql = "DELETE FROM sessions WHERE expires_at < NOW()";
        $this->db->query($sql);
    }
}
