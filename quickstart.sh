#!/bin/bash

################################################################################
# ServerOS Quick Start Script (Development Mode)
# For testing on localhost without full installation
################################################################################

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_info() {
    echo -e "${BLUE}ℹ $1${NC}"
}

print_success() {
    echo -e "${GREEN}✓ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠ $1${NC}"
}

echo -e "${BLUE}"
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║         ServerOS Quick Start (Development Mode)               ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check if PHP is installed
if ! command -v php &> /dev/null; then
    print_warning "PHP is not installed!"
    echo "Install it with: sudo apt install php php-mysql php-cli"
    exit 1
fi

print_success "PHP detected: $(php -v | head -n 1)"

# Check if MySQL/MariaDB is installed
if ! command -v mysql &> /dev/null; then
    print_warning "MySQL/MariaDB is not installed!"
    echo "Install it with: sudo apt install mariadb-server"
    exit 1
fi

print_success "Database server detected"

# Setup database
print_info "Setting up database..."

# Check if database already exists
if mysql -e "USE servermanager" 2>/dev/null; then
    print_warning "Database 'servermanager' already exists. Skipping creation."
else
    print_info "Creating database and importing schema..."
    mysql -e "CREATE DATABASE servermanager;"
    mysql servermanager < init.sql
    print_success "Database created and schema imported"
fi

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

# Update config for localhost
print_info "Configuring for localhost..."
sed -i.bak "s/define('BASE_URL', 'http:\/\/localhost');/define('BASE_URL', 'http:\/\/localhost:8000');/" "${SCRIPT_DIR}/config/config.php"

# Get local IP
LOCAL_IP=$(hostname -I | awk '{print $1}')

echo ""
print_success "Setup complete!"
echo ""
echo -e "${BLUE}Starting PHP development server...${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo -e "  Local:   ${GREEN}http://localhost:8000${NC}"
echo -e "  Network: ${GREEN}http://${LOCAL_IP}:8000${NC}"
echo ""
echo -e "  Username: ${GREEN}admin${NC}"
echo -e "  Password: ${GREEN}admin123${NC}"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
print_info "Press Ctrl+C to stop the server"
echo ""

# Start PHP built-in server
cd "${SCRIPT_DIR}/public"
php -S 0.0.0.0:8000
