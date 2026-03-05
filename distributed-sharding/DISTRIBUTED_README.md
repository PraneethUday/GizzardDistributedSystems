# Distributed Sharding System

A distributed database sharding system built in Go. Each shard runs as an independent server that can be deployed on separate laptops/machines.

## Architecture

```
     Client
       |
    Gateway (port 8000)
       |
  -------------------------
  |    |    |    |
  S1   S2   S3   S4
 8001 8002 8003 8004
```

- **Gateway**: Central API router that receives all client requests and forwards them to the correct shard
- **Shard Nodes**: Independent servers, each owning one SQLite database

## Sharding Formula

```
shard = (userID - 1) % 4 + 1
```

| User ID | Shard |
|---------|-------|
| 1       | 1     |
| 2       | 2     |
| 3       | 3     |
| 4       | 4     |
| 5       | 1     |
| 6       | 2     |
| ...     | ...   |

## Quick Start (Local Testing)

Run everything on one machine:

```bash
# Option 1: Using the start script
./scripts/start-all.sh

# Option 2: Using Make
make run-all

# Option 3: Manual
make build
./bin/node -shard=1 -port=8001 &
./bin/node -shard=2 -port=8002 &
./bin/node -shard=3 -port=8003 &
./bin/node -shard=4 -port=8004 &
./bin/gateway -port=8000
```

## Distributed Setup (4 Laptops)

### Laptop 1 (Shard 1)
```bash
./scripts/start-node.sh 1 8001
# Or: ./bin/node -shard=1 -port=8001
```

### Laptop 2 (Shard 2)
```bash
./scripts/start-node.sh 2 8002
# Or: ./bin/node -shard=2 -port=8002
```

### Laptop 3 (Shard 3)
```bash
./scripts/start-node.sh 3 8003
# Or: ./bin/node -shard=3 -port=8003
```

### Laptop 4 (Shard 4)
```bash
./scripts/start-node.sh 4 8004
# Or: ./bin/node -shard=4 -port=8004
```

### Gateway (Any machine)
```bash
# Replace IPs with actual laptop IP addresses
./scripts/start-gateway.sh 192.168.1.101:8001 192.168.1.102:8002 192.168.1.103:8003 192.168.1.104:8004

# Or directly:
./bin/gateway -port=8000 \
  -node1=192.168.1.101:8001 \
  -node2=192.168.1.102:8002 \
  -node3=192.168.1.103:8003 \
  -node4=192.168.1.104:8004
```

## API Endpoints

### Gateway (port 8000)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /users | Create a user (auto-routed to correct shard) |
| GET | /users/:id | Get user by ID (auto-routed to correct shard) |
| GET | /users | Get all users from all shards |
| GET | /shards | Get shard status and user counts |
| GET | /health | Gateway health check |

### Shard Node (ports 8001-8004)

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | /insert | Insert a user into this shard |
| GET | /user/:id | Get user by ID from this shard |
| GET | /users | Get all users from this shard |
| GET | /health | Node health check |

## Examples

### Create Users (via Gateway)
```bash
# User 1 → Shard 1
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{"id":1,"name":"Alice","email":"alice@example.com"}'

# User 2 → Shard 2
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{"id":2,"name":"Bob","email":"bob@example.com"}'
```

### Get User
```bash
curl http://localhost:8000/users/1
```

### Get All Users
```bash
curl http://localhost:8000/users
```

### Check Shard Status
```bash
curl http://localhost:8000/shards
```

### Direct Shard Access (for testing)
```bash
# Insert directly into shard 1
curl -X POST http://localhost:8001/insert \
  -H "Content-Type: application/json" \
  -d '{"id":1,"name":"Alice","email":"alice@example.com"}'

# Get from shard 1
curl http://localhost:8001/user/1
```

## Building

```bash
# Build everything
make build

# Build only node
make build-node

# Build only gateway
make build-gateway
```

## Make Commands

```bash
make build        # Build all binaries
make run-all      # Run all nodes and gateway locally
make run-node1    # Run shard node 1
make run-node2    # Run shard node 2
make run-node3    # Run shard node 3
make run-node4    # Run shard node 4
make run-gateway  # Run API gateway
make stop         # Stop all processes
make clean        # Clean build artifacts and data
make test         # Test the distributed system
```

## File Structure

```
distributed-sharding/
├── cmd/
│   ├── node/
│   │   └── main.go      # Shard node server
│   └── gateway/
│       └── main.go      # API gateway
├── scripts/
│   ├── start-all.sh     # Start everything locally
│   ├── start-node.sh    # Start a single node
│   └── start-gateway.sh # Start the gateway
├── data/                # SQLite databases (auto-created)
│   ├── shard1.db
│   ├── shard2.db
│   ├── shard3.db
│   └── shard4.db
├── bin/                 # Compiled binaries (auto-created)
│   ├── node
│   └── gateway
├── Makefile
└── README.md
```
