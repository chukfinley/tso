# Network Shares Feature Documentation

## Overview

The Network Shares feature provides comprehensive SMB/Samba share management through an intuitive web interface. This allows you to create, manage, and monitor file shares accessible from Windows, macOS, and Linux clients.

## Key Features

### 1. Share Management
- **Create/Edit/Delete Shares**: Full CRUD operations for network shares
- **Directory Selection**: Choose any directory on your server to share
- **Share Settings**:
  - Browseable (visible in network browser)
  - Read-only mode
  - Guest access (no authentication required)
  - Custom display names and comments

### 2. User Management
- **Dedicated Share Users**: Create users specifically for share access
- **Password Management**: Set and change passwords securely
- **User Status**: Enable/disable users without deletion
- **Samba Integration**: Automatically syncs with Samba user database

### 3. Permission System
- **Granular Permissions**: Per-user, per-share access control
- **Three Permission Levels**:
  - **Read**: View and download files only
  - **Write**: Create, modify, and delete files
  - **Admin**: Full control including permission management
- **Easy Management**: Visual interface for setting permissions

### 4. Advanced Configuration
- **Case Sensitivity**: Configure filename case handling
  - Auto: Automatic detection based on filesystem
  - Yes: Case-sensitive (file.txt ≠ File.txt)
  - No: Case-insensitive (file.txt = File.txt)
- **File Masks**: Set Unix permissions for new files and directories
- **Force User/Group**: Override file ownership
- **Preserve Case**: Control case preservation in filenames

### 5. Monitoring & Logging
- **Samba Status**: Real-time service status monitoring
- **Connected Clients**: View currently connected users
- **Activity Logs**: Track all share operations
- **Configuration Testing**: Validate Samba configuration

## User Interface

### Main Sections

#### Shares Tab
Lists all configured shares with:
- Share name and display name
- Directory path
- Status (Active/Inactive)
- Settings overview (browseable, read-only, guest access)
- Quick actions (Edit, Permissions, Enable/Disable, Delete)

#### Users Tab
Manage share users:
- Username and full name
- Account status
- Actions: Change password, View permissions, Enable/Disable, Delete

#### Activity Logs Tab
View recent operations:
- Share creation/modification/deletion
- Permission changes
- User actions
- Timestamps and IP addresses

### Creating a Share

1. Click "Create New Share" button
2. Fill in Basic Settings:
   - Share Name (internal identifier)
   - Display Name (shown to users)
   - Directory Path (where files are stored)
   - Comment/Description
   - Options (browseable, read-only, guest access)

3. Configure Advanced Settings (optional):
   - Case sensitivity options
   - File creation masks
   - Directory masks
   - Force user/group

4. Save the share

5. Set Permissions:
   - Click "Permissions" button on the share
   - Add users and assign permission levels
   - Save permissions

### Creating a User

1. Go to Users tab
2. Click "Create New User"
3. Enter:
   - Username (alphanumeric, hyphens, underscores)
   - Full Name (optional)
   - Password (minimum 4 characters)
   - Confirm Password
4. Save

### Managing Permissions

#### From Share View:
1. Find the share
2. Click "Permissions" button
3. Click "Add User"
4. Select user and permission level
5. Update or remove as needed

#### From User View:
1. Find the user
2. Click "View Permissions"
3. See all shares the user has access to

## Technical Details

### Database Schema

#### `shares` Table
Stores share configuration:
- Basic settings (name, path, comment)
- Options (browseable, readonly, guest_ok)
- Case sensitivity settings
- File masks
- User/group overrides
- Status and metadata

#### `share_users` Table
Stores share user information:
- Username and full name
- Password hash (reference only)
- Active status
- Creation metadata

#### `share_permissions` Table
Maps users to shares with permission levels:
- Share ID and User ID
- Permission level (read/write/admin)
- Metadata

#### `share_access_log` Table
Tracks all operations:
- Share ID and username
- Action performed
- IP address and details
- Timestamp

### API Endpoints

`/api/share-control.php` provides:

**Share Operations:**
- `list` - Get all shares
- `get` - Get share by ID
- `create` - Create new share
- `update` - Update share
- `delete` - Delete share
- `toggle` - Enable/disable share

**User Operations:**
- `list_users` - Get all users
- `get_user` - Get user by ID
- `create_user` - Create new user
- `update_user_password` - Change password
- `delete_user` - Delete user
- `toggle_user` - Enable/disable user

**Permission Operations:**
- `get_permissions` - Get share permissions
- `get_user_permissions` - Get user's permissions
- `set_permission` - Set user permission on share
- `remove_permission` - Remove permission

**System Operations:**
- `status` - Get Samba status
- `test_config` - Test configuration
- `connected_clients` - List connected clients
- `list_directories` - List available directories
- `get_logs` - Get activity logs

### Samba Integration

The `Share` class (`src/Share.php`) handles:

1. **Configuration Management**:
   - Generates `/etc/samba/smb.conf`
   - Preserves global settings
   - Adds share-specific configurations

2. **User Management**:
   - Creates system users
   - Manages Samba passwords via `smbpasswd`
   - Enables/disables users

3. **Service Control**:
   - Reloads Samba after changes
   - Tests configuration validity
   - Monitors service status

