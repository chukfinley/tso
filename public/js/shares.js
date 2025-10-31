/**
 * Network Shares Management JavaScript
 */

let allUsers = [];
let allShares = [];
let currentShareId = null;

// Initialize on page load
document.addEventListener('DOMContentLoaded', function() {
    loadSambaStatus();
    loadShares();
    loadUsers();
    loadLogs();

    // Refresh every 30 seconds
    setInterval(loadSambaStatus, 30000);
});

// ============ Share Operations ============

function loadShares() {
    fetch('/api/share-control.php?action=list')
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                allShares = data.shares;
                displayShares(data.shares);
            } else {
                showError('Failed to load shares: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to load shares');
        });
}

function displayShares(shares) {
    const container = document.getElementById('shares-list');
    
    if (shares.length === 0) {
        container.innerHTML = '<p style="color: #666;">No shares configured. Click "Create New Share" to get started.</p>';
        return;
    }

    let html = '';
    shares.forEach(share => {
        const isActive = parseInt(share.is_active) === 1;
        const statusClass = isActive ? 'status-active' : 'status-inactive';
        const statusText = isActive ? 'Active' : 'Inactive';
        
        html += `
            <div class="share-item">
                <div class="share-header">
                    <div>
                        <span class="share-title">${escapeHtml(share.share_name)}</span>
                        ${share.display_name && share.display_name !== share.share_name ? 
                            `<span style="color: #aaa; margin-left: 10px;">(${escapeHtml(share.display_name)})</span>` : ''}
                    </div>
                    <span class="share-status ${statusClass}">${statusText}</span>
                </div>
                <div class="share-details">
                    <div><strong>Path:</strong> ${escapeHtml(share.path)}</div>
                    ${share.comment ? `<div><strong>Description:</strong> ${escapeHtml(share.comment)}</div>` : ''}
                    <div>
                        <strong>Options:</strong> 
                        ${share.browseable ? '📁 Browseable' : '🔒 Hidden'} | 
                        ${share.readonly ? '👁️ Read-Only' : '✍️ Read-Write'} | 
                        ${share.guest_ok ? '👤 Guest OK' : '🔐 Auth Required'}
                    </div>
                    ${share.case_sensitive !== 'auto' ? 
                        `<div><strong>Case Sensitive:</strong> ${share.case_sensitive}</div>` : ''}
                </div>
                <div class="share-actions">
                    <button class="btn btn-sm btn-primary" onclick="editShare(${share.id})">
                        <i class="fas fa-edit"></i> Edit
                    </button>
                    <button class="btn btn-sm btn-info" onclick="managePermissions(${share.id})">
                        <i class="fas fa-users"></i> Permissions
                    </button>
                    <button class="btn btn-sm btn-${isActive ? 'warning' : 'success'}" onclick="toggleShare(${share.id})">
                        <i class="fas fa-${isActive ? 'pause' : 'play'}"></i> ${isActive ? 'Disable' : 'Enable'}
                    </button>
                    <button class="btn btn-sm btn-danger" onclick="deleteShare(${share.id}, '${escapeHtml(share.share_name)}')">
                        <i class="fas fa-trash"></i> Delete
                    </button>
                </div>
            </div>
        `;
    });

    container.innerHTML = html;
}

function showCreateShareModal() {
    document.getElementById('shareModalTitle').textContent = 'Create New Share';
    document.getElementById('shareForm').reset();
    document.getElementById('share_id').value = '';
    document.getElementById('share_name').disabled = false;
    currentShareId = null;
    
    // Load users for permissions section
    loadUsersForPermissions();
    
    $('#shareModal').modal('show');
}

function editShare(id) {
    fetch(`/api/share-control.php?action=get&id=${id}`)
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                const share = data.share;
                document.getElementById('shareModalTitle').textContent = 'Edit Share';
                document.getElementById('share_id').value = share.id;
                document.getElementById('share_name').value = share.share_name;
                document.getElementById('share_name').disabled = true; // Can't change share name
                document.getElementById('display_name').value = share.display_name || '';
                document.getElementById('path').value = share.path;
                document.getElementById('comment').value = share.comment || '';
                document.getElementById('browseable').checked = parseInt(share.browseable) === 1;
                document.getElementById('readonly').checked = parseInt(share.readonly) === 1;
                document.getElementById('guest_ok').checked = parseInt(share.guest_ok) === 1;
                document.getElementById('case_sensitive').value = share.case_sensitive;
                document.getElementById('preserve_case').checked = parseInt(share.preserve_case) === 1;
                document.getElementById('short_preserve_case').checked = parseInt(share.short_preserve_case) === 1;
                document.getElementById('create_mask').value = share.create_mask;
                document.getElementById('directory_mask').value = share.directory_mask;
                document.getElementById('force_user').value = share.force_user || '';
                document.getElementById('force_group').value = share.force_group || '';
                
                currentShareId = id;
                
                // Load existing permissions
                loadSharePermissionsForEdit(id);
                
                $('#shareModal').modal('show');
            } else {
                showError('Failed to load share: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to load share');
        });
}

