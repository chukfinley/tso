#!/bin/bash

################################################################################
# TSO Installation Script - Go & TypeScript Version
# Einfaches Installation-Skript - Alles automatisch
################################################################################

# Ensure script is running with bash
if [ -z "$BASH_VERSION" ] && [ -f "$0" ]; then
    exec bash "$0" "$@"
fi

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Verbose mode - check for --verbose or -v flag
VERBOSE=false
if [[ "$*" == *"--verbose"* ]] || [[ "$*" == *"-v"* ]]; then
    VERBOSE=true
    echo -e "${YELLOW}🔍 Verbose-Modus aktiviert - Zeige alle Outputs${NC}"
    echo ""
fi

# Function to run commands with or without output redirection
run_cmd() {
    if [ "$VERBOSE" = true ]; then
        "$@"
    else
        "$@" > /dev/null 2>&1
    fi
}

run_cmd_show() {
    "$@"
}

INSTALL_DIR="/opt/serveros"
DB_NAME="servermanager"
DB_USER="tso"
DB_PASS=$(openssl rand -base64 12 | tr -d "=+/" | cut -c1-12)
SESSION_SECRET=$(openssl rand -hex 32)
BACKEND_PORT=8080

echo -e "${BLUE}"
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║              TSO Installer (Go/TypeScript)                    ║"
echo "║          Automatische Installation - Alles wird eingerichtet ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

# Check if root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}❌ Bitte mit sudo ausführen!${NC}"
    exit 1
fi

# Install dependencies
echo -e "${YELLOW}📦 Installiere Abhängigkeiten...${NC}"
echo -e "${YELLOW}  → Aktualisiere Paket-Listen...${NC}"
if [ "$VERBOSE" = true ]; then
    if ! apt-get update; then
        echo -e "${RED}❌ Fehler beim Aktualisieren der Paket-Listen!${NC}"
        exit 1
    fi
    echo -e "${YELLOW}  → Installiere curl, wget, git, build-essential...${NC}"
    if ! apt-get install -y curl wget git build-essential; then
        echo -e "${RED}❌ Fehler beim Installieren der Basis-Pakete!${NC}"
        exit 1
    fi
else
    if ! apt-get update > /dev/null 2>&1; then
        echo -e "${RED}❌ Fehler beim Aktualisieren der Paket-Listen!${NC}"
        exit 1
    fi
    if ! apt-get install -y curl wget git build-essential > /dev/null 2>&1; then
        echo -e "${RED}❌ Fehler beim Installieren der Basis-Pakete!${NC}"
        exit 1
    fi
fi

