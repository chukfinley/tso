#!/bin/bash
# Auto-install/update monitoring system

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

echo "===================================="
echo "System Monitoring Installer/Updater"
echo "===================================="
echo ""

# Check if monitoring is already installed
if [ -f "$PROJECT_ROOT/public/api/system-stats.php" ]; then
    echo "✓ Monitoring system detected - UPDATING..."
    MODE="update"
else
    echo "✓ Installing monitoring system for the first time..."
    MODE="install"
fi

# Create API directory if it doesn't exist
mkdir -p "$PROJECT_ROOT/public/api"

# Install/Update API endpoint
echo "→ Installing API endpoint..."
cat > "$PROJECT_ROOT/public/api/system-stats.php" << 'EOFAPI'
<?php
require_once __DIR__ . '/../../config/config.php';
require_once SRC_PATH . '/Auth.php';

header('Content-Type: application/json');

$auth = new Auth();
if (!$auth->isLoggedIn()) {
    http_response_code(401);
    echo json_encode(['error' => 'Unauthorized']);
    exit;
}

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

function getCpuInfo() {
    $cpuInfo = [];
    $cpuModel = trim(executeCommand("lscpu | grep 'Model name' | cut -d ':' -f2 | xargs"));
    if (empty($cpuModel)) {
        $cpuModel = trim(executeCommand("cat /proc/cpuinfo | grep 'model name' | head -n1 | cut -d ':' -f2 | xargs"));
    }
    $cores = trim(executeCommand("nproc"));
    $architecture = trim(executeCommand("uname -m"));
    $maxFreq = trim(executeCommand("lscpu | grep 'CPU max MHz' | awk '{print $4}'"));
    if (empty($maxFreq)) {
        $maxFreq = trim(executeCommand("cat /proc/cpuinfo | grep 'cpu MHz' | head -n1 | awk '{print $4}'"));
    }
    $cpuUsage = trim(executeCommand("top -bn1 | grep 'Cpu(s)' | sed 's/.*, *\\([0-9.]*\\)%* id.*/\\1/' | awk '{print 100 - $1}'"));
    if (empty($cpuUsage)) {
        $cpuUsage = trim(executeCommand("mpstat 1 1 | awk '/Average/ {print 100-$NF}'"));
    }
    $loadAvg = sys_getloadavg();
    $cpuInfo = [
        'model' => $cpuModel ?: 'Unknown',
        'cores' => (int)$cores ?: 1,
        'architecture' => $architecture ?: 'Unknown',
        'frequency' => $maxFreq ? round((float)$maxFreq, 2) . ' MHz' : 'N/A',
        'usage' => round((float)$cpuUsage, 1),
        'load_avg' => [
            '1min' => round($loadAvg[0], 2),
            '5min' => round($loadAvg[1], 2),
            '15min' => round($loadAvg[2], 2)
        ]
    ];
    return $cpuInfo;
}

function getMemoryInfo() {
    $memInfo = [];
    $memData = file_get_contents('/proc/meminfo');
    preg_match_all('/^(\w+):\s+(\d+)/m', $memData, $matches);
    $memArray = array_combine($matches[1], $matches[2]);
    $total = ($memArray['MemTotal'] ?? 0) * 1024;
    $free = ($memArray['MemFree'] ?? 0) * 1024;
    $available = ($memArray['MemAvailable'] ?? 0) * 1024;
    $buffers = ($memArray['Buffers'] ?? 0) * 1024;
    $cached = ($memArray['Cached'] ?? 0) * 1024;
    $used = $total - $free - $buffers - $cached;
    $usagePercent = $total > 0 ? ($used / $total) * 100 : 0;
    return [
        'total' => $total,
        'total_formatted' => formatBytes($total),
        'used' => $used,
        'used_formatted' => formatBytes($used),
        'free' => $free,
        'free_formatted' => formatBytes($free),
        'available' => $available,
        'available_formatted' => formatBytes($available),
        'usage_percent' => round($usagePercent, 1)
    ];
}

function getSwapInfo() {
    $swapInfo = [];
    $memData = file_get_contents('/proc/meminfo');
    preg_match_all('/^(\w+):\s+(\d+)/m', $memData, $matches);
    $memArray = array_combine($matches[1], $matches[2]);
    $total = ($memArray['SwapTotal'] ?? 0) * 1024;
    $free = ($memArray['SwapFree'] ?? 0) * 1024;
    $used = $total - $free;
    $usagePercent = $total > 0 ? ($used / $total) * 100 : 0;
    return [
        'total' => $total,
        'total_formatted' => formatBytes($total),
        'used' => $used,
        'used_formatted' => formatBytes($used),
        'free' => $free,
        'free_formatted' => formatBytes($free),
        'usage_percent' => round($usagePercent, 1)
    ];
}

