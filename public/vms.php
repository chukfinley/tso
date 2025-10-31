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

                                        <?php if ($vm['status'] === 'stopped'): ?>
                                            <button onclick="viewBackups(<?php echo $vm['id']; ?>, '<?php echo htmlspecialchars($vm['name']); ?>')" class="btn btn-secondary" style="padding: 6px 12px; font-size: 12px;">💾 Backups</button>
                                        <?php endif; ?>

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
<!-- Create VM Modal -->
<div id="createModal" class="modal">
    <div class="modal-content" style="max-width: 800px;">
        <h2 style="color: #fff; margin-bottom: 20px;">Create New Virtual Machine</h2>
        <form method="POST">
            <input type="hidden" name="action" value="create">

            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 20px;">
                <div class="form-group">
                    <label>VM Name *</label>
                    <input type="text" name="name" class="form-control" required pattern="[a-zA-Z0-9_-]+" title="Only letters, numbers, underscore and dash allowed">
                </div>

                <div class="form-group">
                    <label>Description</label>
                    <input type="text" name="description" class="form-control">
                </div>

                <div class="form-group">
                    <label>CPU Cores *</label>
                    <input type="number" name="cpu_cores" class="form-control" value="2" min="1" max="16" required>
                </div>

                <div class="form-group">
                    <label>RAM (MB) *</label>
                    <input type="number" name="ram_mb" class="form-control" value="2048" min="512" step="512" required>
                </div>

                <div class="form-group">
                    <label>Disk Size (GB) *</label>
                    <input type="number" name="disk_size_gb" class="form-control" value="20" min="1" max="500" required>
                </div>

                <div class="form-group">
                    <label>Disk Format *</label>
                    <select name="disk_format" class="form-control" required>
                        <option value="qcow2">QCOW2 (Recommended)</option>
                        <option value="raw">RAW</option>
                        <option value="vmdk">VMDK</option>
                    </select>
                </div>

                <div class="form-group">
                    <label>Boot Order *</label>
                    <select name="boot_order" class="form-control" required>
                        <option value="cd,hd">CD-ROM, then Hard Disk</option>
                        <option value="hd,cd">Hard Disk, then CD-ROM</option>
                        <option value="cd">CD-ROM only</option>
                        <option value="hd">Hard Disk only</option>
                        <option value="n">Network (PXE)</option>
                    </select>
                </div>

                <div class="form-group">
                    <label>ISO Image</label>
                    <div style="display: flex; gap: 10px; align-items: flex-end;">
                        <select name="iso_path" class="form-control" id="iso_path" style="flex: 1;">
                            <option value="">None</option>
                            <option value="" disabled>Loading...</option>
                        </select>
                        <div style="display: flex; flex-direction: column; gap: 5px;">
                            <input type="file" id="iso_upload_input" accept=".iso" style="display: none;" onchange="uploadIso()">
                            <button type="button" onclick="document.getElementById('iso_upload_input').click()" class="btn btn-secondary" style="white-space: nowrap;">
                                📤 Upload ISO
                            </button>
                            <div id="iso_upload_progress" style="display: none; color: #0f0; font-size: 12px;"></div>
                        </div>
                    </div>
                </div>

                <div class="form-group">
                    <label>
                        <input type="checkbox" name="boot_from_disk" id="boot_from_disk_create"> Boot from Physical Disk
                    </label>
                </div>

                <div class="form-group">
                    <label>Physical Disk Device</label>
                    <select name="physical_disk_device" class="form-control" id="physical_disk_create">
                        <option value="">None</option>
                        <option value="" disabled>Loading...</option>
                    </select>
                </div>

                <div class="form-group">
                    <label>Network Mode *</label>
                    <select name="network_mode" class="form-control" id="network_mode_create" onchange="toggleBridge('create')" required>
                        <option value="nat">NAT (Default)</option>
                        <option value="bridge">Bridged Network</option>
                        <option value="user">User Mode</option>
                        <option value="none">No Network</option>
                    </select>
                </div>

                <div class="form-group" id="bridge_group_create" style="display: none;">
                    <label>Network Bridge</label>
                    <select name="network_bridge" class="form-control" id="network_bridge_create">
                        <option value="">Select Bridge</option>
                        <option value="" disabled>Loading...</option>
                    </select>
                </div>

                <div class="form-group">
                    <label>Display Type *</label>
                    <select name="display_type" class="form-control" required>
                        <option value="spice">SPICE (Recommended)</option>
                        <option value="vnc">VNC</option>
                        <option value="none">Headless</option>
                    </select>
                </div>
            </div>

            <div style="display: flex; gap: 10px; margin-top: 20px;">
                <button type="submit" class="btn btn-primary">Create VM</button>
                <button type="button" onclick="closeCreateModal()" class="btn btn-secondary">Cancel</button>
            </div>
        </form>
    </div>
