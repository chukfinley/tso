# Automatic ISO Share Setup

## Overview

During installation, ServerOS automatically creates a **public ISO network share** that:
- ✅ Allows **guest access** (no login required)
- ✅ Provides **read/write access** for uploading ISO files
- ✅ Integrates with the **VM management page** for mounting ISOs
- ✅ Is accessible from any computer on your network

## Share Details

- **Share Name**: `iso`
- **Display Name**: ISO Images
- **Local Path**: `/opt/serveros/storage/isos`
- **Network Path**: `\\YOUR_SERVER_IP\iso` (Windows) or `smb://YOUR_SERVER_IP/iso` (Linux/Mac)
- **Permissions**: Read/Write for everyone
- **Guest Access**: Enabled (no authentication required)

## How to Access the Share

### From Windows:

1. Open File Explorer
2. In the address bar, type: `\\YOUR_SERVER_IP\iso`
3. Press Enter
4. You can now drag and drop ISO files directly

**Or map as network drive:**
```
Right-click "This PC" → Map network drive
Folder: \\YOUR_SERVER_IP\iso
☐ Reconnect at sign-in
☑ Connect using different credentials (leave unchecked for guest access)
```

### From Linux:

**Browse with file manager:**
```bash
# Nautilus (GNOME)
nautilus smb://YOUR_SERVER_IP/iso

# Or mount manually
sudo mount -t cifs //YOUR_SERVER_IP/iso /mnt/iso -o guest,uid=1000
```

### From macOS:

1. Open Finder
2. Press `Cmd+K` (Go → Connect to Server)
3. Enter: `smb://YOUR_SERVER_IP/iso`
4. Click "Connect"
5. Select "Guest" when prompted

## Using ISOs in Virtual Machines

1. Upload your ISO files to the network share
2. Go to the **VMs** page in ServerOS web UI
3. Click **"Create New VM"**
4. In the "ISO Image" dropdown, your uploaded ISOs will appear
5. Select the ISO and configure your VM
6. Start the VM - it will boot from the ISO

## Technical Details

### Samba Configuration

The share is configured with these settings:
```ini
[iso]
   comment = ISO images for virtual machines - Public upload
   path = /opt/serveros/storage/isos
   browseable = yes
   read only = no
   guest ok = yes
   create mask = 0664
   directory mask = 0775
```

### Security Considerations

**⚠️ Important Notes:**

1. **Guest Access Enabled**: Anyone on your network can upload/delete files
2. **No Authentication**: Files are accessible without login
3. **Network Only**: Share is not accessible from the internet (unless you configure port forwarding)

**To secure the share:**

If you want to require authentication:

1. Go to **Shares** page in ServerOS UI
2. Click "Edit" on the `iso` share
3. Disable "Guest Access"
4. Create share users and set permissions
5. Assign users to the share

### Directory Permissions

```bash
# Local directory permissions
Directory: /opt/serveros/storage/isos
Owner: root:root
Permissions: 0775
```

### Installation

The ISO share is created automatically during:
- Fresh installation
- Updates (if it doesn't already exist)

**Manual creation** (if needed):
```bash
sudo php /opt/serveros/tools/create-iso-share.php
```

## Troubleshooting

### Cannot connect to share

**Check Samba is running:**
```bash
sudo systemctl status smbd
sudo systemctl status nmbd
```

**Restart Samba:**
```bash
sudo systemctl restart smbd nmbd
```

### ISOs not appearing in VM dropdown

**Check directory permissions:**
```bash
ls -la /opt/serveros/storage/isos/
```

**Verify files are .iso extension:**
```bash
ls /opt/serveros/storage/isos/*.iso
```

### "Access Denied" error

**Check guest access is enabled:**
```bash
sudo testparm -s | grep -A 10 "\[iso\]"
```

Should show: `guest ok = yes`

**Check firewall allows Samba:**
```bash
sudo ufw status | grep -E "139|445"
```

If blocked, allow:
```bash
sudo ufw allow 139/tcp
sudo ufw allow 445/tcp
```

### Cannot upload files

**Check share is not read-only:**
```bash
sudo testparm -s | grep -A 10 "\[iso\]" | grep "read only"
```

Should show: `read only = no`

**Check directory is writable:**
```bash
sudo chmod 775 /opt/serveros/storage/isos
```

## Common ISO Sources

Popular Linux distributions you might want to use:

- **Ubuntu Server**: https://ubuntu.com/download/server
- **Debian**: https://www.debian.org/distrib/
- **Alpine Linux**: https://alpinelinux.org/downloads/
- **CentOS Stream**: https://www.centos.org/download/
- **Rocky Linux**: https://rockylinux.org/download
- **Windows Server**: https://www.microsoft.com/en-us/evalcenter/

## Advanced Usage

### Creating Additional ISO Shares

You can create additional ISO shares for organization:

1. Go to **Shares** page
2. Click **"Create Share"**
3. Configure:
   - Share Name: `windows-iso` or `linux-iso`
   - Path: `/opt/serveros/storage/isos/windows`
   - Guest OK: As desired
4. Update VM.php to scan multiple directories if needed

### Automatic ISO Cleanup

Create a cron job to clean old ISOs:

```bash
# Delete ISOs older than 90 days
sudo crontab -e

# Add line:
0 3 * * 0 find /opt/serveros/storage/isos -name "*.iso" -mtime +90 -delete
```

## Related Documentation

- See `MIGRATION.md` for database setup
- See `SHARES-README.txt` for full Samba configuration
- See VM management docs for creating virtual machines
