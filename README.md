# GizzardDistributedSystems

A distributed database sharding system built in **Go** with a **React** frontend dashboard. Demonstrates core distributed systems concepts including data sharding, an API gateway, and four classical distributed algorithms — all with a visual dashboard for real-time interaction.

---

## ✨ Features

- **Distributed Sharding** — User data is automatically partitioned across 4 SQLite-backed shard nodes using modulo-based routing
- **API Gateway** — Central entry point that routes requests to the correct shard, aggregates responses, and exposes algorithm endpoints
- **React Dashboard** — Vite-powered React frontend for managing users and visualizing algorithm behaviour
- **Multi-Machine Support** — Shard nodes can run on separate laptops over a local network

### Distributed Algorithms

| Algorithm | Description |
|-----------|-------------|
| **Vector Clocks** | Thread-safe causal ordering with event logging across nodes |
| **Chandy-Lamport Snapshot** | Consistent global state capture via marker propagation |
| **Bully Leader Election** | Highest-ID node wins leadership with timeout handling |
| **Consistent Hashing** | SHA-256 hash ring with 150 virtual nodes for balanced key distribution |

---

## 🏗️ Architecture

```
                     ┌──────────────────┐
                     │  React Frontend  │
                     │   (Vite :3000)   │
                     └────────┬─────────┘
                              │
                     ┌────────▼─────────┐
                     │   API Gateway    │
                     │    (:8000)       │
                     └──┬───┬───┬───┬──┘
                        │   │   │   │
              ┌─────────┘   │   │   └─────────┐
              ▼             ▼   ▼             ▼
         ┌─────────┐  ┌─────────┐  ┌─────────┐  ┌─────────┐
         │ Node 1  │  │ Node 2  │  │ Node 3  │  │ Node 4  │
         │ :8001   │  │ :8002   │  │ :8003   │  │ :8004   │
         │ SQLite  │  │ SQLite  │  │ SQLite  │  │ SQLite  │
         └─────────┘  └─────────┘  └─────────┘  └─────────┘
```

---

## 📂 Project Structure

```
GizzardDistributedSystems/
├── distributed-sharding/
│   ├── algorithms/            # Distributed algorithm implementations
│   │   ├── vector_clock.go
│   │   ├── snapshot.go
│   │   ├── leader_election.go
│   │   ├── consistent_hashing.go
│   │   └── *_test.go          # Unit tests (33 total)
│   ├── cmd/
│   │   ├── node/main.go       # Shard node binary
│   │   └── gateway/main.go    # API gateway binary
│   ├── handlers/              # HTTP request handlers
│   ├── models/                # Data models
│   ├── repository/            # Database access layer
│   ├── routes/                # Route definitions
│   ├── sharding/              # Shard manager & routing logic
│   ├── scripts/               # Helper scripts
│   ├── frontend/              # React + Vite dashboard
│   │   └── src/
│   │       ├── App.jsx        # Main app with algorithm dashboard
│   │       └── index.css      # Styling
│   ├── main.go                # Standalone single-server entry point
│   ├── Makefile               # Build & run commands
│   └── go.mod
├── DISTRIBUTED_SHARD_SETUP.md # Guide: run shards across multiple laptops
├── LAPTOP_SETUP_GUIDE.md      # Laptop-specific setup instructions
├── WINDOWS_SETUP_GUIDE.md     # Windows-specific setup instructions
└── LICENSE                    # MIT License
```

---

## 🚀 Quick Start

### Prerequisites

- **Go 1.25+**
- **GCC** (required for SQLite via `go-sqlite3`)
- **Node.js & npm** (for the React frontend)

### 1. Clone & Navigate

```bash
git clone https://github.com/<your-username>/GizzardDistributedSystems.git
cd GizzardDistributedSystems/distributed-sharding
```

### 2. Install Go Dependencies

```bash
go mod download
```

### 3. Build

```bash
make build
```

This compiles two binaries into `bin/`:
- `bin/node` — Shard node server
- `bin/gateway` — API gateway