</div>

<!-- Edit VM Modal -->
<div id="editModal" class="modal">
    <div class="modal-content" style="max-width: 800px;">
        <h2 style="color: #fff; margin-bottom: 20px;">Edit Virtual Machine</h2>
        <form method="POST">
            <input type="hidden" name="action" value="update">
            <input type="hidden" name="vm_id" id="edit_vm_id">

            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 20px;">
                <div class="form-group">
                    <label>VM Name *</label>
                    <input type="text" name="name" id="edit_name" class="form-control" required>
                </div>

                <div class="form-group">
                    <label>Description</label>
                    <input type="text" name="description" id="edit_description" class="form-control">
                </div>

                <div class="form-group">
                    <label>CPU Cores *</label>
                    <input type="number" name="cpu_cores" id="edit_cpu_cores" class="form-control" min="1" max="16" required>
                </div>

                <div class="form-group">
                    <label>RAM (MB) *</label>
                    <input type="number" name="ram_mb" id="edit_ram_mb" class="form-control" min="512" step="512" required>
                </div>

                <div class="form-group">
                    <label>Boot Order *</label>
                    <select name="boot_order" id="edit_boot_order" class="form-control" required>
                        <option value="cd,hd">CD-ROM, then Hard Disk</option>
                        <option value="hd,cd">Hard Disk, then CD-ROM</option>
                        <option value="cd">CD-ROM only</option>
                        <option value="hd">Hard Disk only</option>
                        <option value="n">Network (PXE)</option>
                    </select>
                </div>

                <div class="form-group">
                    <label>ISO Image</label>
                    <div style="display: flex; gap: 10px; align-items: flex-end;">
                        <select name="iso_path" id="edit_iso_path" class="form-control" style="flex: 1;">
                            <option value="">None</option>
                        </select>
                        <div style="display: flex; flex-direction: column; gap: 5px;">
                            <input type="file" id="edit_iso_upload_input" accept=".iso" style="display: none;" onchange="uploadIso('edit')">
                            <button type="button" onclick="document.getElementById('edit_iso_upload_input').click()" class="btn btn-secondary" style="white-space: nowrap;">
                                📤 Upload ISO
                            </button>
                            <div id="edit_iso_upload_progress" style="display: none; color: #0f0; font-size: 12px;"></div>
                        </div>
                    </div>
                </div>

                <div class="form-group">
                    <label>
                        <input type="checkbox" name="boot_from_disk" id="edit_boot_from_disk"> Boot from Physical Disk
                    </label>
                </div>

                <div class="form-group">
                    <label>Physical Disk Device</label>
                    <select name="physical_disk_device" id="edit_physical_disk" class="form-control">
                        <option value="">None</option>
                    </select>
                </div>

                <div class="form-group">
                    <label>Network Mode *</label>
                    <select name="network_mode" id="edit_network_mode" class="form-control" onchange="toggleBridge('edit')" required>
                        <option value="nat">NAT (Default)</option>
                        <option value="bridge">Bridged Network</option>
                        <option value="user">User Mode</option>
                        <option value="none">No Network</option>
                    </select>
                </div>

                <div class="form-group" id="bridge_group_edit" style="display: none;">
                    <label>Network Bridge</label>
                    <select name="network_bridge" id="edit_network_bridge" class="form-control">
                        <option value="">Select Bridge</option>
                    </select>
                </div>

                <div class="form-group">
                    <label>Display Type *</label>
                    <select name="display_type" id="edit_display_type" class="form-control" required>
                        <option value="spice">SPICE (Recommended)</option>
                        <option value="vnc">VNC</option>
                        <option value="none">Headless</option>
                    </select>
                </div>
            </div>

            <p style="color: #ff9800; margin-top: 15px;">⚠️ VM must be stopped to update configuration</p>

            <div style="display: flex; gap: 10px; margin-top: 20px;">
                <button type="submit" class="btn btn-primary">Update VM</button>
                <button type="button" onclick="closeEditModal()" class="btn btn-secondary">Cancel</button>
            </div>
        </form>
    </div>
