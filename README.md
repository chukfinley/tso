# TSO 🚀

Debian-basiertes Server-Management-System - gebaut mit **Go** (Backend) und **TypeScript/React** (Frontend).

**Einfache Installation:**

Lokal (wenn Repository bereits vorhanden):
```bash
sudo ./install.sh
```

Von GitHub (ein Befehl, lädt und installiert alles):
```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

## Features

- ✅ **Sicherer Login** - Session-basierte Authentifizierung
- ✅ **Benutzerverwaltung** - Vollständige CRUD-Operationen
- ✅ **Moderne Web-UI** - Dark Theme, responsive Design
- ✅ **Netzwerk-Freigaben** - SMB/Samba-Verwaltung mit Benutzerberechtigungen
- ✅ **Virtuelle Maschinen** - QEMU/KVM-Verwaltung mit SPICE-Konsole
- ✅ **Web-Terminal** - Browser-basiertes Terminal (Admin)
- ✅ **System-Monitoring** - Echtzeit-Systemstatistiken
- ✅ **Aktivitäts-Protokollierung** - Vollständige Protokollierung aller Aktionen

## Installation

### Automatische Installation (Empfohlen)

**Normal (minimaler Output):**
```bash
sudo ./install.sh
```

**Verbose-Modus (zeigt alle Details - besser zum Debuggen):**
```bash
sudo ./install.sh --verbose
# oder kurz:
sudo ./install.sh -v
```

Das Skript installiert automatisch:
- Go, Node.js, MariaDB
- Erstellt die Datenbank
- Baut Backend und Frontend
- Startet den Service
- Konfiguriert Nginx (falls vorhanden)

**Verbose-Modus zeigt:**
- ✅ Alle Command-Outputs
- ✅ Installations-Fortschritt
- ✅ Fehler-Details
- ✅ Service-Logs bei Problemen

### Ein-Zeilen-Installation (von GitHub)

Lädt das gesamte Repository herunter und installiert automatisch **mit vollständigem Output**:

```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

**Was passiert:**
1. ✅ Lädt TSO von GitHub herunter (zeigt Fortschritt)
2. ✅ Installiert alle Abhängigkeiten (Go, Node.js, MariaDB) - **zeigt alle Outputs**
3. ✅ Erstellt Datenbank - **zeigt alle SQL-Operationen**
4. ✅ Baut Backend und Frontend - **zeigt Build-Logs**
5. ✅ Startet den Service - **zeigt Service-Status**

**Hinweis:** Das bootstrap.sh Skript verwendet automatisch den Verbose-Modus, damit Sie alle Details sehen können!

## Zugriff

Nach der Installation:

- **Web-Interface:** `http://localhost` (falls Nginx installiert)
- **Oder direkt:** `http://localhost:8080/api` (Backend)

**Standard-Anmeldedaten:**
- Benutzername: `admin`
- Passwort: `admin123`

⚠️ **SOFORT ÄNDERN!**

## Service-Verwaltung

```bash
sudo systemctl start tso      # Starten
sudo systemctl stop tso       # Stoppen
sudo systemctl restart tso     # Neustart
sudo systemctl status tso     # Status prüfen
```

## Projekt-Struktur

```
tso/
├── go-backend/          # Go Backend
│   ├── main.go         # Server Entry Point
│   ├── auth.go         # Authentifizierung
│   ├── users.go        # Benutzerverwaltung
│   ├── shares.go       # Freigaben-Verwaltung
│   ├── vms.go          # VM-Verwaltung
│   └── system.go       # System-Statistiken
├── frontend/            # TypeScript/React Frontend
│   ├── src/
│   │   ├── pages/      # Seiten-Komponenten
│   │   ├── components/ # Wiederverwendbare Komponenten
│   │   └── api/        # API-Client
│   └── package.json
├── init.sql             # Datenbank-Schema
└── install.sh           # Installations-Skript
```

## Lokale Entwicklung

```bash
# Terminal 1 - Backend
cd go-backend
export DB_HOST=localhost DB_NAME=servermanager DB_USER=root DB_PASS=dein_pass SESSION_SECRET=secret PORT=8080
go run .

# Terminal 2 - Frontend
cd frontend
npm install
npm run dev
```

Dann öffnen: `http://localhost:3000`

Siehe [QUICKSTART-LOCAL.md](QUICKSTART-LOCAL.md) für Details.

## Anforderungen

- Go 1.21+
- Node.js 18+
- MySQL/MariaDB
- Debian 10+ / Ubuntu 20.04+

Optional:
- Samba (für Freigaben)
- QEMU/KVM (für VMs)
- Nginx (für Production)

## Lizenz

Siehe LICENSE Datei.
