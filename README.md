# ServerOS

A Debian-based server management system inspired by Unraid and TrueNAS — built in pure PHP.

## Features

- **Secure Login System** - Session-based authentication with password hashing
- **User Management** - Full CRUD operations for managing system users
- **Modern Web UI** - Dark theme with responsive design similar to Unraid
- **Top Navigation Bar** - Quick access to all management modules
- **Activity Logging** - Track all user actions and system events
- **Role-Based Access** - Admin and User roles with permission controls

## Modules (In Development)

- ✅ Dashboard - System overview and statistics
- ✅ Users - User management (Admin only)
- 🚧 Disks - Storage device management
- 🚧 Shares - Network shares (SMB/NFS)
- 🚧 Docker - Container management
- 🚧 VMs - Virtual machine management
- 🚧 Plugins - Plugin system
- 🚧 Settings - System configuration

## Project Structure

```
/home/user/git/tso/
├── config/
│   └── config.php          # Application configuration
├── public/                 # Web root (point your web server here)
│   ├── css/
│   │   └── style.css       # Main stylesheet
│   ├── js/
│   │   └── main.js         # JavaScript utilities
│   ├── index.php           # Main entry point
│   ├── login.php           # Login page
│   ├── dashboard.php       # Dashboard
│   ├── users.php           # User management
│   └── ...                 # Other module pages
├── src/
│   ├── Database.php        # Database connection handler
│   ├── User.php            # User model
│   └── Auth.php            # Authentication handler
├── views/
│   ├── layout/
│   │   ├── header.php      # HTML header
│   │   ├── navbar.php      # Top navigation bar
│   │   └── footer.php      # HTML footer
│   └── pages/              # Page templates
└── init.sql                # Database schema

```

## Requirements

- PHP 7.4 or higher
- MySQL 5.7+ or MariaDB 10.2+
- Apache/Nginx web server
- Debian 10+ (recommended)

## Quick Installation ⚡

The easiest way to install ServerOS is using our automated installation script:

```bash
# Clone the repository
git clone https://github.com/yourusername/serveros.git
cd serveros

# Run the automated installer
sudo ./install.sh
```

The installer will automatically:
- ✓ Install Apache2, MariaDB, PHP and all dependencies
- ✓ Create and configure the database
- ✓ Deploy application files to `/opt/serveros`
- ✓ Configure Apache virtual host
- ✓ Set proper permissions and security
- ✓ Start all services

**Installation takes 5-10 minutes.** After completion, access the web interface at `http://YOUR_SERVER_IP`

For detailed installation instructions, see [INSTALL.md](INSTALL.md)

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
2. Login with default credentials
3. Access the dashboard to view system overview
4. Manage users via the Users page (Admin only)
5. Other modules are placeholders for future development

## Development Roadmap

### Phase 1: Core System ✅
- [x] User authentication and session management
- [x] User CRUD operations
- [x] Activity logging
- [x] Modern UI with navigation
- [x] Dashboard with system info

### Phase 2: Storage Management (Next)
- [ ] Disk detection and monitoring
- [ ] RAID array management
- [ ] File system operations
- [ ] SMB/NFS share management

### Phase 3: Virtualization
- [ ] Docker container management
- [ ] VM (KVM/QEMU) support
- [ ] Resource allocation

### Phase 4: Advanced Features
- [ ] Plugin system
- [ ] Backup and restore
- [ ] Monitoring and alerts
- [ ] API endpoints

## Security Notes

- All passwords are hashed using bcrypt
- SQL injection protection via prepared statements
- Session-based authentication with timeout
- XSS protection via htmlspecialchars
- CSRF protection (to be implemented)
- Admin-only access for sensitive operations

## Contributing

This is a work in progress. Feel free to contribute by implementing the planned modules!

## License

See LICENSE file for details.

## Credits

Inspired by:
- Unraid - https://unraid.net/
- TrueNAS - https://www.truenas.com/