</div>

<!-- Logs Modal -->
<div id="logsModal" class="modal">
    <div class="modal-content" style="max-width: 900px;">
        <h2 style="color: #fff; margin-bottom: 20px;">VM Logs: <span id="logs_vm_name"></span></h2>
        <div style="background: #000; border: 1px solid #333; border-radius: 5px; padding: 15px; max-height: 500px; overflow-y: auto; font-family: monospace; font-size: 12px;">
            <pre id="logs_content" style="color: #0f0; margin: 0; white-space: pre-wrap;"></pre>
        </div>
        <div style="display: flex; gap: 10px; margin-top: 20px;">
            <button onclick="refreshLogs()" class="btn btn-primary">↻ Refresh</button>
            <button onclick="closeLogsModal()" class="btn btn-secondary">Close</button>
        </div>
    </div>
</div>

<style>
.modal {
    display: none;
    position: fixed;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
    background: rgba(0,0,0,0.8);
    z-index: 9999;
    align-items: center;
    justify-content: center;
    overflow-y: auto;
    padding: 20px;
}

.modal-content {
    background: #242424;
    border: 1px solid #333;
    border-radius: 8px;
    padding: 30px;
    width: 90%;
    max-height: 90vh;
    overflow-y: auto;
}
</style>

<script>
let currentLogVmId = null;

document.addEventListener('DOMContentLoaded', function() {
    loadIsos();
    loadPhysicalDisks();
    loadNetworkBridges();
    setInterval(refreshVMStatus, 3000);
});

function loadIsos() {
    fetch('/api/vm-control.php?action=list_isos')
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                const selects = ['iso_path', 'edit_iso_path'];
                selects.forEach(id => {
                    const select = document.getElementById(id);
                    if (select) {
                        // Save current selection
                        const currentValue = select.value;
                        
                        // Clear existing options except "None"
                        Array.from(select.options).forEach(opt => {
                            if (opt.disabled || opt.value === '') {
                                if (opt.value === '') {
                                    // Keep "None" option
                                    return;
                                }
                                opt.remove();
                            } else {
                                opt.remove();
                            }
                        });
                        
                        // Add ISO options
                        data.isos.forEach(iso => {
                            const option = document.createElement('option');
                            option.value = iso.path;
                            option.textContent = iso.name + ' (' + iso.size_formatted + ')';
                            select.appendChild(option);
                        });
                        
                        // Restore selection if it still exists
                        if (currentValue) {
                            select.value = currentValue;
                        }
                    }
                });
            }
        });
}

