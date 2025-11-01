#!/bin/bash

################################################################################
# TSO Bootstrap Script
# Einfach ausführen - lädt und installiert alles automatisch
# Nutzung: curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
################################################################################

set -e  # Exit on error

REPO_URL="https://github.com/chukfinley/tso.git"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}"
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║              TSO Bootstrap - Automatische Installation         ║"
echo "║          Lädt und installiert TSO vollständig                  ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo -e "${NC}"

if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}❌ Bitte mit sudo ausführen!${NC}"
    echo "   sudo bash bootstrap.sh"
    echo "   Oder: curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash"
    exit 1
fi

# Create temporary directory
TMP_DIR=$(mktemp -d)
echo -e "${YELLOW}📁 Temporäres Verzeichnis:${NC} $TMP_DIR"
cd "$TMP_DIR"

# Trap to cleanup on exit
cleanup() {
    echo ""
    echo -e "${YELLOW}🧹 Aufräumen...${NC}"
    cd /
    rm -rf "$TMP_DIR"
}
trap cleanup EXIT

echo -e "${YELLOW}📥 Lade TSO von GitHub herunter...${NC}"
echo -e "${YELLOW}   Repository:${NC} $REPO_URL"
echo ""

if ! git clone "$REPO_URL" . 2>&1; then
    echo -e "${RED}❌ Fehler beim Herunterladen!${NC}"
    echo -e "${RED}   Prüfe Internet-Verbindung und Repository-URL${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}✓ Repository heruntergeladen${NC}"

if [ ! -f "install.sh" ]; then
    echo -e "${RED}❌ install.sh nicht gefunden!${NC}"
    echo -e "${YELLOW}   Dateien im Repository:${NC}"
    ls -la
    exit 1
fi

echo -e "${GREEN}✓ install.sh gefunden${NC}"
echo ""
echo -e "${YELLOW}🔧 Starte Installation mit Verbose-Modus...${NC}"
echo ""

# Make install.sh executable
chmod +x install.sh

# Run install.sh with verbose mode
if bash install.sh --verbose; then
    echo ""
    echo -e "${GREEN}✅ Installation erfolgreich abgeschlossen!${NC}"
else
    EXIT_CODE=$?
    echo ""
    echo -e "${RED}❌ Installation fehlgeschlagen mit Exit-Code: $EXIT_CODE${NC}"
    echo -e "${YELLOW}🔍 Prüfe die obigen Fehler-Meldungen${NC}"
    exit $EXIT_CODE
fi