function saveShare() {
    const form = document.getElementById('shareForm');
    if (!form.checkValidity()) {
        form.reportValidity();
        return;
    }

    const formData = new FormData(form);
    const shareId = document.getElementById('share_id').value;
    
    formData.append('action', shareId ? 'update' : 'create');
    if (shareId) {
        formData.append('id', shareId);
    }

    // Add checkbox values explicitly
    formData.set('browseable', document.getElementById('browseable').checked ? '1' : '0');
    formData.set('readonly', document.getElementById('readonly').checked ? '1' : '0');
    formData.set('guest_ok', document.getElementById('guest_ok').checked ? '1' : '0');
    formData.set('preserve_case', document.getElementById('preserve_case').checked ? '1' : '0');
    formData.set('short_preserve_case', document.getElementById('short_preserve_case').checked ? '1' : '0');

    fetch('/api/share-control.php', {
        method: 'POST',
        body: formData
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showSuccess(data.message);
                $('#shareModal').modal('hide');
                loadShares();
                
                // Save permissions if this is a new share
                if (!shareId && data.share_id) {
                    saveAllPermissions(data.share_id);
                }
            } else {
                showError('Failed to save share: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to save share');
        });
}

function toggleShare(id) {
    const formData = new FormData();
    formData.append('action', 'toggle');
    formData.append('id', id);

    fetch('/api/share-control.php', {
        method: 'POST',
        body: formData
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showSuccess(data.message);
                loadShares();
            } else {
                showError('Failed to toggle share: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to toggle share');
        });
}

function deleteShare(id, name) {
    if (!confirm(`Are you sure you want to delete the share "${name}"?\n\nThis will NOT delete the directory by default.`)) {
        return;
    }

    const deleteDir = confirm('Do you also want to DELETE the directory and all its contents?\n\nWARNING: This cannot be undone!');

    const formData = new FormData();
    formData.append('action', 'delete');
    formData.append('id', id);
    formData.append('delete_directory', deleteDir ? '1' : '0');

    fetch('/api/share-control.php', {
        method: 'POST',
        body: formData
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showSuccess(data.message);
                loadShares();
            } else {
                showError('Failed to delete share: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to delete share');
        });
}

// ============ User Operations ============

function loadUsers() {
    fetch('/api/share-control.php?action=list_users')
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                allUsers = data.users;
                displayUsers(data.users);
            } else {
                showError('Failed to load users: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to load users');
        });
}

function displayUsers(users) {
    const container = document.getElementById('users-list');
    
    if (users.length === 0) {
        container.innerHTML = '<p style="color: #666;">No users configured. Click "Create New User" to get started.</p>';
        return;
    }

    let html = '';
    users.forEach(user => {
        const isActive = parseInt(user.is_active) === 1;
        const statusClass = isActive ? 'status-active' : 'status-inactive';
        const statusText = isActive ? 'Active' : 'Inactive';
        
        html += `
            <div class="user-item">
                <div class="user-header">
                    <div>
                        <span class="share-title">${escapeHtml(user.username)}</span>
                        ${user.full_name ? 
                            `<span style="color: #aaa; margin-left: 10px;">(${escapeHtml(user.full_name)})</span>` : ''}
                    </div>
                    <span class="share-status ${statusClass}">${statusText}</span>
                </div>
                <div class="share-details">
                    <div><strong>Created:</strong> ${formatDate(user.created_at)}</div>
                </div>
                <div class="share-actions">
                    <button class="btn btn-sm btn-primary" onclick="changeUserPassword(${user.id}, '${escapeHtml(user.username)}')">
                        <i class="fas fa-key"></i> Change Password
                    </button>
                    <button class="btn btn-sm btn-info" onclick="viewUserPermissions(${user.id}, '${escapeHtml(user.username)}')">
                        <i class="fas fa-list"></i> View Permissions
                    </button>
                    <button class="btn btn-sm btn-${isActive ? 'warning' : 'success'}" onclick="toggleUser(${user.id})">
                        <i class="fas fa-${isActive ? 'pause' : 'play'}"></i> ${isActive ? 'Disable' : 'Enable'}
                    </button>
                    <button class="btn btn-sm btn-danger" onclick="deleteUser(${user.id}, '${escapeHtml(user.username)}')">
                        <i class="fas fa-trash"></i> Delete
                    </button>
                </div>
            </div>
        `;
    });

    container.innerHTML = html;
}

