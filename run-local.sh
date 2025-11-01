#!/bin/bash
# Run script for local development

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}TSO Local Development Runner${NC}"
echo "================================"
echo ""

# Check if backend is already running
if lsof -Pi :8080 -sTCP:LISTEN -t >/dev/null ; then
    echo -e "${YELLOW}⚠️  Port 8080 is already in use${NC}"
    echo "Stop the existing process or change PORT in .env"
    exit 1
fi

# Check if frontend is already running
if lsof -Pi :3000 -sTCP:LISTEN -t >/dev/null ; then
    echo -e "${YELLOW}⚠️  Port 3000 is already in use${NC}"
    echo "Stop the existing process first"
    exit 1
fi

# Function to handle cleanup
cleanup() {
    echo ""
    echo -e "${YELLOW}Stopping servers...${NC}"
    kill $BACKEND_PID $FRONTEND_PID 2>/dev/null
    exit
}

trap cleanup SIGINT SIGTERM

# Start backend
echo -e "${GREEN}Starting backend server...${NC}"
cd go-backend

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
else
    echo -e "${YELLOW}⚠️  .env file not found. Using defaults${NC}"
    export DB_HOST=${DB_HOST:-localhost}
    export DB_NAME=${DB_NAME:-servermanager}
    export DB_USER=${DB_USER:-root}
    export DB_PASS=${DB_PASS:-}
    export SESSION_SECRET=${SESSION_SECRET:-default-secret-change-in-production}
    export PORT=${PORT:-8080}
fi

go run . &
BACKEND_PID=$!
cd ..

sleep 2

# Check if backend started successfully
if ! kill -0 $BACKEND_PID 2>/dev/null; then
    echo -e "${RED}❌ Backend failed to start${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Backend running on http://localhost:${PORT}${NC}"

# Start frontend
echo -e "${GREEN}Starting frontend server...${NC}"
cd frontend
npm run dev &
FRONTEND_PID=$!
cd ..

sleep 2

# Check if frontend started successfully
if ! kill -0 $FRONTEND_PID 2>/dev/null; then
    echo -e "${RED}❌ Frontend failed to start${NC}"
    kill $BACKEND_PID 2>/dev/null
    exit 1
fi

echo -e "${GREEN}✅ Frontend running on http://localhost:3000${NC}"
echo ""
echo -e "${GREEN}🚀 Both servers are running!${NC}"
echo ""
echo "Frontend: http://localhost:3000"
echo "Backend API: http://localhost:8080/api"
echo ""
echo "Press Ctrl+C to stop both servers"

# Wait for both processes
wait

