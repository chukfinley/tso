<?php
require_once __DIR__ . '/../config/config.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/User.php';
require_once SRC_PATH . '/Auth.php';

$auth = new Auth();
$auth->requireAdmin(); // Only admins can manage users

$userModel = new User();
$db = Database::getInstance();

$message = '';
$messageType = '';

// Handle form submissions
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $action = $_POST['action'] ?? '';

    if ($action === 'create') {
        $username = trim($_POST['username'] ?? '');
        $email = trim($_POST['email'] ?? '');
        $password = $_POST['password'] ?? '';
        $full_name = trim($_POST['full_name'] ?? '');
        $role = $_POST['role'] ?? 'user';

        if (strlen($password) < PASSWORD_MIN_LENGTH) {
            $message = 'Password must be at least ' . PASSWORD_MIN_LENGTH . ' characters long.';
            $messageType = 'error';
        } else {
            $result = $userModel->create($username, $email, $password, $full_name, $role);
            if ($result) {
                $message = 'User created successfully!';
                $messageType = 'success';

                // Log activity
                $db->query(
                    "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                    [$_SESSION['user_id'], 'user_create', "Created user: $username", $_SERVER['REMOTE_ADDR']]
                );
            } else {
                $message = 'Failed to create user. Username or email may already exist.';
                $messageType = 'error';
            }
        }
    } elseif ($action === 'delete') {
        $userId = intval($_POST['user_id'] ?? 0);
        if ($userId > 0) {
            $deletedUser = $userModel->getById($userId);
            if ($userModel->delete($userId)) {
                $message = 'User deleted successfully!';
                $messageType = 'success';

                // Log activity
                $db->query(
                    "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                    [$_SESSION['user_id'], 'user_delete', "Deleted user: " . $deletedUser['username'], $_SERVER['REMOTE_ADDR']]
                );
            } else {
                $message = 'Failed to delete user. Cannot delete the main admin account.';
                $messageType = 'error';
            }
        }
    } elseif ($action === 'toggle_status') {
        $userId = intval($_POST['user_id'] ?? 0);
        $newStatus = intval($_POST['new_status'] ?? 0);
        if ($userId > 0 && $userId != 1) { // Don't allow disabling main admin
            $userModel->update($userId, ['is_active' => $newStatus]);
            $message = 'User status updated!';
            $messageType = 'success';
            
            // Log activity
            $statusText = $newStatus ? 'activated' : 'deactivated';
            $targetUser = $userModel->getById($userId);
            $db->query(
                "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                [$_SESSION['user_id'], 'user_status_change', "User {$targetUser['username']} $statusText", $_SERVER['REMOTE_ADDR']]
            );
        }
    } elseif ($action === 'update') {
        $userId = intval($_POST['user_id'] ?? 0);
        $email = trim($_POST['email'] ?? '');
        $full_name = trim($_POST['full_name'] ?? '');
        $role = $_POST['role'] ?? 'user';
        
        if ($userId > 0) {
            $updateData = [
                'email' => $email,
                'full_name' => $full_name,
                'role' => $role
            ];
            
            if ($userModel->update($userId, $updateData)) {
                $message = 'User updated successfully!';
                $messageType = 'success';
                
                // Log activity
                $targetUser = $userModel->getById($userId);
                $db->query(
                    "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                    [$_SESSION['user_id'], 'user_update', "Updated user: {$targetUser['username']}", $_SERVER['REMOTE_ADDR']]
                );
            } else {
                $message = 'Failed to update user.';
                $messageType = 'error';
            }
        }
    } elseif ($action === 'change_password') {
        $userId = intval($_POST['user_id'] ?? 0);
        $newPassword = $_POST['new_password'] ?? '';
        
        if ($userId > 0 && strlen($newPassword) >= PASSWORD_MIN_LENGTH) {
            $hashedPassword = password_hash($newPassword, PASSWORD_BCRYPT);
            if ($userModel->update($userId, ['password' => $hashedPassword])) {
                $message = 'Password changed successfully!';
                $messageType = 'success';
                
                // Log activity
                $targetUser = $userModel->getById($userId);
                $db->query(
                    "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                    [$_SESSION['user_id'], 'password_change', "Changed password for user: {$targetUser['username']}", $_SERVER['REMOTE_ADDR']]
                );
            } else {
                $message = 'Failed to change password.';
                $messageType = 'error';
            }
        } else {
            $message = 'Password must be at least ' . PASSWORD_MIN_LENGTH . ' characters long.';
            $messageType = 'error';
        }
    }
}

// Get all users
$users = $userModel->getAll();
$pageTitle = 'User Management';
?>

<?php include VIEWS_PATH . '/layout/header.php'; ?>
<?php include VIEWS_PATH . '/layout/navbar.php'; ?>

