#!/bin/bash
# TSO Quick Deploy Script
# Usage: ./deploy.sh [host]

set -e

HOST="${1:-192.168.122.165}"
USER="root"
REMOTE_DIR="/opt/serveros"

echo "=== TSO Deploy to $HOST ==="

# Build frontend
echo "[1/4] Building frontend..."
cd "$(dirname "$0")/frontend"
npm run build --silent

# Copy frontend
echo "[2/4] Deploying frontend..."
scp -q -r dist/* ${USER}@${HOST}:${REMOTE_DIR}/frontend/dist/

# Copy backend source
echo "[3/4] Deploying backend..."
cd "$(dirname "$0")/go-backend"
scp -q *.go ${USER}@${HOST}:${REMOTE_DIR}/go-backend/

# Rebuild and restart on server
echo "[4/4] Rebuilding and restarting service..."
ssh ${USER}@${HOST} "cd ${REMOTE_DIR}/go-backend && go build -o tso-server . && systemctl restart tso"

echo "=== Deploy complete ==="
echo "Access: http://${HOST}"
