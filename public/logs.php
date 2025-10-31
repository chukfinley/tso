<?php
require_once __DIR__ . '/../config/config.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/User.php';
require_once SRC_PATH . '/Auth.php';
require_once SRC_PATH . '/Logger.php';

$auth = new Auth();
$auth->requireLogin();

$currentUser = $auth->getCurrentUser();
$pageTitle = 'Logs';

$logger = Logger::getInstance();
$db = Database::getInstance();

// Get filter parameters
$filterLevel = $_GET['level'] ?? '';
$filterSearch = $_GET['search'] ?? '';
$filterStartDate = $_GET['start_date'] ?? '';
$filterEndDate = $_GET['end_date'] ?? '';
$page = max(1, intval($_GET['page'] ?? 1));
$limit = 50;
$offset = ($page - 1) * $limit;

// Build filters
$filters = [];
if (!empty($filterLevel)) {
    $filters['level'] = $filterLevel;
}
if (!empty($filterSearch)) {
    $filters['search'] = $filterSearch;
}
if (!empty($filterStartDate)) {
    $filters['start_date'] = $filterStartDate;
}
if (!empty($filterEndDate)) {
    $filters['end_date'] = $filterEndDate;
}

// Get logs
$logs = $logger->getLogs($filters, $limit, $offset);
$totalLogs = $logger->getLogCount($filters);
$totalPages = ceil($totalLogs / $limit);

// Get log statistics
$stats = [
    'total' => $db->fetchOne("SELECT COUNT(*) as count FROM system_logs")['count'] ?? 0,
    'errors' => $db->fetchOne("SELECT COUNT(*) as count FROM system_logs WHERE level = 'error'")['count'] ?? 0,
    'warnings' => $db->fetchOne("SELECT COUNT(*) as count FROM system_logs WHERE level = 'warning'")['count'] ?? 0,
    'info' => $db->fetchOne("SELECT COUNT(*) as count FROM system_logs WHERE level = 'info'")['count'] ?? 0,
    'debug' => $db->fetchOne("SELECT COUNT(*) as count FROM system_logs WHERE level = 'debug'")['count'] ?? 0,
];

// Get recent errors (last 24 hours)
$recentErrors = $db->fetchAll(
    "SELECT COUNT(*) as count FROM system_logs 
     WHERE level = 'error' AND created_at >= DATE_SUB(NOW(), INTERVAL 24 HOUR)"
)[0]['count'] ?? 0;
?>

<?php include VIEWS_PATH . '/layout/header.php'; ?>
<?php include VIEWS_PATH . '/layout/navbar.php'; ?>

