#!/usr/bin/env php
<?php
/**
 * Create Default ISO Share
 * Creates a public ISO share for VM images during installation
 */

// Change to script directory
chdir(dirname(__FILE__));

require_once __DIR__ . '/../config/config.php';
require_once __DIR__ . '/../src/Database.php';
require_once __DIR__ . '/../src/Share.php';

try {
    $share = new Share();

    echo "Creating default ISO share...\n";

    // Check if ISO share already exists
    $existing = $share->getByName('iso');
    if ($existing) {
        echo "✓ ISO share already exists\n";
        exit(0);
    }

    // Create the ISO share with guest access (no login required)
    $shareData = [
        'share_name' => 'iso',
        'display_name' => 'ISO Images',
        'path' => '/opt/serveros/storage/isos',
        'comment' => 'ISO images for virtual machines - Public upload',
        'browseable' => true,
        'readonly' => false,        // Allow uploads
        'guest_ok' => true,         // No login required
        'case_sensitive' => 'auto',
        'preserve_case' => true,
        'short_preserve_case' => true,
        'create_mask' => '0664',
        'directory_mask' => '0775',
        'force_user' => null,
        'force_group' => null,
        'is_active' => true
    ];

    // Create the share
    $shareId = $share->create($shareData);

    echo "✓ ISO share created successfully (ID: $shareId)\n";
    echo "  - Path: /opt/serveros/storage/isos\n";
    echo "  - Network path: \\\\\\\\YOUR_SERVER_IP\\\\iso\n";
    echo "  - Guest access: Enabled (no login required)\n";
    echo "  - Permissions: Read/Write for everyone\n";
    echo "\nUsers can now upload ISO files to this share without authentication.\n";

} catch (Exception $e) {
    echo "✗ Error: " . $e->getMessage() . "\n";
    exit(1);
}
