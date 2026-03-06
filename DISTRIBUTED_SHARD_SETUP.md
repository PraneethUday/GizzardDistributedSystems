# Running Shard 3 on a Separate Laptop

This guide explains how to run Shard 3 on a different laptop while keeping other shards and the gateway on your main laptop, with the React frontend still working.

---

## Overview

```
┌─────────────────────────────────────────────────────────────────┐
│                        MAIN LAPTOP                               │
│  React Frontend (localhost:3000)                                 │
│         │                                                        │
│         ▼                                                        │
│  API Gateway (localhost:8000)                                    │
│         │                                                        │
│    ┌────┴────┬─────────────────┐                                │
│    ▼         ▼                 ▼                                │
│  Node 1   Node 2            Node 4                              │
│  :8001    :8002             :8004                               │
└────┬────────┬─────────────────┬─────────────────────────────────┘
     │        │                 │
     │        │    ┌────────────┼──── Network Connection ────┐
     │        │    │            │                            │
     │        │    │  ┌─────────────────────────────────┐    │
     │        │    │  │      SECOND LAPTOP              │    │
     │        │    │  │                                 │    │
     │        │    └──┼───▶  Node 3 (:8003)             │    │
     │        │       │      192.168.x.x:8003           │    │
     │        │       └─────────────────────────────────┘    │
     │        │                                              │
     └────────┴──────────────────────────────────────────────┘
```

---

## Prerequisites

1. Both laptops must be on the **same network** (WiFi or LAN)
2. Second laptop needs **Go** and **GCC** installed
3. Know the **IP address** of the second laptop

---

## Step 1: Find the Second Laptop's IP Address

### On the Second Laptop:

**macOS:**

```bash
ipconfig getifaddr en0
```

**Linux:**

```bash
hostname -I | awk '{print $1}'
```

**Windows (PowerShell):**

```powershell
(Get-NetIPAddress -AddressFamily IPv4 | Where-Object {$_.InterfaceAlias -notlike "*Loopback*"}).IPAddress
```

**Example output:** `192.168.1.105`

Write down this IP address - you'll need it later.

---

## Step 2: Setup the Second Laptop (Shard 3)

### 2.1 Clone the Repository

```bash
git clone https://github.com/YOUR_USERNAME/GizzardDistributedSystems.git
cd GizzardDistributedSystems/distributed-sharding
```

### 2.2 Install Dependencies

```bash
go mod download
```

### 2.3 Build the Node Binary

```bash
go build -o bin/node ./cmd/node
```

Or using Make:

```bash
make build-node
```

### 2.4 Create Data Directory

```bash
mkdir -p data
```

### 2.5 Start Shard 3

```bash
./bin/node -shard=3 -port=8003 -data=./data
```

**Expected output:**

```
[GIN-debug] Listening and serving HTTP on :8003
Shard Node 3 running on port 8003
```

**Keep this terminal running!**

---

## Step 3: Configure Firewall on Second Laptop

### macOS

The firewall popup should appear automatically. Click **"Allow"**.

Or manually:

```bash
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --add $(pwd)/bin/node
sudo /usr/libexec/ApplicationFirewall/socketfilterfw --unblockapp $(pwd)/bin/node
```

### Linux

```bash
sudo ufw allow 8003/tcp
```

### Windows

```powershell
New-NetFirewallRule -DisplayName "Shard Node 3" -Direction Inbound -LocalPort 8003 -Protocol TCP -Action Allow
```

Or manually:

1. Open Windows Defender Firewall
2. Click "Allow an app through firewall"
3. Add `node.exe` and allow on Private networks

---

## Step 4: Test Connection Between Laptops

### From Main Laptop

Replace `192.168.1.105` with the second laptop's actual IP:

```bash
curl http://192.168.1.105:8003/health
```

**Expected response:**

```json
{ "status": "ok", "shard": 3 }
```

If this doesn't work, check:

- Both laptops are on the same network
- Firewall is allowing port 8003
- Shard 3 is running on the second laptop

---

## Step 5: Start Services on Main Laptop

### 5.1 Start Shards 1, 2, and 4

```bash
cd GizzardDistributedSystems/distributed-sharding

# Build if not already built
make build

# Start shards 1, 2, 4 (NOT shard 3)
./bin/node -shard=1 -port=8001 -data=./data &
./bin/node -shard=2 -port=8002 -data=./data &
./bin/node -shard=4 -port=8004 -data=./data &
```

### 5.2 Start Gateway with Remote Shard 3

**Replace `192.168.1.105` with the second laptop's actual IP:**

```bash
./bin/gateway -port=8000 \
  -node1=localhost:8001 \
  -node2=localhost:8002 \
  -node3=192.168.1.105:8003 \
  -node4=localhost:8004
```

**Expected output:**

```
API Gateway starting on port 8000
Shard 1: localhost:8001
Shard 2: localhost:8002
Shard 3: 192.168.1.105:8003  <-- Remote!
Shard 4: localhost:8004
```

---

## Step 6: Start the Frontend

