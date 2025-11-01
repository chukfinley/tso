# Migration from PHP to Go/TypeScript

This document describes the migration from the PHP version to the new Go/TypeScript version of TSO.

## What Changed

### Architecture
- **Backend**: PHP → Go (Gorilla Mux)
- **Frontend**: PHP templates + jQuery → TypeScript/React with Vite
- **API**: PHP files → RESTful Go handlers
- **Build**: Apache/PHP-FPM → Go binary + static frontend

### What Stayed the Same
- **Database**: Same MySQL/MariaDB schema (`init.sql`)
- **Features**: All functionality preserved
- **Database structure**: Fully compatible, no migration needed

## Old Code Location

All old PHP code has been moved to `archive-old-php/` directory:

```
archive-old-php/
├── public/         # Old PHP pages
├── src/            # Old PHP classes
├── views/          # Old PHP templates
├── config/          # Old PHP configuration
├── tools/           # Old PHP tools
└── scripts/         # Old setup scripts
```

## Database Migration

**No database migration needed!** The database schema is identical, so:

1. Keep your existing database
2. Point the new Go backend to the same database
3. Everything will work with your existing data

## Installation Process

The new installation script (`install.sh`) will:

1. Install Go, Node.js, MySQL dependencies
2. Setup database (creates new or uses existing)
3. Build Go backend
4. Build TypeScript frontend
5. Create systemd service
6. Configure nginx (if available)

## Configuration

### Old (PHP)
- Configuration in `config/config.php`
- PHP constants for database

### New (Go)
- Configuration via environment variables
- Stored in `go-backend/.env`:
  ```
  DB_HOST=localhost
  DB_NAME=servermanager
  DB_USER=tso
  DB_PASS=your_password
  SESSION_SECRET=secret
  PORT=8080
  ```

## Running the Application

### Old (PHP)
```bash
# Run via Apache/PHP-FPM
# Configured at /etc/apache2/sites-available/serveros.conf
```

### New (Go)
```bash
# Run via systemd service
sudo systemctl start tso

# Or manually
cd go-backend
export $(cat .env | xargs)
./tso-server
```

## File Locations

### Old Installation
- Files: `/opt/serveros/public/`
- Config: `/opt/serveros/config/config.php`
- Tools: `/opt/serveros/tools/`

### New Installation
- Backend: `/opt/serveros/go-backend/`
- Frontend: `/opt/serveros/frontend/dist/`
- Config: `/opt/serveros/go-backend/.env`
- Binary: `/opt/serveros/go-backend/tso-server`

## API Changes

### Old API
- PHP files in `public/api/*.php`
- Direct file access: `/api/system-stats.php`
- Form-data and JSON mixed

### New API
- RESTful endpoints: `/api/system/stats`
- JSON only
- Consistent response format

**Endpoint Mapping:**

| Old | New |
|-----|-----|
| `/api/login.php` | `POST /api/login` |
| `/api/system-stats.php` | `GET /api/system/stats` |
| `/api/share-control.php?action=list` | `GET /api/shares` |
| `/api/vm-control.php?action=list` | `GET /api/vms` |

## Frontend Changes

### Old Frontend
- PHP templates with server-side rendering
- jQuery for AJAX
- Bootstrap for styling
- Direct PHP includes

### New Frontend
- React SPA (Single Page Application)
- TypeScript for type safety
- Vite for fast development
- Axios for HTTP requests
- Client-side routing

## Development Workflow

### Old (PHP)
```bash
# Edit PHP files
vim public/dashboard.php
# Refresh browser
```

### New (Go/TypeScript)
```bash
# Terminal 1: Backend
cd go-backend
go run .

# Terminal 2: Frontend
cd frontend
npm run dev

# Edit files, auto-reload
```

## Breaking Changes

1. **No direct PHP execution** - Everything goes through Go API
2. **No PHP sessions** - Uses Go session middleware
3. **No .php files** - All routes through Go router
4. **Frontend is SPA** - No server-side rendering
5. **API format changed** - All endpoints now RESTful JSON

## Compatibility

### What Works
✅ Same database schema
✅ All features preserved
✅ Same user accounts
✅ Same shares and VMs

### What Doesn't Work
❌ Old PHP API endpoints (moved to REST)
❌ Direct .php file access
❌ PHP-specific tools (need to be rewritten in Go)

## Migration Steps

1. **Backup your data:**
   ```bash
   mysqldump -u root -p servermanager > backup.sql
   ```

2. **Run new installer:**
   ```bash
   sudo ./install.sh
   ```

3. **Point installer to existing database:**
   - The installer will detect existing database
   - Choose to keep it or recreate

4. **Test everything:**
   - Login with existing credentials
   - Verify shares, VMs, users

5. **Remove old Apache config** (optional):
   ```bash
   sudo a2dissite serveros
   sudo systemctl reload apache2
   ```

## Rollback

If you need to rollback:

1. Stop new service:
   ```bash
   sudo systemctl stop tso
   ```

2. Restore old files from archive:
   ```bash
   cp -r archive-old-php/* /opt/serveros/
   ```

3. Restart Apache:
   ```bash
   sudo systemctl restart apache2
   ```

## Support

For issues or questions about the migration:

1. Check `README.md` for general info
2. Check `INSTALL.md` for installation issues
3. Check `QUICKSTART-LOCAL.md` for local development
4. Review `README-GO-TS.md` for architecture details