<div class="container" style="margin-top: 80px;">
    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 30px;">
        <h1 style="color: #fff; margin: 0;">System Logs</h1>
        <div style="display: flex; gap: 10px;">
            <button id="refresh-logs" class="btn btn-secondary">🔄 Refresh</button>
            <?php if (isset($_SESSION['role']) && $_SESSION['role'] === 'admin'): ?>
            <button id="clear-old-logs" class="btn btn-secondary">🗑️ Clear Old Logs</button>
            <?php endif; ?>
        </div>
    </div>

    <!-- Statistics Cards -->
    <div class="stats-grid" style="margin-bottom: 30px;">
        <div class="stat-card" style="background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);">
            <div class="stat-content">
                <h3><?php echo number_format($stats['total']); ?></h3>
                <p>Total Logs</p>
            </div>
            <div class="stat-icon">📋</div>
        </div>

        <div class="stat-card" style="background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);">
            <div class="stat-content">
                <h3><?php echo number_format($stats['errors']); ?></h3>
                <p>Errors</p>
            </div>
            <div class="stat-icon">❌</div>
        </div>

        <div class="stat-card" style="background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);">
            <div class="stat-content">
                <h3><?php echo number_format($stats['warnings']); ?></h3>
                <p>Warnings</p>
            </div>
            <div class="stat-icon">⚠️</div>
        </div>

        <div class="stat-card" style="background: linear-gradient(135deg, #30cfd0 0%, #330867 100%);">
            <div class="stat-content">
                <h3><?php echo number_format($stats['info']); ?></h3>
                <p>Info</p>
            </div>
            <div class="stat-icon">ℹ️</div>
        </div>

        <div class="stat-card" style="background: linear-gradient(135deg, #a8edea 0%, #fed6e3 100%);">
            <div class="stat-content">
                <h3><?php echo number_format($recentErrors); ?></h3>
                <p>Errors (24h)</p>
            </div>
            <div class="stat-icon">🔴</div>
        </div>
    </div>

    <!-- Filters -->
    <div class="card" style="margin-bottom: 30px;">
        <div class="card-header">Filters</div>
        <div class="card-body">
            <form id="filter-form" method="GET" style="display: grid; grid-template-columns: repeat(auto-fit, minmax(200px, 1fr)); gap: 15px; align-items: end;">
                <div>
                    <label style="display: block; margin-bottom: 5px; color: #b0b0b0; font-size: 13px;">Level</label>
                    <select name="level" id="filter-level" class="form-control" style="width: 100%;">
                        <option value="">All Levels</option>
                        <option value="error" <?php echo $filterLevel === 'error' ? 'selected' : ''; ?>>Error</option>
                        <option value="warning" <?php echo $filterLevel === 'warning' ? 'selected' : ''; ?>>Warning</option>
                        <option value="info" <?php echo $filterLevel === 'info' ? 'selected' : ''; ?>>Info</option>
                        <option value="debug" <?php echo $filterLevel === 'debug' ? 'selected' : ''; ?>>Debug</option>
                    </select>
                </div>
                <div>
                    <label style="display: block; margin-bottom: 5px; color: #b0b0b0; font-size: 13px;">Search</label>
                    <input type="text" name="search" id="filter-search" class="form-control" 
                           placeholder="Search logs..." value="<?php echo htmlspecialchars($filterSearch); ?>" 
                           style="width: 100%;">
                </div>
                <div>
                    <label style="display: block; margin-bottom: 5px; color: #b0b0b0; font-size: 13px;">Start Date</label>
                    <input type="date" name="start_date" id="filter-start-date" class="form-control" 
                           value="<?php echo htmlspecialchars($filterStartDate); ?>" style="width: 100%;">
                </div>
                <div>
                    <label style="display: block; margin-bottom: 5px; color: #b0b0b0; font-size: 13px;">End Date</label>
                    <input type="date" name="end_date" id="filter-end-date" class="form-control" 
                           value="<?php echo htmlspecialchars($filterEndDate); ?>" style="width: 100%;">
                </div>
                <div>
                    <button type="submit" class="btn btn-primary" style="width: 100%;">Apply Filters</button>
                </div>
            </form>
            <?php if (!empty($filterLevel) || !empty($filterSearch) || !empty($filterStartDate) || !empty($filterEndDate)): ?>
            <div style="margin-top: 15px;">
                <a href="/logs.php" class="btn btn-secondary" style="font-size: 12px;">Clear Filters</a>
            </div>
            <?php endif; ?>
        </div>
    </div>

    <!-- Logs Table -->
    <div class="card">
        <div class="card-header">
            Logs (<?php echo number_format($totalLogs); ?> total)
        </div>
        <div class="card-body" style="padding: 0;">
            <?php if (empty($logs)): ?>
                <div style="padding: 40px; text-align: center; color: #666;">
                    <p>No logs found matching your filters.</p>
                </div>
            <?php else: ?>
                <div style="overflow-x: auto;">
                    <table class="table" style="margin: 0;">
                        <thead>
                            <tr>
                                <th style="width: 80px;">Level</th>
                                <th style="width: 150px;">Time</th>
                                <th style="width: 120px;">User</th>
                                <th style="width: 150px;">IP Address</th>
                                <th>Full Log Details</th>
                            </tr>
                        </thead>
                        <tbody id="logs-table-body">
                            <?php foreach ($logs as $log): 
                                $context = [];
                                if (!empty($log['context'])) {
                                    $context = json_decode($log['context'], true) ?: [];
                                }
                                $rawEntry = $context['entry'] ?? '';
                                $source = $context['source'] ?? '';
                                $logType = $context['type'] ?? '';
                            ?>
                                <tr data-log-id="<?php echo $log['id']; ?>" class="log-row">
                                    <td>
                                        <span class="log-badge log-badge-<?php echo htmlspecialchars($log['level']); ?>">
                                            <?php echo strtoupper($log['level']); ?>
                                        </span>
                                    </td>
                                    <td style="font-size: 12px; color: #b0b0b0; white-space: nowrap;">
                                        <?php echo date('Y-m-d H:i:s', strtotime($log['created_at'])); ?>
                                    </td>
                                    <td>
                                        <?php echo htmlspecialchars($log['username'] ?? 'System'); ?>
                                    </td>
                                    <td style="font-size: 12px; color: #b0b0b0; font-family: monospace;">
                                        <?php echo htmlspecialchars($log['ip_address'] ?? 'N/A'); ?>
                                    </td>
                                    <td>
                                        <div class="log-details-full">
                                            <div class="log-message" style="margin-bottom: 10px;">
                                                <strong style="color: #fff;">Message:</strong>
                                                <div style="color: #b0b0b0; margin-top: 5px; white-space: pre-wrap; word-wrap: break-word; font-family: monospace; font-size: 13px;">
                                                    <?php echo htmlspecialchars($log['message']); ?>
                                                </div>
                                            </div>
                                            <?php if (!empty($rawEntry)): ?>
                                            <div class="log-entry" style="margin-bottom: 10px; padding: 10px; background: rgba(255, 140, 0, 0.05); border-left: 3px solid #ff8c00; border-radius: 4px;">
                                                <strong style="color: #ff8c00;">Raw Log Entry:</strong>
                                                <div style="color: #fff; margin-top: 5px; white-space: pre-wrap; word-wrap: break-word; font-family: monospace; font-size: 12px; line-height: 1.6;">
                                                    <?php echo htmlspecialchars($rawEntry); ?>
                                                </div>
                                            </div>
                                            <?php endif; ?>
                                            <?php if (!empty($source) || !empty($logType)): ?>
                                            <div class="log-meta" style="margin-bottom: 10px; font-size: 12px; color: #888;">
                                                <?php if (!empty($source)): ?>
                                                    <span style="margin-right: 15px;"><strong>Source:</strong> <?php echo htmlspecialchars($source); ?></span>
                                                <?php endif; ?>
                                                <?php if (!empty($logType)): ?>
                                                    <span><strong>Type:</strong> <?php echo htmlspecialchars($logType); ?></span>
                                                <?php endif; ?>
                                            </div>
                                            <?php endif; ?>
                                            <?php if (!empty($context) && count($context) > 0): ?>
                                            <div class="log-context">
                                                <button class="btn-toggle-context" data-log-id="<?php echo $log['id']; ?>" 
                                                        style="background: none; border: 1px solid #444; color: #ff8c00; cursor: pointer; font-size: 11px; padding: 5px 10px; border-radius: 4px; margin-top: 5px;">
                                                    <span class="toggle-text">Show Full Context</span>
                                                </button>
                                                <div class="context-details" id="context-<?php echo $log['id']; ?>" style="display: none; margin-top: 10px; padding: 10px; background: #0a0a0a; border-radius: 4px; border: 1px solid #333;">
                                                    <pre style="color: #fff; font-size: 11px; line-height: 1.6; white-space: pre-wrap; word-wrap: break-word; margin: 0;"><?php echo htmlspecialchars(json_encode($context, JSON_PRETTY_PRINT | JSON_UNESCAPED_SLASHES)); ?></pre>
                                                </div>
                                            </div>
                                            <?php endif; ?>
                                        </div>
                                    </td>
                                </tr>
                            <?php endforeach; ?>
                        </tbody>
                    </table>
                </div>

                <!-- Pagination -->
                <?php if ($totalPages > 1): ?>
                <div style="padding: 20px; display: flex; justify-content: space-between; align-items: center; border-top: 1px solid #333;">
                    <div style="color: #b0b0b0; font-size: 13px;">
                        Page <?php echo $page; ?> of <?php echo $totalPages; ?> 
                        (<?php echo number_format($totalLogs); ?> total logs)
                    </div>
                    <div style="display: flex; gap: 10px;">
                        <?php if ($page > 1): ?>
                            <a href="?page=<?php echo $page - 1; ?><?php echo !empty($filterLevel) ? '&level=' . urlencode($filterLevel) : ''; ?><?php echo !empty($filterSearch) ? '&search=' . urlencode($filterSearch) : ''; ?><?php echo !empty($filterStartDate) ? '&start_date=' . urlencode($filterStartDate) : ''; ?><?php echo !empty($filterEndDate) ? '&end_date=' . urlencode($filterEndDate) : ''; ?>" 
                               class="btn btn-secondary">Previous</a>
                        <?php endif; ?>
                        <?php if ($page < $totalPages): ?>
                            <a href="?page=<?php echo $page + 1; ?><?php echo !empty($filterLevel) ? '&level=' . urlencode($filterLevel) : ''; ?><?php echo !empty($filterSearch) ? '&search=' . urlencode($filterSearch) : ''; ?><?php echo !empty($filterStartDate) ? '&start_date=' . urlencode($filterStartDate) : ''; ?><?php echo !empty($filterEndDate) ? '&end_date=' . urlencode($filterEndDate) : ''; ?>" 
                               class="btn btn-secondary">Next</a>
                        <?php endif; ?>
                    </div>
                </div>
                <?php endif; ?>
            <?php endif; ?>
        </div>
    </div>
