<?php
require_once __DIR__ . '/../config/config.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/User.php';
require_once SRC_PATH . '/Auth.php';

$auth = new Auth();
$auth->requireLogin();

$pageTitle = 'Network Shares';
?>

<?php include VIEWS_PATH . '/layout/header.php'; ?>
<?php include VIEWS_PATH . '/layout/navbar.php'; ?>

<div class="container">
    <h1 style="margin-bottom: 30px; color: #fff;">Network Shares (SMB/Samba)</h1>

    <!-- Status Bar -->
    <div class="card" style="margin-bottom: 20px;">
        <div class="card-body" style="padding: 15px;">
            <div style="display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: 15px;">
                <div>
                    <strong>Samba Status:</strong> 
                    <span id="samba-status" style="padding: 3px 10px; border-radius: 4px; font-weight: bold;">
                        Checking...
                    </span>
                </div>
                <div>
                    <button onclick="testSambaConfig()" class="btn btn-sm btn-secondary">Test Config</button>
                    <button onclick="loadConnectedClients()" class="btn btn-sm btn-info">Connected Clients</button>
                    <button onclick="refreshAll()" class="btn btn-sm btn-primary">Refresh</button>
                </div>
            </div>
        </div>
    </div>

    <!-- Tabs -->
    <ul class="nav nav-tabs" style="margin-bottom: 20px; border-bottom: 2px solid #333;">
        <li class="nav-item">
            <a class="nav-link active" id="shares-tab" href="#shares" data-toggle="tab" style="color: #fff;">Shares</a>
        </li>
        <li class="nav-item">
            <a class="nav-link" id="users-tab" href="#users" data-toggle="tab" style="color: #fff;">Users</a>
        </li>
        <li class="nav-item">
            <a class="nav-link" id="logs-tab" href="#logs" data-toggle="tab" style="color: #fff;">Activity Logs</a>
        </li>
    </ul>

    <div class="tab-content">
        <!-- Shares Tab -->
        <div class="tab-pane fade show active" id="shares">
            <div style="margin-bottom: 20px;">
                <button onclick="showCreateShareModal()" class="btn btn-success">
                    <i class="fas fa-plus"></i> Create New Share
                </button>
            </div>

            <div class="card">
                <div class="card-header">Configured Shares</div>
                <div class="card-body">
                    <div id="shares-list">
                        <p style="color: #666;">Loading shares...</p>
                    </div>
                </div>
            </div>
        </div>

        <!-- Users Tab -->
        <div class="tab-pane fade" id="users">
            <div style="margin-bottom: 20px;">
                <button onclick="showCreateUserModal()" class="btn btn-success">
                    <i class="fas fa-user-plus"></i> Create New User
                </button>
            </div>

            <div class="card">
                <div class="card-header">Share Users</div>
                <div class="card-body">
                    <div id="users-list">
                        <p style="color: #666;">Loading users...</p>
                    </div>
                </div>
            </div>
        </div>

        <!-- Logs Tab -->
        <div class="tab-pane fade" id="logs">
            <div class="card">
                <div class="card-header">Activity Logs</div>
                <div class="card-body">
                    <div id="logs-list">
                        <p style="color: #666;">Loading logs...</p>
                    </div>
                </div>
            </div>
        </div>
    </div>
</div>

