# TSO - Go & TypeScript Rewrite

This is a complete rewrite of TSO (The Server OS) in Go (backend) and TypeScript/React (frontend).

## Architecture

### Backend (Go)
- **Location**: `go-backend/`
- **Framework**: Gorilla Mux for routing
- **Database**: MySQL/MariaDB
- **Authentication**: Session-based with Gorilla Sessions
- **Features**:
  - RESTful API endpoints
  - User management
  - Share management (SMB/Samba)
  - VM management (QEMU/KVM)
  - System stats monitoring
  - Terminal execution
  - Logging

### Frontend (TypeScript/React)
- **Location**: `frontend/`
- **Framework**: React 18 with TypeScript
- **Build Tool**: Vite
- **Router**: React Router v6
- **HTTP Client**: Axios
- **Features**:
  - Modern dark theme UI
  - Real-time system monitoring
  - Share management interface
  - VM management interface
  - User management (admin only)
  - Responsive design

## Prerequisites

### Backend
- Go 1.21 or higher
- MySQL/MariaDB
- Samba (for share management)
- QEMU/KVM (for VM management)

### Frontend
- Node.js 18+ and npm/yarn

## Setup

### Backend Setup

1. Install dependencies:
```bash
cd go-backend
go mod download
```

2. Configure database:
   - Create database: `CREATE DATABASE servermanager;`
   - Import schema: `mysql -u root -p servermanager < ../init.sql`

3. Set environment variables:
```bash
export DB_HOST=localhost
export DB_NAME=servermanager
export DB_USER=root
export DB_PASS=your_password
export SESSION_SECRET=your-secret-key
export PORT=8080
```

4. Run the server:
```bash
go run .
```

### Frontend Setup

1. Install dependencies:
```bash
cd frontend
npm install
```

2. Run development server:
```bash
npm run dev
```

3. Build for production:
```bash
npm run build
```

## API Endpoints

### Authentication
- `POST /api/login` - Login
- `POST /api/logout` - Logout
- `GET /api/auth/check` - Check authentication status

### Users
- `GET /api/users` - List users (admin)
- `POST /api/users` - Create user (admin)
- `GET /api/users/{id}` - Get user
- `PUT /api/users/{id}` - Update user (admin)
- `DELETE /api/users/{id}` - Delete user (admin)
- `PUT /api/users/{id}/password` - Update password
- `GET /api/profile` - Get current user profile
- `PUT /api/profile` - Update profile

### Shares
- `GET /api/shares` - List shares
- `POST /api/shares` - Create share
- `GET /api/shares/{id}` - Get share
- `PUT /api/shares/{id}` - Update share
- `DELETE /api/shares/{id}` - Delete share
- `POST /api/shares/{id}/toggle` - Toggle share status
- `GET /api/shares/{id}/permissions` - Get permissions
- `POST /api/shares/{id}/permissions` - Set permission
- `DELETE /api/shares/{id}/permissions/{userId}` - Remove permission

### Virtual Machines
- `GET /api/vms` - List VMs
- `POST /api/vms` - Create VM
- `GET /api/vms/{id}` - Get VM
- `PUT /api/vms/{id}` - Update VM
- `DELETE /api/vms/{id}` - Delete VM
- `POST /api/vms/{id}/start` - Start VM
- `POST /api/vms/{id}/stop` - Stop VM
- `POST /api/vms/{id}/restart` - Restart VM
- `GET /api/vms/{id}/status` - Get VM status
- `GET /api/vms/{id}/logs` - Get VM logs
- `GET /api/vms/{id}/spice` - Get SPICE connection file

### System
- `GET /api/system/stats` - Get system statistics
- `POST /api/system/update` - Update system (admin)
- `POST /api/system/control` - System control (admin)

### Terminal
- `POST /api/terminal/execute` - Execute command (admin)

### Logs
- `GET /api/logs` - Get system logs
- `GET /api/logs/activity` - Get activity logs

## Features

### Implemented
- ✅ User authentication and session management
- ✅ User management (CRUD operations)
- ✅ Share management with Samba integration
- ✅ VM management with QEMU/KVM
- ✅ System stats monitoring
- ✅ Dashboard with real-time updates
- ✅ Modern responsive UI

### In Progress / Planned
- ⚠️ Terminal web interface
- ⚠️ Advanced logging interface
- ⚠️ Settings page
- ⚠️ File upload for ISO images
- ⚠️ VM backup/restore UI
- ⚠️ Share permissions UI

## Development

### Backend Development
```bash
cd go-backend
go run .
```

### Frontend Development
```bash
cd frontend
npm run dev
```

Frontend will be available at `http://localhost:3000` and proxies API requests to `http://localhost:8080`.

## Production Deployment

### Backend
1. Build the binary:
```bash
cd go-backend
go build -o tso-server .
```

2. Run with systemd or supervisor

### Frontend
1. Build:
```bash
cd frontend
npm run build
```

2. Serve the `dist/` directory with nginx or another web server

## Migration from PHP Version

The Go/TypeScript version maintains the same database schema, so existing databases can be used. However, you'll need to:

1. Update API endpoints in any external integrations
2. Review authentication implementation (session-based, compatible)
3. Check file paths and permissions for Samba/VMs

## License

Same as original TSO project.