<div class="container">
    <h1 style="margin-bottom: 30px; color: #fff;">User Management</h1>

    <?php if ($message): ?>
        <div class="alert alert-<?php echo $messageType; ?>">
            <?php echo htmlspecialchars($message); ?>
        </div>
    <?php endif; ?>

    <!-- Create User Form -->
    <div class="card">
        <div class="card-header">Create New User</div>
        <div class="card-body">
            <form method="POST" action="">
                <input type="hidden" name="action" value="create">

                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 20px;">
                    <div class="form-group">
                        <label for="username">Username *</label>
                        <input type="text" id="username" name="username" class="form-control" required>
                    </div>

                    <div class="form-group">
                        <label for="email">Email *</label>
                        <input type="email" id="email" name="email" class="form-control" required>
                    </div>

                    <div class="form-group">
                        <label for="full_name">Full Name</label>
                        <input type="text" id="full_name" name="full_name" class="form-control">
                    </div>

                    <div class="form-group">
                        <label for="role">Role *</label>
                        <select id="role" name="role" class="form-control" required>
                            <option value="user">User</option>
                            <option value="admin">Admin</option>
                        </select>
                    </div>

                    <div class="form-group">
                        <label for="password">Password * (min. <?php echo PASSWORD_MIN_LENGTH; ?> characters)</label>
                        <input type="password" id="password" name="password" class="form-control" required minlength="<?php echo PASSWORD_MIN_LENGTH; ?>">
                    </div>
                </div>

                <button type="submit" class="btn btn-primary">Create User</button>
            </form>
        </div>
    </div>

    <!-- Users List -->
    <div class="card">
        <div class="card-header">Existing Users</div>
        <div class="card-body">
            <table class="table">
                <thead>
                    <tr>
                        <th>ID</th>
                        <th>Username</th>
                        <th>Email</th>
                        <th>Full Name</th>
                        <th>Role</th>
                        <th>Status</th>
                        <th>Last Login</th>
                        <th>Created</th>
                        <th>Actions</th>
                    </tr>
                </thead>
                <tbody>
                    <?php foreach ($users as $user): ?>
                        <tr>
                            <td><?php echo $user['id']; ?></td>
                            <td><?php echo htmlspecialchars($user['username']); ?></td>
                            <td><?php echo htmlspecialchars($user['email']); ?></td>
                            <td><?php echo htmlspecialchars($user['full_name'] ?? '-'); ?></td>
                            <td>
                                <span class="badge badge-<?php echo $user['role']; ?>">
                                    <?php echo strtoupper($user['role']); ?>
                                </span>
                            </td>
                            <td>
                                <span class="badge badge-<?php echo $user['is_active'] ? 'active' : 'inactive'; ?>">
                                    <?php echo $user['is_active'] ? 'Active' : 'Inactive'; ?>
                                </span>
                            </td>
                            <td><?php echo $user['last_login'] ? date('Y-m-d H:i', strtotime($user['last_login'])) : 'Never'; ?></td>
                            <td><?php echo date('Y-m-d', strtotime($user['created_at'])); ?></td>
                            <td>
                                <button onclick="openEditModal(<?php echo htmlspecialchars(json_encode($user)); ?>)" class="btn btn-secondary" style="padding: 6px 12px; font-size: 12px; margin-right: 5px;">Edit</button>
                                <button onclick="openPasswordModal(<?php echo $user['id']; ?>, '<?php echo htmlspecialchars($user['username']); ?>')" class="btn btn-secondary" style="padding: 6px 12px; font-size: 12px; margin-right: 5px;">Change Password</button>
                                <?php if ($user['id'] != 1): // Don't allow deleting/disabling main admin ?>
                                    <form method="POST" action="" style="display: inline;" onsubmit="return confirm('<?php echo $user['is_active'] ? 'Deactivate' : 'Activate'; ?> this user?');">
                                        <input type="hidden" name="action" value="toggle_status">
                                        <input type="hidden" name="user_id" value="<?php echo $user['id']; ?>">
                                        <input type="hidden" name="new_status" value="<?php echo $user['is_active'] ? 0 : 1; ?>">
                                        <button type="submit" class="btn <?php echo $user['is_active'] ? 'btn-secondary' : 'btn-success'; ?>" style="padding: 6px 12px; font-size: 12px; margin-right: 5px;"><?php echo $user['is_active'] ? 'Disable' : 'Enable'; ?></button>
                                    </form>
                                    <form method="POST" action="" style="display: inline;" onsubmit="return confirmDelete('Are you sure you want to delete this user?');">
                                        <input type="hidden" name="action" value="delete">
                                        <input type="hidden" name="user_id" value="<?php echo $user['id']; ?>">
                                        <button type="submit" class="btn btn-danger" style="padding: 6px 12px; font-size: 12px;">Delete</button>
                                    </form>
                                <?php else: ?>
                                    <span style="color: #666; font-size: 12px;">Protected</span>
                                <?php endif; ?>
                            </td>
                        </tr>
                    <?php endforeach; ?>
                </tbody>
            </table>
        </div>
    </div>
