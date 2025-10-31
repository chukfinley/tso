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

<!-- Auto-refresh Status Indicator -->
<div style="position: fixed; bottom: 20px; right: 20px; background: rgba(26, 26, 26, 0.95); border: 1px solid #333; border-radius: 8px; padding: 12px 20px; display: flex; align-items: center; gap: 12px; z-index: 1000; box-shadow: 0 4px 12px rgba(0,0,0,0.3);">
    <div id="refresh-indicator" style="width: 8px; height: 8px; border-radius: 50%; background: #4caf50; animation: pulse 2s infinite;"></div>
    <div style="color: #b0b0b0; font-size: 13px;">
        <span style="color: #fff; font-weight: 600;">Live:</span>
        <span id="last-update">Just now</span>
    </div>
</div>

<style>
@keyframes pulse {
    0%, 100% {
        opacity: 1;
        transform: scale(1);
    }
    50% {
        opacity: 0.6;
        transform: scale(0.8);
    }
}

.progress-bar {
    transition: width 0.3s ease, background-color 0.3s ease;
}

.stat-value, .progress-text {
    transition: color 0.2s ease;
}

.updating {
    animation: fadeUpdate 0.3s ease;
}

@keyframes fadeUpdate {
    0% { opacity: 1; }
    50% { opacity: 0.7; }
    100% { opacity: 1; }
}
</style>

<script>
let refreshInterval;
let lastUpdateTime = Date.now();
let isPageVisible = true;

// Track page visibility
document.addEventListener('visibilitychange', function() {
    isPageVisible = !document.hidden;
    if (isPageVisible) {
        // Resume updates when page becomes visible
        if (!refreshInterval) {
            startAutoRefresh();
        }
    }
});

function updateSystemStats() {
    fetch('/api/system-stats.php')
        .then(response => {
            if (!response.ok) {
                throw new Error('Network response was not ok: ' + response.status);
            }
            return response.json();
        })
        .then(data => {
            console.log('Received data:', data); // Debug log
            
            if (data.error) {
                console.error('API Error:', data.error);
                return;
            }
            
            if (data.cpu) {
                console.log('Updating CPU:', data.cpu);
                updateCpuStats(data.cpu);
            }
            if (data.memory) {
                console.log('Updating Memory:', data.memory);
                updateMemoryStats(data.memory);
            }
            if (data.swap) {
                console.log('Updating Swap:', data.swap);
                updateSwapStats(data.swap);
            }
            if (data.uptime) {
                console.log('Updating Uptime:', data.uptime);
                updateUptime(data.uptime);
            }
            if (data.network) {
                console.log('Updating Network:', data.network.length, 'interfaces');
                updateNetworkInterfaces(data.network);
            }

            // Update timestamp
            lastUpdateTime = Date.now();
            updateLastUpdateText();

            // Flash indicator
            flashRefreshIndicator();
        })
        .catch(error => {
            console.error('Error fetching system stats:', error);
            const indicator = document.getElementById('refresh-indicator');
            if (indicator) {
                indicator.style.background = '#f44336';
            }
        });
}

function updateCpuStats(cpu) {
    try {
        console.log('updateCpuStats called with:', cpu);
        
        // Update CPU usage
        const cpuUsageBar = document.getElementById('cpu-usage-bar');
        const cpuUsageText = document.getElementById('cpu-usage-text');

        if (cpuUsageBar && cpuUsageText && cpu.usage !== undefined) {
            cpuUsageBar.style.width = cpu.usage + '%';
            cpuUsageText.textContent = cpu.usage + '%';

            // Update color based on usage
            cpuUsageBar.className = 'progress-bar';
            if (cpu.usage > 80) {
                cpuUsageBar.classList.add('danger');
            } else if (cpu.usage > 60) {
                cpuUsageBar.classList.add('warning');
            }

            // Add update animation
            cpuUsageText.classList.add('updating');
            setTimeout(() => cpuUsageText.classList.remove('updating'), 300);
        } else {
            console.warn('CPU usage elements not found or data missing', {
                cpuUsageBar: !!cpuUsageBar,
                cpuUsageText: !!cpuUsageText,
                cpuUsage: cpu.usage
            });
        }

        // Update load average
        const cpuLoad = document.getElementById('cpu-load');
        if (cpuLoad && cpu.load_avg) {
            cpuLoad.textContent = `${cpu.load_avg['1min']} / ${cpu.load_avg['5min']} / ${cpu.load_avg['15min']}`;
            cpuLoad.classList.add('updating');
            setTimeout(() => cpuLoad.classList.remove('updating'), 300);
        } else {
            console.warn('CPU load element not found or data missing', {
                cpuLoad: !!cpuLoad,
                loadAvg: cpu.load_avg
            });
        }
    } catch (error) {
        console.error('Error in updateCpuStats:', error);
    }
}