function uploadIso(mode = 'create') {
    const inputId = mode === 'edit' ? 'edit_iso_upload_input' : 'iso_upload_input';
    const progressId = mode === 'edit' ? 'edit_iso_upload_progress' : 'iso_upload_progress';
    const input = document.getElementById(inputId);
    const progress = document.getElementById(progressId);
    
    if (!input.files || input.files.length === 0) {
        return;
    }
    
    const file = input.files[0];
    
    // Validate file type
    if (!file.name.toLowerCase().endsWith('.iso')) {
        alert('Please select an ISO file (.iso extension)');
        input.value = '';
        return;
    }
    
    // Validate file size (max 20GB)
    const maxSize = 20 * 1024 * 1024 * 1024; // 20GB in bytes
    if (file.size > maxSize) {
        alert('File size too large. Maximum size is 20GB.');
        input.value = '';
        return;
    }
    
    // Show progress
    progress.style.display = 'block';
    progress.textContent = 'Uploading ' + file.name + '...';
    
    // Create form data
    const formData = new FormData();
    formData.append('iso_file', file);
    
    // Upload file
    fetch('/api/vm-control.php?action=upload_iso', {
        method: 'POST',
        body: formData
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            progress.textContent = '✓ Upload complete: ' + data.filename;
            progress.style.color = '#0f0';
            
            // Clear input
            input.value = '';
            
            // Reload ISO list
            loadIsos();
            
            // Auto-select the uploaded ISO if in create mode
            if (mode === 'create') {
                setTimeout(() => {
                    const select = document.getElementById('iso_path');
                    if (select && data.path) {
                        select.value = data.path;
                    }
                }, 500);
            } else {
                setTimeout(() => {
                    const select = document.getElementById('edit_iso_path');
                    if (select && data.path) {
                        select.value = data.path;
                    }
                }, 500);
            }
            
            // Hide progress after 3 seconds
            setTimeout(() => {
                progress.style.display = 'none';
            }, 3000);
        } else {
            progress.textContent = '✗ Error: ' + (data.error || 'Upload failed');
            progress.style.color = '#f00';
            input.value = '';
            
            // Hide progress after 5 seconds
            setTimeout(() => {
                progress.style.display = 'none';
            }, 5000);
        }
    })
    .catch(error => {
        progress.textContent = '✗ Error: ' + error.message;
        progress.style.color = '#f00';
        input.value = '';
        
        // Hide progress after 5 seconds
        setTimeout(() => {
            progress.style.display = 'none';
        }, 5000);
    });
}

function loadPhysicalDisks() {
    fetch('/api/vm-control.php?action=list_disks')
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                const selects = ['physical_disk_create', 'edit_physical_disk'];
                selects.forEach(id => {
                    const select = document.getElementById(id);
                    if (select) {
                        Array.from(select.options).forEach(opt => {
                            if (opt.disabled) opt.remove();
                        });
                        data.disks.forEach(disk => {
                            const option = document.createElement('option');
                            option.value = disk.device;
                            option.textContent = disk.device + ' (' + disk.size + ' - ' + disk.model + ')';
                            select.appendChild(option);
                        });
                    }
                });
            }
        });
}

function loadNetworkBridges() {
    fetch('/api/vm-control.php?action=list_bridges')
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                const selects = ['network_bridge_create', 'edit_network_bridge'];
                selects.forEach(id => {
                    const select = document.getElementById(id);
                    if (select) {
                        Array.from(select.options).forEach(opt => {
                            if (opt.disabled) opt.remove();
                        });
                        data.bridges.forEach(bridge => {
                            const option = document.createElement('option');
                            option.value = bridge;
                            option.textContent = bridge;
                            select.appendChild(option);
                        });
                    }
                });
            }
        });
}

function toggleBridge(mode) {
    const networkMode = document.getElementById('network_mode_' + mode).value;
    const bridgeGroup = document.getElementById('bridge_group_' + mode);
    bridgeGroup.style.display = networkMode === 'bridge' ? 'block' : 'none';
}

function openCreateModal() {
    document.getElementById('createModal').style.display = 'flex';
}

function closeCreateModal() {
    document.getElementById('createModal').style.display = 'none';
}

function openEditModal(vm) {
    document.getElementById('edit_vm_id').value = vm.id;
    document.getElementById('edit_name').value = vm.name;
    document.getElementById('edit_description').value = vm.description || '';
    document.getElementById('edit_cpu_cores').value = vm.cpu_cores;
    document.getElementById('edit_ram_mb').value = vm.ram_mb;
    document.getElementById('edit_boot_order').value = vm.boot_order;
    document.getElementById('edit_iso_path').value = vm.iso_path || '';
    document.getElementById('edit_boot_from_disk').checked = vm.boot_from_disk == 1;
    document.getElementById('edit_physical_disk').value = vm.physical_disk_device || '';
    document.getElementById('edit_network_mode').value = vm.network_mode;
    document.getElementById('edit_network_bridge').value = vm.network_bridge || '';
    document.getElementById('edit_display_type').value = vm.display_type;
    toggleBridge('edit');
    document.getElementById('editModal').style.display = 'flex';
}

