#!/bin/bash
#
# Network Shares Setup Script for TSO
# This script installs and configures Samba for the shares feature
#

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Functions
print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_error() {
    echo -e "${RED}✗ $1${NC}"
}

print_info() {
    echo -e "${YELLOW}ℹ $1${NC}"
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
            print_error "Aborting due to error."
            exit 1
        else
            return $exit_code
        fi
    fi
    
    return 0
}

check_root() {
    if [ "$EUID" -ne 0 ]; then 
        print_error "This script must be run as root"
        echo "Please run: sudo $0"
        exit 1
    fi
}

detect_os() {
    if [ -f /etc/os-release ]; then
        . /etc/os-release
        OS=$ID
        VERSION=$VERSION_ID
    else
        print_error "Cannot detect OS"
        exit 1
    fi
    print_info "Detected OS: $OS $VERSION"
}

install_samba() {
    print_info "Installing Samba..."
    
    case "$OS" in
        ubuntu|debian)
            run_command \
                "apt update 2>&1" \
                "Update package lists"
            
            run_command \
                "apt install -y samba samba-common-bin 2>&1" \
                "Install Samba packages"
            ;;
        centos|rhel|fedora)
            run_command \
                "yum install -y samba samba-client 2>&1" \
                "Install Samba packages"
            ;;
        *)
            print_error "Unsupported OS: $OS"
            exit 1
            ;;
    esac
    
    print_success "Samba installed"
}

check_samba() {
    if command -v smbd &> /dev/null; then
        print_success "Samba is already installed"
        return 0
    else
        print_info "Samba is not installed"
        return 1
    fi
}

setup_sudoers() {
    print_info "Setting up sudoers configuration..."
    
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
    PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
    SUDOERS_SOURCE="$PROJECT_ROOT/config/samba-sudoers"
    SUDOERS_DEST="/etc/sudoers.d/serveros-samba"
    
    if [ ! -f "$SUDOERS_SOURCE" ]; then
        print_error "Sudoers file not found: $SUDOERS_SOURCE"
        exit 1
    fi
    
    # Detect web server user
    WEB_USER="www-data"
    if id "apache" &>/dev/null; then
        WEB_USER="apache"
    elif id "nginx" &>/dev/null; then
        WEB_USER="nginx"
    fi
    
    print_info "Using web server user: $WEB_USER"
    
    # Copy and modify sudoers file
    sed "s/www-data/$WEB_USER/g" "$SUDOERS_SOURCE" > "$SUDOERS_DEST"
    chmod 440 "$SUDOERS_DEST"
    
    # Validate sudoers file
    if visudo -c -f "$SUDOERS_DEST" &>/dev/null; then
        print_success "Sudoers configuration installed"
    else
        print_error "Sudoers configuration is invalid"
        rm -f "$SUDOERS_DEST"
        exit 1
    fi
}

create_directories() {
    print_info "Creating share directories..."
    
    mkdir -p /srv/samba
    chmod 755 /srv/samba
    
    mkdir -p /opt/serveros/logs
    
    print_success "Directories created"
}

configure_samba() {
    print_info "Configuring Samba..."
    
    SAMBA_CONF="/etc/samba/smb.conf"
    SAMBA_BACKUP="/etc/samba/smb.conf.backup.$(date +%Y%m%d%H%M%S)"
    
    # Backup existing config
    if [ -f "$SAMBA_CONF" ]; then
        cp "$SAMBA_CONF" "$SAMBA_BACKUP"
        print_info "Backed up existing config to $SAMBA_BACKUP"
    fi
    
    # Create basic configuration
    cat > "$SAMBA_CONF" << 'EOF'
[global]
   workgroup = WORKGROUP
   server string = Samba Server
   netbios name = SERVER
   security = user
   map to guest = bad user
   dns proxy = no
   
   # Logging
   log file = /var/log/samba/log.%m
   max log size = 1000
   
   # Performance
   socket options = TCP_NODELAY IPTOS_LOWDELAY SO_RCVBUF=131072 SO_SNDBUF=131072
   read raw = yes
   write raw = yes
   
   # Security
   server signing = auto
   client signing = auto

# Shares will be added here by the web interface
EOF
    
    chmod 644 "$SAMBA_CONF"
    print_success "Samba configured"
}