function updateMemoryStats(memory) {
    try {
        console.log('updateMemoryStats called with:', memory);
        
        // Update memory usage
        const memUsageBar = document.getElementById('mem-usage-bar');
        const memUsageText = document.getElementById('mem-usage-text');
        const memUsage = document.getElementById('mem-usage');

        if (memUsageBar && memUsageText && memory.usage_percent !== undefined) {
            memUsageBar.style.width = memory.usage_percent + '%';
            memUsageText.textContent = memory.usage_percent + '%';

            // Update color based on usage
            memUsageBar.className = 'progress-bar';
            if (memory.usage_percent > 85) {
                memUsageBar.classList.add('danger');
            } else if (memory.usage_percent > 70) {
                memUsageBar.classList.add('warning');
            }

            memUsageText.classList.add('updating');
            setTimeout(() => memUsageText.classList.remove('updating'), 300);
        } else {
            console.warn('Memory usage elements not found or data missing', {
                memUsageBar: !!memUsageBar,
                memUsageText: !!memUsageText,
                usagePercent: memory.usage_percent
            });
        }

        if (memUsage && memory.used_formatted && memory.available_formatted) {
            memUsage.textContent = `${memory.used_formatted} / ${memory.available_formatted}`;
            memUsage.classList.add('updating');
            setTimeout(() => memUsage.classList.remove('updating'), 300);
        } else {
            console.warn('Memory usage text element not found or data missing', {
                memUsage: !!memUsage,
                used: memory.used_formatted,
                available: memory.available_formatted
            });
        }
    } catch (error) {
        console.error('Error in updateMemoryStats:', error);
    }
}

function updateSwapStats(swap) {
    try {
        console.log('updateSwapStats called with:', swap);
        
        // Update swap usage
        const swapUsageBar = document.getElementById('swap-usage-bar');
        const swapUsageText = document.getElementById('swap-usage-text');
        const swapUsage = document.getElementById('swap-usage');

        if (swapUsageBar && swapUsageText && swap.usage_percent !== undefined) {
            swapUsageBar.style.width = swap.usage_percent + '%';
            swapUsageText.textContent = swap.usage_percent + '%';

            // Update color based on usage
            swapUsageBar.className = 'progress-bar';
            if (swap.usage_percent > 75) {
                swapUsageBar.classList.add('danger');
            } else if (swap.usage_percent > 50) {
                swapUsageBar.classList.add('warning');
            }

            swapUsageText.classList.add('updating');
            setTimeout(() => swapUsageText.classList.remove('updating'), 300);
        } else {
            console.warn('Swap usage elements not found or data missing', {
                swapUsageBar: !!swapUsageBar,
                swapUsageText: !!swapUsageText,
                usagePercent: swap.usage_percent
            });
        }

        if (swapUsage && swap.used_formatted && swap.free_formatted) {
            swapUsage.textContent = `${swap.used_formatted} / ${swap.free_formatted}`;
            swapUsage.classList.add('updating');
            setTimeout(() => swapUsage.classList.remove('updating'), 300);
        } else {
            console.warn('Swap usage text element not found or data missing', {
                swapUsage: !!swapUsage,
                used: swap.used_formatted,
                free: swap.free_formatted
            });
        }
    } catch (error) {
        console.error('Error in updateSwapStats:', error);
    }
}

function updateUptime(uptime) {
    try {
        console.log('updateUptime called with:', uptime);
        
        const uptimeEl = document.getElementById('system-uptime');
        if (uptimeEl && uptime.formatted) {
            uptimeEl.textContent = uptime.formatted;
            uptimeEl.classList.add('updating');
            setTimeout(() => uptimeEl.classList.remove('updating'), 300);
        } else {
            console.warn('Uptime element not found or data missing', {
                uptimeEl: !!uptimeEl,
                formatted: uptime.formatted
            });
        }

        // Update system time
        const timeEl = document.getElementById('system-time');
        if (timeEl) {
            const now = new Date();
            const formattedTime = now.getFullYear() + '-' +
                String(now.getMonth() + 1).padStart(2, '0') + '-' +
                String(now.getDate()).padStart(2, '0') + ' ' +
                String(now.getHours()).padStart(2, '0') + ':' +
                String(now.getMinutes()).padStart(2, '0') + ':' +
                String(now.getSeconds()).padStart(2, '0');
            timeEl.textContent = formattedTime;
        } else {
            console.warn('System time element not found');
        }
    } catch (error) {
        console.error('Error in updateUptime:', error);
    }
}

