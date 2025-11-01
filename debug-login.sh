#!/bin/bash

# TSO Login Debug Script
# This script checks and fixes common login issues

echo "╔════════════════════════════════════════════════════════════════╗"
echo "║              TSO Login Debug & Fix Tool                       ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""

# Load database credentials from .env file
if [ -f "/opt/serveros/go-backend/.env" ]; then
    echo "✓ Found .env file"
    source /opt/serveros/go-backend/.env
else
    echo "❌ .env file not found at /opt/serveros/go-backend/.env"
    exit 1
fi

# Default values if not in .env
DB_HOST=${DB_HOST:-localhost}
DB_NAME=${DB_NAME:-servermanager}
DB_USER=${DB_USER:-tso}

if [ -z "$DB_PASS" ]; then
    echo "❌ DB_PASS not found in .env file"
    exit 1
fi

echo "Database: $DB_NAME"
echo "User: $DB_USER"
echo ""

# Test database connection
echo "1. Testing database connection..."
if mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "SELECT 1;" > /dev/null 2>&1; then
    echo "   ✓ Database connection successful"
else
    echo "   ❌ Failed to connect to database"
    exit 1
fi

# Check if admin user exists
echo ""
echo "2. Checking admin user..."
ADMIN_EXISTS=$(mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -sse "SELECT COUNT(*) FROM users WHERE username='admin';")

if [ "$ADMIN_EXISTS" -eq 0 ]; then
    echo "   ❌ Admin user does not exist!"
    echo "   Creating admin user..."
    
    # Create admin user with correct bcrypt hash
    ADMIN_HASH='$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'
    mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" <<EOF
INSERT INTO users (username, email, password, full_name, role, is_active) 
VALUES ('admin', 'admin@localhost', '$ADMIN_HASH', 'Administrator', 'admin', 1);
EOF
    
    if [ $? -eq 0 ]; then
        echo "   ✓ Admin user created successfully"
    else
        echo "   ❌ Failed to create admin user"
        exit 1
    fi
else
    echo "   ✓ Admin user exists"
fi

# Get admin user details
echo ""
echo "3. Admin user details:"
mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
SELECT 
    id, 
    username, 
    email, 
    role, 
    is_active,
    LEFT(password, 7) as password_prefix,
    LENGTH(password) as password_length,
    created_at,
    last_login
FROM users 
WHERE username='admin'\G" | grep -v "password:"

# Check password hash format
echo ""
echo "4. Checking password hash format..."
PASSWORD_HASH=$(mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -sse "SELECT password FROM users WHERE username='admin';")
PASSWORD_PREFIX=$(echo "$PASSWORD_HASH" | cut -c1-4)

if [ "$PASSWORD_PREFIX" = "\$2a\$" ] || [ "$PASSWORD_PREFIX" = "\$2b\$" ]; then
    echo "   ✓ Password hash format is correct (bcrypt)"
    echo "   Password prefix: $PASSWORD_PREFIX"
else
    echo "   ❌ Password hash format is incorrect!"
    echo "   Current prefix: $PASSWORD_PREFIX"
    echo "   Expected: \$2a\$ or \$2b\$"
    echo ""
    echo "   Fixing password hash..."
    
    # Use the correct bcrypt hash for "admin123"
    ADMIN_HASH='$2a$10$N9qo8uLOickgx2ZMRZoMyeIjZAgcfl7p92ldGxad68LJZdL17lhWy'
    mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "
UPDATE users 
SET password = '$ADMIN_HASH'
WHERE username = 'admin';"
    
    if [ $? -eq 0 ]; then
        echo "   ✓ Password hash fixed"
    else
        echo "   ❌ Failed to fix password hash"
        exit 1
    fi
fi

# Check if admin is active
echo ""
echo "5. Checking if admin account is active..."
IS_ACTIVE=$(mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -sse "SELECT is_active FROM users WHERE username='admin';")

if [ "$IS_ACTIVE" -eq 1 ]; then
    echo "   ✓ Admin account is active"
else
    echo "   ❌ Admin account is disabled! Enabling..."
    mysql -h "$DB_HOST" -u "$DB_USER" -p"$DB_PASS" "$DB_NAME" -e "UPDATE users SET is_active=1 WHERE username='admin';"
    echo "   ✓ Admin account enabled"
fi

# Check backend service status
echo ""
echo "6. Checking TSO backend service..."
if systemctl is-active --quiet tso; then
    echo "   ✓ TSO service is running"
    
    # Check if backend is listening on port 8080
    if netstat -tuln 2>/dev/null | grep -q ":8080 " || ss -tuln 2>/dev/null | grep -q ":8080 "; then
        echo "   ✓ Backend is listening on port 8080"
    else
        echo "   ⚠ Backend might not be listening on port 8080"
        echo "   Checking listening ports:"
        ss -tuln 2>/dev/null | grep LISTEN | grep -E ":(80|8080|3000)" || netstat -tuln 2>/dev/null | grep LISTEN | grep -E ":(80|8080|3000)"
    fi
else
    echo "   ❌ TSO service is not running!"
    echo "   Starting TSO service..."
    systemctl start tso
    sleep 2
    
    if systemctl is-active --quiet tso; then
        echo "   ✓ TSO service started"
    else
        echo "   ❌ Failed to start TSO service"
        echo ""
        echo "   Service status:"
        systemctl status tso --no-pager -l
        echo ""
        echo "   Recent logs:"
        journalctl -u tso -n 20 --no-pager
        exit 1
    fi
fi

# Check backend logs for errors
echo ""
echo "7. Checking recent backend logs..."
echo "   Last 5 log entries:"
journalctl -u tso -n 5 --no-pager | tail -n 5

# Test API endpoint
echo ""
echo "8. Testing API endpoint..."
API_RESPONSE=$(curl -s -o /dev/null -w "%{http_code}" http://localhost:8080/api/system/info 2>/dev/null)

if [ "$API_RESPONSE" = "200" ]; then
    echo "   ✓ API is responding (HTTP $API_RESPONSE)"
elif [ -z "$API_RESPONSE" ]; then
    echo "   ❌ API is not responding (curl failed)"
else
    echo "   ⚠ API returned HTTP $API_RESPONSE"
fi

echo ""
echo "╔════════════════════════════════════════════════════════════════╗"
echo "║                     Debug Complete                             ║"
echo "╚════════════════════════════════════════════════════════════════╝"
echo ""
echo "Login Credentials:"
echo "  Username: admin"
echo "  Password: admin123"
echo ""
echo "Access URL:"
echo "  http://$(hostname -I | awk '{print $1}'):8080"
echo ""
echo "If login still fails, check:"
echo "  1. Browser console for errors (F12)"
echo "  2. Backend logs: journalctl -u tso -f"
echo "  3. Database connectivity from backend"
echo ""