enable_samba() {
    print_info "Enabling and starting Samba service..."
    
    systemctl enable smbd 2>/dev/null || systemctl enable smb 2>/dev/null || true
    systemctl start smbd 2>/dev/null || systemctl start smb 2>/dev/null || true
    
    # Check if started
    sleep 2
    if systemctl is-active --quiet smbd 2>/dev/null || systemctl is-active --quiet smb 2>/dev/null; then
        print_success "Samba service is running"
    else
        print_error "Failed to start Samba service"
        exit 1
    fi
}

configure_firewall() {
    print_info "Configuring firewall..."
    
    # UFW (Ubuntu/Debian)
    if command -v ufw &> /dev/null; then
        ufw allow Samba 2>/dev/null || true
        print_success "UFW rules added"
    fi
    
    # Firewalld (CentOS/RHEL)
    if command -v firewall-cmd &> /dev/null; then
        firewall-cmd --permanent --add-service=samba 2>/dev/null || true
        firewall-cmd --reload 2>/dev/null || true
        print_success "Firewalld rules added"
    fi
    
    print_info "Note: You may need to manually configure your firewall"
    print_info "Required ports: TCP 445, TCP 139, UDP 137-138"
}

update_database() {
    print_info "Updating database..."
    
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
    PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"
    SQL_FILE="$PROJECT_ROOT/init.sql"
    
    if [ ! -f "$SQL_FILE" ]; then
        print_error "SQL file not found: $SQL_FILE"
        exit 1
    fi
    
    # Read database config
    CONFIG_FILE="$PROJECT_ROOT/config/config.php"
    if [ ! -f "$CONFIG_FILE" ]; then
        print_error "Config file not found: $CONFIG_FILE"
        exit 1
    fi
    
    # Prompt for database credentials
    read -p "Enter MySQL/MariaDB root password: " -s DB_PASSWORD
    echo
    
    mysql -u root -p"$DB_PASSWORD" servermanager < "$SQL_FILE" 2>/dev/null
    
    if [ $? -eq 0 ]; then
        print_success "Database updated"
    else
        print_error "Database update failed"
        print_info "You may need to manually run: mysql -u root -p servermanager < init.sql"
    fi
}

print_summary() {
    echo
    echo "================================================"
    echo -e "${GREEN}Network Shares Setup Complete!${NC}"
    echo "================================================"
    echo
    echo "Next steps:"
    echo "1. Access the web interface at: http://your-server/shares.php"
    echo "2. Create share users in the Users tab"
    echo "3. Create shares in the Shares tab"
    echo "4. Set permissions for users on shares"
    echo
    echo "Connecting to shares:"
    echo "  Windows: \\\\your-server\\share-name"
    echo "  macOS:   smb://your-server/share-name"
    echo "  Linux:   smb://your-server/share-name"
    echo
    echo "For detailed documentation, see: SHARES-SETUP.md"
    echo "================================================"
}

# Main execution
main() {
    echo "================================================"
    echo "TSO Network Shares Setup"
    echo "================================================"
    echo
    
    check_root
    detect_os
    
    echo
    read -p "This will install and configure Samba. Continue? (y/n) " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Setup cancelled"
        exit 0
    fi
    
    echo
    print_info "Starting installation..."
    echo
    
    # Check if Samba is already installed
    if ! check_samba; then
        install_samba
    fi
    
    setup_sudoers
    create_directories
    configure_samba
    enable_samba
    configure_firewall
    
    echo
    read -p "Do you want to update the database now? (y/n) " -n 1 -r
    echo
    if [[ $REPLY =~ ^[Yy]$ ]]; then
        update_database
    else
        print_info "Skipping database update"
        print_info "Remember to run: mysql -u root -p servermanager < init.sql"
    fi
    
    print_summary
}

main "$@"

