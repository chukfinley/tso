#!/usr/bin/env php
<?php
/**
 * Admin Password Reset Utility
 * Creates or resets the admin user with password 'admin123'
 */

echo "╔════════════════════════════════════════════════════════════════╗\n";
echo "║         ServerOS Admin Password Reset Utility                  ║\n";
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
    exit(1);
}

require_once $configPath;

// Connect to database
try {
    $pdo = new PDO(
        "mysql:host=" . DB_HOST . ";dbname=" . DB_NAME . ";charset=utf8mb4",
        DB_USER,
        DB_PASS,
        [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
    );
    echo "✓ Connected to database\n\n";
} catch (PDOException $e) {
    echo "✗ Database connection failed: " . $e->getMessage() . "\n";
    exit(1);
}

// Generate password hash for 'admin123'
$password = 'admin123';
$passwordHash = password_hash($password, PASSWORD_BCRYPT);

echo "Generated password hash for 'admin123':\n";
echo "  " . $passwordHash . "\n\n";

// Check if admin user exists
$stmt = $pdo->prepare("SELECT id FROM users WHERE username = ?");
$stmt->execute(['admin']);
$existingAdmin = $stmt->fetch(PDO::FETCH_ASSOC);

if ($existingAdmin) {
    // Update existing admin user
    echo "Admin user exists. Resetting password...\n";

    $stmt = $pdo->prepare("
        UPDATE users
        SET password = ?,
            email = 'admin@localhost',
            full_name = 'Administrator',
            role = 'admin',
            is_active = 1
        WHERE username = 'admin'
    ");

    $stmt->execute([$passwordHash]);

    echo "✓ Admin password reset successfully!\n";

} else {
    // Create new admin user
    echo "Admin user does not exist. Creating new admin user...\n";

    $stmt = $pdo->prepare("
        INSERT INTO users (username, email, password, full_name, role, is_active)
        VALUES (?, ?, ?, ?, ?, ?)
    ");

    $stmt->execute([
        'admin',
        'admin@localhost',
        $passwordHash,
        'Administrator',
        'admin',
        1
    ]);

    echo "✓ Admin user created successfully!\n";
}

echo "\n";
echo "╔════════════════════════════════════════════════════════════════╗\n";
echo "║                    Reset Complete!                             ║\n";
echo "╚════════════════════════════════════════════════════════════════╝\n";
echo "\n";
echo "You can now login with:\n";
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n";
echo "  Username: admin\n";
echo "  Password: admin123\n";
echo "\n";
echo "⚠  IMPORTANT: Change this password after first login!\n";
echo "\n";
