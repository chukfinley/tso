#!/usr/bin/env php
<?php
/**
 * Database Migration Script
 * Safely adds new tables and columns without dropping existing data
 */

// Change to script directory
chdir(dirname(__FILE__));

require_once __DIR__ . '/../config/config.php';

try {
    $pdo = new PDO(
        "mysql:host=" . DB_HOST . ";dbname=" . DB_NAME,
        DB_USER,
        DB_PASS,
        [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
    );

    echo "Connected to database: " . DB_NAME . "\n";
    echo "Running migrations...\n\n";

    // Check and create virtual_machines table
    echo "Checking virtual_machines table... ";
    $stmt = $pdo->query("SHOW TABLES LIKE 'virtual_machines'");
    if ($stmt->rowCount() === 0) {
        echo "Creating...\n";
        $pdo->exec("
            CREATE TABLE IF NOT EXISTS virtual_machines (
                id INT AUTO_INCREMENT PRIMARY KEY,
                name VARCHAR(100) NOT NULL UNIQUE,
                description TEXT,
                uuid VARCHAR(36) UNIQUE NOT NULL,

                -- Hardware Configuration
                cpu_cores INT DEFAULT 2,
                ram_mb INT DEFAULT 2048,

                -- Disk Configuration
                disk_path VARCHAR(255),
                disk_size_gb INT DEFAULT 20,
                disk_format ENUM('qcow2', 'raw', 'vmdk') DEFAULT 'qcow2',

                -- Boot Configuration
                boot_order VARCHAR(50) DEFAULT 'cd,hd',
                iso_path VARCHAR(255),
                boot_from_disk BOOLEAN DEFAULT FALSE,
                physical_disk_device VARCHAR(50),

                -- Network Configuration
                network_mode ENUM('nat', 'bridge', 'user', 'none') DEFAULT 'nat',
                network_bridge VARCHAR(50),
                mac_address VARCHAR(17),

                -- Display Configuration
                display_type ENUM('spice', 'vnc', 'none') DEFAULT 'spice',
                spice_port INT,
                vnc_port INT,
                spice_password VARCHAR(50),

                -- Status
                status ENUM('stopped', 'running', 'paused', 'error') DEFAULT 'stopped',
                pid INT,

                -- Metadata
                created_by INT,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
                last_started_at TIMESTAMP NULL,

                FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
                INDEX idx_name (name),
                INDEX idx_status (status),
                INDEX idx_uuid (uuid)
            ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
        ");
        echo "✓ Created virtual_machines table\n";
    } else {
        echo "✓ Already exists\n";
    }

    // Check and create vm_backups table
    echo "Checking vm_backups table... ";
    $stmt = $pdo->query("SHOW TABLES LIKE 'vm_backups'");
    if ($stmt->rowCount() === 0) {
        echo "Creating...\n";
        $pdo->exec("
            CREATE TABLE IF NOT EXISTS vm_backups (
                id INT AUTO_INCREMENT PRIMARY KEY,
                vm_id INT NOT NULL,
                vm_name VARCHAR(100) NOT NULL,
                backup_name VARCHAR(255) NOT NULL,
                backup_path VARCHAR(500) NOT NULL,
                backup_size BIGINT,
                compressed BOOLEAN DEFAULT TRUE,
                compression_type ENUM('gzip', 'none') DEFAULT 'gzip',
                status ENUM('creating', 'completed', 'failed', 'restoring') DEFAULT 'creating',
                created_by INT,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                completed_at TIMESTAMP NULL,
                notes TEXT,

                FOREIGN KEY (vm_id) REFERENCES virtual_machines(id) ON DELETE CASCADE,
                FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
                INDEX idx_vm_id (vm_id),
                INDEX idx_status (status),
                INDEX idx_created (created_at)
            ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
        ");
        echo "✓ Created vm_backups table\n";
    } else {
        echo "✓ Already exists\n";
    }

    // Check and create shares table
    echo "Checking shares table... ";
    $stmt = $pdo->query("SHOW TABLES LIKE 'shares'");
    if ($stmt->rowCount() === 0) {
        echo "Creating...\n";
        $pdo->exec("
            CREATE TABLE IF NOT EXISTS shares (
                id INT AUTO_INCREMENT PRIMARY KEY,
                share_name VARCHAR(100) NOT NULL UNIQUE,
                display_name VARCHAR(100),
                path VARCHAR(500) NOT NULL,
                comment TEXT,

                -- Share Settings
                browseable BOOLEAN DEFAULT TRUE,
                readonly BOOLEAN DEFAULT FALSE,
                guest_ok BOOLEAN DEFAULT FALSE,

                -- Case Sensitivity Settings
                case_sensitive ENUM('auto', 'yes', 'no') DEFAULT 'auto',
                preserve_case BOOLEAN DEFAULT TRUE,
                short_preserve_case BOOLEAN DEFAULT TRUE,

                -- Advanced Options
                valid_users TEXT,
                write_list TEXT,
                read_list TEXT,
                admin_users TEXT,

                -- Additional Options
                create_mask VARCHAR(10) DEFAULT '0664',
                directory_mask VARCHAR(10) DEFAULT '0775',
                force_user VARCHAR(50),
                force_group VARCHAR(50),

                -- Status
                is_active BOOLEAN DEFAULT TRUE,

                -- Metadata
                created_by INT,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

                FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
                INDEX idx_share_name (share_name),
                INDEX idx_is_active (is_active)
            ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
        ");
        echo "✓ Created shares table\n";
    } else {
        echo "✓ Already exists\n";
    }

    // Check and create share_users table
    echo "Checking share_users table... ";
    $stmt = $pdo->query("SHOW TABLES LIKE 'share_users'");
    if ($stmt->rowCount() === 0) {
        echo "Creating...\n";
        $pdo->exec("
            CREATE TABLE IF NOT EXISTS share_users (
                id INT AUTO_INCREMENT PRIMARY KEY,
                username VARCHAR(50) NOT NULL UNIQUE,
                full_name VARCHAR(100),
                password_hash VARCHAR(255),
                is_active BOOLEAN DEFAULT TRUE,

                -- Metadata
                created_by INT,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

                FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
                INDEX idx_username (username),
                INDEX idx_is_active (is_active)
            ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
        ");
        echo "✓ Created share_users table\n";
    } else {
        echo "✓ Already exists\n";
    }

    // Check and create share_permissions table
    echo "Checking share_permissions table... ";
    $stmt = $pdo->query("SHOW TABLES LIKE 'share_permissions'");
    if ($stmt->rowCount() === 0) {
        echo "Creating...\n";
        $pdo->exec("
            CREATE TABLE IF NOT EXISTS share_permissions (
                id INT AUTO_INCREMENT PRIMARY KEY,
                share_id INT NOT NULL,
                share_user_id INT NOT NULL,
                permission_level ENUM('read', 'write', 'admin') DEFAULT 'read',

                -- Metadata
                created_by INT,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
                updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,

                FOREIGN KEY (share_id) REFERENCES shares(id) ON DELETE CASCADE,
                FOREIGN KEY (share_user_id) REFERENCES share_users(id) ON DELETE CASCADE,
                FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
                UNIQUE KEY unique_share_user (share_id, share_user_id),
                INDEX idx_share_id (share_id),
                INDEX idx_share_user_id (share_user_id)
            ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
        ");
        echo "✓ Created share_permissions table\n";
    } else {
        echo "✓ Already exists\n";
    }

    // Check and create share_access_log table
    echo "Checking share_access_log table... ";
    $stmt = $pdo->query("SHOW TABLES LIKE 'share_access_log'");
    if ($stmt->rowCount() === 0) {
        echo "Creating...\n";
        $pdo->exec("
            CREATE TABLE IF NOT EXISTS share_access_log (
                id INT AUTO_INCREMENT PRIMARY KEY,
                share_id INT,
                username VARCHAR(50),
                action VARCHAR(50),
                ip_address VARCHAR(45),
                details TEXT,
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

                FOREIGN KEY (share_id) REFERENCES shares(id) ON DELETE CASCADE,
                INDEX idx_share_id (share_id),
                INDEX idx_username (username),
                INDEX idx_created (created_at)
            ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
        ");
        echo "✓ Created share_access_log table\n";
    } else {
        echo "✓ Already exists\n";
    }

    // Check and create system_logs table
    echo "Checking system_logs table... ";
    $stmt = $pdo->query("SHOW TABLES LIKE 'system_logs'");
    if ($stmt->rowCount() === 0) {
        echo "Creating...\n";
        $pdo->exec("
            CREATE TABLE IF NOT EXISTS system_logs (
                id INT AUTO_INCREMENT PRIMARY KEY,
                level ENUM('error', 'warning', 'info', 'debug') NOT NULL DEFAULT 'info',
                message TEXT NOT NULL,
                context TEXT,
                user_id INT,
                ip_address VARCHAR(45),
                created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,

                FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
                INDEX idx_level (level),
                INDEX idx_user_id (user_id),
                INDEX idx_created (created_at),
                INDEX idx_ip_address (ip_address)
            ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
        ");
        echo "✓ Created system_logs table\n";
    } else {
        echo "✓ Already exists\n";
    }

    // Create VM storage directories
    echo "\nCreating VM storage directories... ";
    $vmStorageDir = '/opt/serveros/storage/vms';
    $vmBackupDir = '/opt/serveros/storage/backups';
    $vmIsoDir = '/opt/serveros/storage/isos';

    if (!is_dir($vmStorageDir)) {
        mkdir($vmStorageDir, 0755, true);
    }
    if (!is_dir($vmBackupDir)) {
        mkdir($vmBackupDir, 0755, true);
    }
    if (!is_dir($vmIsoDir)) {
        mkdir($vmIsoDir, 0755, true);
    }

    // Set proper permissions
    exec('chown -R www-data:www-data /opt/serveros/storage');
    echo "✓ Created\n";

    echo "\n✓ All migrations completed successfully!\n";

} catch (PDOException $e) {
    echo "\n✗ Database error: " . $e->getMessage() . "\n";
    exit(1);
} catch (Exception $e) {
    echo "\n✗ Error: " . $e->getMessage() . "\n";
    exit(1);
}
