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
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