### 4. Run the Entire System

```bash
make run-all
```

This starts 4 shard nodes (ports 8001–8004) and the gateway (port 8000).

### 5. Start the Frontend

```bash
cd frontend
npm install
npm run dev
```

Open **http://localhost:3000** in your browser.

---

## 📡 API Reference

### User CRUD (Gateway — `:8000`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/users` | Get all users (aggregated from all shards) |
| `GET` | `/users/:id` | Get user by ID |
| `POST` | `/users` | Create a new user |
| `PUT` | `/users/:id` | Update a user |
| `DELETE` | `/users/:id` | Delete a user |
| `GET` | `/shards` | View shard status and distribution |

### Algorithm Endpoints (Gateway — `:8000`)

| Method | Endpoint | Algorithm |
|--------|----------|-----------|
| `GET` | `/algorithms` | List all available algorithms |
| `GET` | `/clocks` | Vector Clocks — all node clocks |
| `GET` | `/events` | Vector Clocks — event logs |
| `POST` | `/snapshot` | Chandy-Lamport — initiate snapshot |
| `GET` | `/snapshot` | Chandy-Lamport — get results |
| `POST` | `/election/start` | Bully — trigger leader election |
| `GET` | `/election/leader` | Bully — current leader |
| `GET` | `/hash-ring/status` | Consistent Hashing — ring state |
| `POST` | `/hash-ring/lookup` | Consistent Hashing — key lookup |
| `POST` | `/hash-ring/add-node` | Consistent Hashing — add node |
| `DELETE` | `/hash-ring/remove-node/:id` | Consistent Hashing — remove node |

### Example Usage

```bash
# Create a user
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{"id":1,"name":"Alice","email":"alice@example.com"}'

# View shard distribution
curl http://localhost:8000/shards | jq

# Trigger a leader election
curl -X POST http://localhost:8000/election/start | jq

# Initiate a Chandy-Lamport snapshot
curl -X POST http://localhost:8000/snapshot | jq

# Look up a key on the hash ring
curl -X POST http://localhost:8000/hash-ring/lookup \
  -H "Content-Type: application/json" \
  -d '{"user_id": 42}' | jq
```

---

## 🧪 Testing

```bash
# Run all 33 unit tests
go test ./algorithms/ -v

# Vet for code issues
go vet ./...

# Smoke-test the running system
make test
```

---

## 🌐 Multi-Machine Deployment

Shard nodes can be distributed across different machines on the same network. See the detailed guides:

- [**DISTRIBUTED_SHARD_SETUP.md**](DISTRIBUTED_SHARD_SETUP.md) — Running shards on separate laptops
- [**LAPTOP_SETUP_GUIDE.md**](LAPTOP_SETUP_GUIDE.md) — Laptop-specific configuration
- [**WINDOWS_SETUP_GUIDE.md**](WINDOWS_SETUP_GUIDE.md) — Windows-specific setup

**Quick example** — run Shard 3 on a remote laptop:

```bash
# On the remote laptop
./bin/node -shard=3 -port=8003 -data=./data

# On the main laptop (replace IP)
./bin/gateway -port=8000 \
  -node1=localhost:8001 \
  -node2=localhost:8002 \
  -node3=192.168.1.105:8003 \
  -node4=localhost:8004
```

---

## 🛠️ Make Targets

| Command | Description |
|---------|-------------|
| `make build` | Build all binaries |
| `make run-all` | Start all nodes + gateway |
| `make run-node1` | Run shard node 1 on port 8001 |
| `make run-gateway` | Run the API gateway |
| `make stop` | Stop all running processes |
| `make clean` | Remove build artifacts and data |
| `make test` | Smoke-test the distributed system |

---

## 🔧 Tech Stack

| Component | Technology |
|-----------|------------|
| Backend | Go 1.25, Gin |
| Database | SQLite (per-shard) |
| Frontend | React, Vite |
| Algorithms | Pure Go |

---

## 📄 License

This project is licensed under the **MIT License** — see the [LICENSE](LICENSE) file for details.