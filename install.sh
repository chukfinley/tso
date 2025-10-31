#!/bin/bash

################################################################################
# TSO Installation Script
# A Debian-based Server Management System
################################################################################

# Ensure script is running with bash (not sh)
# Only re-exec if running from a file (not piped from stdin)
if [ -z "$BASH_VERSION" ] && [ -f "$0" ]; then
    exec bash "$0" "$@"
fi

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
    echo "║                      TSO Installer                             ║"
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

# Error checking function
run_command() {
    local cmd="$1"
    local description="$2"
    local exit_on_error="${3:-true}"  # Default to exit on error
    
    # Run command and capture output
    local output
    local exit_code
    
    if output=$(eval "$cmd" 2>&1); then
        exit_code=0
    else
        exit_code=$?
    fi
    
    if [[ $exit_code -ne 0 ]]; then
        print_error "Failed: $description"
        echo ""
        echo -e "${RED}Command:${NC} $cmd"
        echo -e "${RED}Exit Code:${NC} $exit_code"
        echo -e "${RED}Output:${NC}"
        echo "$output" | sed 's/^/  /'
        echo ""
        
        if [[ "$exit_on_error" == "true" ]]; then
            print_error "Aborting installation due to error."
            exit 1
        else
            return $exit_code
        fi
    fi
    
    return 0
}

# Verify service is running
verify_service() {
    local service="$1"
    local description="$2"
    
    # Wait a moment for service to start
    sleep 2
    
    if systemctl is-active --quiet "$service"; then
        print_success "$description is running"
        return 0
    else
        print_error "$description failed to start"
        echo ""
        echo -e "${RED}Service Status:${NC}"
        systemctl status "$service" --no-pager -l || true
        echo ""
        print_error "Please check the service logs and configuration."
        exit 1
    fi
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
    run_command \
        "apt-get update -qq 2>&1" \
        "Update package lists"
    
    run_command \
        "apt-get upgrade -y -qq 2>&1" \
        "Upgrade system packages"
    
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

    print_info "Installing QEMU/KVM and virtualization tools..."

    # Install QEMU/KVM packages
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
        qemu-kvm \
        qemu-system-x86 \
        qemu-utils \
        libvirt-daemon-system \
        libvirt-clients \
        bridge-utils \
        virt-manager \
        gzip \
        > /dev/null 2>&1

    # Enable and start libvirtd
    systemctl enable libvirtd > /dev/null 2>&1
    systemctl start libvirtd > /dev/null 2>&1

    # Add www-data to necessary groups for VM management
    usermod -aG kvm www-data > /dev/null 2>&1 || true
    usermod -aG libvirt www-data > /dev/null 2>&1 || true

    print_success "QEMU/KVM and virtualization tools installed"

    print_info "Installing Samba for network shares..."

    # Install Samba packages
    DEBIAN_FRONTEND=noninteractive apt-get install -y -qq \
        samba \
        samba-common-bin \
        > /dev/null 2>&1

    # Enable and start Samba services
    systemctl enable smbd > /dev/null 2>&1
    systemctl enable nmbd > /dev/null 2>&1
    systemctl start smbd > /dev/null 2>&1
    systemctl start nmbd > /dev/null 2>&1

    print_success "Samba installed and started"
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
    run_command \
        "mysql -e 'CREATE DATABASE IF NOT EXISTS ${DB_NAME};' 2>&1" \
        "Create database ${DB_NAME}"

    # Create user with password
    run_command \
        "mysql -e \"CREATE USER IF NOT EXISTS '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}';\" 2>&1" \
        "Create database user ${DB_USER}"
    
    run_command \
        "mysql -e \"GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'localhost';\" 2>&1" \
        "Grant privileges to database user"
    
    run_command \
        "mysql -e 'FLUSH PRIVILEGES;' 2>&1" \
        "Flush MySQL privileges"

    print_success "Database '${DB_NAME}' created"
}

import_schema() {
    print_info "Importing database schema..."

    # Get script directory
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

    if [[ -f "${SCRIPT_DIR}/init.sql" ]]; then
        run_command \
            "mysql ${DB_NAME} < '${SCRIPT_DIR}/init.sql' 2>&1" \
            "Import database schema"
        print_success "Database schema imported"
    else
        print_error "init.sql not found!"
        print_error "Expected location: ${SCRIPT_DIR}/init.sql"
        exit 1
    fi
}

run_database_migrations() {
    print_info "Running database migrations..."

    # Run migration script if it exists
    if [[ -f "${INSTALL_DIR}/tools/migrate-database.php" ]]; then
        php ${INSTALL_DIR}/tools/migrate-database.php
        print_success "Database migrations completed"
    else
        print_warning "Migration script not found, skipping..."
    fi
}

