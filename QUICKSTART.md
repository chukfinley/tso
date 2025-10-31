# Quick Start Guide

Get TSO running in minutes!

## Production Installation

Install on your server with one command:

```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

### What happens during installation:

1. **System Check** (5 seconds)
   - Verifies Debian/Ubuntu OS
   - Checks for root privileges
   - Installs git if needed

2. **Package Installation** (2-5 minutes)
   - Updates system packages
   - Installs Apache2, MariaDB, PHP
   - Installs required PHP extensions

3. **Database Setup** (10 seconds)
   - Creates database
   - Imports schema
   - Sets up default admin user

4. **File Deployment** (5 seconds)
   - Copies files to `/opt/serveros`
   - Configures application
   - Sets permissions

5. **Web Server Configuration** (5 seconds)
   - Creates Apache virtual host
   - Enables required modules
   - Configures security headers

6. **Service Start** (5 seconds)
   - Starts Apache2
   - Starts MariaDB
   - Enables auto-start

### After Installation

You'll see output like this:

```
╔════════════════════════════════════════════════════════════════╗
║                 Installation Completed!                        ║
╚════════════════════════════════════════════════════════════════╝

Access Information:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  URL:      http://192.168.1.100
  Hostname: http://your-hostname

Default Credentials:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Username: admin
  Password: admin123
```

**Next Steps:**
1. Open your browser to the URL shown
2. Login with `admin` / `admin123`
3. **Immediately change the password!**
4. Start exploring TSO

## Development Mode

Want to test locally without full installation?

### Prerequisites

```bash
# Install only what you need
sudo apt install php php-mysql mariadb-server

# Setup database
sudo mysql < init.sql
```

### Run Development Server

```bash
# Clone repository
git clone https://github.com/chukfinley/tso.git
cd tso

# Start built-in PHP server
./quickstart.sh
```

Access at: **http://localhost:8000**

Default login: `admin` / `admin123`

### Development Features

- Uses PHP built-in server (no Apache needed)
- Runs on port 8000
- No file deployment needed
- Quick start/stop
- Perfect for testing changes

## Common Use Cases

### Home Server / NAS

```bash
# Install on your home server
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash

# Access from any device on your network
http://your-server-ip
```

### VPS / Cloud Server

```bash
# Install on Digital Ocean, AWS, etc.
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash

# Configure firewall
sudo ufw allow 80/tcp
sudo ufw allow 443/tcp
```

### Development / Testing

```bash
# Quick local testing
git clone https://github.com/chukfinley/tso.git
cd tso
./quickstart.sh
```

## Accessing After Installation

### By IP Address
```
http://192.168.1.100
```

### By Hostname
```
http://your-hostname
```

### From Same Machine
```
http://localhost
```

## First Login Checklist

After installation, do these immediately:

- [ ] Login with default credentials
- [ ] Change admin password (Users page)
- [ ] Create additional user accounts
- [ ] Test all modules work
- [ ] Bookmark the web interface

## Credentials Location

All installation details saved to:
```
/root/serveros_credentials.txt
```

View anytime:
```bash
sudo cat /root/serveros_credentials.txt
```

## Troubleshooting

### Can't access web interface?

```bash
# Check Apache is running
sudo systemctl status apache2

# Check firewall
sudo ufw status
sudo ufw allow 80/tcp
```

### Forgot password?

Reset admin password:
```bash
# Connect to database
sudo mysql servermanager

# Reset password to 'admin123'
UPDATE users SET password='$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi' WHERE username='admin';
```

### Need to reinstall?

```bash
# Uninstall first
cd /tmp
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/uninstall.sh | sudo bash

# Then reinstall
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

## What's Next?

- Explore the Dashboard
- Create users via Users page
- Check out the placeholder modules
- Read [CONTRIBUTING.md](CONTRIBUTING.md) to help develop features
- Star the repo on GitHub! ⭐

## Getting Help

- 📖 Read [INSTALL.md](INSTALL.md) for detailed setup
- 📖 Read [README.md](README.md) for features and roadmap
- 🐛 Report issues on GitHub
- 💬 Check closed issues for solutions

---

**Ready to get started?** Run the installer now:

```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```
