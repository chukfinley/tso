# TSO Troubleshooting Tools

These utilities help diagnose and fix common issues with TSO.

## Available Tools

### 1. Database Diagnostic Tool

**check-db.php** - Comprehensive database check

```bash
sudo /opt/serveros/tools/check-db.php
```

**What it checks:**
- ✓ Database connection
- ✓ Database schema (tables exist)
- ✓ Admin user exists
- ✓ Password hash is correct
- ✓ Lists all users

**Example Output:**
```
╔════════════════════════════════════════════════════════════════╗
║         TSO Database Diagnostic Tool                      ║
╚════════════════════════════════════════════════════════════════╝

✓ Config found: /opt/serveros/config/config.php

Database Configuration:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Host:     localhost
  Database: servermanager
  User:     serveros
  Password: ************

Testing database connection...
✓ Database connection successful!

Checking database schema...
✓ Users table exists

Checking for admin user...
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
✓ Admin user found:
  ID:       1
  Username: admin
  Email:    admin@localhost
  Role:     admin
  Active:   Yes
  Created:  2024-10-31 12:34:56

Password Information:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Hash: $2y$10$...
  Type: bcrypt

Testing default password 'admin123'...
✓ Password 'admin123' is CORRECT!

You should be able to login with:
  Username: admin
  Password: admin123
```

### 2. Admin Password Reset Tool

**reset-admin.php** - Reset or create admin user

```bash
sudo /opt/serveros/tools/reset-admin.php
```

**What it does:**
- Creates admin user if missing
- Resets admin password to 'admin123'
- Ensures admin has proper role and permissions

**Example Output:**
```
╔════════════════════════════════════════════════════════════════╗
║         TSO Admin Password Reset Utility                  ║
╚════════════════════════════════════════════════════════════════╝

✓ Connected to database

Generated password hash for 'admin123':
  $2y$10$...

Admin user exists. Resetting password...
✓ Admin password reset successfully!

╔════════════════════════════════════════════════════════════════╗
║                    Reset Complete!                             ║
╚════════════════════════════════════════════════════════════════╝

You can now login with:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  Username: admin
  Password: admin123

⚠  IMPORTANT: Change this password after first login!
```

## Common Issues and Solutions

### Issue: "Invalid username or password"

**Cause:** Admin user not created or password hash is incorrect

**Solution:**
```bash
# Check if admin exists and password is correct
sudo /opt/serveros/tools/check-db.php

# If password is wrong, reset it
sudo /opt/serveros/tools/reset-admin.php
```

### Issue: "Database connection failed"

**Cause:** Database credentials are incorrect

**Solution:**
```bash
# Check database credentials
sudo cat /opt/serveros/config/config.php | grep DB_

# Test database connection
sudo /opt/serveros/tools/check-db.php

# If needed, update credentials in config.php
sudo nano /opt/serveros/config/config.php
```

### Issue: "Users table does not exist"

**Cause:** Database schema not imported

**Solution:**
```bash
# Import database schema
sudo mysql servermanager < /opt/serveros/init.sql

# Create admin user
sudo /opt/serveros/tools/reset-admin.php
```

### Issue: Can't access web interface

**Cause:** Apache not configured or not running

**Solution:**
```bash
# Check Apache status
sudo systemctl status apache2

# Restart Apache
sudo systemctl restart apache2

# Check Apache logs
sudo tail -f /var/log/apache2/serveros_error.log
```

## Manual Database Operations

### Connect to database:
```bash
sudo mysql servermanager
```

### Check users:
```sql
SELECT id, username, email, role, is_active FROM users;
```

### Manually reset admin password:
```sql
UPDATE users
SET password = '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi'
WHERE username = 'admin';
```
(Password: admin123)

### Create admin user manually:
```sql
INSERT INTO users (username, email, password, full_name, role, is_active)
VALUES (
    'admin',
    'admin@localhost',
    '$2y$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi',
    'Administrator',
    'admin',
    1
);
```

## Reinstallation

If all else fails, reinstall:

```bash
# Uninstall
cd /tmp
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/uninstall.sh | sudo bash

# Reinstall
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

## Getting More Help

- Check logs: `/var/log/apache2/serveros_error.log`
- Check application logs: `/opt/serveros/logs/`
- Report issues: https://github.com/chukfinley/tso/issues
- Read documentation: `/opt/serveros/` or GitHub repo

## Notes

- All tools require sudo/root access
- Tools work from any directory
- Safe to run multiple times
- Won't corrupt existing data