function closeEditModal() {
    document.getElementById('editModal').style.display = 'none';
}

function controlVM(vmId, action) {
    const confirmMessage = action === 'stop' ? 'Stop this VM?' : action === 'restart' ? 'Restart this VM?' : null;
    if (confirmMessage && !confirm(confirmMessage)) return;

    fetch('/api/vm-control.php', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: action, vm_id: vmId })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            alert(data.message);
            setTimeout(() => location.reload(), 1000);
        } else {
            alert('Error: ' + data.error);
        }
    })
    .catch(error => alert('Error: ' + error.message));
}

function viewLogs(vmId, vmName) {
    currentLogVmId = vmId;
    document.getElementById('logs_vm_name').textContent = vmName;
    document.getElementById('logsModal').style.display = 'flex';
    refreshLogs();
}

function refreshLogs() {
    if (!currentLogVmId) return;
    fetch('/api/vm-control.php', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ action: 'logs', vm_id: currentLogVmId, lines: 200 })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            document.getElementById('logs_content').textContent = data.logs || 'No logs available';
        }
    });
}

function closeLogsModal() {
    document.getElementById('logsModal').style.display = 'none';
    currentLogVmId = null;
}

function refreshVMStatus() {
    const rows = document.querySelectorAll('#vms-tbody tr');
    rows.forEach(row => {
        const vmId = row.getAttribute('data-vm-id');
        fetch('/api/vm-control.php?action=status&vm_id=' + vmId)
            .then(response => response.json())
            .then(data => {
                if (data.success) {
                    const statusBadge = document.getElementById('status-' + vmId);
                    if (statusBadge) {
                        statusBadge.textContent = data.status.charAt(0).toUpperCase() + data.status.slice(1);
                        statusBadge.className = 'badge badge-' + (data.status === 'running' ? 'active' : 'inactive');
                    }
                }
            });
    });
}

document.querySelectorAll('.modal').forEach(modal => {
    modal.addEventListener('click', function(e) {
        if (e.target === this) this.style.display = 'none';
    });
});
</script>

<?php include VIEWS_PATH . '/layout/footer.php'; ?>

<!-- Backups Modal -->
<div id="backupsModal" class="modal">
    <div class="modal-content" style="max-width: 900px;">
        <h2 style="color: #fff; margin-bottom: 20px;">VM Backups: <span id="backups_vm_name"></span></h2>

        <button onclick="createBackup()" class="btn btn-primary" style="margin-bottom: 20px;">+ Create New Backup</button>

        <div id="backups_list">
            <p style="color: #666;">Loading backups...</p>
        </div>

        <div style="display: flex; gap: 10px; margin-top: 20px;">
            <button onclick="refreshBackups()" class="btn btn-primary">↻ Refresh</button>
            <button onclick="closeBackupsModal()" class="btn btn-secondary">Close</button>
        </div>
    </div>
</div>

<script>
let currentBackupVmId = null;
let currentBackupVmName = '';

function viewBackups(vmId, vmName) {
    currentBackupVmId = vmId;
    currentBackupVmName = vmName;
    document.getElementById('backups_vm_name').textContent = vmName;
    document.getElementById('backupsModal').style.display = 'flex';
    refreshBackups();
}

function refreshBackups() {
    if (!currentBackupVmId) return;

    fetch('/api/vm-control.php?action=list_backups&vm_id=' + currentBackupVmId)
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                displayBackups(data.backups);
            }
        })
        .catch(error => console.error('Error fetching backups:', error));
}

