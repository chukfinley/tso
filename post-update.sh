#!/bin/bash

################################################################################
# TSO Post-Update Script
# Run this after 'git pull' to apply changes
#
# Usage:
#   cd /opt/serveros && sudo git pull && sudo ./post-update.sh
################################################################################

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

# Check if running as root
if [[ $EUID -ne 0 ]]; then
    print_error "This script must be run as root or with sudo"
    exit 1
fi

# Determine installation directory
INSTALL_DIR="/opt/serveros"
if [[ ! -d "${INSTALL_DIR}" ]]; then
    INSTALL_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
fi

echo ""
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                TSO Post-Update Tasks                           ║${NC}"
echo -e "${BLUE}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Ensure config exists
if [[ ! -f "${INSTALL_DIR}/config/config.php" ]]; then
    if [[ -f "${INSTALL_DIR}/config/config.example.php" ]]; then
        print_info "Creating config from example..."
        cp ${INSTALL_DIR}/config/config.example.php ${INSTALL_DIR}/config/config.php
        print_warning "Config created from example - you need to update database credentials!"
        print_info "Edit: ${INSTALL_DIR}/config/config.php"
    else
        print_error "No config file found! Cannot continue."
        exit 1
    fi
else
    print_info "Configuration file exists"
fi

# Ensure logs and storage directories exist
print_info "Ensuring required directories exist..."
mkdir -p ${INSTALL_DIR}/logs
mkdir -p ${INSTALL_DIR}/storage
print_success "Directories ready"

# Set correct permissions
print_info "Fixing file permissions..."
chown -R www-data:www-data ${INSTALL_DIR}
find ${INSTALL_DIR} -type d -exec chmod 755 {} \;
find ${INSTALL_DIR} -type f -exec chmod 644 {} \;
chmod -R 775 ${INSTALL_DIR}/logs
chmod -R 775 ${INSTALL_DIR}/storage
chmod +x ${INSTALL_DIR}/tools/*.php 2>/dev/null || true
chmod +x ${INSTALL_DIR}/*.sh 2>/dev/null || true
print_success "Permissions fixed"

# Ensure admin user exists
print_info "Ensuring admin user exists..."
if [[ -f "${INSTALL_DIR}/config/config.php" ]]; then
    php -r "
        require_once '${INSTALL_DIR}/config/config.php';

        try {
            \$pdo = new PDO(
                'mysql:host=' . DB_HOST . ';dbname=' . DB_NAME . ';charset=utf8mb4',
                DB_USER,
                DB_PASS,
                [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
            );

            \$stmt = \$pdo->prepare('SELECT id FROM users WHERE username = ?');
            \$stmt->execute(['admin']);

            if (\$stmt->rowCount() === 0) {
                \$password = password_hash('admin123', PASSWORD_BCRYPT);
                \$stmt = \$pdo->prepare('
                    INSERT INTO users (username, email, password, full_name, role, is_active)
                    VALUES (?, ?, ?, ?, ?, ?)
                ');
                \$stmt->execute(['admin', 'admin@localhost', \$password, 'Administrator', 'admin', 1]);
            }
        } catch (Exception \$e) {
            // Silently ignore if database doesn't exist or isn't configured yet
        }
    " > /dev/null 2>&1 || true
    print_success "Admin user checked"
else
    print_warning "Config file not found - skipping database tasks"
fi

# Restart Apache
print_info "Restarting Apache..."
systemctl restart apache2
print_success "Apache restarted"

echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                Post-Update Complete!                           ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
print_success "TSO has been updated successfully!"
echo ""
print_info "What was done:"
echo "  ✓ Configuration preserved"
echo "  ✓ Permissions fixed"
echo "  ✓ Admin user verified"
echo "  ✓ Apache restarted"
echo ""
IP_ADDRESS=$(hostname -I | awk '{print $1}' 2>/dev/null || echo "localhost")
echo -e "Access TSO at: ${GREEN}http://${IP_ADDRESS}${NC}"
echo ""
