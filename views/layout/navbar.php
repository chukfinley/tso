<?php
$currentPage = basename($_SERVER['PHP_SELF'], '.php');
?>
<nav class="navbar">
    <a href="/dashboard.php" class="navbar-brand">ServerOS</a>
    <ul class="navbar-menu">
        <li><a href="/dashboard.php" class="<?php echo $currentPage === 'dashboard' ? 'active' : ''; ?>">Dashboard</a></li>
        <li><a href="/disks.php" class="<?php echo $currentPage === 'disks' ? 'active' : ''; ?>">Disks</a></li>
        <li><a href="/shares.php" class="<?php echo $currentPage === 'shares' ? 'active' : ''; ?>">Shares</a></li>
        <li><a href="/users.php" class="<?php echo $currentPage === 'users' ? 'active' : ''; ?>">Users</a></li>
        <li><a href="/docker.php" class="<?php echo $currentPage === 'docker' ? 'active' : ''; ?>">Docker</a></li>
        <li><a href="/vms.php" class="<?php echo $currentPage === 'vms' ? 'active' : ''; ?>">VMs</a></li>
        <li><a href="/plugins.php" class="<?php echo $currentPage === 'plugins' ? 'active' : ''; ?>">Plugins</a></li>
        <?php if (isset($_SESSION['role']) && $_SESSION['role'] === 'admin'): ?>
        <li><a href="/terminal.php" class="<?php echo $currentPage === 'terminal' ? 'active' : ''; ?>">Terminal</a></li>
        <?php endif; ?>
        <li><a href="/settings.php" class="<?php echo $currentPage === 'settings' ? 'active' : ''; ?>">Settings</a></li>
    </ul>
    <div class="navbar-right">
        <div class="user-info">
            Logged in as: <span><?php echo htmlspecialchars($_SESSION['username'] ?? 'Guest'); ?></span>
            <?php if (isset($_SESSION['role']) && $_SESSION['role'] === 'admin'): ?>
                <span class="badge badge-admin" style="margin-left: 10px;">Admin</span>
            <?php endif; ?>
        </div>
        <a href="/profile.php" class="btn btn-secondary" style="font-size: 12px; padding: 8px 15px; margin-right: 10px;">Profile</a>
        <a href="/logout.php" class="btn btn-secondary" style="font-size: 12px; padding: 8px 15px;">Logout</a>
    </div>
</nav>
