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
                                <?php if ($user['id'] != 1): // Don't allow deleting main admin ?>
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

<?php include VIEWS_PATH . '/layout/footer.php'; ?>
