#!/bin/bash

################################################################################
# ServerOS Installation Script
# A Debian-based Server Management System
################################################################################

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
INSTALL_DIR="/opt/serveros"
WEB_ROOT="${INSTALL_DIR}/public"
DB_NAME="servermanager"
DB_USER="serveros"
DB_PASS=$(openssl rand -base64 12)
APACHE_CONF="/etc/apache2/sites-available/serveros.conf"
REPO_URL="https://github.com/chukfinley/tso.git"

################################################################################
# Helper Functions
################################################################################

print_header() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                    ServerOS Installer                          ║"
    echo "║          Server Management System Installation Script          ║"
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

check_os() {
    if [[ ! -f /etc/debian_version ]]; then
        print_error "This script is designed for Debian-based systems only"
        exit 1
    fi
    print_success "Debian-based system detected"
}

################################################################################
# Installation Steps
################################################################################

update_system() {
    print_info "Updating system packages..."
    apt-get update -qq
    apt-get upgrade -y -qq
    print_success "System updated"
}

install_dependencies() {
    print_info "Installing LAMP stack and dependencies..."

    # Install packages
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
        apache2 \
        mariadb-server \
        php \
        php-mysql \
        php-cli \
        php-mbstring \
        php-xml \
        php-curl \
        php-zip \
        php-gd \
        libapache2-mod-php \
        curl \
        wget \
        git \
        unzip \
        openssl \
        > /dev/null 2>&1

    print_success "LAMP stack installed"
}

configure_mariadb() {
    print_info "Configuring MariaDB..."

    # Start MariaDB
    systemctl start mariadb
    systemctl enable mariadb > /dev/null 2>&1

    # Secure installation (automated)
    mysql -e "DELETE FROM mysql.user WHERE User='';"
    mysql -e "DELETE FROM mysql.user WHERE User='root' AND Host NOT IN ('localhost', '127.0.0.1', '::1');"
    mysql -e "DROP DATABASE IF EXISTS test;"
    mysql -e "DELETE FROM mysql.db WHERE Db='test' OR Db='test\\_%';"
    mysql -e "FLUSH PRIVILEGES;"

    print_success "MariaDB configured and secured"
}

create_database() {
    print_info "Creating database and user..."

    # Create database
    mysql -e "CREATE DATABASE IF NOT EXISTS ${DB_NAME};"

    # Create user with password
    mysql -e "CREATE USER IF NOT EXISTS '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}';"
    mysql -e "GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'localhost';"
    mysql -e "FLUSH PRIVILEGES;"

    print_success "Database '${DB_NAME}' created"
}

import_schema() {
    print_info "Importing database schema..."

    # Get script directory
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

    if [[ -f "${SCRIPT_DIR}/init.sql" ]]; then
        mysql ${DB_NAME} < "${SCRIPT_DIR}/init.sql"
        print_success "Database schema imported"
    else
        print_error "init.sql not found!"
        exit 1
    fi
}

deploy_files() {
    print_info "Deploying application files..."

    # Get script directory
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

    # Check if we're in a git repository
    if [[ -d "${SCRIPT_DIR}/.git" ]]; then
        # We're in a git repo - clone it to installation directory
        print_info "Cloning git repository to ${INSTALL_DIR}..."

        # Remove old installation if exists
        if [[ -d "${INSTALL_DIR}" ]]; then
            rm -rf ${INSTALL_DIR}
        fi

        # Clone repository
        git clone "${REPO_URL}" ${INSTALL_DIR} > /dev/null 2>&1

        print_success "Git repository cloned"
    else
        # Not in git repo - copy files (fallback)
        print_info "Copying files to ${INSTALL_DIR}..."

        # Create installation directory
        mkdir -p ${INSTALL_DIR}

        # Copy all directories and files
        cp -r "${SCRIPT_DIR}/config" ${INSTALL_DIR}/ 2>/dev/null || true
        cp -r "${SCRIPT_DIR}/public" ${INSTALL_DIR}/ 2>/dev/null || true
        cp -r "${SCRIPT_DIR}/src" ${INSTALL_DIR}/ 2>/dev/null || true
        cp -r "${SCRIPT_DIR}/views" ${INSTALL_DIR}/ 2>/dev/null || true
        cp -r "${SCRIPT_DIR}/tools" ${INSTALL_DIR}/ 2>/dev/null || true
        cp "${SCRIPT_DIR}/init.sql" ${INSTALL_DIR}/ 2>/dev/null || true

        print_success "Files copied"
    fi

    # Create logs and storage directories
    mkdir -p ${INSTALL_DIR}/logs
    mkdir -p ${INSTALL_DIR}/storage

    print_success "Application files deployed"
}