</div>

<!-- Edit User Modal -->
<div id="editModal" style="display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.8); z-index: 9999; align-items: center; justify-content: center;">
    <div style="background: #242424; border: 1px solid #333; border-radius: 8px; padding: 30px; width: 90%; max-width: 500px;">
        <h2 style="color: #fff; margin-bottom: 20px;">Edit User</h2>
        <form method="POST" action="">
            <input type="hidden" name="action" value="update">
            <input type="hidden" name="user_id" id="edit_user_id">
            
            <div class="form-group">
                <label for="edit_username">Username (read-only)</label>
                <input type="text" id="edit_username" class="form-control" disabled>
            </div>
            
            <div class="form-group">
                <label for="edit_email">Email *</label>
                <input type="email" id="edit_email" name="email" class="form-control" required>
            </div>
            
            <div class="form-group">
                <label for="edit_full_name">Full Name</label>
                <input type="text" id="edit_full_name" name="full_name" class="form-control">
            </div>
            
            <div class="form-group">
                <label for="edit_role">Role *</label>
                <select id="edit_role" name="role" class="form-control" required>
                    <option value="user">User</option>
                    <option value="admin">Admin</option>
                </select>
            </div>
            
            <div style="display: flex; gap: 10px; margin-top: 20px;">
                <button type="submit" class="btn btn-primary">Save Changes</button>
                <button type="button" onclick="closeEditModal()" class="btn btn-secondary">Cancel</button>
            </div>
        </form>
    </div>
</div>

<!-- Change Password Modal -->
<div id="passwordModal" style="display: none; position: fixed; top: 0; left: 0; width: 100%; height: 100%; background: rgba(0,0,0,0.8); z-index: 9999; align-items: center; justify-content: center;">
    <div style="background: #242424; border: 1px solid #333; border-radius: 8px; padding: 30px; width: 90%; max-width: 500px;">
        <h2 style="color: #fff; margin-bottom: 20px;">Change Password</h2>
        <p style="color: #b0b0b0; margin-bottom: 20px;">Changing password for: <strong id="password_username" style="color: #ff8c00;"></strong></p>
        <form method="POST" action="">
            <input type="hidden" name="action" value="change_password">
            <input type="hidden" name="user_id" id="password_user_id">
            
            <div class="form-group">
                <label for="new_password">New Password * (min. <?php echo PASSWORD_MIN_LENGTH; ?> characters)</label>
                <input type="password" id="new_password" name="new_password" class="form-control" required minlength="<?php echo PASSWORD_MIN_LENGTH; ?>">
            </div>
            
            <div class="form-group">
                <label for="confirm_password">Confirm Password *</label>
                <input type="password" id="confirm_password" class="form-control" required minlength="<?php echo PASSWORD_MIN_LENGTH; ?>">
            </div>
            
            <div style="display: flex; gap: 10px; margin-top: 20px;">
                <button type="submit" class="btn btn-primary" onclick="return validatePassword()">Change Password</button>
                <button type="button" onclick="closePasswordModal()" class="btn btn-secondary">Cancel</button>
            </div>
        </form>
    </div>
</div>

<script>
function openEditModal(user) {
    document.getElementById('edit_user_id').value = user.id;
    document.getElementById('edit_username').value = user.username;
    document.getElementById('edit_email').value = user.email;
    document.getElementById('edit_full_name').value = user.full_name || '';
    document.getElementById('edit_role').value = user.role;
    document.getElementById('editModal').style.display = 'flex';
}

function closeEditModal() {
    document.getElementById('editModal').style.display = 'none';
}

function openPasswordModal(userId, username) {
    document.getElementById('password_user_id').value = userId;
    document.getElementById('password_username').textContent = username;
    document.getElementById('new_password').value = '';
    document.getElementById('confirm_password').value = '';
    document.getElementById('passwordModal').style.display = 'flex';
}

function closePasswordModal() {
    document.getElementById('passwordModal').style.display = 'none';
}

function validatePassword() {
    const newPassword = document.getElementById('new_password').value;
    const confirmPassword = document.getElementById('confirm_password').value;
    
    if (newPassword !== confirmPassword) {
        alert('Passwords do not match!');
        return false;
    }
    
    if (newPassword.length < <?php echo PASSWORD_MIN_LENGTH; ?>) {
        alert('Password must be at least <?php echo PASSWORD_MIN_LENGTH; ?> characters long!');
        return false;
    }
    
    return true;
}

// Close modals when clicking outside
document.getElementById('editModal').addEventListener('click', function(e) {
    if (e.target === this) closeEditModal();
});

document.getElementById('passwordModal').addEventListener('click', function(e) {
    if (e.target === this) closePasswordModal();
});
</script>

<?php include VIEWS_PATH . '/layout/footer.php'; ?>
