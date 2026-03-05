#!/bin/bash

# Distributed Sharding System - Start Script
# This script starts all shard nodes and the API gateway

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

echo "=========================================="
echo "  Distributed Sharding System"
echo "=========================================="

# Build binaries
echo "Building binaries..."
mkdir -p bin
go build -o bin/node ./cmd/node
go build -o bin/gateway ./cmd/gateway
echo "Build complete."

# Create data directory
mkdir -p data

# Kill any existing processes
echo "Stopping any existing processes..."
pkill -f "bin/node" 2>/dev/null || true
pkill -f "bin/gateway" 2>/dev/null || true
sleep 1

# Start shard nodes
echo ""
echo "Starting shard nodes..."
./bin/node -shard=1 -port=8001 -data=./data &
echo "  Shard 1 started on port 8001"
./bin/node -shard=2 -port=8002 -data=./data &
echo "  Shard 2 started on port 8002"
./bin/node -shard=3 -port=8003 -data=./data &
echo "  Shard 3 started on port 8003"
./bin/node -shard=4 -port=8004 -data=./data &
echo "  Shard 4 started on port 8004"

# Wait for nodes to start
sleep 2

# Start API gateway
echo ""
echo "Starting API Gateway on port 8000..."
echo ""
echo "=========================================="
echo "  System Ready!"
echo "=========================================="
echo ""
echo "Architecture:"
echo ""
echo "     Client"
echo "       |"
echo "    Gateway (port 8000)"
echo "       |"
echo "  -------------------------"
echo "  |    |    |    |"
echo "  S1   S2   S3   S4"
echo " 8001 8002 8003 8004"
echo ""
echo "Endpoints:"
echo "  POST http://localhost:8000/users     - Create user"
echo "  GET  http://localhost:8000/users/:id - Get user"
echo "  GET  http://localhost:8000/users     - List all users"
echo "  GET  http://localhost:8000/shards    - Shard status"
echo ""
echo "Press Ctrl+C to stop all services"
echo ""

./bin/gateway -port=8000 -node1=localhost:8001 -node2=localhost:8002 -node3=localhost:8003 -node4=localhost:8004