function updateNetworkInterfaces(interfaces) {
    try {
        console.log('updateNetworkInterfaces called with:', interfaces);
        
        const container = document.getElementById('network-interfaces');
        if (!container) {
            console.warn('Network interfaces container not found');
            return;
        }

        if (!interfaces || interfaces.length === 0) {
            container.innerHTML = '<p style="color: #666; text-align: center;">No network interfaces found</p>';
            return;
        }

        let html = '<div style="display: grid; gap: 15px;">';

        interfaces.forEach(iface => {
            const statusColor = iface.is_up ? '#4caf50' : '#f44336';
            const statusIcon = iface.is_up ? '●' : '○';

            html += `
                <div style="background: #1a1a1a; border: 1px solid #333; border-radius: 6px; padding: 15px;">
                    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 12px;">
                        <div style="display: flex; align-items: center; gap: 10px;">
                            <span style="font-weight: 600; color: #ff8c00; font-size: 16px;">${iface.name || 'Unknown'}</span>
                            <span style="color: ${statusColor}; font-size: 14px;">${statusIcon} ${iface.status || 'Unknown'}</span>
                        </div>
                        <span style="background: rgba(255, 140, 0, 0.2); color: #ff8c00; padding: 4px 10px; border-radius: 12px; font-size: 12px; font-weight: 600;">${iface.type || 'Unknown'}</span>
                    </div>
                    <div style="display: grid; grid-template-columns: repeat(2, 1fr); gap: 10px; font-size: 13px;">
                        <div>
                            <span style="color: #888;">IP Address:</span>
                            <span style="color: #fff; margin-left: 8px;">${iface.ip || 'N/A'}</span>
                        </div>
                        <div>
                            <span style="color: #888;">Speed:</span>
                            <span style="color: #fff; margin-left: 8px;">${iface.speed || 'N/A'}</span>
                        </div>
                        <div>
                            <span style="color: #888;">MAC:</span>
                            <span style="color: #fff; margin-left: 8px; font-family: monospace; font-size: 12px;">${iface.mac || 'N/A'}</span>
                        </div>
                        <div>
                            <span style="color: #888;">Traffic:</span>
                            <span style="color: #4caf50; margin-left: 8px;">↓ ${iface.rx_formatted || '0 B'}</span>
                            <span style="color: #f44336; margin-left: 8px;">↑ ${iface.tx_formatted || '0 B'}</span>
                        </div>
                    </div>
                </div>
            `;
        });

        html += '</div>';
        container.innerHTML = html;
    } catch (error) {
        console.error('Error in updateNetworkInterfaces:', error);
    }
}

function updateLastUpdateText() {
    const lastUpdateEl = document.getElementById('last-update');
    if (!lastUpdateEl) return;

    const secondsAgo = Math.floor((Date.now() - lastUpdateTime) / 1000);

    if (secondsAgo < 2) {
        lastUpdateEl.textContent = 'Just now';
    } else if (secondsAgo < 60) {
        lastUpdateEl.textContent = secondsAgo + 's ago';
    } else {
        lastUpdateEl.textContent = Math.floor(secondsAgo / 60) + 'm ago';
    }
}

function flashRefreshIndicator() {
    const indicator = document.getElementById('refresh-indicator');
    if (!indicator) return;

    indicator.style.background = '#4caf50';
    indicator.style.transform = 'scale(1.3)';

    setTimeout(() => {
        indicator.style.transform = 'scale(1)';
    }, 150);
}

function startAutoRefresh() {
    console.log('Starting auto-refresh - updates every 1 second');
    
    // Update immediately
    updateSystemStats();

    // Then update every second
    refreshInterval = setInterval(() => {
        if (isPageVisible) {
            updateSystemStats();
        } else {
            console.log('Page not visible, skipping update');
        }
    }, 1000);

    // Update "last update" text more frequently
    setInterval(updateLastUpdateText, 1000);
    
    console.log('Auto-refresh started successfully');
}

// Start auto-refresh when page loads
document.addEventListener('DOMContentLoaded', function() {
    console.log('Dashboard loaded, initializing auto-refresh...');
    startAutoRefresh();
});

// Clean up when page is being unloaded
window.addEventListener('beforeunload', function() {
    if (refreshInterval) {
        clearInterval(refreshInterval);
    }
});
</script>

<?php include VIEWS_PATH . '/layout/footer.php'; ?>
