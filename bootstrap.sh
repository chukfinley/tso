#!/bin/bash

################################################################################
# TSO Bootstrap Script
# Einfach ausführen - lädt und installiert alles automatisch
# Nutzung: curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
################################################################################

set -e

REPO_URL="https://github.com/chukfinley/tso.git"

echo "🚀 TSO Bootstrap - Automatische Installation"
echo "=============================================="
echo ""

if [ "$EUID" -ne 0 ]; then
    echo "❌ Bitte mit sudo ausführen!"
    echo "   sudo bash bootstrap.sh"
    echo "   Oder: curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash"
    exit 1
fi

# Create temporary directory
TMP_DIR=$(mktemp -d)
cd "$TMP_DIR"

echo "📥 Lade TSO herunter..."
if ! git clone "$REPO_URL" . > /dev/null 2>&1; then
    echo "❌ Fehler beim Herunterladen!"
    echo "   Prüfe Internet-Verbindung und Repository-URL"
    exit 1
fi

if [ ! -f "install.sh" ]; then
    echo "❌ install.sh nicht gefunden!"
    exit 1
fi

echo "🔧 Führe Installation aus..."
chmod +x install.sh
bash install.sh

# Cleanup
cd /
rm -rf "$TMP_DIR"
