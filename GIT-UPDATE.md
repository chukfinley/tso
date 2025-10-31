# Git Pull Update Guide

The **fastest** way to update TSO!

## Quick Update (Recommended)

If TSO is installed on your server, simply run:

```bash
cd /opt/serveros && sudo git pull && sudo ./post-update.sh
```

**Done!** Takes only seconds.

## What Happens

### `git pull`
- Downloads only changed files from GitHub
- Much faster than full clone
- Preserves your config.php (it's in .gitignore)

### `post-update.sh`
- Ensures config exists
- Fixes file permissions
- Creates required directories
- Ensures admin user exists
- Restarts Apache

## Why This Is Better

| Method | Time | Bandwidth | Config Safety |
|--------|------|-----------|---------------|
| **git pull** | ~5 seconds | Only changes | ✅ Preserved |
| Bootstrap script | ~2 minutes | Full download | ✅ Preserved |
| update.sh | ~1 minute | Full download | ✅ Preserved |

## Step-by-Step

### 1. SSH into your server

```bash
ssh root@your-server
```

### 2. Navigate to TSO directory

```bash
cd /opt/serveros
```

### 3. Pull latest changes

```bash
sudo git pull
```

### 4. Run post-update tasks

```bash
sudo ./post-update.sh
```

### 5. Done!

Access your server: `http://your-server-ip`

## Troubleshooting

### Error: "config/config.php has local changes"

This happens if you edited config.php. Git won't overwrite it (good!), but you need to tell git to keep your version:

```bash
cd /opt/serveros
sudo git checkout config/config.php  # Reset to example
sudo git pull                         # Get updates
sudo ./post-update.sh                 # Will create config from example
# Then edit config.php with your database credentials
```

Better solution - keep a backup:

```bash
cd /opt/serveros
sudo cp config/config.php ~/config-backup.php  # Backup
sudo git reset --hard                          # Force update
sudo git pull                                  # Get updates
sudo cp ~/config-backup.php config/config.php  # Restore
sudo ./post-update.sh                          # Apply updates
```

### Error: "permission denied"

Make sure you use sudo:

```bash
sudo git pull
sudo ./post-update.sh
```

### Check what changed

Before pulling, see what will change:

```bash
cd /opt/serveros
git fetch
git log HEAD..origin/master --oneline
git diff HEAD..origin/master
```

### Rollback if needed

If update breaks something:

```bash
cd /opt/serveros
git log --oneline -10           # Find previous commit
git reset --hard <commit-hash>  # Rollback
sudo ./post-update.sh           # Apply
```

## Automatic Updates

To automatically pull updates daily:

```bash
sudo crontab -e
```

Add:

```bash
# Update TSO daily at 3 AM
0 3 * * * cd /opt/serveros && git pull && ./post-update.sh >> /var/log/serveros-auto-update.log 2>&1
```

**Warning:** Automatic updates can break things! Only use if:
- You monitor logs regularly
- You have backups
- You can quickly rollback

## Alternative Update Methods

If git pull doesn't work, use these:

### Method 1: Bootstrap Script
```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

### Method 2: Update Script
```bash
sudo /opt/serveros/update.sh
```

### Method 3: Manual
```bash
cd /tmp
git clone https://github.com/chukfinley/tso.git
cd tso
sudo ./install.sh
# Will detect existing installation and offer update
```

## Check Update Status

After updating:

```bash
# Check git status
cd /opt/serveros && git status

# Check current version (commit)
git log -1 --oneline

# Check if Apache is running
systemctl status apache2

# Test database connection
sudo /opt/serveros/tools/check-db.php

# Check for errors
sudo tail -50 /var/log/apache2/serveros_error.log
```

## Config File Management

The `config/config.php` file is **ignored by git** to prevent overwriting your database credentials.

### First Installation

On first install, `config.example.php` is copied to `config.php` and configured automatically.

### After Updates

`config.php` is never touched by git. Your database credentials are safe!

### If You Need to Reset Config

```bash
cd /opt/serveros
sudo cp config/config.example.php config/config.php
# Edit config.php with your database credentials
sudo nano config/config.php
```

## Best Practices

1. **Backup config before major updates:**
   ```bash
   sudo cp /opt/serveros/config/config.php ~/config-backup.php
   ```

2. **Check changelog before updating:**
   ```bash
   cd /opt/serveros
   git fetch
   git log HEAD..origin/master
   ```

3. **Test in staging first** (if you have one)

4. **Update during low traffic** hours

5. **Monitor logs after updating:**
   ```bash
   sudo tail -f /var/log/apache2/serveros_error.log
   ```

## Summary

**Update in 3 seconds:**

```bash
cd /opt/serveros && sudo git pull && sudo ./post-update.sh
```

**That's it!** Your TSO is now up to date.

---

Need help? Check [TROUBLESHOOTING.md](TROUBLESHOOTING.md)