</div>

<style>
.log-badge {
    display: inline-block;
    padding: 4px 10px;
    border-radius: 12px;
    font-size: 11px;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.5px;
}

.log-badge-error {
    background: rgba(244, 67, 54, 0.2);
    color: #f44336;
    border: 1px solid #f44336;
}

.log-badge-warning {
    background: rgba(255, 152, 0, 0.2);
    color: #ff9800;
    border: 1px solid #ff9800;
}

.log-badge-info {
    background: rgba(33, 150, 243, 0.2);
    color: #2196f3;
    border: 1px solid #2196f3;
}

.log-badge-debug {
    background: rgba(156, 39, 176, 0.2);
    color: #9c27b0;
    border: 1px solid #9c27b0;
}

#logs-table-body tr:hover {
    background: rgba(255, 140, 0, 0.05);
}

.log-row td {
    padding: 15px !important;
    vertical-align: top;
}

.log-details-full {
    min-width: 500px;
    max-width: 100%;
}

.btn-toggle-context:hover {
    background: rgba(255, 140, 0, 0.1) !important;
    border-color: #ff8c00 !important;
}
</style>

<script>
document.addEventListener('DOMContentLoaded', function() {
    // Refresh logs button
    const refreshBtn = document.getElementById('refresh-logs');
    if (refreshBtn) {
        refreshBtn.addEventListener('click', function() {
            window.location.reload();
        });
    }

    // Clear old logs button
    const clearBtn = document.getElementById('clear-old-logs');
    if (clearBtn) {
        clearBtn.addEventListener('click', function() {
            if (confirm('Are you sure you want to clear logs older than 30 days? This action cannot be undone.')) {
                fetch('/api/logs.php?action=clear_old')
                    .then(response => response.json())
                    .then(data => {
                        if (data.success) {
                            alert('Old logs cleared successfully');
                            window.location.reload();
                        } else {
                            alert('Failed to clear logs: ' + (data.error || 'Unknown error'));
                        }
                    })
                    .catch(error => {
                        alert('Error: ' + error.message);
                    });
            }
        });
    }

    // Toggle context details
    const toggleButtons = document.querySelectorAll('.btn-toggle-context');
    toggleButtons.forEach(button => {
        button.addEventListener('click', function() {
            const logId = this.getAttribute('data-log-id');
            const contextDiv = document.getElementById('context-' + logId);
            const toggleText = this.querySelector('.toggle-text');
            
            if (contextDiv) {
                if (contextDiv.style.display === 'none') {
                    contextDiv.style.display = 'block';
                    toggleText.textContent = 'Hide Full Context';
                } else {
                    contextDiv.style.display = 'none';
                    toggleText.textContent = 'Show Full Context';
                }
            }
        });
    });

    // Auto-refresh every 30 seconds
    setInterval(function() {
        if (document.visibilityState === 'visible') {
            window.location.reload();
        }
    }, 30000);
});
</script>

<?php include VIEWS_PATH . '/layout/footer.php'; ?>

