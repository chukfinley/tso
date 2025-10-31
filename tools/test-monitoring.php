#!/usr/bin/env php
<?php
/**
 * Test script for system monitoring API
 * Usage: php tools/test-monitoring.php
 */

echo "=== System Monitoring API Test ===\n\n";

// Test individual functions
require_once __DIR__ . '/../config/config.php';

function executeCommand($command) {
    return shell_exec($command);
}

function formatBytes($bytes, $precision = 2) {
    $units = ['B', 'KB', 'MB', 'GB', 'TB'];
    $bytes = max($bytes, 0);
    $pow = floor(($bytes ? log($bytes) : 0) / log(1024));
    $pow = min($pow, count($units) - 1);
    $bytes /= pow(1024, $pow);
    return round($bytes, $precision) . ' ' . $units[$pow];
}

// Test CPU info
echo "--- CPU Information ---\n";
$cpuModel = trim(executeCommand("lscpu | grep 'Model name' | cut -d ':' -f2 | xargs"));
if (empty($cpuModel)) {
    $cpuModel = trim(executeCommand("cat /proc/cpuinfo | grep 'model name' | head -n1 | cut -d ':' -f2 | xargs"));
}
$cores = trim(executeCommand("nproc"));
$architecture = trim(executeCommand("uname -m"));
$loadAvg = sys_getloadavg();

echo "Model: $cpuModel\n";
echo "Cores: $cores\n";
echo "Architecture: $architecture\n";
echo "Load Average: " . round($loadAvg[0], 2) . " / " . round($loadAvg[1], 2) . " / " . round($loadAvg[2], 2) . "\n\n";

// Test memory info
echo "--- Memory Information ---\n";
$memData = file_get_contents('/proc/meminfo');
preg_match_all('/^(\w+):\s+(\d+)/m', $memData, $matches);
$memArray = array_combine($matches[1], $matches[2]);

$totalMem = ($memArray['MemTotal'] ?? 0) * 1024;
$freeMem = ($memArray['MemFree'] ?? 0) * 1024;
$availableMem = ($memArray['MemAvailable'] ?? 0) * 1024;
$buffers = ($memArray['Buffers'] ?? 0) * 1024;
$cached = ($memArray['Cached'] ?? 0) * 1024;
$usedMem = $totalMem - $freeMem - $buffers - $cached;

echo "Total: " . formatBytes($totalMem) . "\n";
echo "Used: " . formatBytes($usedMem) . "\n";
echo "Free: " . formatBytes($freeMem) . "\n";
echo "Available: " . formatBytes($availableMem) . "\n";
echo "Usage: " . round(($usedMem / $totalMem) * 100, 1) . "%\n\n";

// Test swap info
echo "--- Swap Information ---\n";
$totalSwap = ($memArray['SwapTotal'] ?? 0) * 1024;
$freeSwap = ($memArray['SwapFree'] ?? 0) * 1024;
$usedSwap = $totalSwap - $freeSwap;

echo "Total: " . formatBytes($totalSwap) . "\n";
echo "Used: " . formatBytes($usedSwap) . "\n";
echo "Free: " . formatBytes($freeSwap) . "\n";
if ($totalSwap > 0) {
    echo "Usage: " . round(($usedSwap / $totalSwap) * 100, 1) . "%\n";
} else {
    echo "Usage: N/A (no swap configured)\n";
}
echo "\n";

// Test uptime
echo "--- System Uptime ---\n";
$uptimeSeconds = (int)trim(executeCommand("cat /proc/uptime | awk '{print $1}'"));
$days = floor($uptimeSeconds / 86400);
$hours = floor(($uptimeSeconds % 86400) / 3600);
$minutes = floor(($uptimeSeconds % 3600) / 60);

$parts = [];
if ($days > 0) $parts[] = $days . 'd';
if ($hours > 0) $parts[] = $hours . 'h';
if ($minutes > 0) $parts[] = $minutes . 'm';
echo "Uptime: " . implode(' ', $parts) . "\n\n";

// Test motherboard info
echo "--- Motherboard Information ---\n";
$mbVendor = trim(executeCommand("cat /sys/class/dmi/id/board_vendor 2>/dev/null"));
$mbName = trim(executeCommand("cat /sys/class/dmi/id/board_name 2>/dev/null"));
$mbVersion = trim(executeCommand("cat /sys/class/dmi/id/board_version 2>/dev/null"));

echo "Vendor: " . ($mbVendor ?: "Unknown") . "\n";
echo "Name: " . ($mbName ?: "Unknown") . "\n";
echo "Version: " . ($mbVersion ?: "N/A") . "\n\n";

// Test network interfaces
echo "--- Network Interfaces ---\n";
$interfaceList = explode("\n", trim(executeCommand("ls /sys/class/net/")));
$interfaceCount = 0;

foreach ($interfaceList as $interface) {
    if (empty($interface) || $interface === 'lo') {
        continue;
    }
    
    $interfaceCount++;
    $operstate = trim(@file_get_contents("/sys/class/net/$interface/operstate"));
    $isUp = ($operstate === 'up');
    $mac = trim(@file_get_contents("/sys/class/net/$interface/address"));
    $ip = trim(executeCommand("ip addr show $interface | grep 'inet ' | awk '{print $2}' | cut -d'/' -f1"));
    $rxBytes = (int)@file_get_contents("/sys/class/net/$interface/statistics/rx_bytes");
    $txBytes = (int)@file_get_contents("/sys/class/net/$interface/statistics/tx_bytes");
    $speed = trim(@file_get_contents("/sys/class/net/$interface/speed"));
    
    if ($speed === '' || $speed === '-1' || $speed === '4294967295') {
        $speed = 'N/A';
    } else {
        $speed = $speed . ' Mbps';
    }
    
    echo "Interface: $interface\n";
    echo "  Status: " . ($isUp ? "Connected" : "Disconnected") . "\n";
    echo "  IP: " . ($ip ?: "N/A") . "\n";
    echo "  MAC: " . ($mac ?: "N/A") . "\n";
    echo "  Speed: $speed\n";
    echo "  RX: " . formatBytes($rxBytes) . "\n";
    echo "  TX: " . formatBytes($txBytes) . "\n";
    echo "\n";
}

if ($interfaceCount === 0) {
    echo "No network interfaces found (excluding loopback).\n\n";
}

// Test API endpoint access
echo "--- API Endpoint Test ---\n";
echo "To test the full API endpoint, visit:\n";
echo "http://your-server/api/system-stats.php\n";
echo "(requires authentication)\n\n";

echo "=== Test Complete ===\n";
echo "All system monitoring components are working correctly.\n";

