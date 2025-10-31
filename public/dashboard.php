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
                    <div class="stat-value" id="cpu-model">Loading...</div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Cores / Architecture</div>
                    <div class="stat-value" id="cpu-cores">Loading...</div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Frequency</div>
                    <div class="stat-value" id="cpu-freq">Loading...</div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Usage</div>
                    <div class="progress-container">
                        <div class="progress-bar" id="cpu-usage-bar" style="width: 0%"></div>
                        <div class="progress-text" id="cpu-usage-text">0%</div>
                    </div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Load Average (1m / 5m / 15m)</div>
                    <div class="stat-value" id="cpu-load">Loading...</div>
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
                    <div class="stat-value" id="mem-total">Loading...</div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Used / Available</div>
                    <div class="stat-value" id="mem-usage">Loading...</div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Memory Usage</div>
                    <div class="progress-container">
                        <div class="progress-bar" id="mem-usage-bar" style="width: 0%"></div>
                        <div class="progress-text" id="mem-usage-text">0%</div>
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
                    <div class="stat-value" id="swap-total">Loading...</div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Used / Free</div>
                    <div class="stat-value" id="swap-usage">Loading...</div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">Swap Usage</div>
                    <div class="progress-container">
                        <div class="progress-bar" id="swap-usage-bar" style="width: 0%"></div>
                        <div class="progress-text" id="swap-usage-text">0%</div>
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
                    <div class="stat-value" id="mb-info">Loading...</div>
                </div>
                <div class="monitor-stat">
                    <div class="stat-label">System Uptime</div>
                    <div class="stat-value" id="system-uptime">Loading...</div>
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
