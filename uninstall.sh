#!/bin/bash

################################################################################
# ServerOS Uninstallation Script
################################################################################

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Configuration
INSTALL_DIR="/opt/serveros"
DB_NAME="servermanager"
DB_USER="serveros"
APACHE_CONF="/etc/apache2/sites-available/serveros.conf"

print_header() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                  ServerOS Uninstaller                          ║"
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

confirm_uninstall() {
    echo ""
    print_warning "This will completely remove ServerOS from your system!"
    print_warning "Including all data, database, and configurations."
    echo ""
    read -p "Are you sure you want to continue? (yes/no): " -r
    echo ""

    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        print_info "Uninstallation cancelled."
        exit 0
    fi
}

disable_apache_site() {
    print_info "Disabling Apache site..."

    if [[ -f ${APACHE_CONF} ]]; then
        a2dissite serveros > /dev/null 2>&1 || true
        print_success "Apache site disabled"
    fi
}

remove_apache_config() {
    print_info "Removing Apache configuration..."

    if [[ -f ${APACHE_CONF} ]]; then
        rm -f ${APACHE_CONF}
        print_success "Apache configuration removed"
    fi
}

remove_database() {
    print_info "Removing database and user..."

    # Drop database
    mysql -e "DROP DATABASE IF EXISTS ${DB_NAME};" 2>/dev/null || true

    # Drop user
    mysql -e "DROP USER IF EXISTS '${DB_USER}'@'localhost';" 2>/dev/null || true
    mysql -e "FLUSH PRIVILEGES;" 2>/dev/null || true

    print_success "Database and user removed"
}

remove_files() {
    print_info "Removing application files..."

    if [[ -d ${INSTALL_DIR} ]]; then
        # Backup credentials if they exist
        if [[ -f /root/serveros_credentials.txt ]]; then
            mv /root/serveros_credentials.txt /root/serveros_credentials.txt.bak
            print_info "Credentials backed up to: /root/serveros_credentials.txt.bak"
        fi

        rm -rf ${INSTALL_DIR}
        print_success "Application files removed"
    else
        print_warning "Installation directory not found"
    fi
}

restart_apache() {
    print_info "Restarting Apache..."
    systemctl restart apache2
    print_success "Apache restarted"
}

show_completion() {
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║              Uninstallation Completed!                         ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    print_success "ServerOS has been completely removed from your system."
    echo ""

    if [[ -f /root/serveros_credentials.txt.bak ]]; then
        print_info "Old credentials backed up to: /root/serveros_credentials.txt.bak"
    fi

    echo ""
    print_info "The following packages were NOT removed (remove manually if needed):"
    echo "  - Apache2"
    echo "  - MariaDB"
    echo "  - PHP"
    echo ""
    print_info "To remove them completely, run:"
    echo "  sudo apt remove --purge apache2 mariadb-server php php-*"
    echo "  sudo apt autoremove"
    echo ""
}

main() {
    print_header
    check_root
    confirm_uninstall

    echo ""
    print_info "Starting uninstallation process..."
    echo ""

    disable_apache_site
    remove_apache_config
    remove_database
    remove_files
    restart_apache

    show_completion
}

main "$@"
