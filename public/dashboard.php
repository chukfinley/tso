<?php
require_once __DIR__ . '/../config/config.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/User.php';
require_once SRC_PATH . '/Auth.php';

$auth = new Auth();
$auth->requireLogin();

$currentUser = $auth->getCurrentUser();
$pageTitle = 'Dashboard';

// Get system stats (placeholder data for now)
$stats = [
    'total_disks' => 0,
    'total_shares' => 0,
    'total_users' => 0,
    'running_containers' => 0,
    'running_vms' => 0,
];

// Get actual user count
$db = Database::getInstance();
$stats['total_users'] = $db->fetchOne("SELECT COUNT(*) as count FROM users")['count'] ?? 0;

// Helper functions for system monitoring
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

// Get initial system stats for page load
function getCpuInfo() {
    $cpuModel = trim(executeCommand("lscpu | grep 'Model name' | cut -d ':' -f2 | xargs 2>/dev/null") ?: executeCommand("cat /proc/cpuinfo | grep 'model name' | head -n1 | cut -d ':' -f2 | xargs 2>/dev/null"));
    $cores = trim(executeCommand("nproc 2>/dev/null")) ?: '1';
    $architecture = trim(executeCommand("uname -m 2>/dev/null")) ?: 'Unknown';
    $maxFreq = trim(executeCommand("lscpu | grep 'CPU max MHz' | awk '{print $4}' 2>/dev/null"));
    if (empty($maxFreq)) {
        $maxFreq = trim(executeCommand("cat /proc/cpuinfo | grep 'cpu MHz' | head -n1 | awk '{print $4}' 2>/dev/null"));
    }
    $cpuUsage = trim(executeCommand("top -bn1 | grep 'Cpu(s)' | sed 's/.*, *\\([0-9.]*\\)%* id.*/\\1/' | awk '{print 100 - $1}' 2>/dev/null"));
    $loadAvg = sys_getloadavg();
    return [
        'model' => $cpuModel ?: 'Unknown',
        'cores' => (int)$cores,
        'architecture' => $architecture,
        'frequency' => $maxFreq ? round((float)$maxFreq, 2) . ' MHz' : 'N/A',
        'usage' => round((float)$cpuUsage, 1),
        'load_avg' => [
            '1min' => round($loadAvg[0], 2),
            '5min' => round($loadAvg[1], 2),
            '15min' => round($loadAvg[2], 2)
        ]
    ];
}

function getMemoryInfo() {
    $memData = @file_get_contents('/proc/meminfo');
    if (!$memData) return null;
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
        'available' => $available,
        'available_formatted' => formatBytes($available),
        'usage_percent' => round($usagePercent, 1)
    ];
}

function getSwapInfo() {
    $memData = @file_get_contents('/proc/meminfo');
    if (!$memData) return null;
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
        'usage_percent' => round($usagePercent, 1)
    ];
}

function getUptime() {
    $uptimeSeconds = (int)trim(executeCommand("cat /proc/uptime 2>/dev/null | awk '{print $1}'"));
    $days = floor($uptimeSeconds / 86400);
    $hours = floor(($uptimeSeconds % 86400) / 3600);
    $minutes = floor(($uptimeSeconds % 3600) / 60);
    $parts = [];
    if ($days > 0) $parts[] = $days . 'd';
    if ($hours > 0) $parts[] = $hours . 'h';
    if ($minutes > 0) $parts[] = $minutes . 'm';
    return implode(' ', $parts) ?: '0m';
}

function getMotherboardInfo() {
    $vendor = trim(@file_get_contents("/sys/class/dmi/id/board_vendor"));
    $name = trim(@file_get_contents("/sys/class/dmi/id/board_name"));
    return ($vendor && $name && $vendor != 'Unknown' && $name != 'Unknown') ? "$vendor $name" : 'Unknown';
}

// Fetch initial data
$cpuInfo = getCpuInfo();
$memInfo = getMemoryInfo();
$swapInfo = getSwapInfo();
$uptime = getUptime();
$motherboard = getMotherboardInfo();
?>

<?php include VIEWS_PATH . '/layout/header.php'; ?>
<?php include VIEWS_PATH . '/layout/navbar.php'; ?>

