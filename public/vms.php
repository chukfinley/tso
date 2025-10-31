<?php
require_once __DIR__ . '/../config/config.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/User.php';
require_once SRC_PATH . '/Auth.php';
require_once SRC_PATH . '/VM.php';

$auth = new Auth();
$auth->requireLogin();

$vmManager = new VM();
$db = Database::getInstance();

$message = '';
$messageType = '';

// Handle form submissions
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $action = $_POST['action'] ?? '';

    try {
        if ($action === 'create') {
            $data = [
                'name' => trim($_POST['name']),
                'description' => trim($_POST['description'] ?? ''),
                'cpu_cores' => (int)$_POST['cpu_cores'],
                'ram_mb' => (int)$_POST['ram_mb'],
                'disk_size_gb' => (int)$_POST['disk_size_gb'],
                'disk_format' => $_POST['disk_format'],
                'boot_order' => $_POST['boot_order'],
                'iso_path' => $_POST['iso_path'] ?? null,
                'boot_from_disk' => isset($_POST['boot_from_disk']),
                'physical_disk_device' => $_POST['physical_disk_device'] ?? null,
                'network_mode' => $_POST['network_mode'],
                'network_bridge' => $_POST['network_bridge'] ?? null,
                'display_type' => $_POST['display_type']
            ];

            $vmId = $vmManager->create($data);
            $message = 'VM created successfully!';
            $messageType = 'success';

            // Log activity
            $db->query(
                "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                [$_SESSION['user_id'], 'vm_create', "Created VM: {$data['name']}", $_SERVER['REMOTE_ADDR']]
            );
        } elseif ($action === 'update') {
            $vmId = (int)$_POST['vm_id'];
            $data = [
                'name' => trim($_POST['name']),
                'description' => trim($_POST['description'] ?? ''),
                'cpu_cores' => (int)$_POST['cpu_cores'],
                'ram_mb' => (int)$_POST['ram_mb'],
                'boot_order' => $_POST['boot_order'],
                'iso_path' => $_POST['iso_path'] ?? null,
                'boot_from_disk' => isset($_POST['boot_from_disk']),
                'physical_disk_device' => $_POST['physical_disk_device'] ?? null,
                'network_mode' => $_POST['network_mode'],
                'network_bridge' => $_POST['network_bridge'] ?? null,
                'display_type' => $_POST['display_type']
            ];

            $vmManager->update($vmId, $data);
            $message = 'VM updated successfully!';
            $messageType = 'success';

            // Log activity
            $vmData = $vmManager->getById($vmId);
            $db->query(
                "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                [$_SESSION['user_id'], 'vm_update', "Updated VM: {$vmData['name']}", $_SERVER['REMOTE_ADDR']]
            );
        } elseif ($action === 'delete') {
            $vmId = (int)$_POST['vm_id'];
            $vmData = $vmManager->getById($vmId);
            $vmManager->delete($vmId);
            $message = 'VM deleted successfully!';
            $messageType = 'success';

            // Log activity
            $db->query(
                "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                [$_SESSION['user_id'], 'vm_delete', "Deleted VM: {$vmData['name']}", $_SERVER['REMOTE_ADDR']]
            );
        }
    } catch (Exception $e) {
        $message = 'Error: ' . $e->getMessage();
        $messageType = 'error';
    }
}

// Get all VMs
$vms = $vmManager->getAll();
$pageTitle = 'Virtual Machines';
?>

<?php include VIEWS_PATH . '/layout/header.php'; ?>
<?php include VIEWS_PATH . '/layout/navbar.php'; ?>