<!-- Create/Edit Share Modal -->
<div class="modal fade" id="shareModal" tabindex="-1">
    <div class="modal-dialog modal-lg">
        <div class="modal-content" style="background: #1e1e1e; color: #fff;">
            <div class="modal-header">
                <h5 class="modal-title" id="shareModalTitle">Create Share</h5>
                <button type="button" class="close" data-dismiss="modal" style="color: #fff;">
                    <span>&times;</span>
                </button>
            </div>
            <div class="modal-body">
                <form id="shareForm">
                    <input type="hidden" id="share_id" name="share_id">
                    
                    <ul class="nav nav-tabs" style="margin-bottom: 20px;">
                        <li class="nav-item">
                            <a class="nav-link active" data-toggle="tab" href="#basic-settings">Basic</a>
                        </li>
                        <li class="nav-item">
                            <a class="nav-link" data-toggle="tab" href="#permissions-settings">Permissions</a>
                        </li>
                        <li class="nav-item">
                            <a class="nav-link" data-toggle="tab" href="#advanced-settings">Advanced</a>
                        </li>
                    </ul>

                    <div class="tab-content">
                        <!-- Basic Settings -->
                        <div class="tab-pane fade show active" id="basic-settings">
                            <div class="form-group">
                                <label>Share Name *</label>
                                <input type="text" class="form-control" id="share_name" name="share_name" required 
                                       placeholder="e.g., documents" pattern="[a-zA-Z0-9_-]+">
                                <small class="form-text text-muted">Only letters, numbers, hyphens, and underscores</small>
                            </div>

                            <div class="form-group">
                                <label>Display Name</label>
                                <input type="text" class="form-control" id="display_name" name="display_name" 
                                       placeholder="e.g., Company Documents">
                            </div>

                            <div class="form-group">
                                <label>Directory Path *</label>
                                <div class="input-group">
                                    <input type="text" class="form-control" id="path" name="path" required 
                                           placeholder="/srv/samba/documents">
                                    <div class="input-group-append">
                                        <button type="button" class="btn btn-secondary" onclick="browseDirectories()">Browse</button>
                                    </div>
                                </div>
                            </div>

                            <div class="form-group">
                                <label>Comment/Description</label>
                                <textarea class="form-control" id="comment" name="comment" rows="2" 
                                          placeholder="Optional description of this share"></textarea>
                            </div>

                            <div class="form-check">
                                <input type="checkbox" class="form-check-input" id="browseable" name="browseable" checked>
                                <label class="form-check-label" for="browseable">
                                    Browseable (visible in network browser)
                                </label>
                            </div>

                            <div class="form-check">
                                <input type="checkbox" class="form-check-input" id="readonly" name="readonly">
                                <label class="form-check-label" for="readonly">
                                    Read Only (no write access for anyone)
                                </label>
                            </div>

                            <div class="form-check">
                                <input type="checkbox" class="form-check-input" id="guest_ok" name="guest_ok">
                                <label class="form-check-label" for="guest_ok">
                                    Allow Guest Access (no password required)
                                </label>
                            </div>
                        </div>

                        <!-- Permissions Settings -->
                        <div class="tab-pane fade" id="permissions-settings">
                            <div id="share-permissions-section">
                                <p class="text-muted">Set user permissions for this share.</p>
                                <button type="button" class="btn btn-sm btn-primary" onclick="addUserPermission()">
                                    <i class="fas fa-plus"></i> Add User
                                </button>
                                <div id="user-permissions-list" style="margin-top: 15px;">
                                    <!-- Dynamic user permissions will be added here -->
                                </div>
                            </div>
                        </div>

                        <!-- Advanced Settings -->
                        <div class="tab-pane fade" id="advanced-settings">
                            <h6>Case Sensitivity</h6>
                            <div class="form-group">
                                <label>Case Sensitive</label>
                                <select class="form-control" id="case_sensitive" name="case_sensitive">
                                    <option value="auto">Auto</option>
                                    <option value="yes">Yes</option>
                                    <option value="no">No</option>
                                </select>
                            </div>

                            <div class="form-check">
                                <input type="checkbox" class="form-check-input" id="preserve_case" name="preserve_case" checked>
                                <label class="form-check-label" for="preserve_case">
                                    Preserve Case
                                </label>
                            </div>

                            <div class="form-check">
                                <input type="checkbox" class="form-check-input" id="short_preserve_case" name="short_preserve_case" checked>
                                <label class="form-check-label" for="short_preserve_case">
                                    Short Preserve Case
                                </label>
                            </div>

                            <hr>

                            <h6>File Masks</h6>
                            <div class="form-group">
                                <label>Create Mask</label>
                                <input type="text" class="form-control" id="create_mask" name="create_mask" value="0664">
                                <small class="form-text text-muted">Unix permission mask for new files</small>
                            </div>

                            <div class="form-group">
                                <label>Directory Mask</label>
                                <input type="text" class="form-control" id="directory_mask" name="directory_mask" value="0775">
                                <small class="form-text text-muted">Unix permission mask for new directories</small>
                            </div>

                            <hr>

                            <h6>Force User/Group</h6>
                            <div class="form-group">
                                <label>Force User</label>
                                <input type="text" class="form-control" id="force_user" name="force_user" 
                                       placeholder="Optional: force all operations as this user">
                            </div>

                            <div class="form-group">
                                <label>Force Group</label>
                                <input type="text" class="form-control" id="force_group" name="force_group" 
                                       placeholder="Optional: force all operations as this group">
                            </div>
                        </div>
                    </div>
                </form>
            </div>
            <div class="modal-footer">
                <button type="button" class="btn btn-secondary" data-dismiss="modal">Cancel</button>
                <button type="button" class="btn btn-primary" onclick="saveShare()">Save Share</button>
            </div>
        </div>
    </div>
