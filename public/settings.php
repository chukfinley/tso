<?php
require_once __DIR__ . '/../config/config.php';
require_once SRC_PATH . '/Database.php';
require_once SRC_PATH . '/User.php';
require_once SRC_PATH . '/Auth.php';

$auth = new Auth();
$auth->requireLogin();

$pageTitle = 'Settings';
?>

<?php include VIEWS_PATH . '/layout/header.php'; ?>
<?php include VIEWS_PATH . '/layout/navbar.php'; ?>

<div class="container">
    <h1 style="margin-bottom: 30px; color: #fff;">System Settings</h1>

    <!-- System Update Card -->
    <div class="card">
        <div class="card-header">
            <span style="font-size: 20px;">🔄</span> System Update
        </div>
        <div class="card-body">
            <div id="update-status-container"></div>
            
            <div style="margin-bottom: 20px;">
                <p style="color: #b0b0b0; margin-bottom: 10px;">
                    <strong>Current Version:</strong> <?php echo APP_VERSION; ?>
                </p>
                <p style="color: #b0b0b0; margin-bottom: 20px;">
                    Update TSO to the latest version from GitHub. This process will:
                </p>
                <ul style="color: #888; margin-left: 20px; margin-bottom: 20px;">
                    <li>Update all application files</li>
                    <li>Preserve your configuration and database</li>
                    <li>Update the monitoring system</li>
                    <li>Restart services automatically</li>
                </ul>
                <p style="color: #ff9800; margin-bottom: 20px;">
                    <strong>⚠️ Note:</strong> The update process may take 1-2 minutes. Your session will remain active.
                </p>
            </div>

            <button id="update-btn" class="btn btn-primary" onclick="startUpdate()">
                <span id="update-btn-text">🔄 Update System</span>
            </button>

            <div id="update-output" style="display: none; margin-top: 20px;">
                <div style="background: #1a1a1a; border: 1px solid #333; border-radius: 5px; padding: 15px; max-height: 400px; overflow-y: auto;">
                    <pre id="update-log" style="color: #4caf50; margin: 0; font-family: monospace; font-size: 12px; white-space: pre-wrap;"></pre>
                </div>
            </div>
        </div>
    </div>

    <!-- General Settings Card -->
    <div class="card">
        <div class="card-header">General Settings</div>
        <div class="card-body">
            <p style="color: #666;">Additional system settings will be available here in future updates.</p>
        </div>
    </div>
</div>

<script>
function startUpdate() {
    const btn = document.getElementById('update-btn');
    const btnText = document.getElementById('update-btn-text');
    const output = document.getElementById('update-output');
    const log = document.getElementById('update-log');
    const statusContainer = document.getElementById('update-status-container');
    
    // Disable button
    btn.disabled = true;
    btnText.textContent = '⏳ Updating...';
    
    // Show output
    output.style.display = 'block';
    log.textContent = 'Starting system update...\n';
    
    // Clear any previous status
    statusContainer.innerHTML = '';
    
    // Start the update
    fetch('/api/system-update.php', {
        method: 'POST'
    })
    .then(response => response.json())
    .then(data => {
        if (data.success) {
            log.textContent += '\n' + data.output;
            log.textContent += '\n\n✓ Update completed successfully!\n';
            
            // Show success message
            statusContainer.innerHTML = `
                <div class="alert alert-success" style="margin-bottom: 20px;">
                    <strong>✓ Success!</strong> System has been updated to the latest version.
                    <br><br>
                    <button class="btn btn-primary" onclick="window.location.reload()">
                        Reload Page
                    </button>
                </div>
            `;
            
            btn.disabled = false;
            btnText.textContent = '🔄 Update System';
        } else {
            log.textContent += '\n\n✗ Update failed:\n' + (data.error || 'Unknown error');
            
            // Show error message
            statusContainer.innerHTML = `
                <div class="alert alert-error" style="margin-bottom: 20px;">
                    <strong>✗ Error:</strong> ${data.error || 'Update failed. Check the log below.'}
                </div>
            `;
            
            btn.disabled = false;
            btnText.textContent = '🔄 Update System';
        }
    })
    .catch(error => {
        log.textContent += '\n\n✗ Error: ' + error.message;
        
        // Show error message
        statusContainer.innerHTML = `
            <div class="alert alert-error" style="margin-bottom: 20px;">
                <strong>✗ Error:</strong> Failed to connect to update service.
            </div>
        `;
        
        btn.disabled = false;
        btnText.textContent = '🔄 Update System';
    });
}
</script>

<?php include VIEWS_PATH . '/layout/footer.php'; ?>
