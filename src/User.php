<?php
/**
 * User Management Class
 */
class User {
    private $db;

    public function __construct() {
        $this->db = Database::getInstance();
    }

    /**
     * Create a new user
     */
    public function create($username, $email, $password, $full_name, $role = 'user') {
        $hashedPassword = password_hash($password, PASSWORD_BCRYPT);

        $sql = "INSERT INTO users (username, email, password, full_name, role)
                VALUES (?, ?, ?, ?, ?)";

        try {
            $this->db->query($sql, [$username, $email, $hashedPassword, $full_name, $role]);
            return $this->db->lastInsertId();
        } catch (PDOException $e) {
            return false;
        }
    }

    /**
     * Get user by ID
     */
    public function getById($id) {
        $sql = "SELECT * FROM users WHERE id = ?";
        return $this->db->fetchOne($sql, [$id]);
    }

    /**
     * Get user by username
     */
    public function getByUsername($username) {
        $sql = "SELECT * FROM users WHERE username = ?";
        return $this->db->fetchOne($sql, [$username]);
    }

    /**
     * Get user by email
     */
    public function getByEmail($email) {
        $sql = "SELECT * FROM users WHERE email = ?";
        return $this->db->fetchOne($sql, [$email]);
    }

    /**
     * Get all users
     */
    public function getAll() {
        $sql = "SELECT id, username, email, full_name, role, is_active, created_at, last_login
                FROM users ORDER BY created_at DESC";
        return $this->db->fetchAll($sql);
    }

    /**
     * Update user
     */
    public function update($id, $data) {
        $fields = [];
        $values = [];

        foreach ($data as $key => $value) {
            if (in_array($key, ['username', 'email', 'full_name', 'role', 'is_active'])) {
                $fields[] = "$key = ?";
                $values[] = $value;
            }
        }

        if (empty($fields)) {
            return false;
        }

        $values[] = $id;
        $sql = "UPDATE users SET " . implode(', ', $fields) . " WHERE id = ?";

        try {
            $this->db->query($sql, $values);
            return true;
        } catch (PDOException $e) {
            return false;
        }
    }

    /**
     * Update password
     */
    public function updatePassword($id, $newPassword) {
        $hashedPassword = password_hash($newPassword, PASSWORD_BCRYPT);
        $sql = "UPDATE users SET password = ? WHERE id = ?";

        try {
            $this->db->query($sql, [$hashedPassword, $id]);
            return true;
        } catch (PDOException $e) {
            return false;
        }
    }

    /**
     * Delete user
     */
    public function delete($id) {
        // Don't allow deleting user with ID 1 (main admin)
        if ($id == 1) {
            return false;
        }

        $sql = "DELETE FROM users WHERE id = ?";
        try {
            $this->db->query($sql, [$id]);
            return true;
        } catch (PDOException $e) {
            return false;
        }
    }

    /**
     * Update last login timestamp
     */
    public function updateLastLogin($id) {
        $sql = "UPDATE users SET last_login = NOW() WHERE id = ?";
        $this->db->query($sql, [$id]);
    }

    /**
     * Verify password
     */
    public function verifyPassword($password, $hash) {
        return password_verify($password, $hash);
    }
}
