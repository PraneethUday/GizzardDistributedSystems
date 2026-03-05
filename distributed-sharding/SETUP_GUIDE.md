# Distributed Sharding System - Complete Setup Guide

A distributed database sharding system built with Go (Gin framework) and React, demonstrating horizontal data partitioning across multiple SQLite database shards.

## System Architecture

```
                    ┌─────────────────────────────────┐
                    │        React Frontend           │
                    │      http://localhost:3000      │
                    └───────────────┬─────────────────┘
                                    │
                                    ▼
                    ┌─────────────────────────────────┐
                    │         API Gateway             │
                    │      http://localhost:8000      │
                    │   Routes: (userID - 1) % 4      │
                    └───────────────┬─────────────────┘
                                    │
           ┌────────────┬───────────┼───────────┬────────────┐
           ▼            ▼           ▼           ▼            │
    ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐      │
    │  Node 1  │ │  Node 2  │ │  Node 3  │ │  Node 4  │      │
    │:8001     │ │:8002     │ │:8003     │ │:8004     │      │
    │shard1.db │ │shard2.db │ │shard3.db │ │shard4.db │      │
    │Users:    │ │Users:    │ │Users:    │ │Users:    │      │
    │1,5,9,13..│ │2,6,10,14.│ │3,7,11,15.│ │4,8,12,16.│      │
    └──────────┘ └──────────┘ └──────────┘ └──────────┘      │
```

## Prerequisites

