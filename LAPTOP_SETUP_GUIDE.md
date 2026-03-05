# Gizzard Distributed Systems - Laptop Setup Guide

Complete step-by-step guide for setting up the distributed sharding system on a new laptop.

---

## Prerequisites

Install the following before proceeding:

### 1. Install Go (1.21+)

**macOS:**
```bash
brew install go
```

**Windows:**
Download from https://golang.org/dl/

**Linux:**
```bash
sudo apt update
sudo apt install golang-go
```

Verify installation:
```bash
go version
```

### 2. Install Node.js (18+)

**macOS:**
```bash
brew install node
```

**Windows/Linux:**
Download from https://nodejs.org/

Verify installation:
```bash
node --version
npm --version
```

### 3. Install GCC (Required for SQLite compilation)

**macOS:**
```bash
xcode-select --install
```

**Linux:**
```bash
sudo apt install build-essential
```

**Windows:**
Install MinGW or use WSL.

### 4. Install Git

**macOS:**
```bash
brew install git
```

**Windows/Linux:**
Download from https://git-scm.com/

---

## Step 1: Clone the Repository

```bash
git clone https://github.com/YOUR_USERNAME/GizzardDistributedSystems.git
cd GizzardDistributedSystems/distributed-sharding
```

---

## Step 2: Install Go Dependencies

```bash
go mod download
```

This downloads all Go dependencies including:
- Gin web framework
- SQLite driver

---

## Step 3: Build the Project

```bash
make build
```

**Expected output:**
```
Building shard node...
Building API gateway...
```

This creates binaries in the `bin/` directory:
- `bin/node` - Shard node server
- `bin/gateway` - API gateway server

---

## Step 4: Start Backend Services

### Option A: Start All at Once (Recommended)

```bash
make run-all
```

This starts:
- 4 shard nodes (ports 8001-8004)
- 1 API gateway (port 8000)

### Option B: Start Individually (for debugging)

**Terminal 1 - Shard Node 1:**
```bash
./bin/node -shard=1 -port=8001 -data=./data
```

**Terminal 2 - Shard Node 2:**
```bash
./bin/node -shard=2 -port=8002 -data=./data
```

**Terminal 3 - Shard Node 3:**
```bash
./bin/node -shard=3 -port=8003 -data=./data
```

**Terminal 4 - Shard Node 4:**
```bash
./bin/node -shard=4 -port=8004 -data=./data
```

**Terminal 5 - API Gateway:**
```bash
./bin/gateway -port=8000 -node1=localhost:8001 -node2=localhost:8002 -node3=localhost:8003 -node4=localhost:8004
```

---

## Step 5: Install Frontend Dependencies

Open a **new terminal**:

```bash
cd distributed-sharding/frontend
npm install
```

---

## Step 6: Start Frontend

```bash
npm run dev
```

**Expected output:**
```
VITE v5.4.21  ready in 332 ms

➜  Local:   http://localhost:3000/
```

---

## Step 7: Access the Application

Open your browser and go to:
```
http://localhost:3000
```

---

## Verify Everything Works

### Test API Endpoints

**Create a user:**
```bash
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{"id":1,"name":"Test User","email":"test@example.com"}'
```

**Get all users:**
```bash
curl http://localhost:8000/users
```

**Check shard status:**
```bash
curl http://localhost:8000/shards
```

### Run Automated Tests

```bash
make test
```

---

## Stopping Services

```bash
make stop
```

Or manually:
```bash
pkill -f "bin/node"
pkill -f "bin/gateway"
```

---

## Clean Up (Reset Everything)

```bash
make clean
```

This removes:
- Built binaries (`bin/`)
- SQLite database files (`data/`)

---

## Troubleshooting

### Port Already in Use

```bash
# Find process using a port (e.g., 8000)
lsof -i :8000

# Kill it
kill -9 <PID>
```

### Go Module Issues

```bash
go mod tidy
go mod download
```

### Frontend Not Connecting to Backend

Ensure the backend is running on port 8000. Check `frontend/vite.config.js` for proxy settings.

### Permission Denied on Scripts

```bash
chmod +x scripts/*.sh
```

### SQLite Build Errors

Ensure GCC is installed:
```bash
gcc --version
```

On macOS, run:
```bash
xcode-select --install
```

---

## Quick Reference Commands

| Command | Description |
|---------|-------------|
| `make build` | Build all binaries |
| `make run-all` | Start all backend services |
| `make stop` | Stop all services |
| `make clean` | Remove binaries and data |
| `make test` | Run integration tests |
| `cd frontend && npm run dev` | Start frontend |

---

## Architecture Overview

```
     React Frontend (localhost:3000)
              │
              ▼
       API Gateway (localhost:8000)
              │
    ┌─────────┼─────────┐
    ▼         ▼         ▼         ▼
 Node 1    Node 2    Node 3    Node 4
 :8001     :8002     :8003     :8004
shard1.db shard2.db shard3.db shard4.db
```

**Sharding Formula:** `shard = (userID - 1) % 4 + 1`

---

## Project Structure

```
distributed-sharding/
├── bin/              # Compiled binaries
├── cmd/
│   ├── gateway/      # API Gateway source
│   └── node/         # Shard Node source
├── data/             # SQLite databases
├── frontend/         # React frontend
├── handlers/         # HTTP handlers
├── models/           # Data models
├── repository/       # Database operations
├── routes/           # Route definitions
├── scripts/          # Shell scripts
├── sharding/         # Sharding logic
├── Makefile          # Build commands
└── go.mod            # Go dependencies
```

---

## Need Help?

1. Check existing documentation in `DISTRIBUTED_README.md` and `SETUP_GUIDE.md`
2. Ensure all prerequisites are installed with correct versions
3. Try `make clean && make build` to rebuild from scratch
