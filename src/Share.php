<?php
/**
 * Network Share Management Class
 * Handles SMB/Samba share operations
 */
class Share {
    private $db;
    private $sambaConfPath = '/etc/samba/smb.conf';
    private $shareBasePath = '/srv/samba';
    private $isRoot = false;

    public function __construct() {
        $this->db = Database::getInstance();
        
        // Check if running as root
        $this->isRoot = (posix_geteuid() === 0);
        
        // Ensure share base directory exists
        if (!is_dir($this->shareBasePath)) {
            mkdir($this->shareBasePath, 0755, true);
        }
    }

    /**
     * Get command prefix (sudo if not root, empty if root)
     */
    private function getSudoPrefix() {
        return $this->isRoot ? '' : 'sudo ';
    }

    // ============ Share Management ============

    /**
     * Get all shares
     */
    public function getAll() {
        return $this->db->fetchAll("SELECT * FROM shares ORDER BY share_name");
    }

    /**
     * Get share by ID
     */
    public function getById($id) {
        return $this->db->fetchOne("SELECT * FROM shares WHERE id = ?", [$id]);
    }

    /**
     * Get share by name
     */
    public function getByName($name) {
        return $this->db->fetchOne("SELECT * FROM shares WHERE share_name = ?", [$name]);
    }