- **Go** 1.21+ ([Download](https://golang.org/dl/))
- **Node.js** 18+ ([Download](https://nodejs.org/))
- **GCC** (for SQLite compilation on macOS: `xcode-select --install`)

## Directory Structure

```
distributed-sharding/
├── cmd/
│   ├── gateway/main.go    # API Gateway server
│   └── node/main.go       # Shard node server
├── frontend/
│   ├── src/
│   │   ├── App.jsx        # React application
│   │   ├── index.css      # Styles
│   │   └── main.jsx       # Entry point
│   ├── package.json
│   └── vite.config.js
├── handlers/
├── models/
├── repository/
├── routes/
├── sharding/
├── scripts/
├── Makefile
└── go.mod
```

---

## Quick Start (All-in-One)

### Step 1: Build Everything

```bash
cd distributed-sharding
make build
```

**Expected Output:**
```
Building node server...
Building gateway server...
Build complete! Binaries in ./bin/
```

### Step 2: Start All Backend Services

Open a terminal and run:

```bash
make run-all
```

**Expected Output:**
```
Starting shard nodes and gateway...
Starting Node 1 on port 8001...
Starting Node 2 on port 8002...
Starting Node 3 on port 8003...
Starting Node 4 on port 8004...
Starting Gateway on port 8000...
All services started!
```

### Step 3: Start Frontend

Open a **new terminal** and run:

```bash
cd distributed-sharding/frontend
npm install
npm run dev
```

**Expected Output:**
```
  VITE v5.4.21  ready in 332 ms

  ➜  Local:   http://localhost:3000/
  ➜  Network: use --host to expose
```

### Step 4: Open Browser

Navigate to: **http://localhost:3000**

---

## Manual Start (Step-by-Step)

If you prefer to start services individually:

### Terminal 1 - Node 1 (Shard 1)

```bash
cd distributed-sharding
./bin/node -shard=1 -port=8001 -data=./data
```

**Output:**
```
[GIN-debug] Listening and serving HTTP on :8001
Shard Node 1 running on port 8001
Data directory: ./data
```

### Terminal 2 - Node 2 (Shard 2)

```bash
cd distributed-sharding
./bin/node -shard=2 -port=8002 -data=./data
```

### Terminal 3 - Node 3 (Shard 3)

```bash
cd distributed-sharding
./bin/node -shard=3 -port=8003 -data=./data
```

### Terminal 4 - Node 4 (Shard 4)

```bash
cd distributed-sharding
./bin/node -shard=4 -port=8004 -data=./data
```

### Terminal 5 - Gateway

```bash
cd distributed-sharding
./bin/gateway -port=8000
```

**Output:**
```
[GIN-debug] Listening and serving HTTP on :8000
API Gateway running on port 8000
Connected to nodes: [8001, 8002, 8003, 8004]
```

### Terminal 6 - Frontend

```bash
cd distributed-sharding/frontend
npm run dev
```

---

## Testing with cURL

### Create Users

```bash
# User 1 → Shard 1
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{"id": 1, "name": "Alice", "email": "alice@example.com"}'
```

**Expected Output:**
```json
{
  "message": "User created successfully",
  "user": {
    "id": 1,
    "name": "Alice",
    "email": "alice@example.com"
  },
  "shard": 1
}
```

```bash
# User 2 → Shard 2
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{"id": 2, "name": "Bob", "email": "bob@example.com"}'
```

**Expected Output:**
```json
{
  "message": "User created successfully",
  "user": {
    "id": 2,
    "name": "Bob",
    "email": "bob@example.com"
  },
  "shard": 2
}
```

```bash
# User 5 → Shard 1 (because (5-1) % 4 = 0 → shard 1)
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{"id": 5, "name": "Eve", "email": "eve@example.com"}'
```

**Expected Output:**
```json
{
  "message": "User created successfully",
  "user": {
    "id": 5,
    "name": "Eve",
    "email": "eve@example.com"
  },
  "shard": 1
}
```

### Fetch a User

```bash
curl http://localhost:8000/users/1
```

**Expected Output:**
```json
{
  "user": {
    "id": 1,
    "name": "Alice",
    "email": "alice@example.com"
  },
  "shard": 1
}
```

### Get All Users (Aggregated from All Shards)

```bash
curl http://localhost:8000/users
```

**Expected Output:**
```json
{
  "users": [
    {"id": 1, "name": "Alice", "email": "alice@example.com", "shard": 1},
    {"id": 5, "name": "Eve", "email": "eve@example.com", "shard": 1},
    {"id": 2, "name": "Bob", "email": "bob@example.com", "shard": 2}
  ],
  "total": 3
}
```

### Check Shard Status

```bash
curl http://localhost:8000/shards
```

**Expected Output:**
```json
{
  "shards": [
    {"id": 1, "port": 8001, "status": "online", "user_count": 2},
    {"id": 2, "port": 8002, "status": "online", "user_count": 1},
    {"id": 3, "port": 8003, "status": "online", "user_count": 0},
    {"id": 4, "port": 8004, "status": "online", "user_count": 0}
  ]
}
```

---

## Sharding Formula

The system uses consistent hashing to determine which shard stores each user:

```
Shard Number = ((userID - 1) % 4) + 1
```

| User ID | Calculation | Shard |
|---------|-------------|-------|
| 1       | (1-1) % 4 + 1 = 1 | Shard 1 |
| 2       | (2-1) % 4 + 1 = 2 | Shard 2 |
| 3       | (3-1) % 4 + 1 = 3 | Shard 3 |
| 4       | (4-1) % 4 + 1 = 4 | Shard 4 |
| 5       | (5-1) % 4 + 1 = 1 | Shard 1 |
| 6       | (6-1) % 4 + 1 = 2 | Shard 2 |
| 100     | (100-1) % 4 + 1 = 4 | Shard 4 |

---

## Frontend Features

### 1. Create User Form
- Enter User ID, Name, and Email
- Shows which shard the user was stored in
- Displays success/error messages

### 2. Fetch User Form
- Look up any user by ID
- Shows user details and shard location

### 3. Shard Status Dashboard
- Visual grid showing all 4 shards
- User count per shard
- Online/offline status indicators
- Color-coded shard cards

### 4. All Users Table
- Aggregated view of all users across shards
- Color-coded shard badges
- Sortable columns

---

## Stopping Services

### Stop All Services

```bash
make stop
```

Or manually:

```bash
pkill -f "bin/node"
pkill -f "bin/gateway"
```

### Stop Frontend

Press `Ctrl+C` in the frontend terminal.

---

## Clean Up

Remove all build artifacts and data:

```bash
make clean
```

This removes:
- `./bin/` directory
- `./data/` directory (all SQLite databases)
- `./server` binary

---

## Troubleshooting

### Port Already in Use

```bash
# Find process using port 8000
lsof -i :8000

# Kill it
kill -9 <PID>

# Or kill all node/gateway processes
pkill -f "bin/node" && pkill -f "bin/gateway"
```

### CGO Disabled Error

If you see `Binary was compiled with 'CGO_ENABLED=0'`:

```bash
CGO_ENABLED=1 go build -o bin/node ./cmd/node
CGO_ENABLED=1 go build -o bin/gateway ./cmd/gateway
```

### Frontend Can't Connect to Backend

1. Ensure gateway is running on port 8000
2. Check browser console for CORS errors
3. Verify `vite.config.js` has correct proxy settings

### Database Locked

If you see "database is locked" errors:
```bash
# Remove data directory and restart
rm -rf data/
make run-all
```

---

## API Endpoints

| Method | Endpoint | Description |
|--------|----------|-------------|
| POST | `/users` | Create a new user |
| GET | `/users/:id` | Get user by ID |
| GET | `/users` | Get all users from all shards |
| GET | `/shards` | Get shard status information |

---

## Distributed Deployment

To run shards on different machines:

### Machine 1 (Gateway + Shard 1)
```bash
./bin/node -shard=1 -port=8001 -data=./data &
./bin/gateway -port=8000 -nodes="localhost:8001,192.168.1.102:8002,192.168.1.103:8003,192.168.1.104:8004"
```

### Machine 2 (Shard 2)
```bash
./bin/node -shard=2 -port=8002 -data=./data
```

### Machine 3 (Shard 3)
```bash
./bin/node -shard=3 -port=8003 -data=./data
```

### Machine 4 (Shard 4)
```bash
./bin/node -shard=4 -port=8004 -data=./data
```

---

## Screenshots

### React Dashboard
The frontend provides a visual interface showing:
- Real-time shard status
- User distribution across shards
- Create/fetch user forms with instant feedback

### Expected Browser View
When you open http://localhost:3000, you'll see:
1. **Header**: "Distributed Sharding System" title
2. **Shard Status Grid**: 4 colored cards (green for online, gray for offline)
3. **Forms Section**: Create User and Fetch User side by side
4. **Users Table**: All users with shard indicators
5. **Architecture Diagram**: ASCII visualization of the system

---

## Summary

| Service | Port | Purpose |
|---------|------|---------|
| Frontend | 3000 | React UI |
| Gateway | 8000 | API routing |
| Node 1 | 8001 | Shard 1 (users 1,5,9...) |
| Node 2 | 8002 | Shard 2 (users 2,6,10...) |
| Node 3 | 8003 | Shard 3 (users 3,7,11...) |
| Node 4 | 8004 | Shard 4 (users 4,8,12...) |
