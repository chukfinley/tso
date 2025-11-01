# Quick Start Guide - Running TSO Locally

This guide will help you run the TSO application on your local computer for testing.

## Prerequisites

### Required Software
- **Go** 1.21 or higher ([Download](https://golang.org/dl/))
- **Node.js** 18+ and npm ([Download](https://nodejs.org/))
- **MySQL** or **MariaDB** ([Download](https://mariadb.org/download/) or use `sudo apt install mariadb-server`)

### Optional (for full functionality)
- **Samba** (for share management): `sudo apt install samba`
- **QEMU/KVM** (for VM management): `sudo apt install qemu-kvm libvirt-daemon-system`

## Step 1: Database Setup

1. **Start MySQL/MariaDB service:**
```bash
sudo systemctl start mariadb
# or
sudo systemctl start mysql
```

2. **Create database and user:**
```bash
sudo mysql -u root -p
```

Then in MySQL console:
```sql
CREATE DATABASE servermanager;
CREATE USER 'tso'@'localhost' IDENTIFIED BY 'tso_password';
GRANT ALL PRIVILEGES ON servermanager.* TO 'tso'@'localhost';
FLUSH PRIVILEGES;
EXIT;
```

3. **Import database schema:**
```bash
cd /home/user/git/tso
mysql -u tso -ptso_password servermanager < init.sql
```

## Step 2: Backend Setup

1. **Navigate to backend directory:**
```bash
cd /home/user/git/tso/go-backend
```

2. **Install Go dependencies:**
```bash
go mod download
```

3. **Set environment variables:**
```bash
export DB_HOST=localhost
export DB_NAME=servermanager
export DB_USER=tso
export DB_PASS=tso_password
export SESSION_SECRET=your-secret-key-change-this
export PORT=8080
```

Or create a `.env` file (you may need to install a package to load it, or set them manually):
```bash
# .env file
DB_HOST=localhost
DB_NAME=servermanager
DB_USER=tso
DB_PASS=tso_password
SESSION_SECRET=your-secret-key-change-this
PORT=8080
```

4. **Run the backend server:**
```bash
go run .
```

You should see:
```
Server starting on port 8080
```

**Keep this terminal open!** The backend will run on `http://localhost:8080`

## Step 3: Frontend Setup

1. **Open a NEW terminal** and navigate to frontend directory:
```bash
cd /home/user/git/tso/frontend
```

2. **Install dependencies:**
```bash
npm install
```

3. **Start the development server:**
```bash
npm run dev
```

You should see:
```
  VITE v5.x.x  ready in xxx ms

  ➜  Local:   http://localhost:3000/
  ➜  Network: use --host to expose
```

The frontend will automatically proxy API requests to `http://localhost:8080`

## Step 4: Access the Application

1. **Open your browser** and go to:
```
http://localhost:3000
```

2. **Login with default credentials:**
   - If you're using the existing database: use the credentials you already have
   - If starting fresh: You may need to create an admin user first (see below)

## Creating the First Admin User

If you need to create an admin user, you can use the existing PHP tools or create one directly in MySQL:

```sql
mysql -u tso -ptso_password servermanager

INSERT INTO users (username, email, password, full_name, role, is_active) 
VALUES ('admin', 'admin@localhost', '$2a$10$YourHashedPasswordHere', 'Administrator', 'admin', 1);
```

Or use the Go backend to create one (you'll need to temporarily disable auth or create a setup script).

## Troubleshooting

### Database Connection Issues
- Check MySQL is running: `sudo systemctl status mariadb`
- Verify credentials: `mysql -u tso -ptso_password servermanager`
- Check database exists: `SHOW DATABASES;`

### Backend Won't Start
- Check Go version: `go version` (need 1.21+)
- Check dependencies: `go mod download`
- Check port 8080 is free: `lsof -i :8080` or `netstat -an | grep 8080`

### Frontend Won't Start
- Check Node version: `node --version` (need 18+)
- Clear node_modules and reinstall: `rm -rf node_modules && npm install`
- Check port 3000 is free: `lsof -i :3000`

### API Calls Fail
- Ensure backend is running on port 8080
- Check browser console for CORS errors
- Verify the proxy in `vite.config.ts` points to `http://localhost:8080`

## Development Workflow

1. **Backend changes:** Edit Go files in `go-backend/`, server auto-reloads with `go run .`
2. **Frontend changes:** Edit files in `frontend/src/`, Vite auto-reloads in browser
3. **Database changes:** Modify schema in `init.sql` and re-import

## Quick Test Commands

### Test Backend API directly:
```bash
# Test login endpoint
curl -X POST http://localhost:8080/api/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"your_password"}' \
  -c cookies.txt

# Test system stats (requires auth)
curl http://localhost:8080/api/system/stats -b cookies.txt
```

### Test Frontend:
Just open `http://localhost:3000` in your browser!

## Stopping the Servers

- **Backend:** Press `Ctrl+C` in the backend terminal
- **Frontend:** Press `Ctrl+C` in the frontend terminal

## Production Build (Optional)

To build for production:

**Backend:**
```bash
cd go-backend
go build -o tso-server .
./tso-server
```

**Frontend:**
```bash
cd frontend
npm run build
# Serve the dist/ folder with a web server
```

## Next Steps

- Explore the dashboard at `http://localhost:3000/dashboard`
- Create users, shares, and VMs through the UI
- Check system stats in real-time
- Review logs in the Logs section

Happy testing! 🚀