On the main laptop:

```bash
cd distributed-sharding/frontend
npm install  # if not already done
npm run dev
```

---

## Step 7: Test the Distributed System

### Open Browser

Go to: `http://localhost:3000`

### Create Users via API

```bash
# User 1 → Shard 1 (Main laptop)
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{"id":1,"name":"Alice","email":"alice@test.com"}'

# User 2 → Shard 2 (Main laptop)
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{"id":2,"name":"Bob","email":"bob@test.com"}'

# User 3 → Shard 3 (SECOND LAPTOP!)
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{"id":3,"name":"Charlie","email":"charlie@test.com"}'

# User 4 → Shard 4 (Main laptop)
curl -X POST http://localhost:8000/users \
  -H "Content-Type: application/json" \
  -d '{"id":4,"name":"Diana","email":"diana@test.com"}'
```

### Verify Shard Distribution

```bash
curl http://localhost:8000/shards
```

**Expected output:**

```json
{
  "shards": [
    {
      "shard": 1,
      "address": "localhost:8001",
      "user_count": 1,
      "status": "healthy"
    },
    {
      "shard": 2,
      "address": "localhost:8002",
      "user_count": 1,
      "status": "healthy"
    },
    {
      "shard": 3,
      "address": "192.168.1.105:8003",
      "user_count": 1,
      "status": "healthy"
    },
    {
      "shard": 4,
      "address": "localhost:8004",
      "user_count": 1,
      "status": "healthy"
    }
  ]
}
```

### Get All Users

```bash
curl http://localhost:8000/users
```

Should return all 4 users, including Charlie from the remote shard!

---

## Quick Reference

### Second Laptop Commands

```bash
# Start Shard 3
./bin/node -shard=3 -port=8003 -data=./data
```

### Main Laptop Commands

```bash
# Start Shards 1, 2, 4
./bin/node -shard=1 -port=8001 -data=./data &
./bin/node -shard=2 -port=8002 -data=./data &
./bin/node -shard=4 -port=8004 -data=./data &

# Start Gateway (replace IP)
./bin/gateway -port=8000 \
  -node1=localhost:8001 \
  -node2=localhost:8002 \
  -node3=SECOND_LAPTOP_IP:8003 \
  -node4=localhost:8004

# Start Frontend
cd frontend && npm run dev
```

---

## Troubleshooting

### "Connection refused" to Shard 3

1. **Check if Shard 3 is running** on the second laptop
2. **Verify the IP address** is correct
3. **Test network connectivity:**
   ```bash
   ping 192.168.1.105
   ```
4. **Check firewall** on second laptop

### Shard 3 Shows "unhealthy"

The gateway health check failed. Verify:

- Shard 3 is running: `curl http://SECOND_LAPTOP_IP:8003/health`
- No firewall blocking
- Same network/subnet

### Frontend Shows "Error fetching data"

1. Check gateway is running on port 8000
2. Check browser console for errors
3. Verify all shards are responding

### Users Not Appearing from Shard 3

1. Ensure user ID follows sharding formula: IDs 3, 7, 11, 15... go to Shard 3
2. Check Shard 3 is connected: `curl http://localhost:8000/shards`

---

## Stopping Services

### Main Laptop

```bash
# Stop all local processes
pkill -f "bin/node"
pkill -f "bin/gateway"
```

### Second Laptop

Press `Ctrl+C` in the terminal running Shard 3.

---

## Extending to More Laptops

You can run each shard on a different laptop:

| Laptop   | Role               | Command                                                                                        |
| -------- | ------------------ | ---------------------------------------------------------------------------------------------- |
| Laptop A | Gateway + Frontend | `./bin/gateway -port=8000 -node1=IP_B:8001 -node2=IP_C:8002 -node3=IP_D:8003 -node4=IP_E:8004` |
| Laptop B | Shard 1            | `./bin/node -shard=1 -port=8001`                                                               |
| Laptop C | Shard 2            | `./bin/node -shard=2 -port=8002`                                                               |
| Laptop D | Shard 3            | `./bin/node -shard=3 -port=8003`                                                               |
| Laptop E | Shard 4            | `./bin/node -shard=4 -port=8004`                                                               |

Just replace the IPs in the gateway command with each laptop's actual IP address.

---

## Network Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Same WiFi Network                         │
│                   (192.168.1.0/24)                           │
│                                                              │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────┐   │
│  │ Main Laptop  │    │   Laptop 2   │    │   Laptop 3   │   │
│  │ 192.168.1.100│    │ 192.168.1.105│    │ 192.168.1.110│   │
│  │              │    │              │    │              │   │
│  │ Gateway:8000 │◄───│              │    │              │   │
│  │ Node 1:8001  │    │ Node 3:8003  │    │ Node 2:8002  │   │
│  │ Node 4:8004  │    │              │    │              │   │
│  │ Frontend:3000│    │              │    │              │   │
│  └──────────────┘    └──────────────┘    └──────────────┘   │
│                                                              │
└─────────────────────────────────────────────────────────────┘
```