function showCreateUserModal() {
    document.getElementById('userModalTitle').textContent = 'Create New User';
    document.getElementById('userForm').reset();
    document.getElementById('user_id').value = '';
    document.getElementById('username').disabled = false;
    $('#userModal').modal('show');
}

function saveUser() {
    const form = document.getElementById('userForm');
    if (!form.checkValidity()) {
        form.reportValidity();
        return;
    }

    const password = document.getElementById('password').value;
    const passwordConfirm = document.getElementById('password_confirm').value;

    if (password !== passwordConfirm) {
        showError('Passwords do not match');
        return;
    }

    const formData = new FormData(form);
    const userId = document.getElementById('user_id').value;
    
    formData.append('action', userId ? 'update_user_password' : 'create_user');
    if (userId) {
        formData.append('id', userId);
    }

    fetch('/api/share-control.php', {
        method: 'POST',
        body: formData
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showSuccess(data.message);
                $('#userModal').modal('hide');
                loadUsers();
            } else {
                showError('Failed to save user: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to save user');
        });
}

function changeUserPassword(id, username) {
    document.getElementById('userModalTitle').textContent = `Change Password for ${username}`;
    document.getElementById('userForm').reset();
    document.getElementById('user_id').value = id;
    document.getElementById('username').value = username;
    document.getElementById('username').disabled = true;
    document.getElementById('full_name').parentElement.style.display = 'none';
    $('#userModal').modal('show');
    
    // Reset on close
    $('#userModal').on('hidden.bs.modal', function () {
        document.getElementById('full_name').parentElement.style.display = 'block';
    });
}

function toggleUser(id) {
    const formData = new FormData();
    formData.append('action', 'toggle_user');
    formData.append('id', id);

    fetch('/api/share-control.php', {
        method: 'POST',
        body: formData
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showSuccess(data.message);
                loadUsers();
            } else {
                showError('Failed to toggle user: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to toggle user');
        });
}

function deleteUser(id, username) {
    if (!confirm(`Are you sure you want to delete user "${username}"?`)) {
        return;
    }

    const formData = new FormData();
    formData.append('action', 'delete_user');
    formData.append('id', id);

    fetch('/api/share-control.php', {
        method: 'POST',
        body: formData
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showSuccess(data.message);
                loadUsers();
            } else {
                showError('Failed to delete user: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to delete user');
        });
}

// ============ Permission Management ============

function managePermissions(shareId) {
    const share = allShares.find(s => s.id == shareId);
    if (!share) return;

    document.getElementById('permissionsModalTitle').textContent = `Manage Permissions: ${share.share_name}`;
    
    fetch(`/api/share-control.php?action=get_permissions&share_id=${shareId}`)
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                displayPermissions(shareId, data.permissions);
                $('#permissionsModal').modal('show');
            } else {
                showError('Failed to load permissions: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to load permissions');
        });
}

function displayPermissions(shareId, permissions) {
    const container = document.getElementById('permissions-content');
    
    let html = '<div style="margin-bottom: 15px;">';
    html += '<button class="btn btn-sm btn-success" onclick="addPermissionRow(' + shareId + ')"><i class="fas fa-plus"></i> Add User</button>';
    html += '</div>';

    if (permissions.length === 0) {
        html += '<p style="color: #666;">No permissions set. Add users above.</p>';
    } else {
        html += '<table class="table table-dark table-striped">';
        html += '<thead><tr><th>User</th><th>Permission</th><th>Actions</th></tr></thead>';
        html += '<tbody>';
        
        permissions.forEach(perm => {
            html += `
                <tr>
                    <td>${escapeHtml(perm.username)}${perm.full_name ? ' (' + escapeHtml(perm.full_name) + ')' : ''}</td>
                    <td>
                        <select class="form-control form-control-sm" onchange="updatePermission(${shareId}, ${perm.share_user_id}, this.value)">
                            <option value="read" ${perm.permission_level === 'read' ? 'selected' : ''}>Read Only</option>
                            <option value="write" ${perm.permission_level === 'write' ? 'selected' : ''}>Read/Write</option>
                            <option value="admin" ${perm.permission_level === 'admin' ? 'selected' : ''}>Admin</option>
                        </select>
                    </td>
                    <td>
                        <button class="btn btn-sm btn-danger" onclick="removePermission(${shareId}, ${perm.share_user_id})">
                            <i class="fas fa-trash"></i>
                        </button>
                    </td>
                </tr>
            `;
        });
        
        html += '</tbody></table>';
    }

    container.innerHTML = html;
}

