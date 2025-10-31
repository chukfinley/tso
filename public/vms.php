<?php
require_once __DIR__ . '/../config/config.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/User.php';
require_once SRC_PATH . '/Auth.php';

$auth = new Auth();
$auth->requireLogin();

$pageTitle = 'Virtual Machines';
?>

<?php include VIEWS_PATH . '/layout/header.php'; ?>
<?php include VIEWS_PATH . '/layout/navbar.php'; ?>

<div class="container">
    <h1 style="margin-bottom: 30px; color: #fff;">Virtual Machines</h1>

    <div class="card">
        <div class="card-header">VM Management</div>
        <div class="card-body">
            <p style="color: #666;">This feature will be implemented next. Here you can manage virtual machines (KVM/QEMU).</p>
        </div>
    </div>
</div>

<?php include VIEWS_PATH . '/layout/footer.php'; ?>