</div>

<!-- Create/Edit User Modal -->
<div class="modal fade" id="userModal" tabindex="-1">
    <div class="modal-dialog">
        <div class="modal-content" style="background: #1e1e1e; color: #fff;">
            <div class="modal-header">
                <h5 class="modal-title" id="userModalTitle">Create User</h5>
                <button type="button" class="close" data-dismiss="modal" style="color: #fff;">
                    <span>&times;</span>
                </button>
            </div>
            <div class="modal-body">
                <form id="userForm">
                    <input type="hidden" id="user_id" name="user_id">
                    
                    <div class="form-group">
                        <label>Username *</label>
                        <input type="text" class="form-control" id="username" name="username" required 
                               pattern="[a-zA-Z0-9_-]+" placeholder="e.g., john">
                        <small class="form-text text-muted">Only letters, numbers, hyphens, and underscores</small>
                    </div>

                    <div class="form-group">
                        <label>Full Name</label>
                        <input type="text" class="form-control" id="full_name" name="full_name" 
                               placeholder="e.g., John Doe">
                    </div>

                    <div class="form-group">
                        <label>Password *</label>
                        <input type="password" class="form-control" id="password" name="password" required 
                               minlength="4">
                        <small class="form-text text-muted">Minimum 4 characters</small>
                    </div>

                    <div class="form-group">
                        <label>Confirm Password *</label>
                        <input type="password" class="form-control" id="password_confirm" name="password_confirm" required>
                    </div>
                </form>
            </div>
            <div class="modal-footer">
                <button type="button" class="btn btn-secondary" data-dismiss="modal">Cancel</button>
                <button type="button" class="btn btn-primary" onclick="saveUser()">Save User</button>
            </div>
        </div>
    </div>
</div>

<!-- Manage Permissions Modal -->
<div class="modal fade" id="permissionsModal" tabindex="-1">
    <div class="modal-dialog modal-lg">
        <div class="modal-content" style="background: #1e1e1e; color: #fff;">
            <div class="modal-header">
                <h5 class="modal-title" id="permissionsModalTitle">Manage Permissions</h5>
                <button type="button" class="close" data-dismiss="modal" style="color: #fff;">
                    <span>&times;</span>
                </button>
            </div>
            <div class="modal-body">
                <div id="permissions-content">
                    <!-- Dynamic content will be loaded here -->
                </div>
            </div>
            <div class="modal-footer">
                <button type="button" class="btn btn-secondary" data-dismiss="modal">Close</button>
            </div>
        </div>
    </div>
</div>

<style>
    .share-item, .user-item {
        border: 1px solid #333;
        border-radius: 6px;
        padding: 15px;
        margin-bottom: 15px;
        background: #2a2a2a;
    }

    .share-item:hover, .user-item:hover {
        background: #333;
    }

    .share-header, .user-header {
        display: flex;
        justify-content: space-between;
        align-items: center;
        margin-bottom: 10px;
    }

    .share-title {
        font-size: 18px;
        font-weight: bold;
        color: #4CAF50;
    }

    .share-status {
        padding: 3px 10px;
        border-radius: 4px;
        font-size: 12px;
        font-weight: bold;
    }

    .status-active {
        background: #4CAF50;
        color: white;
    }

    .status-inactive {
        background: #666;
        color: white;
    }

    .share-details {
        color: #aaa;
        font-size: 14px;
        margin-bottom: 10px;
    }

    .share-actions {
        display: flex;
        gap: 10px;
        flex-wrap: wrap;
    }

    .permission-row {
        display: flex;
        gap: 10px;
        margin-bottom: 10px;
        align-items: center;
    }

    .permission-row select,
    .permission-row input {
        flex: 1;
    }

    .log-entry {
        padding: 10px;
        border-bottom: 1px solid #333;
    }

    .log-entry:last-child {
        border-bottom: none;
    }

    .nav-tabs .nav-link {
        color: #aaa;
        border: none;
        background: transparent;
    }

    .nav-tabs .nav-link.active {
        color: #fff;
        background: #333;
        border-bottom: 2px solid #4CAF50 !important;
    }

    .tab-pane {
        padding-top: 15px;
    }
</style>

<script src="/js/shares.js"></script>

<?php include VIEWS_PATH . '/layout/footer.php'; ?>