create_default_shares() {
    print_info "Creating default network shares..."

    # Create default ISO share for VM images
    if [[ -f "${INSTALL_DIR}/tools/create-iso-share.php" ]]; then
        php ${INSTALL_DIR}/tools/create-iso-share.php
        print_success "Default ISO share created"
    else
        print_warning "ISO share creation script not found, skipping..."
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

    # Update base URL to HTTPS
    IP_ADDRESS=$(hostname -I | awk '{print $1}')
    HOSTNAME=$(hostname -f 2>/dev/null || hostname)
    # Use IP address for BASE_URL if available, otherwise use hostname
    BASE_HOST="${IP_ADDRESS:-${HOSTNAME}}"
    sed -i "s#define('BASE_URL', 'http://localhost');#define('BASE_URL', 'https://${BASE_HOST}');#" ${INSTALL_DIR}/config/config.php

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
    print_info "Configuring Apache2 with SSL/HTTPS..."

    # Get server IP address
    IP_ADDRESS=$(hostname -I | awk '{print $1}')
    HOSTNAME=$(hostname -f 2>/dev/null || hostname)

    # SSL certificate directory
    SSL_DIR="/etc/ssl/serveros"
    mkdir -p ${SSL_DIR}

    # Generate self-signed SSL certificate that works with IP addresses
    print_info "Generating self-signed SSL certificate..."
    
    # Create certificate configuration for IP address
    cat > /tmp/serveros-ssl.conf << EOF
[req]
distinguished_name = req_distinguished_name
req_extensions = v3_req
prompt = no

[req_distinguished_name]
C = US
ST = State
L = City
O = ServerOS
CN = ${IP_ADDRESS}

[v3_req]
keyUsage = keyEncipherment, dataEncipherment
extendedKeyUsage = serverAuth
subjectAltName = @alt_names

[alt_names]
IP.1 = ${IP_ADDRESS}
DNS.1 = ${HOSTNAME}
DNS.2 = localhost
EOF

    # Generate private key and certificate
    run_command \
        "openssl req -x509 -nodes -days 3650 -newkey rsa:2048 -keyout ${SSL_DIR}/serveros.key -out ${SSL_DIR}/serveros.crt -config /tmp/serveros-ssl.conf -extensions v3_req 2>&1" \
        "Generate SSL certificate"

    # Set proper permissions
    run_command \
        "chmod 600 ${SSL_DIR}/serveros.key" \
        "Set SSL key permissions"
    
    run_command \
        "chmod 644 ${SSL_DIR}/serveros.crt" \
        "Set SSL certificate permissions"
    
    rm -f /tmp/serveros-ssl.conf

    print_success "SSL certificate generated"

    # Create Apache configuration with HTTP redirect and HTTPS
    cat > ${APACHE_CONF} << EOF
# HTTP VirtualHost - Redirect to HTTPS
<VirtualHost *:80>
    ServerName ${HOSTNAME}
    ServerAdmin admin@localhost

    # Redirect all HTTP traffic to HTTPS
    RewriteEngine On
    RewriteCond %{HTTPS} off
    RewriteRule ^(.*)$ https://%{HTTP_HOST}%{REQUEST_URI} [R=301,L]

    ErrorLog \${APACHE_LOG_DIR}/serveros_error.log
    CustomLog \${APACHE_LOG_DIR}/serveros_access.log combined
</VirtualHost>

# HTTPS VirtualHost - Main site
<VirtualHost *:443>
    ServerName ${HOSTNAME}
    ServerAdmin admin@localhost
    DocumentRoot ${WEB_ROOT}

    # SSL Configuration
    SSLEngine on
    SSLCertificateFile ${SSL_DIR}/serveros.crt
    SSLCertificateKeyFile ${SSL_DIR}/serveros.key

    # SSL Security Settings
    SSLProtocol all -SSLv2 -SSLv3
    SSLCipherSuite HIGH:!aNULL:!MD5
    SSLHonorCipherOrder on

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
    Header always set Strict-Transport-Security "max-age=31536000; includeSubDomains"
</VirtualHost>
EOF

    # Enable required Apache modules
    run_command \
        "a2enmod rewrite 2>&1" \
        "Enable Apache rewrite module"
    
    run_command \
        "a2enmod headers 2>&1" \
        "Enable Apache headers module"
    
    run_command \
        "a2enmod ssl 2>&1" \
        "Enable Apache SSL module"

    PHP_VERSION=$(php -r 'echo PHP_MAJOR_VERSION.".".PHP_MINOR_VERSION;' 2>/dev/null || echo "")
    if [[ -n "$PHP_VERSION" ]]; then
        run_command \
            "a2enmod php${PHP_VERSION} 2>&1" \
            "Enable Apache PHP module" \
            "false"  # Don't exit on error for PHP module
    fi

    # Disable default site
    a2dissite 000-default > /dev/null 2>&1 || true

    # Enable TSO site
    run_command \
        "a2ensite serveros 2>&1" \
        "Enable TSO Apache site"

    # Test Apache configuration
    run_command \
        "apache2ctl configtest 2>&1" \
        "Test Apache configuration"

    print_success "Apache configured with SSL/HTTPS"
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

configure_sudo() {
    print_info "Configuring sudo permissions..."

    # Copy main sudoers file
    if [ -f "${INSTALL_DIR}/config/serveros-sudoers" ]; then
        cp "${INSTALL_DIR}/config/serveros-sudoers" /etc/sudoers.d/serveros
        chmod 0440 /etc/sudoers.d/serveros
    else
        print_warning "Main sudoers config not found"
    fi

    # Copy Samba sudoers file
    if [ -f "${INSTALL_DIR}/config/samba-sudoers" ]; then
        cp "${INSTALL_DIR}/config/samba-sudoers" /etc/sudoers.d/serveros-samba
        chmod 0440 /etc/sudoers.d/serveros-samba
    else
        print_warning "Samba sudoers config not found"
    fi

    print_success "Sudo permissions configured"
}

restart_services() {
    print_info "Restarting services..."

    # Restart Apache
    run_command \
        "systemctl restart apache2 2>&1" \
        "Restart Apache service"
    
    # Enable Apache to start on boot
    run_command \
        "systemctl enable apache2 2>&1" \
        "Enable Apache on boot"

    # Verify Apache is running
    verify_service "apache2" "Apache2"
    
    # Verify SSL is working by checking if SSL module is loaded
    if apache2ctl -M 2>/dev/null | grep -q ssl_module; then
        print_success "Apache SSL module is loaded"
    else
        print_warning "Apache SSL module may not be loaded - HTTPS may not work"
        echo "Checking Apache configuration..."
        apache2ctl -M 2>&1 | grep -i ssl || true
    fi
    
    print_success "Services restarted and verified"
}

configure_firewall() {
    print_info "Configuring firewall..."

    # Check if UFW is installed
    if command -v ufw &> /dev/null; then
        ufw allow 80/tcp > /dev/null 2>&1 || true
        ufw allow 443/tcp > /dev/null 2>&1 || true
        print_success "Firewall configured (HTTP redirects to HTTPS)"
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
    print_success "TSO has been successfully installed!"
    echo ""
    echo -e "${BLUE}Access Information:${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo -e "  URL:      ${GREEN}https://${IP_ADDRESS}${NC}"
    echo -e "  Hostname: ${GREEN}https://${HOSTNAME}${NC}"
    echo ""
    echo -e "${YELLOW}  Note: Using self-signed certificate (browser warnings are expected)${NC}"
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
    echo "  2. Access is via HTTPS only (HTTP automatically redirects)"
    echo "  3. Self-signed certificate is in use (browser warnings are normal)"
    echo "  4. Database credentials saved in: ${INSTALL_DIR}/config/config.php"
    echo "  5. Keep database password secure: ${DB_PASS}"
    echo ""

    # Save credentials to file
    cat > /root/serveros_credentials.txt << EOF
TSO Installation Credentials
==================================
Generated: $(date)

Web Access:
-----------
URL: https://${IP_ADDRESS}
Hostname: https://${HOSTNAME}
Note: Using self-signed certificate (browser warnings are expected)

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

    # Deploy new files (will overwrite application files but not config)
    print_info "Updating application files..."
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
    
    # Copy all application files from source to installation directory
    cp -r "${SCRIPT_DIR}/public" ${INSTALL_DIR}/ 2>/dev/null || true
    cp -r "${SCRIPT_DIR}/src" ${INSTALL_DIR}/ 2>/dev/null || true
    cp -r "${SCRIPT_DIR}/views" ${INSTALL_DIR}/ 2>/dev/null || true
    cp -r "${SCRIPT_DIR}/tools" ${INSTALL_DIR}/ 2>/dev/null || true
    cp -r "${SCRIPT_DIR}/scripts" ${INSTALL_DIR}/ 2>/dev/null || true
    cp "${SCRIPT_DIR}/init.sql" ${INSTALL_DIR}/ 2>/dev/null || true
    
    print_success "Application files updated"

    # Restore config
    print_info "Restoring configuration..."
    cp /tmp/serveros-config-backup.php ${INSTALL_DIR}/config/config.php
    rm -f /tmp/serveros-config-backup.php
    print_success "Configuration restored"
    
    # Install/Update monitoring system
    if [ -f "${INSTALL_DIR}/scripts/install-monitoring.sh" ]; then
        echo ""
        print_info "Updating monitoring system..."
        bash "${INSTALL_DIR}/scripts/install-monitoring.sh"
    fi

    # Ensure admin user exists with correct password
    create_admin_user

    # Run database migrations
    run_database_migrations

    # Create default shares
    create_default_shares

    # Update permissions
    set_permissions

    # Configure sudo permissions
    configure_sudo

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
        print_warning "TSO is already installed at ${INSTALL_DIR}"
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
        run_database_migrations
        create_default_shares
        configure_apache
        set_permissions
        configure_sudo
        configure_firewall
        restart_services
        
        # Install monitoring system
        if [ -f "$SCRIPT_DIR/scripts/install-monitoring.sh" ]; then
            echo ""
            echo "Installing monitoring system..."
            bash "$SCRIPT_DIR/scripts/install-monitoring.sh"
        fi

        # Show completion info
        show_completion_info
    fi
}

# Run main installation
main "$@"