function addPermissionRow(shareId) {
    const availableUsers = allUsers.filter(user => parseInt(user.is_active) === 1);
    
    if (availableUsers.length === 0) {
        showError('No active users available. Create users first.');
        return;
    }

    const container = document.getElementById('permissions-content');
    const form = document.createElement('div');
    form.className = 'permission-row';
    form.innerHTML = `
        <select class="form-control" id="new-user-select">
            <option value="">Select User...</option>
            ${availableUsers.map(u => `<option value="${u.id}">${escapeHtml(u.username)}</option>`).join('')}
        </select>
        <select class="form-control" id="new-perm-select">
            <option value="read">Read Only</option>
            <option value="write">Read/Write</option>
            <option value="admin">Admin</option>
        </select>
        <button class="btn btn-primary" onclick="saveNewPermission(${shareId})">Add</button>
        <button class="btn btn-secondary" onclick="this.parentElement.remove()">Cancel</button>
    `;
    
    container.insertBefore(form, container.firstChild.nextSibling);
}

function saveNewPermission(shareId) {
    const userId = document.getElementById('new-user-select').value;
    const permLevel = document.getElementById('new-perm-select').value;

    if (!userId) {
        showError('Please select a user');
        return;
    }

    updatePermission(shareId, userId, permLevel, () => {
        managePermissions(shareId); // Reload
    });
}

function updatePermission(shareId, userId, permissionLevel, callback) {
    const formData = new FormData();
    formData.append('action', 'set_permission');
    formData.append('share_id', shareId);
    formData.append('user_id', userId);
    formData.append('permission_level', permissionLevel);

    fetch('/api/share-control.php', {
        method: 'POST',
        body: formData
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showSuccess(data.message);
                if (callback) callback();
            } else {
                showError('Failed to update permission: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to update permission');
        });
}

function removePermission(shareId, userId) {
    if (!confirm('Remove this permission?')) return;

    const formData = new FormData();
    formData.append('action', 'remove_permission');
    formData.append('share_id', shareId);
    formData.append('user_id', userId);

    fetch('/api/share-control.php', {
        method: 'POST',
        body: formData
    })
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showSuccess(data.message);
                managePermissions(shareId); // Reload
            } else {
                showError('Failed to remove permission: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to remove permission');
        });
}

function viewUserPermissions(userId, username) {
    fetch(`/api/share-control.php?action=get_user_permissions&user_id=${userId}`)
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                document.getElementById('permissionsModalTitle').textContent = `Permissions for ${username}`;
                displayUserPermissions(data.permissions);
                $('#permissionsModal').modal('show');
            } else {
                showError('Failed to load permissions: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to load permissions');
        });
}

function displayUserPermissions(permissions) {
    const container = document.getElementById('permissions-content');
    
    if (permissions.length === 0) {
        container.innerHTML = '<p style="color: #666;">This user has no share permissions assigned.</p>';
        return;
    }

    let html = '<table class="table table-dark table-striped">';
    html += '<thead><tr><th>Share</th><th>Permission</th></tr></thead>';
    html += '<tbody>';
    
    permissions.forEach(perm => {
        html += `
            <tr>
                <td>${escapeHtml(perm.share_name)}</td>
                <td>
                    <span class="badge badge-${perm.permission_level === 'admin' ? 'danger' : perm.permission_level === 'write' ? 'warning' : 'info'}">
                        ${perm.permission_level}
                    </span>
                </td>
            </tr>
        `;
    });
    
    html += '</tbody></table>';
    container.innerHTML = html;
}

// Permission handling in share modal
function loadUsersForPermissions() {
    document.getElementById('user-permissions-list').innerHTML = '';
}

function loadSharePermissionsForEdit(shareId) {
    fetch(`/api/share-control.php?action=get_permissions&share_id=${shareId}`)
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                // Permissions are managed via separate modal
                // Just show count
                const count = data.permissions.length;
                document.getElementById('user-permissions-list').innerHTML = 
                    `<p class="text-muted">${count} user(s) have permissions on this share. Use "Manage Permissions" button to edit.</p>`;
            }
        });
}