function getUptime() {
    $uptimeSeconds = (int)trim(executeCommand("cat /proc/uptime | awk '{print $1}'"));
    $days = floor($uptimeSeconds / 86400);
    $hours = floor(($uptimeSeconds % 86400) / 3600);
    $minutes = floor(($uptimeSeconds % 3600) / 60);
    $parts = [];
    if ($days > 0) $parts[] = $days . 'd';
    if ($hours > 0) $parts[] = $hours . 'h';
    if ($minutes > 0) $parts[] = $minutes . 'm';
    return [
        'seconds' => $uptimeSeconds,
        'formatted' => implode(' ', $parts) ?: '0m'
    ];
}

function getMotherboardInfo() {
    $vendor = trim(executeCommand("cat /sys/class/dmi/id/board_vendor 2>/dev/null"));
    $name = trim(executeCommand("cat /sys/class/dmi/id/board_name 2>/dev/null"));
    $version = trim(executeCommand("cat /sys/class/dmi/id/board_version 2>/dev/null"));
    if (empty($vendor)) {
        $vendor = trim(executeCommand("dmidecode -s baseboard-manufacturer 2>/dev/null"));
    }
    if (empty($name)) {
        $name = trim(executeCommand("dmidecode -s baseboard-product-name 2>/dev/null"));
    }
    return [
        'vendor' => $vendor ?: 'Unknown',
        'name' => $name ?: 'Unknown',
        'version' => $version ?: 'N/A'
    ];
}

function getNetworkInfo() {
    $interfaces = [];
    $interfaceList = explode("\n", trim(executeCommand("ls /sys/class/net/")));
    foreach ($interfaceList as $interface) {
        if (empty($interface) || $interface === 'lo') {
            continue;
        }
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
        $type = 'Unknown';
        if (strpos($interface, 'eth') !== false || strpos($interface, 'en') !== false) {
            $type = 'Ethernet';
        } elseif (strpos($interface, 'wlan') !== false || strpos($interface, 'wl') !== false) {
            $type = 'Wireless';
        } elseif (strpos($interface, 'docker') !== false || strpos($interface, 'veth') !== false || strpos($interface, 'br') !== false) {
            $type = 'Virtual';
        }
        $interfaces[] = [
            'name' => $interface,
            'status' => $isUp ? 'Connected' : 'Disconnected',
            'is_up' => $isUp,
            'type' => $type,
            'ip' => $ip ?: 'N/A',
            'mac' => $mac ?: 'N/A',
            'speed' => $speed,
            'rx_bytes' => $rxBytes,
            'rx_formatted' => formatBytes($rxBytes),
            'tx_bytes' => $txBytes,
            'tx_formatted' => formatBytes($txBytes)
        ];
    }
    return $interfaces;
}

$stats = [
    'cpu' => getCpuInfo(),
    'memory' => getMemoryInfo(),
    'swap' => getSwapInfo(),
    'uptime' => getUptime(),
    'motherboard' => getMotherboardInfo(),
    'network' => getNetworkInfo(),
    'timestamp' => time()
];

echo json_encode($stats, JSON_PRETTY_PRINT);
EOFAPI

echo "✓ API endpoint installed"

# Check if dashboard already has monitoring - update it
if grep -q "system-monitoring-grid" "$PROJECT_ROOT/public/dashboard.php" 2>/dev/null; then
    echo "✓ Dashboard monitoring UI already present"
else
    echo "→ Dashboard needs monitoring UI update"
    echo "  (Manual update required - monitoring UI should already be in dashboard.php)"
fi

# Check CSS
if grep -q "system-monitoring-grid" "$PROJECT_ROOT/public/css/style.css" 2>/dev/null; then
    echo "✓ CSS styles already present"
else
    echo "  (CSS styles should already be in style.css)"
fi

# Check JavaScript
if grep -q "fetchSystemStats" "$PROJECT_ROOT/public/js/main.js" 2>/dev/null; then
    echo "✓ JavaScript monitoring code already present"
else
    echo "  (JavaScript should already be in main.js)"
fi

echo ""
echo "===================================="
if [ "$MODE" = "update" ]; then
    echo "✓ Monitoring system UPDATED!"
else
    echo "✓ Monitoring system INSTALLED!"
fi
echo "===================================="
echo ""
echo "The monitoring dashboard is ready!"
echo "Log in to view real-time system statistics."
echo ""

