<?php
require_once __DIR__ . '/../config/config.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/User.php';
require_once SRC_PATH . '/Auth.php';

$auth = new Auth();
$auth->requireLogin(); // Any logged-in user can access their profile

$userModel = new User();
$db = Database::getInstance();

$message = '';
$messageType = '';

// Get current user data
$currentUser = $userModel->getById($_SESSION['user_id']);

// Handle form submissions
if ($_SERVER['REQUEST_METHOD'] === 'POST') {
    $action = $_POST['action'] ?? '';

    if ($action === 'update_profile') {
        $email = trim($_POST['email'] ?? '');
        $full_name = trim($_POST['full_name'] ?? '');

        if ($email && filter_var($email, FILTER_VALIDATE_EMAIL)) {
            $updateData = [
                'email' => $email,
                'full_name' => $full_name
            ];

            if ($userModel->update($_SESSION['user_id'], $updateData)) {
                $message = 'Profile updated successfully!';
                $messageType = 'success';

                // Log activity
                $db->query(
                    "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                    [$_SESSION['user_id'], 'profile_update', "Updated own profile", $_SERVER['REMOTE_ADDR']]
                );

                // Refresh user data
                $currentUser = $userModel->getById($_SESSION['user_id']);
            } else {
                $message = 'Failed to update profile. Email may already be in use.';
                $messageType = 'error';
            }
        } else {
            $message = 'Please provide a valid email address.';
            $messageType = 'error';
        }
    } elseif ($action === 'change_password') {
        $oldPassword = $_POST['old_password'] ?? '';
        $newPassword = $_POST['new_password'] ?? '';
        $confirmPassword = $_POST['confirm_password'] ?? '';

        if (strlen($newPassword) < PASSWORD_MIN_LENGTH) {
            $message = 'Password must be at least ' . PASSWORD_MIN_LENGTH . ' characters long.';
            $messageType = 'error';
        } elseif ($newPassword !== $confirmPassword) {
            $message = 'New passwords do not match.';
            $messageType = 'error';
        } elseif (!password_verify($oldPassword, $currentUser['password'])) {
            $message = 'Incorrect old password.';
            $messageType = 'error';
        } else {
            $hashedPassword = password_hash($newPassword, PASSWORD_BCRYPT);
            if ($userModel->update($_SESSION['user_id'], ['password' => $hashedPassword])) {
                $message = 'Password changed successfully!';
                $messageType = 'success';

                // Log activity
                $db->query(
                    "INSERT INTO activity_log (user_id, action, description, ip_address) VALUES (?, ?, ?, ?)",
                    [$_SESSION['user_id'], 'password_change', "Changed own password", $_SERVER['REMOTE_ADDR']]
                );

                // Refresh user data
                $currentUser = $userModel->getById($_SESSION['user_id']);
            } else {
                $message = 'Failed to change password.';
                $messageType = 'error';
            }
        }
    }
}

$pageTitle = 'My Profile';
?>

<?php include VIEWS_PATH . '/layout/header.php'; ?>
<?php include VIEWS_PATH . '/layout/navbar.php'; ?>

<div class="container">
    <h1 style="margin-bottom: 30px; color: #fff;">My Profile</h1>

    <?php if ($message): ?>
        <div class="alert alert-<?php echo $messageType; ?>">
            <?php echo htmlspecialchars($message); ?>
        </div>
    <?php endif; ?>

    <!-- Profile Information Card -->
    <div class="card">
        <div class="card-header">Profile Information</div>
        <div class="card-body">
            <form method="POST" action="">
                <input type="hidden" name="action" value="update_profile">

                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 20px;">
                    <div class="form-group">
                        <label for="username">Username</label>
                        <input type="text" id="username" class="form-control" value="<?php echo htmlspecialchars($currentUser['username']); ?>" disabled>
                        <small style="color: #888;">Username cannot be changed</small>
                    </div>

                    <div class="form-group">
                        <label for="role">Role</label>
                        <input type="text" id="role" class="form-control" value="<?php echo htmlspecialchars(ucfirst($currentUser['role'])); ?>" disabled>
                    </div>

                    <div class="form-group">
                        <label for="email">Email *</label>
                        <input type="email" id="email" name="email" class="form-control" value="<?php echo htmlspecialchars($currentUser['email']); ?>" required>
                    </div>

                    <div class="form-group">
                        <label for="full_name">Full Name</label>
                        <input type="text" id="full_name" name="full_name" class="form-control" value="<?php echo htmlspecialchars($currentUser['full_name'] ?? ''); ?>">
                    </div>
                </div>

                <div style="margin-top: 20px;">
                    <p style="color: #888; font-size: 14px;">
                        <strong>Account Created:</strong> <?php echo date('F j, Y', strtotime($currentUser['created_at'])); ?>
                    </p>
                    <p style="color: #888; font-size: 14px;">
                        <strong>Last Login:</strong> <?php echo $currentUser['last_login'] ? date('F j, Y g:i A', strtotime($currentUser['last_login'])) : 'Never'; ?>
                    </p>
                </div>

                <button type="submit" class="btn btn-primary" style="margin-top: 20px;">Update Profile</button>
            </form>
        </div>
    </div>

    <!-- Change Password Card -->
    <div class="card">
        <div class="card-header">Change Password</div>
        <div class="card-body">
            <form method="POST" action="" onsubmit="return validatePasswordForm()">
                <input type="hidden" name="action" value="change_password">

                <div style="display: grid; grid-template-columns: 1fr; gap: 20px; max-width: 500px;">
                    <div class="form-group">
                        <label for="old_password">Old Password *</label>
                        <input type="password" id="old_password" name="old_password" class="form-control" required>
                    </div>

                    <div class="form-group">
                        <label for="new_password">New Password * (min. <?php echo PASSWORD_MIN_LENGTH; ?> characters)</label>
                        <input type="password" id="new_password" name="new_password" class="form-control" required minlength="<?php echo PASSWORD_MIN_LENGTH; ?>">
                    </div>

                    <div class="form-group">
                        <label for="confirm_password">Confirm New Password *</label>
                        <input type="password" id="confirm_password" name="confirm_password" class="form-control" required minlength="<?php echo PASSWORD_MIN_LENGTH; ?>">
                    </div>
                </div>

                <button type="submit" class="btn btn-primary" style="margin-top: 20px;">Change Password</button>
            </form>
        </div>
    </div>
</div>

<script>
function validatePasswordForm() {
    const newPassword = document.getElementById('new_password').value;
    const confirmPassword = document.getElementById('confirm_password').value;

    if (newPassword !== confirmPassword) {
        alert('New passwords do not match!');
        return false;
    }

    if (newPassword.length < <?php echo PASSWORD_MIN_LENGTH; ?>) {
        alert('Password must be at least <?php echo PASSWORD_MIN_LENGTH; ?> characters long!');
        return false;
    }

    return true;
}
</script>

<?php include VIEWS_PATH . '/layout/footer.php'; ?>