4. **Permission Sync**:
   - Updates share user lists
   - Configures read/write lists
   - Sets admin users

### Security

#### Sudo Configuration
The web server requires specific sudo privileges:
- `smbpasswd` - Manage Samba passwords
- `useradd`/`userdel` - System user management
- `systemctl` - Service control
- `testparm` - Configuration testing
- `smbstatus` - Status monitoring

Configured in `/etc/sudoers.d/serveros-samba` (installed via setup script)

#### Password Handling
- Passwords are hashed in the database (reference only)
- Actual authentication handled by Samba
- Minimum password length enforced
- Password changes require confirmation

#### Access Control
- All API endpoints require authentication
- Session validation on every request
- SQL injection protection via prepared statements
- Input validation and sanitization

## Client Access

### Windows
```
Network Path: \\server-ip\share-name
Example: \\192.168.1.100\documents

In File Explorer:
1. Type address in location bar
2. Enter username and password
3. Optionally save credentials
```

### macOS
```
In Finder:
1. Press Cmd + K
2. Enter: smb://server-ip/share-name
3. Connect and authenticate

Example: smb://192.168.1.100/documents
```

### Linux
```
# Mount command
sudo mount -t cifs //server-ip/share-name /mnt/point -o username=user

# File manager (Nautilus, Dolphin, etc.)
smb://server-ip/share-name

# Command line access
smbclient //server-ip/share-name -U username
```

## Performance Considerations

### Network Settings
Default configuration includes:
- TCP_NODELAY for reduced latency
- Increased socket buffers (128KB)
- Raw read/write enabled

### File Operations
- Adjust create_mask and directory_mask based on needs
- Use force_user/force_group to simplify permissions
- Consider read-only shares for frequently accessed data

### Monitoring
- Check connected clients regularly
- Review activity logs for unusual patterns
- Monitor disk space on share directories

## Troubleshooting

### Common Issues

1. **"Samba Status: Stopped"**
   - Service not running
   - Run setup script or start manually
   - Check logs: `journalctl -u smbd`

2. **"Permission Denied" when accessing share**
   - Verify user has permission on share
   - Check directory permissions on server
   - Ensure share is active
   - Verify correct password

3. **Configuration test failed**
   - Syntax error in configuration
   - Check `/etc/samba/smb.conf`
   - Review error output from testparm

4. **Can't create/modify users**
   - Sudo permissions not configured
   - Run setup script again
   - Check `/etc/sudoers.d/serveros-samba`

5. **Share not visible on network**
   - Check "Browseable" option is enabled
   - Verify Samba is running
   - Check firewall rules (ports 445, 139)

### Logs

**Samba Logs:**
- `/var/log/samba/log.smbd`
- `/var/log/samba/log.nmbd`

**Application Logs:**
- Activity Logs tab in web interface
- Database: `share_access_log` table

**System Logs:**
```bash
# Samba service logs
sudo journalctl -u smbd -n 100

# Follow live logs
sudo journalctl -u smbd -f
```

## Best Practices

1. **Security**:
   - Use strong passwords
   - Disable guest access unless necessary
   - Regularly review permissions
   - Keep Samba updated

2. **Organization**:
   - Use descriptive share and user names
   - Document share purposes in comments
   - Group related shares logically
   - Regular permission audits

3. **Performance**:
   - Don't over-share (only share what's needed)
   - Monitor disk space
   - Use read-only for static content
   - Consider separate shares for different purposes

4. **Maintenance**:
   - Regular backups of share data
   - Test configuration after changes
   - Monitor activity logs
   - Keep user list current

## Integration

### With TSO Features
- **User Management**: Separate from system users
- **Activity Logs**: Integrated with TSO logging
- **Dashboard**: Status visible from main dashboard (potential)
- **Backups**: Can integrate with backup system (potential)

### External Tools
- Standard SMB/CIFS protocol
- Compatible with all SMB clients
- Can be monitored via standard tools
- Supports Windows, macOS, Linux, BSD, etc.

## Future Enhancements

Potential future additions:
- NFS share support
- Quota management
- Automatic backup integration
- Enhanced activity monitoring
- User groups
- Share templates
- Bulk operations
- Advanced ACLs
- Integration with LDAP/AD

## Quick Reference

### Common Tasks

**Create a read-only share:**
1. Create share
2. Check "Read Only" option
3. Add users with any permission level

**Create a team workspace:**
1. Create share (read-write)
2. Add team members
3. Set permissions: Write for editors, Read for viewers

**Enable guest share:**
1. Create share
2. Check "Allow Guest Access"
3. Optionally set as read-only

**Change user password:**
1. Go to Users tab
2. Click "Change Password" on user
3. Enter new password twice

**Disable share temporarily:**
1. Find share
2. Click "Disable" button
3. Re-enable when needed

## Support

For questions or issues:
1. Check this documentation
2. Review SHARES-SETUP.md
3. Check Activity Logs
4. Test configuration
5. Review Samba logs
6. Consult TSO documentation

## Credits

This feature integrates with:
- Samba (SMB/CIFS server)
- Bootstrap (UI framework)
- jQuery (JavaScript library)
- MariaDB/MySQL (Database)

---

For installation instructions, see `SHARES-SETUP.md`