function displayBackups(backups) {
    const container = document.getElementById('backups_list');

    if (backups.length === 0) {
        container.innerHTML = '<p style="color: #666;">No backups found. Create one to get started!</p>';
        return;
    }

    let html = '<table class="table"><thead><tr><th>Backup Name</th><th>Size</th><th>Status</th><th>Created</th><th>Actions</th></tr></thead><tbody>';

    backups.forEach(backup => {
        const size = backup.backup_size ? formatBytes(backup.backup_size) : 'N/A';
        const created = new Date(backup.created_at).toLocaleString();
        const statusClass = backup.status === 'completed' ? 'badge-active' : backup.status === 'creating' ? 'badge-inactive' : 'badge-admin';

        html += '<tr>';
        html += '<td><strong>' + backup.backup_name + '</strong><br><small style="color: #888;">' + (backup.notes || '') + '</small></td>';
        html += '<td>' + size + '</td>';
        html += '<td><span class="badge ' + statusClass + '">' + backup.status + '</span></td>';
        html += '<td>' + created + '</td>';
        html += '<td>';

        if (backup.status === 'completed') {
            html += '<button onclick="restoreBackup(' + backup.id + ', \'' + backup.backup_name + '\')" class="btn btn-success" style="padding: 6px 12px; font-size: 12px;">↺ Restore</button> ';
        }

        html += '<button onclick="deleteBackup(' + backup.id + ', \'' + backup.backup_name + '\')" class="btn btn-danger" style="padding: 6px 12px; font-size: 12px;">🗑 Delete</button>';
        html += '</td></tr>';
    });

    html += '</tbody></table>';
    container.innerHTML = html;
}

function createBackup() {
    const notes = prompt('Enter backup notes (optional):');
    if (notes === null) return;

    fetch('/api/vm-control.php', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            action: 'create_backup',
            vm_id: currentBackupVmId,
            notes: notes
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            alert('Backup started! This may take a few minutes...');
            refreshBackups();
            checkBackupProgress(data.backup_id);
        } else {
            alert('Error: ' + data.error);
        }
    })
    .catch(error => alert('Error: ' + error.message));
}

function checkBackupProgress(backupId) {
    const interval = setInterval(() => {
        fetch('/api/vm-control.php', {
            method: 'POST',
            headers: { 'Content-Type': 'application/json' },
            body: JSON.stringify({
                action: 'check_backup_status',
                backup_id: backupId
            })
        })
        .then(response => response.json())
        .then(data => {
            if (data.success && data.status === 'completed') {
                clearInterval(interval);
                refreshBackups();
                alert('Backup completed successfully!');
            } else if (data.success && data.status === 'failed') {
                clearInterval(interval);
                refreshBackups();
                alert('Backup failed!');
            }
        });
    }, 3000);
}

function restoreBackup(backupId, backupName) {
    if (!confirm('Are you sure you want to restore from backup "' + backupName + '"? This will overwrite the current VM disk!')) {
        return;
    }

    fetch('/api/vm-control.php', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            action: 'restore_backup',
            backup_id: backupId
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            alert('Backup restored successfully!');
            closeBackupsModal();
        } else {
            alert('Error: ' + data.error);
        }
    })
    .catch(error => alert('Error: ' + error.message));
}

function deleteBackup(backupId, backupName) {
    if (!confirm('Are you sure you want to delete backup "' + backupName + '"?')) {
        return;
    }

    fetch('/api/vm-control.php', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
            action: 'delete_backup',
            backup_id: backupId
        })
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            alert('Backup deleted successfully!');
            refreshBackups();
        } else {
            alert('Error: ' + data.error);
        }
    })
    .catch(error => alert('Error: ' + error.message));
}

function closeBackupsModal() {
    document.getElementById('backupsModal').style.display = 'none';
    currentBackupVmId = null;
}

function formatBytes(bytes) {
    if (bytes === 0) return '0 Bytes';
    const k = 1024;
    const sizes = ['Bytes', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));
    return Math.round(bytes / Math.pow(k, i) * 100) / 100 + ' ' + sizes[i];
}
</script>
