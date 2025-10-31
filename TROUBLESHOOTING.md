# ServerOS Troubleshooting Guide

Common issues and their solutions.

## Login Issues

### ❌ "Invalid username or password"

**Solution:**

```bash
# Step 1: Check if admin user exists
sudo /opt/serveros/tools/check-db.php

# Step 2: Reset admin password
sudo /opt/serveros/tools/reset-admin.php

# Step 3: Try logging in again with admin/admin123
```

**Alternative - Manual database fix:**
```bash
sudo mysql servermanager -e "
UPDATE users
SET password = '\$2y\$10\$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'
WHERE username = 'admin';
"
```

## Installation Issues

### ❌ Installation fails with database errors

**Solution:**
```bash
# Check if MariaDB is running
sudo systemctl status mariadb
sudo systemctl start mariadb

# Re-run installation
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

### ❌ Can't access web interface after installation

**Solution:**
```bash
# Check Apache is running
sudo systemctl status apache2
sudo systemctl restart apache2

# Check firewall
sudo ufw allow 80/tcp
sudo ufw status

# Get your server IP
hostname -I
```

### ❌ "Connection refused" or blank page

**Solution:**
```bash
# Check Apache configuration
sudo apache2ctl -t

# Check Apache error logs
sudo tail -50 /var/log/apache2/serveros_error.log

# Restart Apache
sudo systemctl restart apache2
```

## Database Issues

### ❌ "Database connection failed"

**Solution:**
```bash
# Check database credentials
sudo cat /opt/serveros/config/config.php | grep DB_

# Test database connection
sudo mysql -u serveros -p servermanager
# (Use password from config.php)

# If credentials are wrong, update them
sudo nano /opt/serveros/config/config.php
```

### ❌ "Table 'users' doesn't exist"

**Solution:**
```bash
# Import database schema
sudo mysql servermanager < /opt/serveros/init.sql

# Create admin user
sudo /opt/serveros/tools/reset-admin.php
```

## Permission Issues

### ❌ "Permission denied" errors in browser

**Solution:**
```bash
# Fix ownership
sudo chown -R www-data:www-data /opt/serveros

# Fix permissions
sudo chmod -R 755 /opt/serveros
sudo chmod -R 775 /opt/serveros/logs
sudo chmod -R 775 /opt/serveros/storage

# Restart Apache
sudo systemctl restart apache2
```

### ❌ Can't write to logs or storage

**Solution:**
```bash
# Make directories writable
sudo chmod -R 775 /opt/serveros/logs
sudo chmod -R 775 /opt/serveros/storage
sudo chown -R www-data:www-data /opt/serveros/logs
sudo chown -R www-data:www-data /opt/serveros/storage
```

## PHP Issues

### ❌ "PHP Fatal error" or blank pages

**Solution:**
```bash
# Check PHP error log
sudo tail -50 /var/log/apache2/serveros_error.log

# Check PHP version (needs 7.4+)
php -v

# Restart Apache
sudo systemctl restart apache2
```

### ❌ "Call to undefined function"

**Solution:**
```bash
# Install missing PHP extensions
sudo apt install php-mysql php-mbstring php-xml php-curl

# Restart Apache
sudo systemctl restart apache2
```

## Network/Access Issues

### ❌ Can't access from other computers

**Solution:**
```bash
# Check firewall
sudo ufw allow 80/tcp
sudo ufw status

# Get server IP
hostname -I

# Access from another computer:
# http://YOUR_SERVER_IP
```

### ❌ "Forbidden" error

**Solution:**
```bash
# Check Apache configuration
sudo cat /etc/apache2/sites-available/serveros.conf

# Ensure site is enabled
sudo a2ensite serveros
sudo systemctl restart apache2
```

## User Management Issues

### ❌ Can't create new users

**Symptoms:** "Failed to create user" error

**Solution:**
```bash
# Check database connection
sudo /opt/serveros/tools/check-db.php

# Check if you're logged in as admin
# Only admins can create users
```

### ❌ Locked out (forgot admin password)

**Solution:**
```bash
# Reset admin password to 'admin123'
sudo /opt/serveros/tools/reset-admin.php

# Or manually via database:
sudo mysql servermanager -e "
UPDATE users
SET password = '\$2y\$10\$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'
WHERE username = 'admin';
"
```

## Performance Issues

### ❌ Slow loading or timeouts

**Solution:**
```bash
# Check system resources
free -h
df -h

# Check running processes
top

# Restart services
sudo systemctl restart apache2 mariadb
```

## Diagnostic Tools

### Quick Health Check
```bash
# Database check
sudo /opt/serveros/tools/check-db.php

# Apache status
sudo systemctl status apache2

# MariaDB status
sudo systemctl status mariadb

# View logs
sudo tail -50 /var/log/apache2/serveros_error.log
```

### View All Logs
```bash
# Apache error log
sudo tail -f /var/log/apache2/serveros_error.log

# Apache access log
sudo tail -f /var/log/apache2/serveros_access.log

# Application logs (if any)
sudo ls -lah /opt/serveros/logs/
```

## Complete Reinstallation

If nothing else works:

```bash
# 1. Backup any data you need
sudo cp -r /opt/serveros /tmp/serveros-backup

# 2. Uninstall
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/uninstall.sh | sudo bash

# 3. Reinstall
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

## Getting Help

If you're still stuck:

1. **Check logs:**
   ```bash
   sudo /opt/serveros/tools/check-db.php
   sudo tail -100 /var/log/apache2/serveros_error.log
   ```

2. **Gather system info:**
   ```bash
   cat /etc/os-release
   php -v
   mysql --version
   ```

3. **Report issue on GitHub:**
   - Include error messages
   - Include logs (remove sensitive data!)
   - Include system info
   - Steps to reproduce

## Quick Reference

| Problem | Solution Command |
|---------|-----------------|
| Reset admin password | `sudo /opt/serveros/tools/reset-admin.php` |
| Check database | `sudo /opt/serveros/tools/check-db.php` |
| Restart Apache | `sudo systemctl restart apache2` |
| Restart MariaDB | `sudo systemctl restart mariadb` |
| View errors | `sudo tail -50 /var/log/apache2/serveros_error.log` |
| Fix permissions | `sudo chown -R www-data:www-data /opt/serveros` |
| Reinstall | See "Complete Reinstallation" above |

## Prevention

- Always change default password after installation
- Keep system updated: `sudo apt update && sudo apt upgrade`
- Regular backups of database
- Monitor logs periodically
- Don't edit core files unless you know what you're doing

---

**Still need help?** Open an issue: https://github.com/chukfinley/tso/issues
