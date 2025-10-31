# TSO 🚀

A Debian-based server management system inspired by Unraid and TrueNAS — built in pure PHP.

**One-line Installation:**
```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Platform](https://img.shields.io/badge/platform-Debian%20%7C%20Ubuntu-orange.svg)](https://debian.org)
[![PHP](https://img.shields.io/badge/PHP-7.4%2B-777BB4.svg)](https://php.net)

## Features

- **Secure Login System** - Session-based authentication with password hashing
- **User Management** - Full CRUD operations for managing system users
- **Modern Web UI** - Dark theme with responsive design similar to Unraid
- **Top Navigation Bar** - Quick access to all management modules
- **Activity Logging** - Track all user actions and system events
- **Role-Based Access** - Admin and User roles with permission controls
- **Network Shares** - Full SMB/Samba share management with user permissions
- **Virtual Machines** - QEMU/KVM VM creation, management, and SPICE console access
- **Web Terminal** - Browser-based terminal access for administrators
- **System Updates** - Built-in system update functionality via web interface
- **System Logs** - Comprehensive logging and log viewing interface

## Modules

- ✅ **Dashboard** - System overview and statistics
- ✅ **Users** - User management (Admin only)
- ✅ **Shares** - Network shares (SMB/Samba) with full CRUD, user management, and permissions
- ✅ **VMs** - Virtual machine management (QEMU/KVM) with console access
- ✅ **Logs** - System and activity log viewing with filtering
- ✅ **Settings** - System configuration and updates
- ✅ **Terminal** - Web-based terminal access (Admin only)
- ✅ **Profile** - User profile management
- 🚧 **Disks** - Storage device management (in development)
- 🚧 **Docker** - Container management (in development)
- 🚧 **Plugins** - Plugin system (in development)

## Project Structure

```
/home/user/git/tso/
├── config/
│   ├── config.php          # Application configuration
│   ├── config.example.php  # Configuration template
│   └── samba-sudoers       # Samba sudo permissions
├── public/                 # Web root (point your web server here)
│   ├── api/                # API endpoints
│   │   ├── share-control.php    # Share management API
│   │   ├── vm-control.php       # VM management API
│   │   ├── system-stats.php     # System statistics API
│   │   ├── system-update.php    # System update API
│   │   ├── terminal-exec.php    # Terminal execution API
│   │   └── logs.php             # Logs API
│   ├── css/
│   │   └── style.css       # Main stylesheet
│   ├── js/
│   │   ├── main.js         # JavaScript utilities
│   │   └── shares.js       # Shares module JavaScript
│   ├── index.php           # Main entry point
│   ├── login.php           # Login page
│   ├── dashboard.php       # Dashboard
│   ├── users.php           # User management
│   ├── shares.php          # Network shares management
│   ├── vms.php             # Virtual machine management
│   ├── logs.php            # System logs viewer
│   ├── settings.php        # System settings
│   ├── terminal.php        # Web terminal
│   ├── profile.php         # User profile
│   ├── disks.php           # Disk management (placeholder)
│   ├── docker.php          # Docker management (placeholder)
│   └── plugins.php         # Plugin management (placeholder)
├── src/
│   ├── Database.php        # Database connection handler
│   ├── User.php            # User model
│   ├── Auth.php            # Authentication handler
│   ├── Share.php           # Share management class
│   ├── VM.php              # VM management class
│   ├── Logger.php          # Logging handler
│   └── ErrorHandler.php    # Error handling
├── views/
│   ├── layout/
│   │   ├── header.php      # HTML header
│   │   ├── navbar.php      # Top navigation bar
│   │   └── footer.php      # HTML footer
│   └── pages/              # Page templates
├── tools/                  # Management utilities
│   ├── reset-admin.php     # Reset admin password
│   ├── check-db.php        # Database health check
│   └── migrate-database.php # Database migrations
├── scripts/                # Setup scripts
│   └── setup-shares.sh     # Samba setup script
├── storage/                # Storage directory
└── init.sql                # Database schema

```

## Requirements

- PHP 7.4 or higher
- MySQL 5.7+ or MariaDB 10.2+
- Apache/Nginx web server
- Debian 10+ (recommended)

## Quick Installation ⚡

Install TSO with a single command:

```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

That's it! The installer will automatically:
- ✓ Install Apache2, MariaDB, PHP and all dependencies
- ✓ Create and configure the database
- ✓ Deploy application files to `/opt/serveros`
- ✓ Configure Apache virtual host
- ✓ Set proper permissions and security
- ✓ Start all services

**Installation takes 5-10 minutes.** After completion, access the web interface at `http://YOUR_SERVER_IP`

### Alternative: Git Clone Method

```bash
git clone https://github.com/chukfinley/tso.git
cd tso
sudo ./install.sh
```

For detailed installation instructions, see [INSTALL.md](INSTALL.md)

## Updating TSO 🔄

### Quick Update (Fastest!)

On your server, simply run:

```bash
cd /opt/serveros && sudo git pull && sudo ./post-update.sh
```

**Takes 5 seconds!** Updates only changed files.

### Alternative Methods

Re-run the installer:
```bash
curl -sSL https://raw.githubusercontent.com/chukfinley/tso/master/bootstrap.sh | sudo bash
```

Or use the update script:
```bash
sudo /opt/serveros/update.sh
```

### What Gets Updated

- ✅ Application files updated to latest version
- ✅ Tools and utilities updated
- ✅ **Configuration preserved** (database credentials)
- ✅ **Database and users preserved**
- ✅ **Logs and storage preserved**

**No data loss!** Your settings and users are safe.

For detailed update instructions, see [GIT-UPDATE.md](GIT-UPDATE.md)

## Manual Installation

If you prefer manual installation:

### 1. Install Dependencies

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install LAMP stack
sudo apt install -y apache2 mariadb-server php php-mysql php-cli php-mbstring php-xml

# Start and enable services
sudo systemctl start apache2 mariadb
sudo systemctl enable apache2 mariadb
```

### 2. Configure Database

```bash
# Secure MySQL installation
sudo mysql_secure_installation

# Import database schema
sudo mysql -u root -p < init.sql
```

### 3. Configure Web Server

**Apache Configuration:**

```bash
# Create virtual host configuration
sudo nano /etc/apache2/sites-available/serveros.conf
```

Add the following configuration:

```apache
<VirtualHost *:80>
    ServerName serveros.local
    DocumentRoot /home/user/git/tso/public

    <Directory /home/user/git/tso/public>
        Options -Indexes +FollowSymLinks
        AllowOverride All
        Require all granted
    </Directory>

    ErrorLog ${APACHE_LOG_DIR}/serveros_error.log
    CustomLog ${APACHE_LOG_DIR}/serveros_access.log combined
</VirtualHost>
```

Enable the site and rewrite module:

```bash
sudo a2ensite serveros.conf
sudo a2enmod rewrite
sudo systemctl restart apache2
```

### 4. Set Permissions

```bash
# Set proper ownership
sudo chown -R www-data:www-data /home/user/git/tso

# Set proper permissions
sudo chmod -R 755 /home/user/git/tso
sudo chmod -R 775 /home/user/git/tso/public
```

### 5. Configure Database Connection

Edit `config/config.php` and update database credentials:

```php
define('DB_HOST', 'localhost');
define('DB_NAME', 'servermanager');
define('DB_USER', 'root');
define('DB_PASS', 'your_password');
```

## Default Credentials

After installation, login with:

- **Username:** `admin`
- **Password:** `admin123`

**⚠️ Important:** Change the default password immediately after first login!

## Usage

1. Navigate to `http://serveros.local` (or your configured domain)
2. Login with default credentials (`admin` / `admin123`)
3. Access the dashboard to view system overview and statistics
4. **Manage Users** - Create, edit, and delete system users (Admin only)
5. **Network Shares** - Create and manage SMB/Samba shares with user permissions and access control
6. **Virtual Machines** - Create and manage QEMU/KVM virtual machines with SPICE console access
7. **View Logs** - Browse system logs and activity logs with filtering options
8. **System Settings** - Configure system settings and update TSO from the web interface
9. **Web Terminal** - Access a browser-based terminal (Admin only)
10. **Profile** - Manage your user profile and preferences

For detailed guides on specific features:
- **Shares**: See [SHARES-QUICKSTART.md](SHARES-QUICKSTART.md) and [SHARES-FEATURES.md](SHARES-FEATURES.md)
- **VMs**: Create and manage virtual machines through the VMs interface

## Development Roadmap

### Phase 1: Core System ✅ **COMPLETE**
- [x] User authentication and session management
- [x] User CRUD operations
- [x] Activity logging
- [x] Modern UI with navigation
- [x] Dashboard with system info
- [x] System settings and update functionality
- [x] Web terminal access
- [x] Profile management

### Phase 2: Storage Management ✅ **MOSTLY COMPLETE**
- [x] SMB/Samba share management with full CRUD operations
- [x] Share user management and permissions (Read/Write/Admin)
- [x] Guest access and security settings
- [x] Activity logging for shares
- [x] Samba configuration management
- [ ] Disk detection and monitoring
- [ ] RAID array management
- [ ] File system operations
- [ ] NFS share support

### Phase 3: Virtualization ✅ **COMPLETE**
- [x] VM (QEMU/KVM) support with full management
- [x] VM creation, editing, and deletion
- [x] SPICE console access via web browser
- [x] Resource allocation (CPU, RAM, disk)
- [x] Network configuration (NAT, bridge)
- [x] ISO and disk image management
- [ ] Docker container management (in development)

### Phase 4: Advanced Features 🚧 **IN PROGRESS**
- [ ] Disk management interface
- [ ] Docker container management
- [ ] Plugin system
- [ ] Backup and restore
- [ ] Enhanced monitoring and alerts
- [x] API endpoints (Shares, VMs, System)

## Troubleshooting

### Login Issues - "Invalid username or password"

If you can't login after installation:

```bash
# Reset admin password
sudo /opt/serveros/tools/reset-admin.php

# Or check database status
sudo /opt/serveros/tools/check-db.php
```

**For more issues:** See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for comprehensive solutions.

### Common Issues:
- **Can't access web interface** - Check firewall: `sudo ufw allow 80/tcp`
- **Database errors** - Run: `sudo /opt/serveros/tools/check-db.php`
- **Permission errors** - Run: `sudo chown -R www-data:www-data /opt/serveros`
- **Forgot password** - Run: `sudo /opt/serveros/tools/reset-admin.php`

## Security Notes

- All passwords are hashed using bcrypt
- SQL injection protection via prepared statements
- Session-based authentication with timeout
- XSS protection via htmlspecialchars
- CSRF protection (to be implemented)
- Admin-only access for sensitive operations

## Contributing

TSO is actively developed. Major features implemented:
- ✅ Complete SMB/Samba share management system
- ✅ Full QEMU/KVM virtual machine management
- ✅ Web terminal interface
- ✅ Comprehensive logging system
- ✅ System update functionality

Areas that could use contributions:
- 🚧 Disk management and RAID support
- 🚧 Docker container management
- 🚧 Plugin system architecture
- 🚧 NFS share support
- 🚧 Enhanced monitoring and alerting

Feel free to contribute by implementing these planned modules or improving existing features!

## License

See LICENSE file for details.

## Credits

Inspired by:
- Unraid - https://unraid.net/
- TrueNAS - https://www.truenas.com/