<div class="container">
    <h1 style="margin-bottom: 30px; color: #fff;">Welcome, <?php echo htmlspecialchars($currentUser['full_name'] ?? $currentUser['username']); ?>!</h1>

    <!-- Stats Grid -->
    <div class="stats-grid">
        <div class="stat-card">
            <div class="stat-content">
                <h3><?php echo $stats['total_disks']; ?></h3>
                <p>Total Disks</p>
            </div>
            <div class="stat-icon">💾</div>
        </div>

        <div class="stat-card">
            <div class="stat-content">
                <h3><?php echo $stats['total_shares']; ?></h3>
                <p>Shares</p>
            </div>
            <div class="stat-icon">📁</div>
        </div>

        <div class="stat-card">
            <div class="stat-content">
                <h3><?php echo $stats['total_users']; ?></h3>
                <p>Users</p>
            </div>
            <div class="stat-icon">👥</div>
        </div>

        <div class="stat-card">
            <div class="stat-content">
                <h3><?php echo $stats['running_containers']; ?></h3>
                <p>Docker Containers</p>
            </div>
            <div class="stat-icon">🐳</div>
        </div>
    </div>

    <!-- System Monitoring Grid -->
    <div class="system-monitoring-grid">
        <!-- CPU Card -->
        <div class="card monitor-card">
            <div class="card-header">
                <span class="monitor-icon">⚙️</span> CPU
            </div>
            <div class="card-body">
                <div class="monitor-stat">
                    <div class="stat-label">Model</div>
                    <div class="stat-value" id="cpu-model"><?php echo htmlspecialchars($cpuInfo['model']); ?></div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Cores / Architecture</div>
                    <div class="stat-value" id="cpu-cores"><?php echo $cpuInfo['cores']; ?> Cores / <?php echo htmlspecialchars($cpuInfo['architecture']); ?></div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Frequency</div>
                    <div class="stat-value" id="cpu-freq"><?php echo htmlspecialchars($cpuInfo['frequency']); ?></div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Usage</div>
                    <div class="progress-container">
                        <div class="progress-bar<?php if($cpuInfo['usage'] > 80) echo ' danger'; elseif($cpuInfo['usage'] > 60) echo ' warning'; ?>" id="cpu-usage-bar" style="width: <?php echo $cpuInfo['usage']; ?>%"></div>
                        <div class="progress-text" id="cpu-usage-text"><?php echo $cpuInfo['usage']; ?>%</div>
                    </div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Load Average (1m / 5m / 15m)</div>
                    <div class="stat-value" id="cpu-load"><?php echo $cpuInfo['load_avg']['1min']; ?> / <?php echo $cpuInfo['load_avg']['5min']; ?> / <?php echo $cpuInfo['load_avg']['15min']; ?></div>
                </div>
            </div>
        </div>

        <!-- Memory Card -->
        <div class="card monitor-card">
            <div class="card-header">
                <span class="monitor-icon">💾</span> Memory (RAM)
            </div>
            <div class="card-body">
                <div class="monitor-stat">
                    <div class="stat-label">Total Memory</div>
                    <div class="stat-value" id="mem-total"><?php echo $memInfo['total_formatted']; ?></div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Used / Available</div>
                    <div class="stat-value" id="mem-usage"><?php echo $memInfo['used_formatted']; ?> / <?php echo $memInfo['available_formatted']; ?></div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Memory Usage</div>
                    <div class="progress-container">
                        <div class="progress-bar<?php if($memInfo['usage_percent'] > 85) echo ' danger'; elseif($memInfo['usage_percent'] > 70) echo ' warning'; ?>" id="mem-usage-bar" style="width: <?php echo $memInfo['usage_percent']; ?>%"></div>
                        <div class="progress-text" id="mem-usage-text"><?php echo $memInfo['usage_percent']; ?>%</div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Swap Card -->
        <div class="card monitor-card">
            <div class="card-header">
                <span class="monitor-icon">💿</span> Swap
            </div>
            <div class="card-body">
                <div class="monitor-stat">
                    <div class="stat-label">Total Swap</div>
                    <div class="stat-value" id="swap-total"><?php echo $swapInfo['total_formatted']; ?></div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Used / Free</div>
                    <div class="stat-value" id="swap-usage"><?php echo $swapInfo['used_formatted']; ?> / <?php echo formatBytes($swapInfo['total'] - $swapInfo['used']); ?></div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Swap Usage</div>
                    <div class="progress-container">
                        <div class="progress-bar<?php if($swapInfo['usage_percent'] > 75) echo ' danger'; elseif($swapInfo['usage_percent'] > 50) echo ' warning'; ?>" id="swap-usage-bar" style="width: <?php echo $swapInfo['usage_percent']; ?>%"></div>
                        <div class="progress-text" id="swap-usage-text"><?php echo $swapInfo['usage_percent']; ?>%</div>
                    </div>
                </div>
            </div>
        </div>

        <!-- Motherboard & Uptime Card -->
        <div class="card monitor-card">
            <div class="card-header">
                <span class="monitor-icon">🖥️</span> System Info
            </div>
            <div class="card-body">
                <div class="monitor-stat">
                    <div class="stat-label">Motherboard</div>
                    <div class="stat-value" id="mb-info"><?php echo htmlspecialchars($motherboard); ?></div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">System Uptime</div>
                    <div class="stat-value" id="system-uptime"><?php echo htmlspecialchars($uptime); ?></div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Hostname</div>
                    <div class="stat-value"><?php echo gethostname(); ?></div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">System Time</div>
                    <div class="stat-value" id="system-time"><?php echo date('Y-m-d H:i:s'); ?></div>
                </div>
            </div>
        </div>
    </div>

    <!-- Network Interfaces Card -->
    <div class="card">
        <div class="card-header">
            <span class="monitor-icon">🌐</span> Network Interfaces
        </div>
        <div class="card-body">
            <div id="network-interfaces">
                <p style="color: #666; text-align: center;">Loading network information...</p>
            </div>
        </div>
    </div>

    <!-- System Overview -->
    <div class="card">
        <div class="card-header">System Overview</div>
        <div class="card-body">
            <table class="table">
                <tr>
                    <td><strong>Server Software:</strong></td>
                    <td><?php echo $_SERVER['SERVER_SOFTWARE'] ?? 'Unknown'; ?></td>
                </tr>
                <tr>
                    <td><strong>PHP Version:</strong></td>
                    <td><?php echo PHP_VERSION; ?></td>
                </tr>
                <tr>
                    <td><strong>App Version:</strong></td>
                    <td><?php echo APP_VERSION; ?></td>
                </tr>
            </table>
        </div>
    </div>

    <!-- Recent Activity -->
    <div class="card">
        <div class="card-header">Recent Activity</div>
        <div class="card-body">
            <?php
            $recentActivity = $db->fetchAll(
                "SELECT al.*, u.username
                 FROM activity_log al
                 LEFT JOIN users u ON al.user_id = u.id
                 ORDER BY al.created_at DESC
                 LIMIT 10"
            );

            if (empty($recentActivity)):
            ?>
                <p style="color: #666;">No recent activity.</p>
            <?php else: ?>
                <table class="table">
                    <thead>
                        <tr>
                            <th>User</th>
                            <th>Action</th>
                            <th>Description</th>
                            <th>IP Address</th>
                            <th>Time</th>
                        </tr>
                    </thead>
                    <tbody>
                        <?php foreach ($recentActivity as $activity): ?>
                            <tr>
                                <td><?php echo htmlspecialchars($activity['username'] ?? 'System'); ?></td>
                                <td><?php echo htmlspecialchars($activity['action']); ?></td>
                                <td><?php echo htmlspecialchars($activity['description']); ?></td>
                                <td><?php echo htmlspecialchars($activity['ip_address']); ?></td>
                                <td><?php echo date('Y-m-d H:i', strtotime($activity['created_at'])); ?></td>
                            </tr>
                        <?php endforeach; ?>
                    </tbody>
                </table>
            <?php endif; ?>
        </div>
    </div>
</div>

<?php include VIEWS_PATH . '/layout/footer.php'; ?>