function addUserPermission() {
    showInfo('Please save the share first, then use "Manage Permissions" button to add users.');
}

function saveAllPermissions(shareId) {
    // Permissions are managed separately via the Manage Permissions modal
}

// ============ System Operations ============

function loadSambaStatus() {
    fetch('/api/share-control.php?action=status')
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                const statusEl = document.getElementById('samba-status');
                if (data.status.running) {
                    statusEl.textContent = 'Running';
                    statusEl.style.background = '#4CAF50';
                    statusEl.style.color = 'white';
                } else {
                    statusEl.textContent = 'Stopped';
                    statusEl.style.background = '#f44336';
                    statusEl.style.color = 'white';
                }
            }
        })
        .catch(error => {
            console.error('Error:', error);
            const statusEl = document.getElementById('samba-status');
            statusEl.textContent = 'Unknown';
            statusEl.style.background = '#666';
        });
}

function testSambaConfig() {
    fetch('/api/share-control.php?action=test_config')
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                showSuccess('Samba configuration is valid!');
            } else {
                showError('Configuration test failed:\n' + data.output);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to test configuration');
        });
}

function loadConnectedClients() {
    fetch('/api/share-control.php?action=connected_clients')
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                if (data.clients.length === 0) {
                    showInfo('No clients currently connected');
                } else {
                    let message = 'Connected Clients:\n\n';
                    data.clients.forEach(client => {
                        message += `User: ${client.username}, Machine: ${client.machine}\n`;
                    });
                    alert(message);
                }
            } else {
                showError('Failed to get connected clients');
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to get connected clients');
        });
}

function browseDirectories() {
    fetch('/api/share-control.php?action=list_directories')
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                const dirs = data.directories;
                if (dirs.length === 0) {
                    showInfo('No common directories found');
                    return;
                }

                let html = '<div class="list-group">';
                dirs.forEach(dir => {
                    html += `
                        <a href="#" class="list-group-item list-group-item-action" 
                           onclick="selectDirectory('${dir.path}'); return false;">
                            ${dir.path} ${dir.writable ? '✓' : '(read-only)'}
                        </a>
                    `;
                });
                html += '</div>';

                // Show in a simple alert for now (could be improved with a better modal)
                const path = prompt('Enter directory path (or choose from common paths):\n\nCommon paths:\n' + 
                    dirs.map(d => d.path).join('\n'));
                if (path) {
                    document.getElementById('path').value = path;
                }
            }
        })
        .catch(error => {
            console.error('Error:', error);
        });
}

function selectDirectory(path) {
    document.getElementById('path').value = path;
}

function loadLogs() {
    fetch('/api/share-control.php?action=get_logs&limit=50')
        .then(response => response.json())
        .then(data => {
            if (data.success) {
                displayLogs(data.logs);
            } else {
                showError('Failed to load logs: ' + data.error);
            }
        })
        .catch(error => {
            console.error('Error:', error);
            showError('Failed to load logs');
        });
}

function displayLogs(logs) {
    const container = document.getElementById('logs-list');
    
    if (logs.length === 0) {
        container.innerHTML = '<p style="color: #666;">No activity logs yet.</p>';
        return;
    }

    let html = '';
    logs.forEach(log => {
        html += `
            <div class="log-entry">
                <div style="display: flex; justify-content: space-between; margin-bottom: 5px;">
                    <strong>${escapeHtml(log.action)}</strong>
                    <span style="color: #888;">${formatDate(log.created_at)}</span>
                </div>
                <div style="color: #aaa;">
                    Share: ${escapeHtml(log.share_name || 'N/A')} | 
                    User: ${escapeHtml(log.username)} | 
                    IP: ${escapeHtml(log.ip_address)}
                </div>
                ${log.details ? `<div style="color: #666; font-size: 12px;">${escapeHtml(log.details)}</div>` : ''}
            </div>
        `;
    });

    container.innerHTML = html;
}

function refreshAll() {
    loadSambaStatus();
    loadShares();
    loadUsers();
    loadLogs();
    showSuccess('Refreshed');
}

// ============ Utility Functions ============

function escapeHtml(text) {
    if (!text) return '';
    const div = document.createElement('div');
    div.textContent = text;
    return div.innerHTML;
}

function formatDate(dateString) {
    if (!dateString) return '';
    const date = new Date(dateString);
    return date.toLocaleString();
}

function showSuccess(message) {
    alert('✓ ' + message);
}

function showError(message) {
    alert('✗ Error: ' + message);
}

function showInfo(message) {
    alert('ℹ ' + message);
}

