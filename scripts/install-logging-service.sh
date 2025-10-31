#!/bin/bash
# Install TSO Background Logging Service
# Ensures logging runs 24/7 in the background

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
INSTALL_DIR="/opt/serveros"

# Detect installation directory
if [ -d "$INSTALL_DIR" ] && [ -f "$INSTALL_DIR/config/config.php" ]; then
    # Use standard installation directory
    :
elif [ -d "$PROJECT_ROOT" ] && [ -f "$PROJECT_ROOT/config/config.php" ]; then
    INSTALL_DIR="$PROJECT_ROOT"
else
    echo "ERROR: Cannot find TSO installation directory"
    exit 1
fi

echo "===================================="
echo "TSO Logging Service Installer"
echo "===================================="
echo ""
echo "Installation directory: $INSTALL_DIR"
echo ""

# Check if running as root
if [ "$EUID" -ne 0 ]; then
    echo "ERROR: This script must be run as root (use sudo)"
    exit 1
fi

# Ensure daemon script exists
DAEMON_SCRIPT="$PROJECT_ROOT/scripts/logging-daemon.php"
if [ ! -f "$DAEMON_SCRIPT" ]; then
    echo "ERROR: Daemon script not found: $DAEMON_SCRIPT"
    exit 1
fi

# Make daemon script executable
chmod +x "$DAEMON_SCRIPT"

# Ensure logs directory exists
mkdir -p "$INSTALL_DIR/logs"
chown -R www-data:www-data "$INSTALL_DIR/logs"
chmod 755 "$INSTALL_DIR/logs"

# Create systemd service file
SYSTEMD_SERVICE="/etc/systemd/system/tso-logging.service"

echo "→ Creating systemd service unit..."

cat > "$SYSTEMD_SERVICE" << EOF
[Unit]
Description=TSO Background Logging Daemon
Documentation=https://github.com/chukfinley/tso
After=network.target mysql.service mariadb.service apache2.service

[Service]
Type=simple
User=www-data
Group=www-data
WorkingDirectory=$INSTALL_DIR
Environment="TSO_INSTALL_DIR=$INSTALL_DIR"
ExecStart=/usr/bin/php $DAEMON_SCRIPT
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=tso-logging

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$INSTALL_DIR/logs

# Resource limits
MemoryLimit=512M
CPUQuota=50%

[Install]
WantedBy=multi-user.target
EOF

echo "✓ Systemd service file created: $SYSTEMD_SERVICE"

# Reload systemd
echo "→ Reloading systemd daemon..."
systemctl daemon-reload

# Enable service to start on boot
echo "→ Enabling logging service..."
systemctl enable tso-logging.service

# Start the service
echo "→ Starting logging service..."
if systemctl start tso-logging.service; then
    echo "✓ Logging service started"
else
    echo "⚠ Warning: Failed to start service immediately"
    echo "  Service will auto-start on next boot"
fi

# Wait a moment and check status
sleep 2
if systemctl is-active --quiet tso-logging.service; then
    echo "✓ Logging service is running"
    
    # Show status
    echo ""
    echo "Service Status:"
    systemctl status tso-logging.service --no-pager -l | head -20
else
    echo "⚠ Warning: Service is not active"
    echo "  Check status with: systemctl status tso-logging.service"
    echo "  Check logs with: journalctl -u tso-logging.service -f"
fi

echo ""
echo "===================================="
echo "✓ Logging Service Installed!"
echo "===================================="
echo ""
echo "The logging daemon is now configured to run 24/7 in the background."
echo ""
echo "Useful commands:"
echo "  • Check status:  systemctl status tso-logging.service"
echo "  • View logs:     journalctl -u tso-logging.service -f"
echo "  • Restart:       systemctl restart tso-logging.service"
echo "  • Stop:          systemctl stop tso-logging.service"
echo "  • Start:         systemctl start tso-logging.service"
echo ""
echo "The daemon will:"
echo "  • Auto-start on system boot"
echo "  • Auto-restart if it crashes"
echo "  • Log data from pages, services, commands, PHP, and updates"
echo ""

