#!/bin/bash
# Development startup script for Truth-telling
# Prerequisites: Go 1.24+, Node 18+, MongoDB 5.0 running on localhost:27017
#
# Quick start:
#   1. Install and start MongoDB 5.0 (brew install mongodb-community@5.0 OR docker run -d -p 27017:27017 mongo:5.0)
#   2. Run this script: ./dev.sh
#
# This starts both the Go backend (:5000) and the Next.js frontend (:3000)

set -e

# Backend env vars (modify as needed)
export DB_DEBUG_URI="${DB_DEBUG_URI:-localhost:27017}"
export DASHBOARD_USER="${DASHBOARD_USER:-admin}"
export DASHBOARD_PASSWORD="${DASHBOARD_PASSWORD:-admin123}"
FRONTEND_PORT="${FRONTEND_PORT:-3033}"

echo "=== Truth-telling Development ==="
echo ""
echo "Backend:  http://localhost:5000"
echo "Frontend: http://localhost:$FRONTEND_PORT"
echo "Admin:    http://localhost:5000/pool/ (user: $DASHBOARD_USER)"
echo ""

# Check MongoDB
if ! command -v mongosh &>/dev/null && ! command -v mongo &>/dev/null; then
  echo "WARNING: mongosh not found. Make sure MongoDB is running on $DB_DEBUG_URI"
fi

# Build and start backend
echo "[1/2] Starting Go backend..."
cd backend
if [ ! -f server ]; then
  echo "  Building..."
  go build -o server .
fi
./server &
BACKEND_PID=$!
cd ..

# Wait for backend
sleep 2
if ! kill -0 $BACKEND_PID 2>/dev/null; then
  echo "ERROR: Backend failed to start. Is MongoDB running on $DB_DEBUG_URI?"
  exit 1
fi
echo "  Backend running (PID $BACKEND_PID)"

# Seed questions if database is empty
echo "  Checking if questions need seeding..."
SEED_RESPONSE=$(curl -s -X POST http://localhost:5000/dashboard/count -u "$DASHBOARD_USER:$DASHBOARD_PASSWORD")
if echo "$SEED_RESPONSE" | grep -q '"questions":0'; then
  echo "  Seeding starter questions..."
  for q in \
    "What is something you have never told anyone?" \
    "What is the biggest risk you have ever taken?" \
    "What do you wish you had done differently in your life?" \
    "What is the most important lesson you learned the hard way?" \
    "Who has had the biggest impact on your life and why?" \
    "What is something you pretend to understand but actually do not?" \
    "What is your biggest fear that you rarely talk about?" \
    "If you could change one decision from your past, what would it be?" \
    "What do you think people misunderstand about you?" \
    "What keeps you up at night?" \
    "What is the kindest thing someone has done for you?" \
    "What is something you believe that most people disagree with?" \
    "What would you do if you knew you could not fail?" \
    "What is the hardest truth you have had to accept?" \
    "What memory do you keep coming back to?" \
    "What do you value most in a friendship?" \
    "What part of yourself do you hide from others?" \
    "What is something you are proud of but never talk about?" \
    "If you could send a message to your younger self, what would it say?" \
    "What does success mean to you, honestly?"
  do
    curl -s -X POST http://localhost:5000/dashboard/questions/add \
      -u "$DASHBOARD_USER:$DASHBOARD_PASSWORD" \
      -H "Content-Type: application/json" \
      -d "{\"content\":\"$q\"}" > /dev/null
  done
  echo "  Seeded 20 questions"
fi

# Start frontend
echo "[2/2] Starting Next.js frontend..."
cd frontend
if [ ! -d node_modules ]; then
  echo "  Installing dependencies..."
  npm install
fi
NEXT_PUBLIC_API_URL=http://localhost:5000 npx next dev --port "$FRONTEND_PORT" &
FRONTEND_PID=$!
cd ..

echo ""
echo "Both services running. Press Ctrl+C to stop."
echo ""

# Cleanup on exit
cleanup() {
  echo ""
  echo "Shutting down..."
  kill $FRONTEND_PID 2>/dev/null
  kill $BACKEND_PID 2>/dev/null
  wait
}
trap cleanup EXIT INT TERM

wait
