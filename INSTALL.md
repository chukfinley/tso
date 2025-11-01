# TSO Installation Guide

Complete installation guide for TSO (The Server OS) - Go & TypeScript version.

## Prerequisites

### Required Software

- **Go** 1.21 or higher
- **Node.js** 18+ and npm
- **MySQL** 5.7+ or **MariaDB** 10.2+
- **Debian** 10+ or **Ubuntu** 20.04+ (recommended)

### Optional Software

- **Samba** (for network share management)
- **QEMU/KVM** (for virtual machine management)
- **Nginx** or **Apache** (for production web serving)

## Quick Installation

### Automated Installation

```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

Or:

```bash
git clone https://github.com/chukfinley/tso.git
cd tso
sudo ./install.sh
```

## Manual Installation

### Step 1: Install System Dependencies

**Debian/Ubuntu:**

```bash
sudo apt update
sudo apt install -y curl wget git build-essential

# Install Go
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc

# Install Node.js
curl -fsSL https://deb.nodesource.com/setup_18.x | sudo -E bash -
sudo apt install -y nodejs

# Install MariaDB
sudo apt install -y mariadb-server
sudo systemctl start mariadb
sudo systemctl enable mariadb

# Optional: Install Samba and QEMU
sudo apt install -y samba qemu-kvm libvirt-daemon-system
```

### Step 2: Setup Database

```bash
sudo mysql -u root -p
```

Run:

```sql
CREATE DATABASE servermanager CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
CREATE USER 'tso'@'localhost' IDENTIFIED BY 'your_secure_password';
GRANT ALL PRIVILEGES ON servermanager.* TO 'tso'@'localhost';
FLUSH PRIVILEGES;
EXIT;
```

Import schema:

```bash
cd /path/to/tso
mysql -u tso -p servermanager < init.sql
```

### Step 3: Setup Backend

```bash
cd /path/to/tso/go-backend

# Install Go dependencies
go mod download

# Create .env file
cat > .env <<EOF
DB_HOST=localhost
DB_NAME=servermanager
DB_USER=tso
DB_PASS=your_secure_password
SESSION_SECRET=$(openssl rand -hex 32)
PORT=8080
EOF

# Build backend
go build -o tso-server .

# Test run
./tso-server
```

### Step 4: Setup Frontend

```bash
cd /path/to/tso/frontend

# Install dependencies
npm install

# Build for production
npm run build
```

### Step 5: Create Systemd Service

Create `/etc/systemd/system/tso.service`:

```ini
[Unit]
Description=TSO Server Management System
After=network.target mariadb.service

[Service]
Type=simple
User=root
WorkingDirectory=/opt/serveros/go-backend
EnvironmentFile=/opt/serveros/go-backend/.env
ExecStart=/opt/serveros/go-backend/tso-server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

Enable and start:

```bash
sudo systemctl daemon-reload
sudo systemctl enable tso
sudo systemctl start tso
sudo systemctl status tso
```

### Step 6: Configure Web Server (Optional)

#### Nginx Configuration

Create `/etc/nginx/sites-available/tso`:

```nginx
server {
    listen 80;
    server_name your-domain.com;
    
    root /opt/serveros/frontend/dist;
    index index.html;
    
    location / {
        try_files $uri $uri/ /index.html;
    }
    
    location /api {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

Enable:

```bash
sudo ln -s /etc/nginx/sites-available/tso /etc/nginx/sites-enabled/
sudo nginx -t
sudo systemctl reload nginx
```

## Post-Installation

### 1. Create Admin User

If the database is fresh, create an admin user:

```sql
mysql -u tso -p servermanager

INSERT INTO users (username, email, password, full_name, role, is_active) 
VALUES ('admin', 'admin@localhost', '$2y$10$YourHashedPasswordHere', 'Administrator', 'admin', 1);
```

Or use the default credentials (if created by installer):
- Username: `admin`
- Password: `admin123`

**⚠️ Change the default password immediately!**

### 2. Access Web Interface

- **Local development**: `http://localhost:3000`
- **Production (with nginx)**: `http://your-server-ip`
- **Direct backend**: `http://localhost:8080/api`

### 3. Verify Installation

1. Login to web interface
2. Check dashboard shows system stats
3. Verify all services are running:
   ```bash
   sudo systemctl status tso
   sudo systemctl status mariadb
   ```

## Local Development Setup

For local testing without systemd:

### Terminal 1 - Backend:

```bash
cd go-backend
export $(cat .env | xargs)  # Or set manually
go run .
```

### Terminal 2 - Frontend:

```bash
cd frontend
npm run dev
```

Access at `http://localhost:3000`

## Troubleshooting

### Backend Won't Start

1. Check database connection:
   ```bash
   mysql -u tso -p servermanager
   ```

2. Verify environment variables:
   ```bash
   cat go-backend/.env
   ```

3. Check port availability:
   ```bash
   sudo lsof -i :8080
   ```

4. View logs:
   ```bash
   sudo journalctl -u tso -f
   ```

### Frontend Won't Build

1. Check Node version:
   ```bash
   node --version  # Should be 18+
   ```

2. Clear and reinstall:
   ```bash
   rm -rf node_modules package-lock.json
   npm install
   ```

3. Check TypeScript errors:
   ```bash
   npm run build  # Will show errors
   ```

### Database Connection Issues

1. Verify MySQL is running:
   ```bash
   sudo systemctl status mariadb
   ```

2. Test connection:
   ```bash
   mysql -u tso -p servermanager
   ```

3. Check credentials in `.env` file

## Updating

To update an existing installation:

```bash
cd /opt/serveros
sudo git pull
cd go-backend
go mod download
go build -o tso-server .
cd ../frontend
npm install
npm run build
sudo systemctl restart tso
```

## Uninstallation

To remove TSO:

```bash
sudo systemctl stop tso
sudo systemctl disable tso
sudo rm /etc/systemd/system/tso.service
sudo rm -rf /opt/serveros
sudo systemctl daemon-reload
```

To remove database (optional):

```bash
sudo mysql -u root -p
DROP DATABASE servermanager;
DROP USER 'tso'@'localhost';
```

## Next Steps

- Change default admin password
- Configure Samba shares
- Create virtual machines
- Set up SSL/TLS certificates
- Configure firewall rules
- Set up backup schedule

For more information, see [README.md](README.md) and [QUICKSTART-LOCAL.md](QUICKSTART-LOCAL.md).