<div class="container">
    <h1 style="margin-bottom: 30px; color: #fff;">Virtual Machines</h1>

    <?php if ($message): ?>
        <div class="alert alert-<?php echo $messageType; ?>">
            <?php echo htmlspecialchars($message); ?>
        </div>
    <?php endif; ?>

    <!-- Create VM Button -->
    <div style="margin-bottom: 20px;">
        <button onclick="openCreateModal()" class="btn btn-primary">+ Create New VM</button>
    </div>

    <!-- VMs List -->
    <div class="card">
        <div class="card-header">Virtual Machines</div>
        <div class="card-body">
            <?php if (empty($vms)): ?>
                <p style="color: #666;">No virtual machines found. Create one to get started!</p>
            <?php else: ?>
                <table class="table">
                    <thead>
                        <tr>
                            <th>Name</th>
                            <th>Status</th>
                            <th>CPU</th>
                            <th>RAM</th>
                            <th>Network</th>
                            <th>Display</th>
                            <th>Actions</th>
                        </tr>
                    </thead>
                    <tbody id="vms-tbody">
                        <?php foreach ($vms as $vm): ?>
                            <tr data-vm-id="<?php echo $vm['id']; ?>">
                                <td>
                                    <strong><?php echo htmlspecialchars($vm['name']); ?></strong><br>
                                    <small style="color: #888;"><?php echo htmlspecialchars($vm['description']); ?></small>
                                </td>
                                <td>
                                    <span class="badge badge-<?php echo $vm['status'] === 'running' ? 'active' : 'inactive'; ?>" id="status-<?php echo $vm['id']; ?>">
                                        <?php echo ucfirst($vm['status']); ?>
                                    </span>
                                </td>
                                <td><?php echo $vm['cpu_cores']; ?> cores</td>
                                <td><?php echo number_format($vm['ram_mb'] / 1024, 1); ?> GB</td>
                                <td><?php echo ucfirst($vm['network_mode']); ?></td>
                                <td><?php echo strtoupper($vm['display_type']); ?></td>
                                <td>
                                    <div style="display: flex; gap: 5px; flex-wrap: wrap;">
                                        <?php if ($vm['status'] === 'stopped'): ?>
                                            <button onclick="controlVM(<?php echo $vm['id']; ?>, 'start')" class="btn btn-success" style="padding: 6px 12px; font-size: 12px;">▶ Start</button>
                                        <?php else: ?>
                                            <button onclick="controlVM(<?php echo $vm['id']; ?>, 'stop')" class="btn btn-secondary" style="padding: 6px 12px; font-size: 12px;">⏹ Stop</button>
                                            <button onclick="controlVM(<?php echo $vm['id']; ?>, 'restart')" class="btn btn-secondary" style="padding: 6px 12px; font-size: 12px;">↻ Restart</button>
                                        <?php endif; ?>

                                        <?php if ($vm['display_type'] === 'spice'): ?>
                                            <a href="/download-spice.php?vm_id=<?php echo $vm['id']; ?>" class="btn btn-secondary" style="padding: 6px 12px; font-size: 12px; text-decoration: none;">📥 SPICE</a>
                                        <?php endif; ?>

                                        <button onclick="viewLogs(<?php echo $vm['id']; ?>, '<?php echo htmlspecialchars($vm['name']); ?>')" class="btn btn-secondary" style="padding: 6px 12px; font-size: 12px;">📋 Logs</button>
                                        <button onclick="openEditModal(<?php echo htmlspecialchars(json_encode($vm)); ?>)" class="btn btn-secondary" style="padding: 6px 12px; font-size: 12px;">⚙ Edit</button>

                                        <?php if ($vm['status'] === 'stopped'): ?>
                                            <form method="POST" style="display: inline;" onsubmit="return confirm('Are you sure you want to delete this VM? This will delete all associated data.');">
                                                <input type="hidden" name="action" value="delete">
                                                <input type="hidden" name="vm_id" value="<?php echo $vm['id']; ?>">
                                                <button type="submit" class="btn btn-danger" style="padding: 6px 12px; font-size: 12px;">🗑 Delete</button>
                                            </form>
                                        <?php endif; ?>
                                    </div>
                                </td>
                            </tr>
                        <?php endforeach; ?>
                    </tbody>
                </table>
            <?php endif; ?>
        </div>
    </div>
</div>

<!-- Modals and Scripts included in separate file due to size -->
