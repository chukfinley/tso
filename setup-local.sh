#!/bin/bash
# Quick setup script for local testing

set -e

echo "🚀 TSO Local Setup Script"
echo "=========================="
echo ""

# Check prerequisites
echo "Checking prerequisites..."

if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed. Please install Go 1.21+"
    exit 1
fi

if ! command -v node &> /dev/null; then
    echo "❌ Node.js is not installed. Please install Node.js 18+"
    exit 1
fi

if ! command -v mysql &> /dev/null; then
    echo "❌ MySQL/MariaDB is not installed. Please install MariaDB"
    exit 1
fi

echo "✅ Prerequisites check passed"
echo ""

# Database setup
echo "Setting up database..."
read -p "MySQL root password: " -s ROOT_PASS
echo ""

read -p "Database user (default: tso): " DB_USER
DB_USER=${DB_USER:-tso}

read -p "Database password (default: tso_password): " -s DB_PASS
DB_PASS=${DB_PASS:-tso_password}
echo ""

DB_NAME="servermanager"

# Create database and user
echo "Creating database and user..."
mysql -u root -p${ROOT_PASS} <<EOF
CREATE DATABASE IF NOT EXISTS ${DB_NAME};
CREATE USER IF NOT EXISTS '${DB_USER}'@'localhost' IDENTIFIED BY '${DB_PASS}';
GRANT ALL PRIVILEGES ON ${DB_NAME}.* TO '${DB_USER}'@'localhost';
FLUSH PRIVILEGES;
EOF

# Import schema
echo "Importing database schema..."
if [ -f "init.sql" ]; then
    mysql -u ${DB_USER} -p${DB_PASS} ${DB_NAME} < init.sql
    echo "✅ Database schema imported"
else
    echo "⚠️  init.sql not found, skipping schema import"
fi

echo ""

# Backend setup
echo "Setting up backend..."
cd go-backend

if [ ! -f "go.mod" ]; then
    go mod init github.com/chukfinley/tso
fi

go mod download
echo "✅ Backend dependencies installed"

# Create .env file
cat > .env <<EOF
DB_HOST=localhost
DB_NAME=${DB_NAME}
DB_USER=${DB_USER}
DB_PASS=${DB_PASS}
SESSION_SECRET=$(openssl rand -hex 32)
PORT=8080
EOF

echo "✅ Backend configuration created (.env file)"
cd ..

echo ""

# Frontend setup
echo "Setting up frontend..."
cd frontend

if [ ! -d "node_modules" ]; then
    npm install
    echo "✅ Frontend dependencies installed"
else
    echo "✅ Frontend dependencies already installed"
fi

cd ..

echo ""
echo "✅ Setup complete!"
echo ""
echo "To run the application:"
echo ""
echo "1. Start backend (in one terminal):"
echo "   cd go-backend"
echo "   export \$(cat .env | xargs)"
echo "   go run ."
echo ""
echo "2. Start frontend (in another terminal):"
echo "   cd frontend"
echo "   npm run dev"
echo ""
echo "3. Open http://localhost:3000 in your browser"
echo ""

