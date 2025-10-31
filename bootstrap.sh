#!/bin/bash

################################################################################
# ServerOS Bootstrap Installer
# Quick one-liner installation script
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
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
TEMP_DIR="/tmp/serveros-install-$$"
INSTALL_DIR="/opt/serveros"

################################################################################
# Helper Functions
################################################################################

print_header() {
    echo -e "${BLUE}"
    echo "╔════════════════════════════════════════════════════════════════╗"
    echo "║              ServerOS Bootstrap Installer                      ║"
    echo "║          Installing from: github.com/chukfinley/tso           ║"
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
        echo ""
        echo "Try: curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash"
        exit 1
    fi
}

check_os() {
    if [[ ! -f /etc/debian_version ]]; then
        print_error "This script is designed for Debian-based systems only"
        print_info "Supported: Debian 10+, Ubuntu 20.04+"
        exit 1
    fi
    print_success "Debian-based system detected"
}

check_dependencies() {
    print_info "Checking for required tools..."

    # Check for git
    if ! command -v git &> /dev/null; then
        print_warning "Git not found, installing..."
        apt-get update -qq
        apt-get install -y -qq git
    fi

    # Check for curl
    if ! command -v curl &> /dev/null; then
        print_warning "Curl not found, installing..."
        apt-get update -qq
        apt-get install -y -qq curl
    fi

    print_success "Required tools available"
}

clone_repository() {
    print_info "Cloning ServerOS repository..."

    # Clean up old temp directory if exists
    rm -rf "${TEMP_DIR}"

    # Clone repository
    if git clone --depth 1 "${REPO_URL}" "${TEMP_DIR}" > /dev/null 2>&1; then
        print_success "Repository cloned successfully"
    else
        print_error "Failed to clone repository from ${REPO_URL}"
        exit 1
    fi
}

run_installer() {
    print_info "Running ServerOS installer..."
    echo ""

    cd "${TEMP_DIR}"

    # Make installer executable
    chmod +x install.sh

    # Run the actual installer
    ./install.sh
}

cleanup() {
    print_info "Cleaning up temporary files..."
    rm -rf "${TEMP_DIR}"
    print_success "Cleanup complete"
}

show_success() {
    echo ""
    echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║            Bootstrap Installation Complete!                    ║${NC}"
    echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    print_success "ServerOS has been installed successfully!"
    echo ""
    print_info "Check the output above for access credentials"
    print_info "Credentials also saved to: /root/serveros_credentials.txt"
    echo ""
}

handle_error() {
    print_error "Installation failed!"
    print_info "Cleaning up..."
    rm -rf "${TEMP_DIR}" 2>/dev/null || true
    exit 1
}

################################################################################
# Main Execution
################################################################################

# Set error handler
trap handle_error ERR

main() {
    print_header

    echo ""
    print_info "Starting bootstrap installation..."
    echo ""

    # Pre-flight checks
    check_root
    check_os
    check_dependencies

    # Clone and install
    clone_repository
    run_installer

    # Cleanup
    cleanup

    # Show success
    show_success
}

# Run main
main "$@"
