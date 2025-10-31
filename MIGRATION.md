# Database Migration Instructions

## Fix 500 Error on VM Page

The 500 error is caused by missing database tables (virtual_machines, vm_backups, shares, etc.).

### Option 1: Run Migration Script Directly (RECOMMENDED - Takes seconds)

```bash
sudo php /opt/serveros/tools/migrate-database.php
```

This will:
- ✓ Check for and create missing tables
- ✓ Create VM storage directories
- ✓ Set proper permissions
- ✓ Preserve all existing data

### Option 2: Re-run Install Script (Automatic Update)

```bash
cd /home/user/git/tso
sudo ./install.sh
```

When prompted, choose "yes" to update. This will:
- ✓ Update all application files
- ✓ Run database migrations automatically
- ✓ Preserve your config and database
- ✓ Install/verify all QEMU/KVM packages

### Option 3: Pull Latest Changes (Fastest if on server)

```bash
cd /opt/serveros
sudo git pull
sudo php tools/migrate-database.php
sudo systemctl restart apache2
```

## What Gets Installed

### Database Tables Created
- `virtual_machines` - VM configuration and state
- `vm_backups` - VM backup management
- `shares` - Network shares configuration
- `share_users` - Samba users
- `share_permissions` - Share access control
- `share_access_log` - Share access logging

### Storage Directories Created
- `/opt/serveros/storage/vms` - VM disk images
- `/opt/serveros/storage/backups` - VM backups
- `/opt/serveros/storage/isos` - ISO files for VMs

### QEMU/KVM Packages (Already in install.sh)
- qemu-kvm
- qemu-system-x86
- qemu-utils
- libvirt-daemon-system
- libvirt-clients
- bridge-utils
- virt-manager
- gzip

## After Migration

Once the migration completes:

1. Visit http://YOUR_SERVER_IP/vms.php
2. The page should load without 500 error
3. You can now:
   - Create new VMs with QEMU/KVM
   - Configure CPU, RAM, disk, network
   - Mount ISO images
   - Start/Stop/Restart VMs
   - View VM logs
   - Download SPICE connection files
   - Create and restore VM backups

## Troubleshooting

If you still get errors after migration:

1. Check Apache error log:
   ```bash
   sudo tail -50 /var/log/apache2/error.log
   ```

2. Check if migration completed:
   ```bash
   sudo mysql servermanager -e "SHOW TABLES LIKE 'virtual_machines'"
   ```

3. Verify QEMU is installed:
   ```bash
   which qemu-system-x86_64
   ```

4. Check www-data permissions:
   ```bash
   groups www-data
   # Should show: kvm libvirt
   ```
