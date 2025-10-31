# TSO - Installation Guide

This guide will help you install TSO on a Debian-based system.

## Quick Installation (Recommended)

Install TSO with a single command:

```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

**That's it!** No need to clone the repository or install dependencies manually.

### Alternative: Git Clone Method

If you prefer to clone first:

```bash
git clone https://github.com/chukfinley/tso.git
cd tso
sudo ./install.sh
```

### What the Installer Does

The installer will:
- ✓ Update your system
- ✓ Install Apache2, MariaDB, and PHP
- ✓ Create and configure the database
- ✓ Deploy application files to `/opt/serveros`
- ✓ Configure Apache virtual host
- ✓ Set proper permissions
- ✓ Configure firewall rules
- ✓ Start all services

## Installation Time

The entire installation process takes approximately **5-10 minutes** depending on your internet connection and system specifications.

---

## Updating Existing Installation

**Good news:** The installer can update existing installations!

If you already have TSO installed and want to update to the latest version:

```bash
# Method 1: Re-run the bootstrap (recommended)
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash

# Method 2: Use the update script
sudo /opt/serveros/update.sh

# Method 3: Git clone and install
git clone https://github.com/chukfinley/tso.git
cd tso
sudo ./install.sh
# Will detect existing installation and offer to update
```

### What Gets Updated:
- ✅ PHP application files
- ✅ HTML/CSS/JavaScript
- ✅ Backend classes
- ✅ Templates
- ✅ Tools and utilities

### What Gets Preserved:
- ✅ Database credentials in config.php
- ✅ All database tables and data
- ✅ All user accounts
- ✅ Logs
- ✅ Storage data
- ✅ Apache configuration

**No manual database migration needed!** The update is safe and automatic.

---

## System Requirements

### Minimum Requirements:
- **OS:** Debian 10+ or Ubuntu 20.04+
- **RAM:** 512 MB (1 GB recommended)
- **Disk:** 2 GB free space
- **CPU:** 1 core (2+ recommended)
- **Network:** Internet connection for package installation

### Required Privileges:
- Root or sudo access

## What Gets Installed

### Packages:
- `apache2` - Web server
- `mariadb-server` - Database server
- `php` + extensions - Application runtime
- `curl`, `wget`, `git` - Utilities
- `openssl` - Security tools

### Services:
- Apache2 (enabled and started)
- MariaDB (enabled and started)

### Files:
- Application: `/opt/serveros/`
- Apache config: `/etc/apache2/sites-available/serveros.conf`
- Credentials: `/root/serveros_credentials.txt`

## Post-Installation

After installation completes, you'll see:

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

Database Information:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Database: servermanager
  User:     serveros
  Password: [randomly generated]
```

### First Login Steps:

1. **Access the web interface:**
   ```
   http://YOUR_SERVER_IP
   ```

2. **Login with default credentials:**
   - Username: `admin`
   - Password: `admin123`

3. **⚠️ IMMEDIATELY change the default password:**
   - Go to Users page
   - Update the admin user password

4. **Create additional users** (if needed)
   - Navigate to Users → Create New User

## Accessing Your Credentials

All installation credentials are saved to:
```
/root/serveros_credentials.txt
```

View them anytime:
```bash
sudo cat /root/serveros_credentials.txt
```

## Troubleshooting

### Installation fails with "command not found"
Make sure the script is executable:
```bash
chmod +x install.sh
sudo ./install.sh
```

### Cannot access web interface
1. Check if Apache is running:
   ```bash
   sudo systemctl status apache2
   ```

2. Check firewall:
   ```bash
   sudo ufw status
   sudo ufw allow 80/tcp
   ```

3. Check Apache logs:
   ```bash
   sudo tail -f /var/log/apache2/serveros_error.log
   ```

### Database connection errors
1. Verify database credentials in:
   ```bash
   sudo nano /opt/serveros/config/config.php
   ```

2. Check if MariaDB is running:
   ```bash
   sudo systemctl status mariadb
   ```

3. Test database connection:
   ```bash
   sudo mysql -u serveros -p servermanager
   ```

### Permission issues
Reset permissions:
```bash
sudo chown -R www-data:www-data /opt/serveros
sudo chmod -R 755 /opt/serveros
sudo chmod -R 775 /opt/serveros/logs
sudo chmod -R 775 /opt/serveros/storage
```

## Manual Installation

If you prefer to install manually, follow the detailed steps in [README.md](README.md#installation).

## Uninstallation

To completely remove TSO:

```bash
cd serveros
sudo ./uninstall.sh
```

This will:
- Remove all application files
- Drop the database
- Remove Apache configuration
- Backup credentials to `/root/serveros_credentials.txt.bak`

## Updating

To update TSO to the latest version:

```bash
cd serveros
git pull origin master
sudo cp -r public src views config /opt/serveros/
sudo systemctl restart apache2
```

## Security Recommendations

1. **Change default password immediately** after first login
2. **Use strong passwords** (min 12 characters, mixed case, numbers, symbols)
3. **Enable firewall:**
   ```bash
   sudo ufw enable
   sudo ufw allow 22/tcp  # SSH
   sudo ufw allow 80/tcp  # HTTP
   sudo ufw allow 443/tcp # HTTPS (if using SSL)
   ```
4. **Setup SSL/HTTPS** for production use
5. **Regular backups** of database and configuration
6. **Keep system updated:**
   ```bash
   sudo apt update && sudo apt upgrade
   ```

## Getting Help

- Check logs: `/var/log/apache2/serveros_error.log`
- Application logs: `/opt/serveros/logs/`
- Report issues: GitHub Issues
- Documentation: [README.md](README.md)

## Next Steps

After successful installation:

1. ✓ Login to web interface
2. ✓ Change admin password
3. ✓ Explore the Dashboard
4. ✓ Create additional users
5. ✓ Start configuring your system modules

Welcome to TSO! 🚀
