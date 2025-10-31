#!/bin/bash

################################################################################
# TSO Uninstallation Script
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
KEEP_DATABASE=false

print_header() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║                    TSO Uninstaller                             ║"
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
    # Check for force flag
    if [[ "$1" == "--force" ]] || [[ "$1" == "-f" ]]; then
        print_warning "Force mode - skipping confirmation"
        return 0
    fi

    echo ""
    print_warning "This will remove TSO from your system!"
    echo ""
    echo "What would you like to do?"
    echo "  1) Remove everything (app files + database)"
    echo "  2) Remove app files only (keep database)"
    echo "  3) Cancel"
    echo ""
    read -p "Enter your choice (1-3): " -r
    echo ""

    case $REPLY in
        1)
            KEEP_DATABASE=false
            print_info "Will remove everything"
            ;;
        2)
            KEEP_DATABASE=true
            print_info "Will keep database"
            ;;
        3|*)
            print_info "Uninstallation cancelled."
            exit 0
            ;;
    esac

    echo ""
    read -p "Are you absolutely sure? Type 'yes' to confirm: " -r
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

backup_database() {
    print_info "Backing up database..."

    BACKUP_FILE="/root/serveros_db_backup_$(date +%Y%m%d_%H%M%S).sql"

    if mysqldump ${DB_NAME} > ${BACKUP_FILE} 2>/dev/null; then
        print_success "Database backed up to: ${BACKUP_FILE}"
    else
        print_warning "Could not backup database (it may not exist)"
    fi
}

remove_database() {
    if [[ "$KEEP_DATABASE" == true ]]; then
        print_info "Keeping database as requested..."
        backup_database
        print_success "Database preserved"
        return 0
    fi

    print_info "Removing database and user..."

    # Backup before removing
    backup_database

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

    if [[ "$KEEP_DATABASE" == true ]]; then
        print_success "TSO application files removed (database preserved)"
    else
        print_success "TSO has been completely removed from your system."
    fi

    echo ""

    # Show backup files
    BACKUP_FILES=$(ls -t /root/serveros_db_backup_*.sql 2>/dev/null | head -1)
    if [[ -n "$BACKUP_FILES" ]]; then
        print_info "Database backup: $BACKUP_FILES"
    fi

    if [[ -f /root/serveros_credentials.txt.bak ]]; then
        print_info "Credentials backup: /root/serveros_credentials.txt.bak"
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

    if [[ "$KEEP_DATABASE" == true ]]; then
        echo ""
        print_info "To reinstall TSO with existing database:"
        echo "  curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash"
        echo ""
    fi
}

show_usage() {
    echo "Usage: sudo ./uninstall.sh [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  --force, -f          Skip confirmation prompts (removes everything)"
    echo "  --keep-db            Remove app but keep database"
    echo "  --help, -h           Show this help message"
    echo ""
    echo "Examples:"
    echo "  sudo ./uninstall.sh              # Interactive mode"
    echo "  sudo ./uninstall.sh --force      # Remove everything without asking"
    echo "  sudo ./uninstall.sh --keep-db    # Remove app, keep database"
    echo ""
}

main() {
    # Parse arguments
    FORCE_MODE=false

    for arg in "$@"; do
        case $arg in
            --force|-f)
                FORCE_MODE=true
                ;;
            --keep-db)
                KEEP_DATABASE=true
                ;;
            --help|-h)
                show_usage
                exit 0
                ;;
        esac
    done

    print_header
    check_root

    if [[ "$FORCE_MODE" == true ]]; then
        confirm_uninstall "--force"
    else
        confirm_uninstall
    fi

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
