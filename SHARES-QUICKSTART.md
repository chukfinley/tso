# Network Shares - Quick Start Guide

Get your network shares up and running in 5 minutes!

## Quick Installation

### Option 1: Automated Setup (Recommended)
```bash
cd /home/user/git/tso
sudo bash scripts/setup-shares.sh
```

The script will:
- Install Samba
- Configure permissions
- Set up database
- Start services
- Configure firewall

### Option 2: Manual Setup
```bash
# 1. Install Samba
sudo apt install -y samba samba-common-bin

# 2. Setup permissions
sudo cp config/samba-sudoers /etc/sudoers.d/serveros-samba
sudo chmod 440 /etc/sudoers.d/serveros-samba

# 3. Create directories
sudo mkdir -p /srv/samba
sudo chmod 755 /srv/samba

# 4. Update database
mysql -u root -p servermanager < init.sql

# 5. Start Samba
sudo systemctl enable smbd
sudo systemctl start smbd
```

## First Steps

### 1. Access the Interface
Open your browser and navigate to:
```
http://your-server/shares.php
```

### 2. Create Your First User
- Go to **Users** tab
- Click **Create New User**
- Enter username: `testuser`
- Enter password: `Password123`
- Click **Save User**

### 3. Create Your First Share
- Go to **Shares** tab
- Click **Create New Share**
- Enter share name: `documents`
- Enter path: `/srv/samba/documents`
- Check **Browseable**
- Uncheck **Guest OK** (for security)
- Click **Save Share**

### 4. Set Permissions
- Find your new share
- Click **Permissions** button
- Click **Add User**
- Select `testuser`
- Choose permission: **Write**
- Click **Add**

### 5. Connect to Your Share

**From Windows:**
```
\\your-server-ip\documents
```

**From macOS:**
```
smb://your-server-ip/documents
```

**From Linux:**
```
smb://your-server-ip/documents
```

Enter username: `testuser` and password: `Password123`

## Common Tasks

### Create a Public Share (Guest Access)
1. Create share as above
2. Check **Allow Guest Access**
3. Optionally check **Read Only** for safety
4. No permissions needed

### Create a Read-Only Share
1. Create share
2. Check **Read Only** option
3. Add users with any permission level
4. All users will have read-only access

### Change User Password
1. Go to **Users** tab
2. Find the user
3. Click **Change Password**
4. Enter new password twice
5. Click **Save**

### Disable Share Temporarily
1. Find the share
2. Click **Disable** button
3. Share will be hidden from network
4. Click **Enable** to restore

## Troubleshooting

### Can't see share on network
- Check share is **Active** (not disabled)
- Check **Browseable** is enabled
- Verify Samba is running (check status bar)
- Check firewall allows ports 445, 139

### Can't connect to share
- Verify username and password
- Check user has permission on share
- Ensure user is **Active** (not disabled)
- Try from Users tab > View Permissions

### Permission denied
- Check user has Write permission (not just Read)
- Verify directory exists on server
- Check directory permissions: `ls -la /srv/samba/`

### Samba not running
- Click **Test Config** button
- Check for errors
- Restart: `sudo systemctl restart smbd`
- View logs: `sudo journalctl -u smbd -n 50`

## Files Created

All share-related files are located in:
```
/home/user/git/tso/
├── config/samba-sudoers           # Sudo permissions
├── public/
│   ├── api/share-control.php      # API
│   ├── js/shares.js               # JavaScript
│   └── shares.php                 # Main page
├── scripts/setup-shares.sh        # Setup script
├── src/Share.php                  # Backend logic
├── SHARES-SETUP.md                # Detailed setup guide
├── SHARES-FEATURES.md             # Feature documentation
└── SHARES-QUICKSTART.md           # This file
```

## Database Tables

Four new tables added to `servermanager` database:
- `shares` - Share configurations
- `share_users` - Share user accounts
- `share_permissions` - Access control
- `share_access_log` - Activity logs

## Network Ports

Samba uses:
- TCP 445 (SMB)
- TCP 139 (NetBIOS)
- UDP 137-138 (NetBIOS)

Make sure these are open in your firewall.

## Next Steps

1. **Read the documentation:**
   - `SHARES-SETUP.md` - Complete setup guide
   - `SHARES-FEATURES.md` - All features explained

2. **Create more shares:**
   - Team workspaces
   - Department shares
   - Project folders

3. **Organize users:**
   - Create users by department
   - Set appropriate permissions
   - Document share purposes

4. **Monitor activity:**
   - Check **Activity Logs** tab
   - Review connected clients
   - Monitor disk usage

## Security Tips

1. **Use strong passwords** (minimum 8 characters)
2. **Disable guest access** unless necessary
3. **Use read-only** for shared resources
4. **Review permissions** regularly
5. **Check activity logs** for unusual access

## Support

Need help?
1. Check the **Activity Logs** tab
2. Click **Test Config** button
3. Read `SHARES-SETUP.md`
4. Review Samba logs: `/var/log/samba/`

## Features Overview

✅ Create unlimited shares
✅ User management with passwords
✅ Granular permissions (Read/Write/Admin)
✅ Guest access option
✅ Read-only shares
✅ Case sensitivity control
✅ Real-time status monitoring
✅ Activity logging
✅ Connected client viewer
✅ Configuration testing

## Example: Team Workspace

Create a team workspace in 3 minutes:

```bash
# 1. Create users
- john (Write permission)
- jane (Write permission)
- bob (Read permission)

# 2. Create share
- Name: team_workspace
- Path: /srv/samba/team_workspace
- Browseable: Yes
- Guest: No

# 3. Set permissions
- john: Write
- jane: Write
- bob: Read

# 4. Connect
Windows: \\server\team_workspace
macOS: smb://server/team_workspace
Linux: smb://server/team_workspace
```

Done! Your team can now collaborate on files.

---

For more details, see `SHARES-SETUP.md` and `SHARES-FEATURES.md`

