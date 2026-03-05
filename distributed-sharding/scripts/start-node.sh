#!/bin/bash

# Start a single shard node
# Usage: ./start-node.sh <shard_id> <port>
#
# Example:
#   ./start-node.sh 1 8001
#   ./start-node.sh 2 8002

SHARD_ID=${1:-1}
PORT=${2:-$((8000 + SHARD_ID))}
DATA_DIR=${3:-"./data"}

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR/.."

echo "=========================================="
echo "  Shard Node $SHARD_ID"
echo "=========================================="
echo ""
echo "  Port: $PORT"
echo "  Data: $DATA_DIR/shard$SHARD_ID.db"
echo ""

# Build if needed
if [ ! -f bin/node ]; then
    echo "Building node binary..."
    mkdir -p bin
    go build -o bin/node ./cmd/node
fi

# Create data directory
mkdir -p "$DATA_DIR"

# Start the node
echo "Starting shard node $SHARD_ID on port $PORT..."
echo ""
./bin/node -shard=$SHARD_ID -port=$PORT -data="$DATA_DIR"