# Install Go if not present
if ! command -v go &> /dev/null; then
    echo -e "${YELLOW}  → Installiere Go (neueste Version)...${NC}"
    if [ "$VERBOSE" = true ]; then
        if ! bash <(curl -sL https://git.io/go-installer); then
            echo -e "${RED}❌ Fehler beim Installieren von Go!${NC}"
            exit 1
        fi
    else
        if ! bash <(curl -sL https://git.io/go-installer) > /dev/null 2>&1; then
            echo -e "${RED}❌ Fehler beim Installieren von Go!${NC}"
            exit 1
        fi
    fi
    export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> /etc/profile
    # Verify Go installation by checking for actual binary
    if [ -f "/usr/local/go/bin/go" ]; then
        GO_VERIFY_BIN="/usr/local/go/bin/go"
    elif [ -f "$HOME/go/bin/go" ]; then
        GO_VERIFY_BIN="$HOME/go/bin/go"
    else
        echo -e "${RED}❌ Go Installation fehlgeschlagen! Binärdatei nicht gefunden.${NC}"
        exit 1
    fi
    if [ "$VERBOSE" = true ]; then
        GO_VERSION=$("$GO_VERIFY_BIN" version 2>&1 || echo "unknown")
        echo -e "${GREEN}  ✓ Go installiert: $GO_VERSION${NC}"
    fi
else
    GO_VERSION=$(go version 2>&1 || echo "unknown")
    if [ "$VERBOSE" = true ]; then
        echo -e "${GREEN}  ✓ Go bereits installiert: $GO_VERSION${NC}"
    fi
    # Ensure PATH is set
    export PATH=$PATH:/usr/local/go/bin
    export PATH=$PATH:$HOME/go/bin
fi

# Install Node.js if not present
if ! command -v node &> /dev/null; then
    echo -e "${YELLOW}  → Installiere Node.js...${NC}"
    if [ "$VERBOSE" = true ]; then
        echo -e "${YELLOW}    Füge NodeSource Repository hinzu (Node.js 20.x)...${NC}"
        # Use Node.js 20.x (LTS) instead of 18.x
        if ! curl -fsSL https://deb.nodesource.com/setup_20.x | bash -; then
            echo -e "${RED}❌ Fehler beim Hinzufügen des NodeSource Repositories!${NC}"
            exit 1
        fi
        echo -e "${YELLOW}    Installiere Node.js...${NC}"
        if ! apt-get install -y nodejs; then
            echo -e "${RED}❌ Fehler beim Installieren von Node.js!${NC}"
            exit 1
        fi
    else
        # Use Node.js 20.x (LTS) instead of 18.x
        # Suppress deprecation warnings for non-interactive install
        DEBIAN_FRONTEND=noninteractive curl -fsSL https://deb.nodesource.com/setup_20.x | bash - > /dev/null 2>&1
        if [ ${PIPESTATUS[0]} -ne 0 ]; then
            echo -e "${RED}❌ Fehler beim Hinzufügen des NodeSource Repositories!${NC}"
            exit 1
        fi
        if ! apt-get install -y nodejs > /dev/null 2>&1; then
            echo -e "${RED}❌ Fehler beim Installieren von Node.js!${NC}"
            exit 1
        fi
    fi
    if [ "$VERBOSE" = true ]; then
        NODE_VERSION=$(node --version 2>&1 || echo "unknown")
        echo -e "${GREEN}  ✓ Node.js installiert: $NODE_VERSION${NC}"
    fi
else
    NODE_VERSION=$(node --version 2>&1 || echo "unknown")
    if [ "$VERBOSE" = true ]; then
        echo -e "${GREEN}  ✓ Node.js bereits installiert: $NODE_VERSION${NC}"
    fi
fi

# Install MariaDB if not present
if ! command -v mysql &> /dev/null; then
    echo -e "${YELLOW}  → Installiere MariaDB...${NC}"
    if [ "$VERBOSE" = true ]; then
        if ! apt-get install -y mariadb-server; then
            echo -e "${RED}❌ Fehler beim Installieren von MariaDB!${NC}"
            exit 1
        fi
        echo -e "${YELLOW}    Starte MariaDB Service...${NC}"
        if ! systemctl start mariadb; then
            echo -e "${YELLOW}⚠️  MariaDB konnte nicht gestartet werden, versuche es manuell${NC}"
        fi
        systemctl enable mariadb || true
    else
        if ! apt-get install -y mariadb-server > /dev/null 2>&1; then
            echo -e "${RED}❌ Fehler beim Installieren von MariaDB!${NC}"
            exit 1
        fi
        systemctl start mariadb || true
        systemctl enable mariadb > /dev/null 2>&1 || true
    fi
    if [ "$VERBOSE" = true ]; then
        echo -e "${GREEN}  ✓ MariaDB installiert${NC}"
    fi
else
    if [ "$VERBOSE" = true ]; then
        echo -e "${GREEN}  ✓ MariaDB bereits installiert${NC}"
    fi
fi

echo -e "${GREEN}✓ Abhängigkeiten installiert${NC}"

# Setup database
echo -e "${YELLOW}🗄️  Richte Datenbank ein...${NC}"
if [ "$VERBOSE" = true ]; then
    echo -e "${YELLOW}  → Erstelle Datenbank und Benutzer...${NC}"
fi

# Try to connect to MariaDB (might need password or might not)
DB_SETUP_OUTPUT=$(mysql -u root <<EOF 2>&1
CREATE DATABASE IF NOT EXISTS $DB_NAME CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
DROP USER IF EXISTS '$DB_USER'@'localhost';
CREATE USER '$DB_USER'@'localhost' IDENTIFIED BY '$DB_PASS';
GRANT ALL PRIVILEGES ON $DB_NAME.* TO '$DB_USER'@'localhost';
FLUSH PRIVILEGES;
EOF
)
DB_SETUP_STATUS=$?

if [ "$VERBOSE" = true ]; then
    echo "$DB_SETUP_OUTPUT"
fi

if [ $DB_SETUP_STATUS -ne 0 ]; then
    echo -e "${RED}❌ Fehler beim Einrichten der Datenbank!${NC}"
    if [ "$VERBOSE" = true ]; then
        echo -e "${RED}Fehler-Details:${NC}"
        echo "$DB_SETUP_OUTPUT"
    fi
    echo -e "${YELLOW}   Prüfe ob MariaDB läuft: systemctl status mariadb${NC}"
    echo -e "${YELLOW}   Möglicherweise benötigt MariaDB ein Root-Passwort${NC}"
    echo -e "${YELLOW}   Versuche: sudo mysql_secure_installation${NC}"
    exit 1
fi

# Import schema
if [ -f "init.sql" ]; then
    if [ "$VERBOSE" = true ]; then
        echo -e "${YELLOW}  → Importiere Datenbank-Schema...${NC}"
    fi
    
    SCHEMA_OUTPUT=$(mysql -u $DB_USER -p$DB_PASS $DB_NAME < init.sql 2>&1)
    SCHEMA_STATUS=$?
    
    if [ "$VERBOSE" = true ]; then
        if [ -n "$SCHEMA_OUTPUT" ]; then
            echo "$SCHEMA_OUTPUT"
        fi
    fi
    
    if [ $SCHEMA_STATUS -ne 0 ]; then
        echo -e "${RED}❌ Fehler beim Importieren des Datenbank-Schemas!${NC}"
        if [ "$VERBOSE" = true ]; then
            echo -e "${RED}Fehler-Details:${NC}"
            echo "$SCHEMA_OUTPUT"
        fi
        exit 1
    fi
    
    echo -e "${GREEN}✓ Datenbank-Schema importiert${NC}"
else
    echo -e "${RED}❌ init.sql nicht gefunden!${NC}"
    if [ "$VERBOSE" = true ]; then
        echo -e "${YELLOW}   Aktuelles Verzeichnis: $(pwd)${NC}"
        echo -e "${YELLOW}   Verfügbare Dateien:${NC}"
        ls -la *.sql 2>/dev/null || echo "   Keine .sql Dateien gefunden"
    fi
    exit 1
fi

# Create admin user
if [ "$VERBOSE" = true ]; then
    echo -e "${YELLOW}  → Erstelle Admin-Benutzer...${NC}"
fi

# Try to generate bcrypt hash using Go if available
ADMIN_PASSWORD="admin123"
ADMIN_HASH=""
if command -v go &> /dev/null || [ -f "$GO_BIN" ]; then
    # Generate hash using Go
    TEMP_HASH_FILE=$(mktemp)
    cat > "$TEMP_HASH_FILE" <<'GOEOF'
package main
import (
    "fmt"
    "golang.org/x/crypto/bcrypt"
)
func main() {
    hash, _ := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
    fmt.Print(string(hash))
}
GOEOF
    if [ -f "$GO_BIN" ]; then
        cd "$(dirname "$TEMP_HASH_FILE")"
        if "$GO_BIN" run "$TEMP_HASH_FILE" 2>/dev/null | grep -q '^\$2a\$'; then
            ADMIN_HASH=$("$GO_BIN" run "$TEMP_HASH_FILE" 2>/dev/null)
        fi
        rm -f "$TEMP_HASH_FILE"
    fi
fi

# Fallback to a pre-generated Go-compatible bcrypt hash if Go generation failed
if [ -z "$ADMIN_HASH" ] || [ ${#ADMIN_HASH} -lt 50 ]; then
    # Pre-generated bcrypt hash for "admin123" (Go-compatible $2a$ format)
    ADMIN_HASH='$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'
fi

# Use REPLACE instead of INSERT IGNORE to update password hash if user exists
mysql -u $DB_USER -p$DB_PASS $DB_NAME <<EOF
REPLACE INTO users (username, email, password, full_name, role, is_active) 
VALUES ('admin', 'admin@localhost', '$ADMIN_HASH', 'Administrator', 'admin', 1);
EOF

if [ "$VERBOSE" = true ]; then
    echo -e "${GREEN}  ✓ Admin-Benutzer erstellt${NC}"
fi

# Create directories
echo -e "${YELLOW}📁 Erstelle Verzeichnisse...${NC}"
mkdir -p "$INSTALL_DIR/go-backend"
mkdir -p "$INSTALL_DIR/frontend"
mkdir -p "$INSTALL_DIR/logs"
mkdir -p "$INSTALL_DIR/storage/isos"
mkdir -p "$INSTALL_DIR/vms"
mkdir -p "$INSTALL_DIR/logs/vms"
mkdir -p /srv/samba

# Copy backend files
echo -e "${YELLOW}🔧 Kopiere Backend-Dateien...${NC}"
if [ "$VERBOSE" = true ]; then
    cp -v go-backend/* "$INSTALL_DIR/go-backend/"
else
    cp -r go-backend/* "$INSTALL_DIR/go-backend/"
fi

# Create .env file for backend
if [ "$VERBOSE" = true ]; then
    echo -e "${YELLOW}  → Erstelle .env Datei...${NC}"
fi
cat > "$INSTALL_DIR/go-backend/.env" <<EOF
DB_HOST=localhost
DB_NAME=$DB_NAME
DB_USER=$DB_USER
DB_PASS=$DB_PASS
SESSION_SECRET=$SESSION_SECRET
PORT=$BACKEND_PORT
INSTALL_DIR=$INSTALL_DIR
EOF

# Build backend
echo -e "${YELLOW}🔨 Baue Backend...${NC}"
cd "$INSTALL_DIR/go-backend"
export PATH=$PATH:/usr/local/go/bin:$HOME/.go/bin:$HOME/go/bin:/root/.go/bin

# Find Go binary (avoid aliases by checking file directly)
GO_BIN=""
if [ -f "/usr/local/go/bin/go" ]; then
    GO_BIN="/usr/local/go/bin/go"
elif [ -f "$HOME/.go/bin/go" ]; then
    GO_BIN="$HOME/.go/bin/go"
elif [ -f "$HOME/go/bin/go" ]; then
    GO_BIN="$HOME/go/bin/go"
elif [ -f "/root/.go/bin/go" ]; then
    GO_BIN="/root/.go/bin/go"
elif command -v go &> /dev/null; then
    # Last resort: use command -v but verify it's actually a file
    GO_BIN=$(command -v go)
    if [ ! -f "$GO_BIN" ]; then
        GO_BIN=""
    fi
fi

if [ -z "$GO_BIN" ] || [ ! -f "$GO_BIN" ]; then
    echo -e "${RED}❌ Go nicht gefunden! Bitte manuell installieren.${NC}"
    exit 1
fi

# Unalias go if it's aliased (won't affect our $GO_BIN)
unalias go 2>/dev/null || true

# Verify Go is available
if ! "$GO_BIN" version &> /dev/null; then
    echo -e "${RED}❌ Go ist nicht funktionsfähig!${NC}"
    exit 1
fi

if [ "$VERBOSE" = true ]; then
    echo -e "${YELLOW}  → Prüfe Go Version...${NC}"
    "$GO_BIN" version
    
    echo -e "${YELLOW}  → Initialisiere und synchronisiere Go Module...${NC}"
    GO_TIDY_OUTPUT=$("$GO_BIN" mod tidy 2>&1)
    GO_TIDY_STATUS=$?
    if [ $GO_TIDY_STATUS -ne 0 ]; then
        echo -e "${RED}❌ Fehler bei go mod tidy!${NC}"
        echo "$GO_TIDY_OUTPUT"
        exit 1
    fi
    
    echo -e "${YELLOW}  → Installiere Go Dependencies...${NC}"
    GO_DOWNLOAD_OUTPUT=$("$GO_BIN" mod download 2>&1)
    GO_DOWNLOAD_STATUS=$?
    if [ $GO_DOWNLOAD_STATUS -ne 0 ]; then
        echo -e "${RED}❌ Fehler beim Herunterladen der Dependencies!${NC}"
        echo "$GO_DOWNLOAD_OUTPUT"
        exit 1
    fi
    
    echo -e "${YELLOW}  → Verifiziere go.sum Datei...${NC}"
    if [ ! -f "go.sum" ]; then
        echo -e "${YELLOW}    Warnung: go.sum nicht gefunden, führe go mod tidy erneut aus...${NC}"
        "$GO_BIN" mod tidy
    fi
    
    echo -e "${YELLOW}  → Kompiliere Backend...${NC}"
    if ! "$GO_BIN" build -o tso-server . 2>&1 | tee /tmp/go-build-output.log; then
        echo -e "${RED}❌ Fehler beim Kompilieren des Backends!${NC}"
        echo -e "${YELLOW}Build-Output:${NC}"
        cat /tmp/go-build-output.log
        exit 1
    fi
else
    GO_TIDY_OUTPUT=$("$GO_BIN" mod tidy 2>&1)
    GO_TIDY_STATUS=$?
    if [ $GO_TIDY_STATUS -ne 0 ]; then
        echo -e "${RED}❌ Fehler bei go mod tidy!${NC}"
        echo "$GO_TIDY_OUTPUT"
        echo -e "${YELLOW}   Führen Sie install.sh mit --verbose aus für Details${NC}"
        exit 1
    fi
    
    GO_DOWNLOAD_OUTPUT=$("$GO_BIN" mod download 2>&1)
    GO_DOWNLOAD_STATUS=$?
    if [ $GO_DOWNLOAD_STATUS -ne 0 ]; then
        echo -e "${RED}❌ Fehler beim Herunterladen der Dependencies!${NC}"
        echo "$GO_DOWNLOAD_OUTPUT"
        echo -e "${YELLOW}   Führen Sie install.sh mit --verbose aus für Details${NC}"
        exit 1
    fi
    
    # Verify go.sum exists after tidy
    if [ ! -f "go.sum" ]; then
        echo -e "${YELLOW}  → go.sum fehlt, führe go mod tidy erneut aus...${NC}"
        "$GO_BIN" mod tidy
    fi
    
    GO_BUILD_OUTPUT=$("$GO_BIN" build -o tso-server . 2>&1)
    GO_BUILD_STATUS=$?
    if [ $GO_BUILD_STATUS -ne 0 ]; then
        echo -e "${RED}❌ Fehler beim Kompilieren des Backends!${NC}"
        echo "$GO_BUILD_OUTPUT"
        echo -e "${YELLOW}   Führen Sie install.sh mit --verbose aus für Details${NC}"
        exit 1
    fi
fi
echo -e "${GREEN}✓ Backend gebaut${NC}"

# Copy frontend files
echo -e "${YELLOW}🎨 Kopiere Frontend-Dateien...${NC}"
cd -
if [ "$VERBOSE" = true ]; then
    cp -v -r frontend/* "$INSTALL_DIR/frontend/"
else
    cp -r frontend/* "$INSTALL_DIR/frontend/"
fi

# Build frontend
echo -e "${YELLOW}🔨 Baue Frontend...${NC}"
cd "$INSTALL_DIR/frontend"
if [ "$VERBOSE" = true ]; then
    echo -e "${YELLOW}  → Installiere npm Dependencies...${NC}"
    npm install
    echo -e "${YELLOW}  → Baue Frontend...${NC}"
    npm run build
else
    npm install --silent > /dev/null 2>&1
    npm run build > /dev/null 2>&1
fi
echo -e "${GREEN}✓ Frontend gebaut${NC}"

cd -

# Create systemd service
echo -e "${YELLOW}⚙️  Erstelle Systemd-Service...${NC}"
cat > /etc/systemd/system/tso.service <<EOF
[Unit]
Description=TSO Server Management System
After=network.target mariadb.service

[Service]
Type=simple
User=root
WorkingDirectory=$INSTALL_DIR/go-backend
EnvironmentFile=$INSTALL_DIR/go-backend/.env
Environment="PATH=/usr/local/go/bin:/usr/local/sbin:/usr/local/bin:/usr/sbin:/usr/bin:/sbin:/bin"
Environment="INSTALL_DIR=$INSTALL_DIR"
ExecStart=$INSTALL_DIR/go-backend/tso-server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
EOF

if [ "$VERBOSE" = true ]; then
    echo -e "${GREEN}  ✓ Systemd-Service erstellt${NC}"
fi

systemctl daemon-reload
systemctl enable tso > /dev/null 2>&1

# Setup nginx if available
if command -v nginx &> /dev/null; then
    echo -e "${YELLOW}🌐 Konfiguriere Nginx...${NC}"
    cat > /etc/nginx/sites-available/tso <<EOF
server {
    listen 80;
    server_name _;
    
    root $INSTALL_DIR/frontend/dist;
    index index.html;
    
    location / {
        try_files \$uri \$uri/ /index.html;
    }
    
    location /api {
        proxy_pass http://localhost:$BACKEND_PORT;
        proxy_http_version 1.1;
        proxy_set_header Upgrade \$http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host \$host;
        proxy_cache_bypass \$http_upgrade;
        proxy_set_header X-Real-IP \$remote_addr;
        proxy_set_header X-Forwarded-For \$proxy_add_x_forwarded_for;
    }
}
EOF
    rm -f /etc/nginx/sites-enabled/default
    ln -sf /etc/nginx/sites-available/tso /etc/nginx/sites-enabled/
    if [ "$VERBOSE" = true ]; then
        nginx -t
        systemctl reload nginx
    else
        nginx -t > /dev/null 2>&1 && systemctl reload nginx > /dev/null 2>&1 || true
    fi
    echo -e "${GREEN}✓ Nginx konfiguriert${NC}"
fi

# Start service
echo -e "${YELLOW}🚀 Starte Service...${NC}"
systemctl start tso
sleep 2

if systemctl is-active --quiet tso; then
    echo -e "${GREEN}✓ Service läuft${NC}"
else
    echo -e "${YELLOW}⚠️  Service-Status prüfen: systemctl status tso${NC}"
    if [ "$VERBOSE" = true ]; then
        echo -e "${YELLOW}  → Service-Logs:${NC}"
        systemctl status tso --no-pager
        echo ""
        echo -e "${YELLOW}  → Letzte Journal-Logs:${NC}"
        journalctl -u tso -n 20 --no-pager
    fi
fi

# Done!
# Get public IP (first non-localhost IP)
PUBLIC_IP=$(hostname -I | awk '{print $1}' 2>/dev/null)
if [ -z "$PUBLIC_IP" ] || [ "$PUBLIC_IP" = "127.0.0.1" ]; then
    # Fallback: try ip command to get first non-loopback IPv4 address
    PUBLIC_IP=$(ip -4 addr show | grep 'inet ' | awk '{print $2}' | cut -d'/' -f1 | grep -v '^127\.' | head -n1)
fi
if [ -z "$PUBLIC_IP" ]; then
    PUBLIC_IP="localhost"
fi

# Determine port based on nginx availability
if command -v nginx &> /dev/null && systemctl is-active --quiet nginx 2>/dev/null; then
    PORT=""
else
    PORT=":$BACKEND_PORT"
fi

# Show only the URL at the end
echo ""
echo "http://${PUBLIC_IP}${PORT}"
