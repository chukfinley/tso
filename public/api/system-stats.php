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

// Get CPU information
function getCpuInfo() {
    $cpuInfo = [];
    
    // CPU model
    $cpuModel = trim(executeCommand("lscpu | grep 'Model name' | cut -d ':' -f2 | xargs"));
    if (empty($cpuModel)) {
        $cpuModel = trim(executeCommand("cat /proc/cpuinfo | grep 'model name' | head -n1 | cut -d ':' -f2 | xargs"));
    }
    
    // CPU cores
    $cores = trim(executeCommand("nproc"));
    
    // CPU architecture
    $architecture = trim(executeCommand("uname -m"));
    
    // CPU frequency
    $maxFreq = trim(executeCommand("lscpu | grep 'CPU max MHz' | awk '{print $4}'"));
    if (empty($maxFreq)) {
        $maxFreq = trim(executeCommand("cat /proc/cpuinfo | grep 'cpu MHz' | head -n1 | awk '{print $4}'"));
    }
    
    // CPU usage
    $cpuUsage = trim(executeCommand("top -bn1 | grep 'Cpu(s)' | sed 's/.*, *\\([0-9.]*\\)%* id.*/\\1/' | awk '{print 100 - $1}'"));
    if (empty($cpuUsage)) {
        // Alternative method
        $cpuUsage = trim(executeCommand("mpstat 1 1 | awk '/Average/ {print 100-$NF}'"));
    }
    
    // Load average
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

// Get memory information
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

// Get swap information
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

// Get system uptime
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

// Get motherboard information
function getMotherboardInfo() {
    $vendor = trim(executeCommand("cat /sys/class/dmi/id/board_vendor 2>/dev/null"));
    $name = trim(executeCommand("cat /sys/class/dmi/id/board_name 2>/dev/null"));
    $version = trim(executeCommand("cat /sys/class/dmi/id/board_version 2>/dev/null"));
    
    // Try alternative method if above fails
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

// Get network information
function getNetworkInfo() {
    $interfaces = [];
    
    // Get list of network interfaces
    $interfaceList = explode("\n", trim(executeCommand("ls /sys/class/net/")));
    
    foreach ($interfaceList as $interface) {
        if (empty($interface) || $interface === 'lo') {
            continue;
        }
        
        // Check if interface is up
        $operstate = trim(@file_get_contents("/sys/class/net/$interface/operstate"));
        $isUp = ($operstate === 'up');
        
        // Get MAC address
        $mac = trim(@file_get_contents("/sys/class/net/$interface/address"));
        
        // Get IP address
        $ip = trim(executeCommand("ip addr show $interface | grep 'inet ' | awk '{print $2}' | cut -d'/' -f1"));
        
        // Get statistics
        $rxBytes = (int)@file_get_contents("/sys/class/net/$interface/statistics/rx_bytes");
        $txBytes = (int)@file_get_contents("/sys/class/net/$interface/statistics/tx_bytes");
        
        // Get speed (in Mbps)
        $speed = trim(@file_get_contents("/sys/class/net/$interface/speed"));
        if ($speed === '' || $speed === '-1' || $speed === '4294967295') {
            $speed = 'N/A';
        } else {
            $speed = $speed . ' Mbps';
        }
        
        // Get connection type
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

// Collect all system stats
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

