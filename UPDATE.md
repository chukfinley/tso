# ServerOS Update Guide

Keep your ServerOS installation up-to-date with the latest features and fixes!

## Quick Update (Recommended)

The easiest way to update:

```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

**That's it!** The script will:
1. Detect your existing installation
2. Ask for confirmation
3. Backup your configuration
4. Update all application files
5. Restore your configuration
6. Restart services

## Alternative Update Methods

### Method 1: Dedicated Update Script

```bash
sudo /opt/serveros/update.sh
```

Non-interactive, automatic update. Perfect for automation!

### Method 2: Git Clone and Install

```bash
git clone https://github.com/chukfinley/tso.git
cd tso
sudo ./install.sh
```

The installer will detect your existing installation and offer to update.

### Method 3: Manual Update

```bash
# Clone latest version
git clone https://github.com/chukfinley/tso.git /tmp/serveros-latest
cd /tmp/serveros-latest

# Backup config
sudo cp /opt/serveros/config/config.php /tmp/config-backup.php

# Update files
sudo cp -r public /opt/serveros/
sudo cp -r src /opt/serveros/
sudo cp -r views /opt/serveros/
sudo cp -r tools /opt/serveros/

# Restore config
sudo cp /tmp/config-backup.php /opt/serveros/config/config.php

# Fix permissions
sudo chown -R www-data:www-data /opt/serveros
sudo chmod -R 755 /opt/serveros

# Restart Apache
sudo systemctl restart apache2

# Cleanup
rm -rf /tmp/serveros-latest
rm /tmp/config-backup.php
```

## What Gets Updated

### ✅ Updated Files:
- All PHP files in `public/` (pages)
- All PHP files in `src/` (backend classes)
- All template files in `views/`
- All CSS files in `public/css/`
- All JavaScript files in `public/js/`
- All tools in `tools/`
- Database schema file `init.sql` (reference only)

### ✅ Preserved Data:
- **Configuration** (`config/config.php`)
  - Database credentials
  - Custom settings
- **Database**
  - All tables and data
  - User accounts and passwords
  - Activity logs
- **Storage**
  - Logs in `logs/`
  - Uploaded files in `storage/`
- **Apache Configuration**
  - Virtual host settings
  - Custom Apache configs

## Update Frequency

We recommend updating:
- **Security fixes:** Immediately
- **Bug fixes:** Within a week
- **New features:** At your convenience

Check for updates:
```bash
# On GitHub
https://github.com/chukfinley/tso/commits/master

# Or watch the repository for notifications
```

## Rollback

If an update causes issues:

### Quick Rollback:

```bash
# Database is unchanged, so just reinstall previous version
git clone https://github.com/chukfinley/tso.git
cd tso
git checkout <previous-commit-hash>
sudo ./install.sh
```

### Full Reinstall:

```bash
# Uninstall
sudo /opt/serveros/uninstall.sh

# Reinstall
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

Note: Uninstall removes the database! Only use if necessary.

## Testing After Update

After updating, verify:

1. **Login works:**
   - Go to `http://your-server-ip`
   - Login with your credentials
   - Should work without issues

2. **Check dashboard:**
   - System info displays correctly
   - No PHP errors

3. **Test user management (if admin):**
   - View users list
   - Create test user
   - Delete test user

4. **Check logs:**
   ```bash
   sudo tail -50 /var/log/apache2/serveros_error.log
   ```
   Should show no errors

5. **Verify tools work:**
   ```bash
   sudo /opt/serveros/tools/check-db.php
   ```

## Troubleshooting Updates

### Update script not found

```bash
# Download latest update script
cd /tmp
curl -O https://raw.githubusercontent.com/chukfinley/tso/master/update.sh
chmod +x update.sh
sudo ./update.sh
```

### Permission errors after update

```bash
sudo chown -R www-data:www-data /opt/serveros
sudo chmod -R 755 /opt/serveros
sudo chmod -R 775 /opt/serveros/logs
sudo chmod -R 775 /opt/serveros/storage
sudo systemctl restart apache2
```

### Can't login after update

```bash
# Reset admin password
sudo /opt/serveros/tools/reset-admin.php
```

### Database errors after update

```bash
# Check database connection
sudo /opt/serveros/tools/check-db.php

# If needed, check credentials
sudo cat /opt/serveros/config/config.php | grep DB_
```

### Apache won't start

```bash
# Check Apache configuration
sudo apache2ctl configtest

# Check error logs
sudo tail -100 /var/log/apache2/error.log

# Restart Apache
sudo systemctl restart apache2
```

## Automated Updates

To automatically update ServerOS (use with caution):

```bash
# Create cron job (runs weekly on Sunday at 3 AM)
sudo crontab -e

# Add this line:
0 3 * * 0 /opt/serveros/update.sh >> /var/log/serveros-update.log 2>&1
```

**Warning:** Automated updates can break your system if new version has issues. Only use if you:
- Have backups
- Monitor logs regularly
- Can quickly rollback if needed

## Update Checklist

Before updating:
- [ ] Check changelog for breaking changes
- [ ] Backup database (optional, but recommended):
  ```bash
  sudo mysqldump servermanager > ~/serveros-backup-$(date +%Y%m%d).sql
  ```
- [ ] Note current version/commit
- [ ] Ensure you can rollback if needed

During update:
- [ ] Run update command
- [ ] Watch for errors
- [ ] Verify completion message

After update:
- [ ] Test login
- [ ] Check dashboard
- [ ] Verify no errors in logs
- [ ] Test critical functions

## Getting Help

If update fails or causes issues:

1. **Check logs:**
   ```bash
   sudo tail -100 /var/log/apache2/serveros_error.log
   ```

2. **Run diagnostics:**
   ```bash
   sudo /opt/serveros/tools/check-db.php
   ```

3. **Check GitHub issues:**
   https://github.com/chukfinley/tso/issues

4. **Report the problem:**
   - Include error messages
   - Include steps to reproduce
   - Include system info

## Best Practices

1. **Test first:** If possible, test update on a clone/staging server
2. **Backup:** Keep database backups before major updates
3. **Off-hours:** Update during low-traffic times
4. **Monitor:** Watch logs after updating
5. **Document:** Note when you updated and what version

---

**Keep ServerOS updated for the latest features and security fixes!**

Update now:
```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```
