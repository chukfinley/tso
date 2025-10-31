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

    <!-- System Overview -->
    <div class="card">
        <div class="card-header">System Overview</div>
        <div class="card-body">
            <table class="table">
                <tr>
                    <td><strong>Hostname:</strong></td>
                    <td><?php echo gethostname(); ?></td>
                </tr>
                <tr>
                    <td><strong>System Time:</strong></td>
                    <td><?php echo date('Y-m-d H:i:s'); ?></td>
                </tr>
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