    /**
     * Create new share
     */
    public function create($data) {
        $shareName = $data['share_name'];
        
        // Validate share name (no spaces or special characters)
        if (!preg_match('/^[a-zA-Z0-9_-]+$/', $shareName)) {
            throw new Exception("Share name can only contain letters, numbers, hyphens, and underscores");
        }

        // Check if share already exists
        if ($this->getByName($shareName)) {
            throw new Exception("Share with this name already exists");
        }

        // Set default path if not provided
        $path = $data['path'] ?? ($this->shareBasePath . '/' . $shareName);
        
        // Create directory if it doesn't exist
        if (!is_dir($path)) {
            if (!mkdir($path, 0775, true)) {
                throw new Exception("Failed to create share directory");
            }
        }

        // Insert into database
        $sql = "INSERT INTO shares (
            share_name, display_name, path, comment,
            browseable, readonly, guest_ok,
            case_sensitive, preserve_case, short_preserve_case,
            valid_users, write_list, read_list, admin_users,
            create_mask, directory_mask, force_user, force_group,
            is_active, created_by
        ) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)";

        // Helper function to normalize boolean values to integers
        $normalizeBool = function($value, $default = false) {
            if ($value === null || $value === '') {
                return $default ? 1 : 0;
            }
            if (is_bool($value)) {
                return $value ? 1 : 0;
            }
            if (is_string($value)) {
                $value = strtolower(trim($value));
                if ($value === '' || $value === '0' || $value === 'false' || $value === 'no') {
                    return 0;
                }
                return 1;
            }
            return (int)$value ? 1 : 0;
        };

        $params = [
            $shareName,
            $data['display_name'] ?? $shareName,
            $path,
            $data['comment'] ?? '',
            $normalizeBool($data['browseable'] ?? null, true),
            $normalizeBool($data['readonly'] ?? null, false),
            $normalizeBool($data['guest_ok'] ?? null, false),
            $data['case_sensitive'] ?? 'auto',
            $normalizeBool($data['preserve_case'] ?? null, true),
            $normalizeBool($data['short_preserve_case'] ?? null, true),
            $data['valid_users'] ?? '',
            $data['write_list'] ?? '',
            $data['read_list'] ?? '',
            $data['admin_users'] ?? '',
            $data['create_mask'] ?? '0664',
            $data['directory_mask'] ?? '0775',
            $data['force_user'] ?? null,
            $data['force_group'] ?? null,
            $normalizeBool($data['is_active'] ?? null, true),
            $_SESSION['user_id'] ?? null
        ];

        $this->db->query($sql, $params);
        $shareId = $this->db->lastInsertId();

        // Update Samba configuration
        $this->updateSambaConfig();
        $this->reloadSamba();

        // Log action
        $this->logAccess($shareId, $_SESSION['username'] ?? 'system', 'create', $_SERVER['REMOTE_ADDR'] ?? '', 'Share created');

        return $shareId;
    }

    /**
     * Update share
     */
    public function update($id, $data) {
        $share = $this->getById($id);
        if (!$share) {
            throw new Exception("Share not found");
        }

        // Helper function to normalize boolean values to integers
        $normalizeBool = function($value) {
            if ($value === null) {
                return null; // Keep null for updates so we can skip unchanged fields
            }
            if (is_bool($value)) {
                return $value ? 1 : 0;
            }
            if (is_string($value)) {
                $value = strtolower(trim($value));
                if ($value === '' || $value === '0' || $value === 'false' || $value === 'no') {
                    return 0;
                }
                return 1;
            }
            return (int)$value ? 1 : 0;
        };

        $updates = [];
        $params = [];

        // Boolean fields that need normalization
        $booleanFields = ['browseable', 'readonly', 'guest_ok', 'preserve_case', 'short_preserve_case', 'is_active'];

        $allowedFields = [
            'display_name', 'path', 'comment', 'browseable', 'readonly', 'guest_ok',
            'case_sensitive', 'preserve_case', 'short_preserve_case',
            'valid_users', 'write_list', 'read_list', 'admin_users',
            'create_mask', 'directory_mask', 'force_user', 'force_group', 'is_active'
        ];

        foreach ($allowedFields as $field) {
            if (isset($data[$field])) {
                $updates[] = "$field = ?";
                // Normalize boolean fields to integers
                if (in_array($field, $booleanFields)) {
                    $params[] = $normalizeBool($data[$field]);
                } else {
                    $params[] = $data[$field];
                }
            }
        }

        if (empty($updates)) {
            return true;
        }

        $params[] = $id;
        $sql = "UPDATE shares SET " . implode(', ', $updates) . " WHERE id = ?";
        $this->db->query($sql, $params);

        // Update Samba configuration
        $this->updateSambaConfig();
        $this->reloadSamba();

        // Log action
        $this->logAccess($id, $_SESSION['username'] ?? 'system', 'update', $_SERVER['REMOTE_ADDR'] ?? '', 'Share updated');

        return true;
    }

    /**
     * Delete share
     */
    public function delete($id, $deleteDirectory = false) {
        $share = $this->getById($id);
        if (!$share) {
            throw new Exception("Share not found");
        }

        // Delete from database (this will cascade to permissions)
        $this->db->query("DELETE FROM shares WHERE id = ?", [$id]);

        // Delete directory if requested
        if ($deleteDirectory && is_dir($share['path'])) {
            $this->deleteDirectory($share['path']);
        }

        // Update Samba configuration
        $this->updateSambaConfig();
        $this->reloadSamba();

        // Log action
        $this->logAccess($id, $_SESSION['username'] ?? 'system', 'delete', $_SERVER['REMOTE_ADDR'] ?? '', 'Share deleted');

        return true;
    }

    /**
     * Toggle share active status
     */
    public function toggleActive($id) {
        $share = $this->getById($id);
        if (!$share) {
            throw new Exception("Share not found");
        }

        $newStatus = !$share['is_active'];
        $this->db->query("UPDATE shares SET is_active = ? WHERE id = ?", [$newStatus, $id]);

        // Update Samba configuration
        $this->updateSambaConfig();
        $this->reloadSamba();

        return $newStatus;
    }

    // ============ Share User Management ============

    /**
     * Get all share users
     */
    public function getAllUsers() {
        return $this->db->fetchAll("SELECT * FROM share_users ORDER BY username");
    }

    /**
     * Get share user by ID
     */
    public function getUserById($id) {
        return $this->db->fetchOne("SELECT * FROM share_users WHERE id = ?", [$id]);
    }

    /**
     * Get share user by username
     */
    public function getUserByUsername($username) {
        return $this->db->fetchOne("SELECT * FROM share_users WHERE username = ?", [$username]);
    }

    /**
     * Create share user
     */
    public function createUser($username, $password, $fullName = null) {
        // Validate username
        if (!preg_match('/^[a-zA-Z0-9_-]+$/', $username)) {
            throw new Exception("Username can only contain letters, numbers, hyphens, and underscores");
        }

        // Check if user already exists
        if ($this->getUserByUsername($username)) {
            throw new Exception("User with this username already exists");
        }

        // Create Samba user
        $this->createSambaUser($username, $password);

        // Insert into database
        $sql = "INSERT INTO share_users (username, full_name, password_hash, is_active, created_by) 
                VALUES (?, ?, ?, ?, ?)";
        
        $this->db->query($sql, [
            $username,
            $fullName,
            password_hash($password, PASSWORD_DEFAULT), // Store hash for reference
            true,
            $_SESSION['user_id'] ?? null
        ]);

        return $this->db->lastInsertId();
    }

    /**
     * Update share user password
     */
    public function updateUserPassword($id, $newPassword) {
        $user = $this->getUserById($id);
        if (!$user) {
            throw new Exception("User not found");
        }

        // Update Samba password
        $this->updateSambaPassword($user['username'], $newPassword);

        // Update database
        $this->db->query(
            "UPDATE share_users SET password_hash = ? WHERE id = ?",
            [password_hash($newPassword, PASSWORD_DEFAULT), $id]
        );

        return true;
    }

    /**
     * Delete share user
     */
    public function deleteUser($id) {
        $user = $this->getUserById($id);
        if (!$user) {
            throw new Exception("User not found");
        }

        // Delete from Samba
        $this->deleteSambaUser($user['username']);

        // Delete from database (cascade to permissions)
        $this->db->query("DELETE FROM share_users WHERE id = ?", [$id]);

        return true;
    }

    /**
     * Toggle user active status
     */
    public function toggleUserActive($id) {
        $user = $this->getUserById($id);
        if (!$user) {
            throw new Exception("User not found");
        }

        $newStatus = !$user['is_active'];
        $this->db->query("UPDATE share_users SET is_active = ? WHERE id = ?", [$newStatus, $id]);

        // Enable/disable Samba user
        $this->toggleSambaUser($user['username'], $newStatus);

        return $newStatus;
    }

    // ============ Permission Management ============

    /**
     * Get share permissions
     */
    public function getSharePermissions($shareId) {
        $sql = "SELECT sp.*, su.username, su.full_name 
                FROM share_permissions sp
                JOIN share_users su ON sp.share_user_id = su.id
                WHERE sp.share_id = ?
                ORDER BY su.username";
        
        return $this->db->fetchAll($sql, [$shareId]);
    }

    /**
     * Get user permissions
     */
    public function getUserPermissions($userId) {
        $sql = "SELECT sp.*, s.share_name, s.display_name 
                FROM share_permissions sp
                JOIN shares s ON sp.share_id = s.id
                WHERE sp.share_user_id = ?
                ORDER BY s.share_name";
        
        return $this->db->fetchAll($sql, [$userId]);
    }

    /**
     * Set share permission
     */
    public function setPermission($shareId, $userId, $permissionLevel) {
        // Validate permission level
        $validLevels = ['read', 'write', 'admin'];
        if (!in_array($permissionLevel, $validLevels)) {
            throw new Exception("Invalid permission level");
        }

        // Check if permission exists
        $existing = $this->db->fetchOne(
            "SELECT * FROM share_permissions WHERE share_id = ? AND share_user_id = ?",
            [$shareId, $userId]
        );

        if ($existing) {
            // Update existing permission
            $this->db->query(
                "UPDATE share_permissions SET permission_level = ? WHERE share_id = ? AND share_user_id = ?",
                [$permissionLevel, $shareId, $userId]
            );
        } else {
            // Create new permission
            $this->db->query(
                "INSERT INTO share_permissions (share_id, share_user_id, permission_level, created_by) 
                 VALUES (?, ?, ?, ?)",
                [$shareId, $userId, $permissionLevel, $_SESSION['user_id'] ?? null]
            );
        }

        // Update share configuration with new permissions
        $this->updateSharePermissions($shareId);

        return true;
    }

    /**
     * Remove share permission
     */
    public function removePermission($shareId, $userId) {
        $this->db->query(
            "DELETE FROM share_permissions WHERE share_id = ? AND share_user_id = ?",
            [$shareId, $userId]
        );

        // Update share configuration
        $this->updateSharePermissions($shareId);

        return true;
    }

    /**
     * Update share permissions in database based on explicit permissions
     */
    private function updateSharePermissions($shareId) {
        $permissions = $this->getSharePermissions($shareId);
        
        $validUsers = [];
        $readList = [];
        $writeList = [];
        $adminUsers = [];

        foreach ($permissions as $perm) {
            $validUsers[] = $perm['username'];
            
            if ($perm['permission_level'] === 'admin') {
                $adminUsers[] = $perm['username'];
                $writeList[] = $perm['username'];
            } elseif ($perm['permission_level'] === 'write') {
                $writeList[] = $perm['username'];
            } elseif ($perm['permission_level'] === 'read') {
                $readList[] = $perm['username'];
            }
        }

        // Update share
        $this->db->query(
            "UPDATE shares SET valid_users = ?, read_list = ?, write_list = ?, admin_users = ? WHERE id = ?",
            [
                implode(',', $validUsers),
                implode(',', $readList),
                implode(',', $writeList),
                implode(',', $adminUsers),
                $shareId
            ]
        );

        // Update Samba configuration
        $this->updateSambaConfig();
        $this->reloadSamba();
    }

    // ============ Samba Integration ============

    /**
     * Update Samba configuration file
     */
    private function updateSambaConfig() {
        $shares = $this->getAll();
        
        // Read existing smb.conf to preserve global settings
        $config = file_exists($this->sambaConfPath) ? file_get_contents($this->sambaConfPath) : '';
        
        // Find the end of global section
        $globalEnd = strpos($config, '[');
        if ($globalEnd === false) {
            $globalEnd = strlen($config);
        }
        
        $globalSection = substr($config, 0, $globalEnd);
        
        // If no global section exists, create one
        if (empty(trim($globalSection))) {
            $globalSection = "[global]\n";
            $globalSection .= "   workgroup = WORKGROUP\n";
            $globalSection .= "   server string = Samba Server\n";
            $globalSection .= "   netbios name = SERVER\n";
            $globalSection .= "   security = user\n";
            $globalSection .= "   map to guest = bad user\n";
            $globalSection .= "   dns proxy = no\n\n";
        }

        // Build share sections
        $shareConfig = $globalSection;
        
        foreach ($shares as $share) {
            if (!$share['is_active']) {
                continue; // Skip inactive shares
            }

            $shareConfig .= "[{$share['share_name']}]\n";
            $shareConfig .= "   comment = " . ($share['comment'] ?: $share['display_name']) . "\n";
            $shareConfig .= "   path = {$share['path']}\n";
            $shareConfig .= "   browseable = " . ($share['browseable'] ? 'yes' : 'no') . "\n";
            $shareConfig .= "   read only = " . ($share['readonly'] ? 'yes' : 'no') . "\n";
            $shareConfig .= "   guest ok = " . ($share['guest_ok'] ? 'yes' : 'no') . "\n";
            
            // Case sensitivity
            $shareConfig .= "   case sensitive = {$share['case_sensitive']}\n";
            $shareConfig .= "   preserve case = " . ($share['preserve_case'] ? 'yes' : 'no') . "\n";
            $shareConfig .= "   short preserve case = " . ($share['short_preserve_case'] ? 'yes' : 'no') . "\n";
            
            // Permissions
            if (!empty($share['valid_users'])) {
                $shareConfig .= "   valid users = {$share['valid_users']}\n";
            }
            if (!empty($share['write_list'])) {
                $shareConfig .= "   write list = {$share['write_list']}\n";
            }
            if (!empty($share['read_list'])) {
                $shareConfig .= "   read list = {$share['read_list']}\n";
            }
            if (!empty($share['admin_users'])) {
                $shareConfig .= "   admin users = {$share['admin_users']}\n";
            }
            
            // Masks
            $shareConfig .= "   create mask = {$share['create_mask']}\n";
            $shareConfig .= "   directory mask = {$share['directory_mask']}\n";
            
            // Force user/group
            if (!empty($share['force_user'])) {
                $shareConfig .= "   force user = {$share['force_user']}\n";
            }
            if (!empty($share['force_group'])) {
                $shareConfig .= "   force group = {$share['force_group']}\n";
            }
            
            $shareConfig .= "\n";
        }

        // Write configuration (requires sudo)
        $tempFile = '/tmp/smb.conf.' . uniqid();
        file_put_contents($tempFile, $shareConfig);
        
        // Ensure /etc/samba directory exists
        $sambaDir = dirname($this->sambaConfPath);
        if (!is_dir($sambaDir)) {
            // Try PHP native mkdir first if we're root (more reliable)
            if ($this->isRoot) {
                if (!mkdir($sambaDir, 0755, true)) {
                    throw new Exception("Failed to create Samba directory: $sambaDir");
                }
            } else {
                // Use sudo for non-root users
                $sudo = $this->getSudoPrefix();
                exec("$sudo mkdir -p $sambaDir 2>&1", $output, $returnCode);
                if ($returnCode !== 0) {
                    throw new Exception("Failed to create Samba directory: " . implode("\n", $output));
                }
            }
        }
        
        // Get sudo prefix for commands below
        $sudo = $this->getSudoPrefix();
        
        // Move to actual location
        exec("$sudo mv $tempFile {$this->sambaConfPath} 2>&1", $output, $returnCode);
        
        if ($returnCode !== 0) {
            throw new Exception("Failed to update Samba configuration: " . implode("\n", $output));
        }

        // Set proper permissions
        exec("$sudo chmod 644 {$this->sambaConfPath}");
        
        return true;
    }

    /**
     * Reload Samba service
     */
    private function reloadSamba() {
        $sudo = $this->getSudoPrefix();
        exec("$sudo systemctl reload smbd 2>&1", $output, $returnCode);
        
        if ($returnCode !== 0) {
            // Try alternative command
            exec("$sudo service smbd reload 2>&1", $output2, $returnCode2);
            if ($returnCode2 !== 0) {
                throw new Exception("Failed to reload Samba service");
            }
        }

        return true;
    }

    /**
     * Create Samba user
     */
    private function createSambaUser($username, $password) {
        $sudo = $this->getSudoPrefix();
        // Create system user if it doesn't exist
        exec("id -u $username > /dev/null 2>&1", $output, $returnCode);
        if ($returnCode !== 0) {
            exec("$sudo useradd -M -s /sbin/nologin $username 2>&1", $output, $returnCode);
            if ($returnCode !== 0) {
                throw new Exception("Failed to create system user");
            }
        }

        // Add to Samba
        $cmd = sprintf(
            '(echo %s; echo %s) | %s smbpasswd -a -s %s 2>&1',
            escapeshellarg($password),
            escapeshellarg($password),
            $sudo,
            escapeshellarg($username)
        );
        
        exec($cmd, $output, $returnCode);
        
        if ($returnCode !== 0) {
            throw new Exception("Failed to create Samba user: " . implode("\n", $output));
        }

        return true;
    }

    /**
     * Update Samba user password
     */
    private function updateSambaPassword($username, $password) {
        $sudo = $this->getSudoPrefix();
        $cmd = sprintf(
            '(echo %s; echo %s) | %s smbpasswd -s %s 2>&1',
            escapeshellarg($password),
            escapeshellarg($password),
            $sudo,
            escapeshellarg($username)
        );
        
        exec($cmd, $output, $returnCode);
        
        if ($returnCode !== 0) {
            throw new Exception("Failed to update Samba password");
        }

        return true;
    }

    /**
     * Delete Samba user
     */
    private function deleteSambaUser($username) {
        $sudo = $this->getSudoPrefix();
        exec("$sudo smbpasswd -x $username 2>&1", $output, $returnCode);
        // Don't throw error if user doesn't exist in Samba
        
        // Optionally delete system user
        exec("$sudo userdel $username 2>&1");
        
        return true;
    }

    /**
     * Toggle Samba user status
     */
    private function toggleSambaUser($username, $enable) {
        $sudo = $this->getSudoPrefix();
        $flag = $enable ? '-e' : '-d';
        exec("$sudo smbpasswd $flag $username 2>&1", $output, $returnCode);
        
        if ($returnCode !== 0) {
            throw new Exception("Failed to toggle Samba user status");
        }

        return true;
    }

    /**
     * Test Samba configuration
     */
    public function testSambaConfig() {
        $sudo = $this->getSudoPrefix();
        exec("$sudo testparm -s {$this->sambaConfPath} 2>&1", $output, $returnCode);
        
        return [
            'valid' => $returnCode === 0,
            'output' => implode("\n", $output)
        ];
    }

    /**
     * Get Samba status
     */
    public function getSambaStatus() {
        $sudo = $this->getSudoPrefix();
        exec("$sudo systemctl is-active smbd 2>&1", $output, $returnCode);
        $active = trim(implode('', $output)) === 'active';

        if (!$active) {
            // Try alternative service name
            exec("$sudo service smbd status 2>&1", $output2, $returnCode2);
            $active = $returnCode2 === 0;
        }

        return [
            'running' => $active,
            'status' => $active ? 'running' : 'stopped'
        ];
    }

    /**
     * Get connected clients
     */
    public function getConnectedClients() {
        $sudo = $this->getSudoPrefix();
        exec("$sudo smbstatus -b 2>&1", $output, $returnCode);
        
        if ($returnCode !== 0) {
            return [];
        }

        $clients = [];
        foreach ($output as $line) {
            if (preg_match('/^(\S+)\s+(\S+)\s+(\S+)\s+(.+)$/', $line, $matches)) {
                $clients[] = [
                    'pid' => $matches[1],
                    'username' => $matches[2],
                    'machine' => $matches[3],
                    'connected_at' => $matches[4]
                ];
            }
        }

        return $clients;
    }

    // ============ Directory Operations ============

    /**
     * List available directories for sharing
     */
    public function listAvailableDirectories($path = '/') {
        $directories = [];
        
        // Common share locations
        $commonPaths = [
            '/srv/samba',
            '/home',
            '/mnt',
            '/media',
            '/opt/shares'
        ];

        foreach ($commonPaths as $commonPath) {
            if (is_dir($commonPath)) {
                $directories[] = [
                    'path' => $commonPath,
                    'name' => basename($commonPath),
                    'parent' => dirname($commonPath),
                    'writable' => is_writable($commonPath)
                ];
            }
        }

        return $directories;
    }

    /**
     * Create share directory
     */
    public function createDirectory($path, $mode = 0775) {
        if (!mkdir($path, $mode, true)) {
            throw new Exception("Failed to create directory");
        }

        $sudo = $this->getSudoPrefix();
        exec("$sudo chown root:root $path");
        exec("$sudo chmod $mode $path");

        return true;
    }

    /**
     * Delete directory recursively
     */
    private function deleteDirectory($path) {
        if (!is_dir($path)) {
            return false;
        }

        $files = array_diff(scandir($path), ['.', '..']);
        
        foreach ($files as $file) {
            $filePath = $path . '/' . $file;
            is_dir($filePath) ? $this->deleteDirectory($filePath) : unlink($filePath);
        }

        return rmdir($path);
    }

    // ============ Logging ============

    /**
     * Log share access
     */
    private function logAccess($shareId, $username, $action, $ipAddress, $details = '') {
        $this->db->query(
            "INSERT INTO share_access_log (share_id, username, action, ip_address, details) 
             VALUES (?, ?, ?, ?, ?)",
            [$shareId, $username, $action, $ipAddress, $details]
        );
    }

    /**
     * Get share access logs
     */
    public function getAccessLogs($shareId = null, $limit = 100) {
        if ($shareId) {
            $sql = "SELECT sal.*, s.share_name 
                    FROM share_access_log sal
                    JOIN shares s ON sal.share_id = s.id
                    WHERE sal.share_id = ?
                    ORDER BY sal.created_at DESC
                    LIMIT ?";
            return $this->db->fetchAll($sql, [$shareId, $limit]);
        } else {
            $sql = "SELECT sal.*, s.share_name 
                    FROM share_access_log sal
                    LEFT JOIN shares s ON sal.share_id = s.id
                    ORDER BY sal.created_at DESC
                    LIMIT ?";
            return $this->db->fetchAll($sql, [$limit]);
        }
    }
}

