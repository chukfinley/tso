#!/bin/bash

################################################################################
# ServerOS Update Script
# Updates ServerOS to latest version from GitHub
################################################################################

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
REPO_URL="https://github.com/chukfinley/tso.git"
TEMP_DIR="/tmp/serveros-update-$$"
INSTALL_DIR="/opt/serveros"

print_header() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                  ServerOS Update Script                        ║"
    echo "╚════════════════════════════════════════════════════════════════╝"
    echo -e "${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

check_root() {
    if [[ $EUID -ne 0 ]]; then
        print_error "This script must be run as root or with sudo"
        exit 1
    fi
}

check_installation() {
    if [[ ! -d "${INSTALL_DIR}" ]] || [[ ! -f "${INSTALL_DIR}/config/config.php" ]]; then
        print_error "ServerOS is not installed!"
        print_info "Run the installer first:"
        echo "  curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash"
        exit 1
    fi
    print_success "Existing installation found"
}

backup_config() {
    print_info "Backing up configuration..."
    cp ${INSTALL_DIR}/config/config.php /tmp/serveros-config-backup.php
    print_success "Configuration backed up"
}

clone_latest() {
    print_info "Fetching latest version from GitHub..."

    rm -rf "${TEMP_DIR}"

    if git clone --depth 1 "${REPO_URL}" "${TEMP_DIR}" > /dev/null 2>&1; then
        print_success "Latest version downloaded"
    else
        print_error "Failed to download latest version"
        exit 1
    fi
}

update_files() {
    print_info "Updating application files..."

    # Update main directories
    cp -r "${TEMP_DIR}/public" ${INSTALL_DIR}/ 2>/dev/null || true
    cp -r "${TEMP_DIR}/src" ${INSTALL_DIR}/ 2>/dev/null || true
    cp -r "${TEMP_DIR}/views" ${INSTALL_DIR}/ 2>/dev/null || true
    cp -r "${TEMP_DIR}/tools" ${INSTALL_DIR}/ 2>/dev/null || true

    # Update init.sql
    cp "${TEMP_DIR}/init.sql" ${INSTALL_DIR}/ 2>/dev/null || true

    print_success "Application files updated"
}

restore_config() {
    print_info "Restoring configuration..."
    cp /tmp/serveros-config-backup.php ${INSTALL_DIR}/config/config.php
    rm -f /tmp/serveros-config-backup.php
    print_success "Configuration restored"
}

fix_permissions() {
    print_info "Fixing permissions..."
    chown -R www-data:www-data ${INSTALL_DIR}
    find ${INSTALL_DIR} -type d -exec chmod 755 {} \;
    find ${INSTALL_DIR} -type f -exec chmod 644 {} \;
    chmod -R 775 ${INSTALL_DIR}/logs
    chmod -R 775 ${INSTALL_DIR}/storage
    chmod +x ${INSTALL_DIR}/tools/*.php 2>/dev/null || true
    print_success "Permissions fixed"
}

ensure_admin() {
    print_info "Ensuring admin user exists..."

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
                echo 'Admin user created';
            }
        } catch (Exception \$e) {
            // Silently ignore if table doesn't exist yet
        }
    " > /dev/null 2>&1 || true

    print_success "Admin user checked"
}

restart_services() {
    print_info "Restarting Apache..."
    systemctl restart apache2
    print_success "Apache restarted"
}

cleanup() {
    print_info "Cleaning up..."
    rm -rf "${TEMP_DIR}"
    print_success "Cleanup complete"
}

show_completion() {
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                Update Completed Successfully!                  ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    print_success "ServerOS has been updated to the latest version!"
    echo ""
    print_info "What was updated:"
    echo "  ✓ Application files (PHP, HTML, CSS, JS)"
    echo "  ✓ Tools and utilities"
    echo "  ✓ Database schema file"
    echo ""
    print_info "What was preserved:"
    echo "  ✓ Configuration (database credentials)"
    echo "  ✓ Database and all users"
    echo "  ✓ Logs and storage"
    echo ""
    IP_ADDRESS=$(hostname -I | awk '{print $1}')
    echo "Access your ServerOS at: ${GREEN}http://${IP_ADDRESS}${NC}"
    echo ""
}

main() {
    print_header

    check_root
    check_installation

    echo ""
    print_info "Starting update process..."
    echo ""

    backup_config
    clone_latest
    update_files
    restore_config
    fix_permissions
    ensure_admin
    restart_services
    cleanup

    show_completion
}

main "$@"
