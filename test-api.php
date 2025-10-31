#!/usr/bin/env php
<?php
/**
 * Test script to verify system-stats API is working
 * Run: php test-api.php
 */

// Simulate the API call
$_SERVER['REQUEST_URI'] = '/api/system-stats.php';

// Mock authentication for testing
class MockAuth {
    public function isLoggedIn() { return true; }
}

// Test the API endpoint
echo "Testing System Stats API...\n";
echo str_repeat("=", 60) . "\n\n";

// Include the actual API file to test it
ob_start();
try {
    // Override Auth class for testing
    class Auth {
        public function isLoggedIn() { return true; }
    }
    
    require_once __DIR__ . '/public/api/system-stats.php';
    $output = ob_get_clean();
    
    echo "API Response:\n";
    echo str_repeat("-", 60) . "\n";
    echo $output . "\n";
    echo str_repeat("-", 60) . "\n\n";
    
    // Validate JSON
    $data = json_decode($output, true);
    if (json_last_error() === JSON_ERROR_NONE) {
        echo "✓ JSON is valid\n\n";
        
        echo "Data structure:\n";
        echo "- CPU: " . (isset($data['cpu']) ? '✓' : '✗') . "\n";
        echo "- Memory: " . (isset($data['memory']) ? '✓' : '✗') . "\n";
        echo "- Swap: " . (isset($data['swap']) ? '✓' : '✗') . "\n";
        echo "- Uptime: " . (isset($data['uptime']) ? '✓' : '✗') . "\n";
        echo "- Motherboard: " . (isset($data['motherboard']) ? '✓' : '✗') . "\n";
        echo "- Network: " . (isset($data['network']) ? '✓' : '✗') . "\n";
        
        if (isset($data['cpu'])) {
            echo "\nCPU Data:\n";
            echo "  - Usage: " . ($data['cpu']['usage'] ?? 'N/A') . "%\n";
            echo "  - Load: " . ($data['cpu']['load_avg']['1min'] ?? 'N/A') . "\n";
        }
        
        if (isset($data['memory'])) {
            echo "\nMemory Data:\n";
            echo "  - Used: " . ($data['memory']['used_formatted'] ?? 'N/A') . "\n";
            echo "  - Available: " . ($data['memory']['available_formatted'] ?? 'N/A') . "\n";
            echo "  - Usage: " . ($data['memory']['usage_percent'] ?? 'N/A') . "%\n";
        }
        
        if (isset($data['swap'])) {
            echo "\nSwap Data:\n";
            echo "  - Used: " . ($data['swap']['used_formatted'] ?? 'N/A') . "\n";
            echo "  - Free: " . ($data['swap']['free_formatted'] ?? 'N/A') . "\n";
            echo "  - Usage: " . ($data['swap']['usage_percent'] ?? 'N/A') . "%\n";
        }
        
        if (isset($data['uptime'])) {
            echo "\nUptime Data:\n";
            echo "  - Formatted: " . ($data['uptime']['formatted'] ?? 'N/A') . "\n";
        }
        
        if (isset($data['network'])) {
            echo "\nNetwork Data:\n";
            echo "  - Interfaces: " . count($data['network']) . "\n";
        }
        
    } else {
        echo "✗ JSON is invalid: " . json_last_error_msg() . "\n";
    }
    
} catch (Exception $e) {
    ob_end_clean();
    echo "✗ Error: " . $e->getMessage() . "\n";
}

echo "\n" . str_repeat("=", 60) . "\n";
echo "Test complete!\n";

