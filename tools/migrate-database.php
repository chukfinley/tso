#!/usr/bin/env php
<?php
/**
 * Database Migration Script
 * 
 * This script handles incremental database migrations for updates.
 * All base tables are defined in init.sql and created during fresh installations.
 * 
 * Only add NEW tables/columns here that are added AFTER the current version.
 * Existing tables (users, sessions, activity_log, virtual_machines, vm_backups,
 * shares, share_users, share_permissions, share_access_log, system_logs) are
 * already in init.sql and should NOT be checked here.
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

    // ============================================================================
    // ADD NEW MIGRATIONS HERE
    // ============================================================================
    // Only add checks for tables/columns that are NEW and not in init.sql
    // Example:
    // 
    // // Check and create new_feature table (added in version X.Y.Z)
    // echo "Checking new_feature table... ";
    // $stmt = $pdo->query("SHOW TABLES LIKE 'new_feature'");
    // if ($stmt->rowCount() === 0) {
    //     echo "Creating...\n";
    //     $pdo->exec("
    //         CREATE TABLE IF NOT EXISTS new_feature (
    //             id INT AUTO_INCREMENT PRIMARY KEY,
    //             ...
    //         ) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci
    //     ");
    //     echo "✓ Created new_feature table\n";
    // } else {
    //     echo "✓ Already exists\n";
    // }

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
