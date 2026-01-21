# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Build and Run Commands

```bash
# Install dependencies
go mod download

# Build binary
go build -o tso-server .

# Run server (loads .env automatically)
go run .

# Build and run
go build -o tso-server . && ./tso-server

# Generate bcrypt password hash (utility)
go run ./cmd/fix-admin-password/
```

No Makefile or test suite exists. Manual/integration testing only.

## Architecture Overview

This is a **single-package Go backend** (`package main`) for TSO (The Server OS), a Debian-based server management system. All code lives in the root with flat file organization.

### File Organization

| File | Purpose |
|------|---------|
| `main.go` | Entry point, router setup, CORS, frontend serving |
| `models.go` | All data structs (User, Share, VirtualMachine, etc.) |
| `database.go` | MySQL connection helper |
| `auth.go` | Login/logout handlers, session middleware |
| `users.go` | User CRUD handlers |
| `shares.go` | Samba share management + system commands |
| `vms.go` | QEMU/KVM VM lifecycle + backups |
| `system.go` | System stats, terminal execution, logs |

### Key Patterns

**Handler structure** - All handlers follow this pattern:
```go
func SomeHandler(w http.ResponseWriter, r *http.Request) {
    db, err := NewDatabase()  // New connection per request
    defer db.Close()
    // Parse request, execute logic, return JSON
    json.NewEncoder(w).Encode(response)
}
```

**Auth middleware** - Composable wrappers:
```go
RequireAuth(handler)                    // Requires login
RequireAuth(RequireAdmin(handler))      // Requires admin role
```

**System operations** - Direct `exec.Command()` calls for Samba, QEMU, shell commands.

### Dual-Port Architecture

- **API server**: Port 8080 (configurable via `PORT`)
- **Frontend server**: Port 80 (configurable via `FRONTEND_PORT`) - serves React SPA from `frontend/dist`

Frontend is optional; API works headless if not found.

## Configuration

Environment variables only (no config files). Loaded from `.env` in working directory or `/opt/serveros/go-backend/.env`.

| Variable | Default | Description |
|----------|---------|-------------|
| `DB_HOST` | localhost | MySQL host |
| `DB_NAME` | servermanager | Database name |
| `DB_USER` | root | Database user |
| `DB_PASS` | (empty) | Database password |
| `SESSION_SECRET` | (insecure default) | Cookie encryption key |
| `PORT` | 8080 | API port |
| `FRONTEND_PORT` | 80 | Frontend SPA port |
| `INSTALL_DIR` | /opt/serveros | Installation base path |
| `ALLOWED_ORIGINS` | (auto-detected) | CORS origins, comma-separated |

## Database

MySQL/MariaDB with these core tables:
- `users` - Dashboard admin/user accounts (bcrypt passwords)
- `shares`, `share_users`, `share_permissions` - Samba configuration
- `virtual_machines`, `vm_backups` - QEMU/KVM VMs
- `activity_log`, `system_logs` - Audit trail

No ORM - direct SQL with `database/sql` and manual row scanning.

## API Routes

All routes prefixed with `/api`. Auth routes:
- `POST /login`, `POST /logout`, `GET /auth/check`

Protected routes use `RequireAuth()`. Admin-only routes additionally use `RequireAdmin()`.

Domain prefixes: `/users`, `/shares`, `/share-users`, `/vms`, `/system`, `/terminal`, `/logs`

## Adding New Handlers

1. Add handler function in appropriate domain file (or create new file)
2. Register route in `main.go` router section
3. Use `RequireAuth()` / `RequireAdmin()` wrappers as needed
4. For new tables, add to schema and create migration logic

## Key Dependencies

- `github.com/gorilla/mux` - HTTP routing
- `github.com/gorilla/sessions` - Cookie-based sessions
- `github.com/go-sql-driver/mysql` - MySQL driver
- `golang.org/x/crypto/bcrypt` - Password hashing
