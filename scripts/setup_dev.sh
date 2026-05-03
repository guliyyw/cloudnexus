#!/usr/bin/env bash
set -euo pipefail

echo "=== CloudNexus Dev Environment Setup ==="

# Check Go
if ! command -v go &>/dev/null; then
    echo "ERROR: Go is required but not installed."
    exit 1
fi
echo "Go: $(go version)"

# Check Node
if ! command -v node &>/dev/null; then
    echo "ERROR: Node.js is required but not installed."
    exit 1
fi
echo "Node: $(node --version)"

# Start dependencies
echo ""
echo "Starting PostgreSQL, Redis, MinIO..."
docker compose -f deploy/docker-compose.single.yml up -d

# Install frontend deps
echo ""
echo "Installing frontend dependencies..."
cd client && npm install

echo ""
echo "Setup complete. Run the following to start:"
echo "  cd server && go run ./cmd/user-file-svc & go run ./cmd/im-svc & go run ./cmd/docker-svc &"
echo "  cd client && npm run dev"