configure_app() {
    print_info "Configuring application..."

    # Update config.php with database credentials
    sed -i "s/define('DB_HOST', 'localhost');/define('DB_HOST', 'localhost');/" ${INSTALL_DIR}/config/config.php
    sed -i "s/define('DB_NAME', 'servermanager');/define('DB_NAME', '${DB_NAME}');/" ${INSTALL_DIR}/config/config.php
    sed -i "s/define('DB_USER', 'root');/define('DB_USER', '${DB_USER}');/" ${INSTALL_DIR}/config/config.php
    sed -i "s/define('DB_PASS', '');/define('DB_PASS', '${DB_PASS}');/" ${INSTALL_DIR}/config/config.php

    # Update base URL
    HOSTNAME=$(hostname -f 2>/dev/null || hostname)
    sed -i "s#define('BASE_URL', 'http://localhost');#define('BASE_URL', 'http://${HOSTNAME}');#" ${INSTALL_DIR}/config/config.php

    # Disable error display for production
    sed -i "s/ini_set('display_errors', 1);/ini_set('display_errors', 0);/" ${INSTALL_DIR}/config/config.php

    print_success "Application configured"
}

create_admin_user() {
    print_info "Creating default admin user..."

    # Create admin user using PHP
    php -r "
        require_once '${INSTALL_DIR}/config/config.php';
        require_once '${INSTALL_DIR}/src/Database.php';

        try {
            \$pdo = new PDO(
                'mysql:host=' . DB_HOST . ';dbname=' . DB_NAME . ';charset=utf8mb4',
                DB_USER,
                DB_PASS,
                [PDO::ATTR_ERRMODE => PDO::ERRMODE_EXCEPTION]
            );

            // Check if admin exists
            \$stmt = \$pdo->prepare('SELECT id FROM users WHERE username = ?');
            \$stmt->execute(['admin']);

            if (\$stmt->rowCount() === 0) {
                // Create admin user
                \$password = password_hash('admin123', PASSWORD_BCRYPT);
                \$stmt = \$pdo->prepare('
                    INSERT INTO users (username, email, password, full_name, role, is_active)
                    VALUES (?, ?, ?, ?, ?, ?)
                ');
                \$stmt->execute(['admin', 'admin@localhost', \$password, 'Administrator', 'admin', 1]);
                echo 'Admin user created';
            } else {
                echo 'Admin user already exists';
            }
        } catch (Exception \$e) {
            echo 'Error: ' . \$e->getMessage();
            exit(1);
        }
    "

    print_success "Admin user ready (admin/admin123)"
}

configure_apache() {
    print_info "Configuring Apache2..."

    # Create Apache configuration
    cat > ${APACHE_CONF} << EOF
<VirtualHost *:80>
    ServerName $(hostname -f 2>/dev/null || hostname)
    ServerAdmin admin@localhost
    DocumentRoot ${WEB_ROOT}

    <Directory ${WEB_ROOT}>
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted

        # Redirect to login if not logged in
        DirectoryIndex index.php
    </Directory>

    # Deny access to sensitive directories
    <Directory ${INSTALL_DIR}/config>
        Require all denied
    </Directory>

    <Directory ${INSTALL_DIR}/src>
        Require all denied
    </Directory>

    <Directory ${INSTALL_DIR}/views>
        Require all denied
    </Directory>

    ErrorLog \${APACHE_LOG_DIR}/serveros_error.log
    CustomLog \${APACHE_LOG_DIR}/serveros_access.log combined

    # Security Headers
    Header always set X-Frame-Options "SAMEORIGIN"
    Header always set X-Content-Type-Options "nosniff"
    Header always set X-XSS-Protection "1; mode=block"
</VirtualHost>
EOF

    # Enable required Apache modules
    a2enmod rewrite > /dev/null 2>&1
    a2enmod headers > /dev/null 2>&1
    a2enmod php$(php -r 'echo PHP_MAJOR_VERSION.".".PHP_MINOR_VERSION;') > /dev/null 2>&1 || true

    # Disable default site
    a2dissite 000-default > /dev/null 2>&1 || true

    # Enable ServerOS site
    a2ensite serveros > /dev/null 2>&1

    print_success "Apache configured"
}

set_permissions() {
    print_info "Setting file permissions..."

    # Set ownership to www-data
    chown -R www-data:www-data ${INSTALL_DIR}

    # Set directory permissions
    find ${INSTALL_DIR} -type d -exec chmod 755 {} \;

    # Set file permissions
    find ${INSTALL_DIR} -type f -exec chmod 644 {} \;

    # Make logs and storage writable
    chmod -R 775 ${INSTALL_DIR}/logs
    chmod -R 775 ${INSTALL_DIR}/storage

    print_success "Permissions set"
}

restart_services() {
    print_info "Restarting services..."

    systemctl restart apache2
    systemctl enable apache2 > /dev/null 2>&1

    print_success "Services restarted"
}

configure_firewall() {
    print_info "Configuring firewall..."

    # Check if UFW is installed
    if command -v ufw &> /dev/null; then
        ufw allow 80/tcp > /dev/null 2>&1 || true
        ufw allow 443/tcp > /dev/null 2>&1 || true
        print_success "Firewall configured (HTTP/HTTPS allowed)"
    else
        print_warning "UFW not installed, skipping firewall configuration"
    fi
}

show_completion_info() {
    IP_ADDRESS=$(hostname -I | awk '{print $1}')
    HOSTNAME=$(hostname -f 2>/dev/null || hostname)

    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                 Installation Completed!                        ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    print_success "ServerOS has been successfully installed!"
    echo ""
    echo -e "${BLUE}Access Information:${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "  URL:      ${GREEN}http://${IP_ADDRESS}${NC}"
    echo -e "  Hostname: ${GREEN}http://${HOSTNAME}${NC}"
    echo ""
    echo -e "${BLUE}Default Credentials:${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "  Username: ${GREEN}admin${NC}"
    echo -e "  Password: ${GREEN}admin123${NC}"
    echo ""
    echo -e "${BLUE}Database Information:${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "  Database: ${GREEN}${DB_NAME}${NC}"
    echo -e "  User:     ${GREEN}${DB_USER}${NC}"
    echo -e "  Password: ${GREEN}${DB_PASS}${NC}"
    echo ""
    echo -e "${BLUE}Installation Directory:${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "  Path: ${GREEN}${INSTALL_DIR}${NC}"
    echo ""
    echo -e "${YELLOW}⚠  IMPORTANT SECURITY NOTES:${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "  1. Change the default admin password immediately!"
    echo "  2. Database credentials saved in: ${INSTALL_DIR}/config/config.php"
    echo "  3. Keep database password secure: ${DB_PASS}"
    echo ""

    # Save credentials to file
    cat > /root/serveros_credentials.txt << EOF
ServerOS Installation Credentials
==================================
Generated: $(date)

Web Access:
-----------
URL: http://${IP_ADDRESS}
Hostname: http://${HOSTNAME}

Default Login:
--------------
Username: admin
Password: admin123

Database:
---------
Database: ${DB_NAME}
User: ${DB_USER}
Password: ${DB_PASS}

Installation Path:
------------------
${INSTALL_DIR}

Apache Config:
--------------
${APACHE_CONF}

IMPORTANT: Change default admin password immediately!
EOF

    chmod 600 /root/serveros_credentials.txt
    print_success "Credentials saved to: /root/serveros_credentials.txt"
    echo ""
}

################################################################################
# Update Detection
################################################################################

detect_existing_installation() {
    if [[ -d "${INSTALL_DIR}" ]] && [[ -f "${INSTALL_DIR}/config/config.php" ]]; then
        return 0  # Installation exists
    else
        return 1  # No installation
    fi
}

perform_update() {
    print_info "Existing installation detected - performing update..."
    echo ""

    # Backup current config
    print_info "Backing up current configuration..."
    cp ${INSTALL_DIR}/config/config.php /tmp/serveros-config-backup.php
    print_success "Configuration backed up"

    # Check if installation is a git repository
    if [[ -d "${INSTALL_DIR}/.git" ]]; then
        print_info "Git repository detected - pulling latest changes..."
        cd ${INSTALL_DIR}
        git pull origin master > /dev/null 2>&1 || git pull origin main > /dev/null 2>&1
        print_success "Latest changes pulled from git"
    else
        print_info "No git repository - fetching latest version..."
        # Deploy new files (will overwrite application files but not config)
        SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
        cp -r "${SCRIPT_DIR}/public" ${INSTALL_DIR}/ 2>/dev/null || true
        cp -r "${SCRIPT_DIR}/src" ${INSTALL_DIR}/ 2>/dev/null || true
        cp -r "${SCRIPT_DIR}/views" ${INSTALL_DIR}/ 2>/dev/null || true
        cp -r "${SCRIPT_DIR}/tools" ${INSTALL_DIR}/ 2>/dev/null || true
        cp "${SCRIPT_DIR}/init.sql" ${INSTALL_DIR}/ 2>/dev/null || true
        print_success "Application files updated"
    fi

    # Restore config
    print_info "Restoring configuration..."
    cp /tmp/serveros-config-backup.php ${INSTALL_DIR}/config/config.php
    rm -f /tmp/serveros-config-backup.php
    print_success "Configuration restored"

    # Ensure admin user exists with correct password
    create_admin_user

    # Update permissions
    set_permissions

    # Restart services
    restart_services

    echo ""
    print_success "Update completed successfully!"
    echo ""
    print_info "Configuration preserved from existing installation"
    print_info "Application files updated to latest version"
    print_info "Database and users unchanged"
    echo ""
    print_info "To update in the future, you can simply run:"
    echo "  cd ${INSTALL_DIR} && sudo git pull && sudo ./post-update.sh"
    echo ""
}

################################################################################
# Main Installation Flow
################################################################################

main() {
    print_header

    # Pre-flight checks
    print_info "Running pre-flight checks..."
    check_root
    check_os

    # Check for --force flag for non-interactive updates
    FORCE_UPDATE=false
    if [[ "$1" == "--force" ]] || [[ "$1" == "-f" ]]; then
        FORCE_UPDATE=true
    fi

    # Check if this is an update or new installation
    if detect_existing_installation; then
        echo ""
        print_warning "ServerOS is already installed at ${INSTALL_DIR}"
        echo ""

        if [[ "$FORCE_UPDATE" == true ]]; then
            print_info "Force update mode - proceeding automatically..."
            perform_update
        else
            echo "Options:"
            echo "  1) Update existing installation (preserves config & database)"
            echo "  2) Cancel and exit"
            echo ""
            read -p "Enter your choice (yes to update, no to cancel): " -r
            echo ""

            if [[ $REPLY =~ ^[Yy][Ee][Ss]$|^[Yy]$|^1$ ]]; then
                perform_update
            else
                print_info "Update cancelled."
                echo ""
                print_info "To update without prompts, use: sudo ./install.sh --force"
                exit 0
            fi
        fi
    else
        # New installation
        echo ""
        print_info "Starting fresh installation..."
        echo ""

        update_system
        install_dependencies
        configure_mariadb
        create_database
        import_schema
        deploy_files
        configure_app
        create_admin_user
        configure_apache
        set_permissions
        configure_firewall
        restart_services

        # Show completion info
        show_completion_info
    fi
}

# Run main installation
main "$@"
