# ServerOS Uninstallation Guide

How to safely remove ServerOS from your system.

## Quick Uninstall

### Complete Removal (removes everything)

```bash
sudo /opt/serveros/uninstall.sh
```

Follow the prompts to:
1. Choose what to remove (app files, database, or both)
2. Confirm the action

### Keep Database (remove app only)

```bash
sudo /opt/serveros/uninstall.sh --keep-db
```

Good for reinstalling later with same data.

### Force Mode (no prompts)

```bash
sudo /opt/serveros/uninstall.sh --force
```

**Warning:** Removes everything without asking!

## What Gets Removed

### Always Removed:
- ✓ Application files from `/opt/serveros`
- ✓ Apache configuration
- ✓ Credentials file

### Optionally Removed:
- Database and all data (optional)
- MariaDB/Apache/PHP packages (manual)

### Never Removed (automatic backups):
- Database backup (saved to `/root/serveros_db_backup_*.sql`)
- Credentials backup (saved to `/root/serveros_credentials.txt.bak`)

## Uninstall Options

### Interactive Mode (Default)

```bash
sudo /opt/serveros/uninstall.sh
```

**Prompts:**
```
What would you like to do?
  1) Remove everything (app files + database)
  2) Remove app files only (keep database)
  3) Cancel
```

### Command Line Options

```bash
# Remove everything without prompts
sudo /opt/serveros/uninstall.sh --force

# Remove app, keep database
sudo /opt/serveros/uninstall.sh --keep-db

# Show help
sudo /opt/serveros/uninstall.sh --help
```

## Step-by-Step Uninstallation

### 1. Backup Important Data (Optional)

```bash
# Backup database
sudo mysqldump servermanager > ~/serveros-backup.sql

# Backup config
sudo cp /opt/serveros/config/config.php ~/config-backup.php
```

### 2. Run Uninstaller

```bash
sudo /opt/serveros/uninstall.sh
```

### 3. Choose What to Remove

```
What would you like to do?
  1) Remove everything (app files + database)
  2) Remove app files only (keep database)
  3) Cancel

Enter your choice (1-3): 1
```

### 4. Confirm

```
Are you absolutely sure? Type 'yes' to confirm: yes
```

### 5. Uninstallation Process

The script will:
- ✓ Disable Apache site
- ✓ Remove Apache configuration
- ✓ Backup database (automatic)
- ✓ Remove database (if selected)
- ✓ Remove application files
- ✓ Restart Apache

### 6. Cleanup Complete

```
╔════════════════════════════════════════════════════════════════╗
║              Uninstallation Completed!                         ║
╚════════════════════════════════════════════════════════════════╝

✓ ServerOS has been completely removed from your system.

ℹ Database backup: /root/serveros_db_backup_20241031_120000.sql
ℹ Credentials backup: /root/serveros_credentials.txt.bak
```

## Reinstalling After Uninstall

### With Fresh Database

```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

### With Existing Database

If you kept the database (option 2) or have a backup:

```bash
# Restore database if you have a backup
sudo mysql servermanager < ~/serveros-backup.sql

# Reinstall
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

The installer will detect the existing database and use it!

## Removing System Packages

ServerOS uninstaller does NOT remove Apache, MariaDB, or PHP.

To remove them manually:

```bash
# Remove packages
sudo apt remove --purge apache2 mariadb-server php php-*

# Clean up
sudo apt autoremove
sudo apt autoclean

# Remove configuration files
sudo rm -rf /etc/apache2
sudo rm -rf /etc/mysql
sudo rm -rf /var/lib/mysql  # ⚠️ Removes ALL databases!
```

**Warning:** This removes ALL MySQL/MariaDB databases, not just ServerOS!

## Partial Uninstall

### Remove App Only (Keep Everything Else)

```bash
sudo rm -rf /opt/serveros
```

Apache, database, and packages remain.

### Remove Apache Config Only

```bash
sudo a2dissite serveros
sudo rm /etc/apache2/sites-available/serveros.conf
sudo systemctl restart apache2
```

### Remove Database Only

```bash
sudo mysql -e "DROP DATABASE servermanager;"
sudo mysql -e "DROP USER 'serveros'@'localhost';"
```

## Troubleshooting

### Uninstall Script Not Found

Download it:

```bash
cd /tmp
curl -O https://raw.githubusercontent.com/chukfinley/tso/master/uninstall.sh
chmod +x uninstall.sh
sudo ./uninstall.sh
```

### Permission Denied

Make sure you use `sudo`:

```bash
sudo ./uninstall.sh
```

### Database Won't Delete

Stop MariaDB first:

```bash
sudo systemctl stop mariadb
sudo mysql -e "DROP DATABASE servermanager;"
sudo mysql -e "DROP USER 'serveros'@'localhost';"
sudo systemctl start mariadb
```

### Apache Won't Restart

Check the error:

```bash
sudo apache2ctl configtest
sudo systemctl status apache2
```

If other sites exist, Apache should restart fine. If not:

```bash
sudo a2ensite 000-default
sudo systemctl restart apache2
```

## Verify Uninstallation

Check that ServerOS is completely removed:

```bash
# Check files
ls -la /opt/serveros
# Should show: cannot access '/opt/serveros': No such file or directory

# Check Apache config
ls -la /etc/apache2/sites-available/serveros.conf
# Should show: No such file or directory

# Check database
sudo mysql -e "SHOW DATABASES LIKE 'servermanager';"
# Should show: Empty set (if database was removed)

# Check processes
ps aux | grep serveros
# Should show: nothing (except your grep command)
```

## Backup Files Location

After uninstall, backups are saved to:

- **Database:** `/root/serveros_db_backup_YYYYMMDD_HHMMSS.sql`
- **Credentials:** `/root/serveros_credentials.txt.bak`

View backups:

```bash
ls -lh /root/serveros*
```

## Clean Reinstall

For a completely fresh start:

### 1. Full Uninstall

```bash
sudo /opt/serveros/uninstall.sh --force
```

### 2. Remove Packages

```bash
sudo apt remove --purge apache2 mariadb-server php php-*
sudo apt autoremove
```

### 3. Remove All Data

```bash
sudo rm -rf /var/lib/mysql
sudo rm -rf /etc/apache2
sudo rm -rf /etc/mysql
sudo rm -rf /root/serveros*
```

### 4. Reinstall

```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

## FAQ

### Will uninstall delete my data?

If you choose option 1 (remove everything), yes. But it creates a backup first.

Choose option 2 (keep database) to preserve data.

### Can I reinstall after uninstalling?

Yes! Just run the installer again:

```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

### Where are my backups?

- Database: `/root/serveros_db_backup_*.sql`
- Credentials: `/root/serveros_credentials.txt.bak`

### Will Apache/MySQL be removed?

No, only ServerOS files. You need to manually remove packages if desired.

### Can I cancel during uninstall?

Yes, press Ctrl+C at any time before confirming.

---

**Need help?** See [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
