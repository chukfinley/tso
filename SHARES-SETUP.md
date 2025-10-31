# Network Shares Setup Guide

This document explains how to set up and use the Network Shares feature in TSO (The Server OS).

## Features

The Network Shares module provides full SMB/Samba share management with:

- **Share Management**: Create, edit, delete, and configure network shares
- **User Management**: Create dedicated share users with passwords
- **Permission System**: Granular per-share, per-user permissions (read, write, admin)
- **Guest Access**: Optional guest access for shares
- **Case Sensitivity**: Configure case sensitivity settings (auto, yes, no)
- **Advanced Options**: File masks, force user/group, and more
- **Activity Logging**: Track all share operations
- **Real-time Status**: Monitor Samba service and connected clients

## Prerequisites

Before using the shares feature, you need to install and configure Samba:

### 1. Install Samba

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install -y samba samba-common-bin

# CentOS/RHEL
sudo yum install -y samba samba-client
```

### 2. Set up Database Tables

Run the database migration to create the necessary tables:

```bash
mysql -u root -p servermanager < init.sql
```

Or manually run only the shares-related tables from the `init.sql` file.

### 3. Configure Sudoers

The web server needs permission to manage Samba without password prompts:

```bash
sudo cp config/samba-sudoers /etc/sudoers.d/serveros-samba
sudo chmod 440 /etc/sudoers.d/serveros-samba
```

**Important**: Edit the file if your web server runs as a different user (default is `www-data`).

### 4. Create Share Base Directory

```bash
sudo mkdir -p /srv/samba
sudo chmod 755 /srv/samba
```

### 5. Start Samba Service

```bash
sudo systemctl enable smbd
sudo systemctl start smbd

# Verify it's running
sudo systemctl status smbd
```

## Usage

### Creating a Share

1. Navigate to **Shares** in the sidebar
2. Click **Create New Share**
3. Fill in the basic settings:
   - **Share Name**: Internal name (e.g., `documents`)
   - **Display Name**: Friendly name shown to users
   - **Directory Path**: Path to the directory to share
   - **Comment**: Optional description
   - **Options**: Browseable, Read-only, Guest access

4. Configure permissions (after creating the share)
5. Save the share

### Creating Share Users

1. Go to the **Users** tab
2. Click **Create New User**
3. Enter:
   - **Username**: User login name
   - **Full Name**: Optional display name
   - **Password**: User's password (minimum 4 characters)
4. Save the user

### Setting Permissions

1. Find the share in the **Shares** tab
2. Click **Permissions** button
3. Click **Add User**
4. Select the user and permission level:
   - **Read Only**: Can view and download files
   - **Read/Write**: Can create, modify, and delete files
   - **Admin**: Full control including permissions
5. Save

### Advanced Settings

In the share modal, go to the **Advanced** tab for:

- **Case Sensitivity**: 
  - `auto`: Automatic detection
  - `yes`: Case-sensitive filenames
  - `no`: Case-insensitive filenames
  
- **File Masks**: Unix permissions for new files and directories
  - Create Mask: Default `0664` (rw-rw-r--)
  - Directory Mask: Default `0775` (rwxrwxr-x)

- **Force User/Group**: Force all operations as a specific user/group

## Connecting to Shares

### Windows

1. Open File Explorer
2. In the address bar, type: `\\<server-ip>\<share-name>`
3. Enter username and password when prompted

Example: `\\192.168.1.100\documents`

### macOS

1. Open Finder
2. Press `Cmd + K`
3. Enter: `smb://<server-ip>/<share-name>`
4. Connect and enter credentials

### Linux

```bash
# Mount temporarily
sudo mount -t cifs //<server-ip>/<share-name> /mnt/share -o username=<user>

# Or use file manager
# In Nautilus/Files: smb://<server-ip>/<share-name>
```

## Troubleshooting

### Samba Service Not Running

Check the status:
```bash
sudo systemctl status smbd
```

View logs:
```bash
sudo journalctl -u smbd -n 50
```

Restart if needed:
```bash
sudo systemctl restart smbd
```

### Configuration Test Failed

Test the Samba configuration manually:
```bash
sudo testparm -s
```

View the configuration file:
```bash
sudo cat /etc/samba/smb.conf
```

### Permission Denied Errors

1. Check directory permissions:
```bash
ls -la /srv/samba/<share-dir>
```

2. Ensure the directory exists and is accessible
3. Check file masks in advanced settings
4. Verify user has write permissions if needed

### User Can't Connect

1. Verify user is active (Users tab)
2. Check user has permission on the share (Permissions button)
3. Test password by resetting it (Change Password button)
4. Verify Samba user exists:
```bash
sudo pdbedit -L
```

### Sudoers Issues

If you get permission denied errors in the logs:

1. Verify sudoers file is installed:
```bash
sudo cat /etc/sudoers.d/serveros-samba
```

2. Check file permissions (must be 440):
```bash
ls -l /etc/sudoers.d/serveros-samba
```

3. Test sudo commands manually:
```bash
sudo -u www-data sudo smbpasswd -h
```

## Security Considerations

1. **Passwords**: Use strong passwords for share users
2. **Guest Access**: Only enable for non-sensitive data
3. **Network**: Consider firewall rules to restrict SMB access
4. **Permissions**: Follow principle of least privilege
5. **Backups**: Regularly backup shared data

## Default Ports

- SMB: TCP 445
- NetBIOS: TCP 139, UDP 137-138

Make sure these ports are open if accessing from remote networks.

## Activity Logs

The **Activity Logs** tab shows:
- Share creation/modification/deletion
- Permission changes
- System operations

Use this for auditing and troubleshooting.

## Tips

1. **Test Configuration**: Use the "Test Config" button to verify Samba configuration
2. **Monitor Clients**: Use "Connected Clients" button to see who's connected
3. **Read-Only Shares**: Good for distributing files without modification risk
4. **Guest Shares**: Useful for public file drops, but less secure
5. **Case Sensitivity**: Set to `auto` for best compatibility with Windows clients

## Support

For issues or questions:
1. Check the Activity Logs tab
2. Test Samba configuration
3. Review system logs: `/var/log/samba/`
4. Check TSO logs: `/opt/serveros/logs/`

## See Also

- [Samba Documentation](https://www.samba.org/samba/docs/)
- [SMB Protocol Guide](https://wiki.samba.org/index.php/User_Documentation)

