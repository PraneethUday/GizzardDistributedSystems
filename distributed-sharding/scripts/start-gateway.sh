#!/bin/bash

# Start the API Gateway
# Usage: ./start-gateway.sh [node1] [node2] [node3] [node4]
#
# Each node should be in the format host:port
#
# Example for local testing:
#   ./start-gateway.sh
#
# Example for distributed setup with 4 laptops:
#   ./start-gateway.sh 192.168.1.101:8001 192.168.1.102:8002 192.168.1.103:8003 192.168.1.104:8004

SCRIPT_DIR_ENV="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# Load .env file if it exists
if [ -f "$SCRIPT_DIR_ENV/../.env" ]; then
    source "$SCRIPT_DIR_ENV/../.env"
fi

NODE1=${1:-"${SHARD1_HOST:-localhost}:${SHARD1_PORT:-8001}"}
NODE2=${2:-"${SHARD2_HOST:-localhost}:${SHARD2_PORT:-8002}"}
NODE3=${3:-"${SHARD3_HOST:-localhost}:${SHARD3_PORT:-8003}"}
NODE4=${4:-"${SHARD4_HOST:-localhost}:${SHARD4_PORT:-8004}"}
PORT=${5:-${GATEWAY_PORT:-8000}}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "=========================================="
echo "  API Gateway"
echo "=========================================="
echo ""
echo "  Gateway Port: $PORT"
echo ""
echo "  Shard Nodes:"
echo "    Shard 1: $NODE1"
echo "    Shard 2: $NODE2"
echo "    Shard 3: $NODE3"
echo "    Shard 4: $NODE4"
echo ""

# Build if needed
if [ ! -f bin/gateway ]; then
    echo "Building gateway binary..."
    mkdir -p bin
    go build -o bin/gateway ./cmd/gateway
fi

echo "Starting API Gateway..."
echo ""
echo "Architecture:"
echo ""
echo "     Client"
echo "       |"
echo "    Gateway (:$PORT)"
echo "       |"
echo "  -------------------------"
echo "  |    |    |    |"
echo "  S1   S2   S3   S4"
echo ""
echo "Endpoints:"
echo "  POST /users     - Create user (auto-routed to correct shard)"
echo "  GET  /users/:id - Get user by ID"
echo "  GET  /users     - Get all users from all shards"
echo "  GET  /shards    - Get shard status"
echo ""

./bin/gateway -port=$PORT -node1="$NODE1" -node2="$NODE2" -node3="$NODE3" -node4="$NODE4"
