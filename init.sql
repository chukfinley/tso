-- Server Management System Database Schema

CREATE DATABASE IF NOT EXISTS servermanager;
USE servermanager;

-- Users Table
CREATE TABLE IF NOT EXISTS users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) UNIQUE NOT NULL,
    email VARCHAR(100) UNIQUE NOT NULL,
    password VARCHAR(255) NOT NULL,
    full_name VARCHAR(100),
    role ENUM('admin', 'user') DEFAULT 'user',
    is_active TINYINT(1) DEFAULT 1,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_login TIMESTAMP NULL,
    INDEX idx_username (username),
    INDEX idx_email (email),
    INDEX idx_role (role)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Default admin user will be created by installation script
-- This ensures the password hash is generated correctly

-- Sessions Table
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(255) PRIMARY KEY,
    user_id INT NOT NULL,
    ip_address VARCHAR(45),
    user_agent VARCHAR(255),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_expires (expires_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Activity Log Table
CREATE TABLE IF NOT EXISTS activity_log (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT,
    action VARCHAR(100) NOT NULL,
    description TEXT,
    ip_address VARCHAR(45),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_user_id (user_id),
    INDEX idx_action (action),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Virtual Machines Table
CREATE TABLE IF NOT EXISTS virtual_machines (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL UNIQUE,
    description TEXT,
    uuid VARCHAR(36) UNIQUE NOT NULL,

    -- Hardware Configuration
    cpu_cores INT DEFAULT 2,
    ram_mb INT DEFAULT 2048,

    -- Extended CPU Configuration
    cpu_type VARCHAR(50) DEFAULT 'host',
    cpu_pinning VARCHAR(255),
    numa_topology TEXT,

    -- Extended Memory Configuration
    balloon_enabled BOOLEAN DEFAULT TRUE,
    hugepages_enabled BOOLEAN DEFAULT FALSE,

    -- Disk Configuration
    disk_path VARCHAR(255),
    disk_size_gb INT DEFAULT 20,
    disk_format ENUM('qcow2', 'raw', 'vmdk') DEFAULT 'qcow2',
    cache_mode VARCHAR(20) DEFAULT 'writeback',
    discard_enabled BOOLEAN DEFAULT TRUE,

    -- Boot Configuration
    boot_order VARCHAR(50) DEFAULT 'cd,hd',
    iso_path VARCHAR(255),
    boot_from_disk BOOLEAN DEFAULT FALSE,
    physical_disk_device VARCHAR(50),
    firmware_type ENUM('bios', 'uefi') DEFAULT 'bios',
    secure_boot BOOLEAN DEFAULT FALSE,
    tpm_enabled BOOLEAN DEFAULT FALSE,

    -- Network Configuration
    network_mode ENUM('nat', 'bridge', 'user', 'none') DEFAULT 'nat',
    network_bridge VARCHAR(50),
    mac_address VARCHAR(17),
    network_model VARCHAR(20) DEFAULT 'virtio',
    vlan_id INT,
    bandwidth_limit_down INT,
    bandwidth_limit_up INT,

    -- Display Configuration
    display_type ENUM('spice', 'vnc', 'none') DEFAULT 'spice',
    spice_port INT,
    vnc_port INT,
    spice_password VARCHAR(50),
    vnc_password VARCHAR(50),
    qmp_socket_path VARCHAR(255),

    -- Status
    status ENUM('stopped', 'running', 'paused', 'error') DEFAULT 'stopped',
    pid INT,

    -- Options
    autostart BOOLEAN DEFAULT FALSE,
    autostart_delay INT DEFAULT 0,
    tags VARCHAR(500),
    os_type VARCHAR(50),
    os_version VARCHAR(50),

    -- Template
    template_id INT,

    -- Metadata
    created_by INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    last_started_at TIMESTAMP NULL,

    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_name (name),
    INDEX idx_status (status),
    INDEX idx_uuid (uuid),
    INDEX idx_autostart (autostart)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- VM Backups Table
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Network Shares Table
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
    valid_users TEXT,  -- Comma-separated list of allowed users
    write_list TEXT,   -- Comma-separated list of users with write access
    read_list TEXT,    -- Comma-separated list of users with read-only access
    admin_users TEXT,  -- Comma-separated list of admin users
    
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Share Users Table (for Samba users)
CREATE TABLE IF NOT EXISTS share_users (
    id INT AUTO_INCREMENT PRIMARY KEY,
    username VARCHAR(50) NOT NULL UNIQUE,
    full_name VARCHAR(100),
    password_hash VARCHAR(255),  -- We'll store a reference, actual password in Samba
    is_active BOOLEAN DEFAULT TRUE,
    
    -- Metadata
    created_by INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_username (username),
    INDEX idx_is_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Share Permissions Table (explicit per-share permissions)
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Share Access Log Table
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- System Logs Table (Centralized logging for all errors and logs)
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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Notifications Table (In-App Notifications)
CREATE TABLE IF NOT EXISTS notifications (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT,
    type ENUM('info', 'warning', 'error', 'success') DEFAULT 'info',
    title VARCHAR(255) NOT NULL,
    message TEXT,
    source VARCHAR(100),
    is_read BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    INDEX idx_user_id (user_id),
    INDEX idx_is_read (is_read),
    INDEX idx_created (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Alert Rules Table
CREATE TABLE IF NOT EXISTS alert_rules (
    id INT AUTO_INCREMENT PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    condition_type VARCHAR(50) NOT NULL,
    threshold FLOAT NOT NULL,
    comparison ENUM('gt', 'lt', 'eq') DEFAULT 'gt',
    severity ENUM('info', 'warning', 'critical') DEFAULT 'warning',
    is_active BOOLEAN DEFAULT TRUE,
    created_by INT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE SET NULL,
    INDEX idx_condition (condition_type),
    INDEX idx_active (is_active)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- Dashboard Config Table (Widget Layout)
CREATE TABLE IF NOT EXISTS dashboard_config (
    id INT AUTO_INCREMENT PRIMARY KEY,
    user_id INT UNIQUE,
    layout JSON,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
