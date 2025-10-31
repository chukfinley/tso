#!/usr/bin/env php
<?php
/**
 * Database Diagnostic Script
 * Checks database connection and default admin user
 */

echo "╔════════════════════════════════════════════════════════════════╗\n";
echo "║           TSO Database Diagnostic Tool                         ║\n";
echo "╚════════════════════════════════════════════════════════════════╝\n\n";

// Determine installation directory
$possiblePaths = [
    '/opt/serveros',
    dirname(__DIR__),
];

$configPath = null;
foreach ($possiblePaths as $path) {
    if (file_exists($path . '/config/config.php')) {
        $configPath = $path . '/config/config.php';
        break;
    }
}

if (!$configPath) {
    echo "✗ Error: Could not find config.php\n";
    echo "  Checked paths:\n";
    foreach ($possiblePaths as $path) {
        echo "  - $path/config/config.php\n";
    }
    exit(1);
}

echo "✓ Config found: $configPath\n\n";

require_once $configPath;

echo "Database Configuration:\n";
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n";
echo "  Host:     " . DB_HOST . "\n";
echo "  Database: " . DB_NAME . "\n";
echo "  User:     " . DB_USER . "\n";
echo "  Password: " . (DB_PASS ? str_repeat('*', strlen(DB_PASS)) : '(empty)') . "\n";
echo "\n";

// Test database connection
echo "Testing database connection...\n";
try {
    $pdo = new PDO(
        "mysql:host=" . DB_HOST . ";dbname=" . DB_NAME . ";charset=utf8mb4",
        DB_USER,
        DB_PASS,
        [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
    );
    echo "✓ Database connection successful!\n\n";
} catch (PDOException $e) {
    echo "✗ Database connection failed!\n";
    echo "  Error: " . $e->getMessage() . "\n";
    exit(1);
}

// Check if users table exists
echo "Checking database schema...\n";
try {
    $stmt = $pdo->query("SHOW TABLES LIKE 'users'");
    if ($stmt->rowCount() === 0) {
        echo "✗ Users table does not exist!\n";
        echo "  Run: mysql " . DB_NAME . " < /opt/serveros/init.sql\n";
        exit(1);
    }
    echo "✓ Users table exists\n\n";
} catch (PDOException $e) {
    echo "✗ Error checking schema: " . $e->getMessage() . "\n";
    exit(1);
}

// Check for admin user
echo "Checking for admin user...\n";
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n";
try {
    $stmt = $pdo->query("SELECT id, username, email, role, is_active, created_at FROM users WHERE username = 'admin'");
    $admin = $stmt->fetch(PDO::FETCH_ASSOC);

    if (!$admin) {
        echo "✗ Admin user not found!\n";
        echo "\n";
        echo "To create admin user, run:\n";
        echo "  sudo /opt/serveros/tools/reset-admin.php\n";
        exit(1);
    }

    echo "✓ Admin user found:\n";
    echo "  ID:       " . $admin['id'] . "\n";
    echo "  Username: " . $admin['username'] . "\n";
    echo "  Email:    " . $admin['email'] . "\n";
    echo "  Role:     " . $admin['role'] . "\n";
    echo "  Active:   " . ($admin['is_active'] ? 'Yes' : 'No') . "\n";
    echo "  Created:  " . $admin['created_at'] . "\n";
    echo "\n";

    // Check password hash
    $stmt = $pdo->query("SELECT password FROM users WHERE username = 'admin'");
    $result = $stmt->fetch(PDO::FETCH_ASSOC);
    $passwordHash = $result['password'];

    echo "Password Information:\n";
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n";
    echo "  Hash: " . substr($passwordHash, 0, 40) . "...\n";
    echo "  Type: " . (strpos($passwordHash, '$2y$') === 0 ? 'bcrypt' : 'unknown') . "\n";
    echo "\n";

    // Test password verification
    echo "Testing default password 'admin123'...\n";
    if (password_verify('admin123', $passwordHash)) {
        echo "✓ Password 'admin123' is CORRECT!\n";
        echo "\n";
        echo "You should be able to login with:\n";
        echo "  Username: admin\n";
        echo "  Password: admin123\n";
    } else {
        echo "✗ Password 'admin123' does NOT match!\n";
        echo "\n";
        echo "To reset the password, run:\n";
        echo "  sudo /opt/serveros/tools/reset-admin.php\n";
    }

} catch (PDOException $e) {
    echo "✗ Error checking admin user: " . $e->getMessage() . "\n";
    exit(1);
}

echo "\n";
echo "All user accounts:\n";
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n";
$stmt = $pdo->query("SELECT id, username, email, role, is_active FROM users ORDER BY id");
$users = $stmt->fetchAll(PDO::FETCH_ASSOC);

foreach ($users as $user) {
    echo "  [{$user['id']}] {$user['username']} ({$user['email']}) - {$user['role']} - " .
         ($user['is_active'] ? 'Active' : 'Inactive') . "\n";
}

echo "\n✓ Diagnostic complete!\n";
