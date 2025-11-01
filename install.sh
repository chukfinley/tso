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
    echo -e "${YELLOW}  → Installiere Go...${NC}"
    GO_VERSION="1.21.5"
    if [ "$VERBOSE" = true ]; then
        echo -e "${YELLOW}    Lade Go ${GO_VERSION} herunter...${NC}"
        if ! wget https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz; then
            echo -e "${RED}❌ Fehler beim Herunterladen von Go!${NC}"
            exit 1
        fi
        echo -e "${YELLOW}    Entpacke Go...${NC}"
        if ! tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz; then
            echo -e "${RED}❌ Fehler beim Entpacken von Go!${NC}"
            exit 1
        fi
    else
        if ! wget -q https://go.dev/dl/go${GO_VERSION}.linux-amd64.tar.gz; then
            echo -e "${RED}❌ Fehler beim Herunterladen von Go!${NC}"
            exit 1
        fi
        if ! tar -C /usr/local -xzf go${GO_VERSION}.linux-amd64.tar.gz; then
            echo -e "${RED}❌ Fehler beim Entpacken von Go!${NC}"
            exit 1
        fi
    fi
    rm -f go${GO_VERSION}.linux-amd64.tar.gz
    export PATH=$PATH:/usr/local/go/bin
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    if [ "$VERBOSE" = true ]; then
        echo -e "${GREEN}  ✓ Go installiert${NC}"
    fi
else
    GO_VERSION=$(go version 2>&1 || echo "unknown")
    if [ "$VERBOSE" = true ]; then
        echo -e "${GREEN}  ✓ Go bereits installiert: $GO_VERSION${NC}"
    fi
fi

# Install Node.js if not present
if ! command -v node &> /dev/null; then
    echo -e "${YELLOW}  → Installiere Node.js...${NC}"
    if [ "$VERBOSE" = true ]; then
        echo -e "${YELLOW}    Füge NodeSource Repository hinzu...${NC}"
        if ! curl -fsSL https://deb.nodesource.com/setup_18.x | bash -; then
            echo -e "${RED}❌ Fehler beim Hinzufügen des NodeSource Repositories!${NC}"
            exit 1
        fi
        echo -e "${YELLOW}    Installiere Node.js...${NC}"
        if ! apt-get install -y nodejs; then
            echo -e "${RED}❌ Fehler beim Installieren von Node.js!${NC}"
            exit 1
        fi
    else
        if ! curl -fsSL https://deb.nodesource.com/setup_18.x | bash - > /dev/null 2>&1; then
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
ADMIN_HASH='$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'  # Password: admin123
if [ "$VERBOSE" = true ]; then
    echo -e "${YELLOW}  → Erstelle Admin-Benutzer...${NC}"
fi
mysql -u $DB_USER -p$DB_PASS $DB_NAME <<EOF
INSERT IGNORE INTO users (username, email, password, full_name, role, is_active) 
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
EOF

# Build backend
echo -e "${YELLOW}🔨 Baue Backend...${NC}"
cd "$INSTALL_DIR/go-backend"
export PATH=$PATH:/usr/local/go/bin

if [ "$VERBOSE" = true ]; then
    echo -e "${YELLOW}  → Installiere Go Dependencies...${NC}"
    go mod download
    echo -e "${YELLOW}  → Kompiliere Backend...${NC}"
    go build -o tso-server .
else
    go mod download > /dev/null 2>&1
    go build -o tso-server . > /dev/null 2>&1
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
echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║                  Installation abgeschlossen!                  ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Installationsverzeichnis:${NC} $INSTALL_DIR"
echo -e "${BLUE}Datenbank:${NC} $DB_NAME"
echo -e "${BLUE}Datenbank-Benutzer:${NC} $DB_USER"
echo -e "${BLUE}Backend-Port:${NC} $BACKEND_PORT"
echo ""
if [ "$VERBOSE" = true ]; then
    echo -e "${YELLOW}Verbose-Modus: Alle Details wurden angezeigt${NC}"
    echo ""
fi
if command -v nginx &> /dev/null; then
    echo -e "${GREEN}🌐 Web-Interface:${NC} http://$(hostname -I | awk '{print $1}')"
    echo -e "${GREEN}   Oder:${NC} http://localhost"
else
    echo -e "${GREEN}🌐 Backend API:${NC} http://localhost:$BACKEND_PORT/api"
    echo -e "${YELLOW}⚠️  Frontend muss separat bereitgestellt werden${NC}"
fi
echo ""
echo -e "${YELLOW}⚠️  Standard-Anmeldedaten:${NC}"
echo -e "   ${BLUE}Benutzername:${NC} admin"
echo -e "   ${BLUE}Passwort:${NC} admin123"
echo -e "${RED}   ⚠️  BITTE SOFORT ÄNDERN!${NC}"
echo ""
echo -e "${BLUE}Service-Befehle:${NC}"
echo "   sudo systemctl start tso    # Starten"
echo "   sudo systemctl stop tso     # Stoppen"
echo "   sudo systemctl restart tso  # Neustart"
echo "   sudo systemctl status tso   # Status prüfen"
if [ "$VERBOSE" = true ]; then
    echo "   sudo journalctl -u tso -f  # Logs live anzeigen"
fi
echo ""
echo -e "${GREEN}✅ Fertig! Viel Erfolg! 🚀${NC}"
