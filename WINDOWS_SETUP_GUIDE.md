# Gizzard Distributed Systems - Windows Setup Guide

Complete step-by-step guide for setting up the distributed sharding system on a Windows laptop.

---

## Prerequisites

### 1. Install Go (1.21+)

1. Download from https://golang.org/dl/
2. Run the `.msi` installer
3. Follow installation wizard (use default settings)
4. Restart your terminal/PowerShell

**Verify installation:**

```powershell
go version
```

### 2. Install Node.js (18+)

1. Download LTS version from https://nodejs.org/
2. Run the `.msi` installer
3. Check "Automatically install necessary tools" if prompted
4. Restart your terminal/PowerShell

**Verify installation:**

```powershell
node --version
npm --version
```

### 3. Install GCC (Required for SQLite)

**Option A: Using MSYS2 (Recommended)**

1. Download MSYS2 from https://www.msys2.org/
2. Run the installer
3. Open "MSYS2 MINGW64" terminal and run:
   ```bash
   pacman -S mingw-w64-x86_64-gcc
   ```
4. Add to PATH: `C:\msys64\mingw64\bin`

**Option B: Using Chocolatey**

```powershell
choco install mingw
```

**Option C: Using Scoop**

```powershell
scoop install gcc
```

**Verify installation:**

```powershell
gcc --version
```

### 4. Install Git

1. Download from https://git-scm.com/download/win
2. Run the installer
3. Use recommended settings (Git Bash, etc.)

**Verify installation:**

```powershell
git --version
```

### 5. Install Make (Optional but Recommended)

**Using Chocolatey:**

```powershell
choco install make
```

**Using Scoop:**

```powershell
scoop install make
```

**Alternative:** Use Git Bash which includes make.

---

## Step 1: Clone the Repository

**PowerShell or Git Bash:**

```powershell
git clone https://github.com/YOUR_USERNAME/GizzardDistributedSystems.git
cd GizzardDistributedSystems\distributed-sharding
```

---

## Step 2: Install Go Dependencies

```powershell
go mod download
```

---

## Step 3: Build the Project

**Option A: Using Make (if installed)**

```powershell
make build
```

**Option B: Manual Build (PowerShell)**

```powershell
# Create bin directory
New-Item -ItemType Directory -Force -Path bin

# Build shard node
go build -o bin/node.exe ./cmd/node

# Build gateway
go build -o bin/gateway.exe ./cmd/gateway
```

**Option C: Using Git Bash**

```bash
mkdir -p bin
go build -o bin/node.exe ./cmd/node
go build -o bin/gateway.exe ./cmd/gateway
```

---

## Step 4: Create Data Directory

```powershell
New-Item -ItemType Directory -Force -Path data
```

Or in Git Bash:

```bash
mkdir -p data
```

---

## Step 5: Start Backend Services

### Option A: Start All at Once (Git Bash)

```bash
# In Git Bash
./bin/node.exe -shard=1 -port=8001 -data=./data &
./bin/node.exe -shard=2 -port=8002 -data=./data &
./bin/node.exe -shard=3 -port=8003 -data=./data &
./bin/node.exe -shard=4 -port=8004 -data=./data &
sleep 2
./bin/gateway.exe -port=8000 -node1=localhost:8001 -node2=localhost:8002 -node3=localhost:8003 -node4=localhost:8004
```

### Option B: Start Individually (PowerShell - Recommended)

Open **5 separate PowerShell windows**:

**Window 1 - Shard Node 1:**

```powershell
cd GizzardDistributedSystems\distributed-sharding
.\bin\node.exe -shard=1 -port=8001 -data=.\data
```

**Window 2 - Shard Node 2:**

```powershell
cd GizzardDistributedSystems\distributed-sharding
.\bin\node.exe -shard=2 -port=8002 -data=.\data
```

**Window 3 - Shard Node 3:**

```powershell
cd GizzardDistributedSystems\distributed-sharding
.\bin\node.exe -shard=3 -port=8003 -data=.\data
```

**Window 4 - Shard Node 4:**

```powershell
cd GizzardDistributedSystems\distributed-sharding
.\bin\node.exe -shard=4 -port=8004 -data=.\data
```

**Window 5 - API Gateway:**

```powershell
cd GizzardDistributedSystems\distributed-sharding
.\bin\gateway.exe -port=8000 -node1=localhost:8001 -node2=localhost:8002 -node3=localhost:8003 -node4=localhost:8004
```

### Option C: Using PowerShell Jobs (Single Window)

```powershell
cd GizzardDistributedSystems\distributed-sharding

# Start nodes as background jobs
Start-Job -ScriptBlock { Set-Location $using:PWD; .\bin\node.exe -shard=1 -port=8001 -data=.\data }
Start-Job -ScriptBlock { Set-Location $using:PWD; .\bin\node.exe -shard=2 -port=8002 -data=.\data }
Start-Job -ScriptBlock { Set-Location $using:PWD; .\bin\node.exe -shard=3 -port=8003 -data=.\data }
Start-Job -ScriptBlock { Set-Location $using:PWD; .\bin\node.exe -shard=4 -port=8004 -data=.\data }

# Wait for nodes to start
Start-Sleep -Seconds 2

# Start gateway in foreground
.\bin\gateway.exe -port=8000 -node1=localhost:8001 -node2=localhost:8002 -node3=localhost:8003 -node4=localhost:8004
```

---

## Step 6: Install Frontend Dependencies

Open a **new PowerShell window**:

```powershell
cd GizzardDistributedSystems\distributed-sharding\frontend
npm install
```

---

## Step 7: Start Frontend

```powershell
npm run dev
```

**Expected output:**

```
VITE v5.4.21  ready in 332 ms

➜  Local:   http://localhost:3000/
```

---

## Step 8: Access the Application

Open your browser and go to:

```
http://localhost:3000
```

---

## Verify Everything Works

### Test Using PowerShell

**Create a user:**

```powershell
Invoke-RestMethod -Uri "http://localhost:8000/users" -Method Post -ContentType "application/json" -Body '{"id":1,"name":"Test User","email":"test@example.com"}'
```

**Get all users:**

```powershell
Invoke-RestMethod -Uri "http://localhost:8000/users"
```

**Check shard status:**

```powershell
Invoke-RestMethod -Uri "http://localhost:8000/shards"
```

### Test Using curl (Git Bash)

```bash
# Create a user
curl -X POST http://localhost:8000/users -H "Content-Type: application/json" -d '{"id":1,"name":"Test User","email":"test@example.com"}'

# Get all users
curl http://localhost:8000/users

# Check shard status
curl http://localhost:8000/shards
```

---

## Stopping Services

### Kill All Processes

**PowerShell:**

```powershell
# Stop node processes
Get-Process node -ErrorAction SilentlyContinue | Stop-Process -Force

# Stop gateway process
Get-Process gateway -ErrorAction SilentlyContinue | Stop-Process -Force

# Stop PowerShell jobs (if using jobs)
Get-Job | Stop-Job
Get-Job | Remove-Job
```

**Or simply close all PowerShell windows running the services.**

---

## Clean Up (Reset Everything)

**PowerShell:**

```powershell
cd GizzardDistributedSystems\distributed-sharding

# Remove binaries
Remove-Item -Recurse -Force bin -ErrorAction SilentlyContinue

# Remove database files
Remove-Item -Recurse -Force data -ErrorAction SilentlyContinue
```

---

## Troubleshooting

### "go: command not found"

Add Go to your PATH:

1. Open System Properties → Advanced → Environment Variables
2. Add `C:\Go\bin` to PATH
3. Restart PowerShell

### "gcc: command not found"

Ensure GCC is in your PATH:

1. If using MSYS2, add `C:\msys64\mingw64\bin` to PATH
2. Restart PowerShell

### CGO_ENABLED Error

Set environment variable before building:

```powershell
$env:CGO_ENABLED=1
go build -o bin/node.exe ./cmd/node
```

### Port Already in Use

```powershell
# Find process using port 8000
netstat -ano | findstr :8000

# Kill by PID (replace 1234 with actual PID)
taskkill /PID 1234 /F
```

### Windows Firewall Blocking

Allow the apps through Windows Firewall:

1. Open Windows Security → Firewall & network protection
2. Click "Allow an app through firewall"
3. Add `node.exe` and `gateway.exe`

### Go Module Issues

```powershell
go mod tidy
go mod download
```

### Frontend Not Connecting

Ensure backend is running on port 8000. Check browser console for CORS errors.

---

## Quick Reference Commands (PowerShell)

| Command                                           | Description    |
| ------------------------------------------------- | -------------- |
| `go build -o bin/node.exe ./cmd/node`             | Build node     |
| `go build -o bin/gateway.exe ./cmd/gateway`       | Build gateway  |
| `.\bin\node.exe -shard=1 -port=8001 -data=.\data` | Start node 1   |
| `.\bin\gateway.exe -port=8000 ...`                | Start gateway  |
| `Get-Process node \| Stop-Process`                | Stop all nodes |
| `cd frontend; npm run dev`                        | Start frontend |

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

## Windows Batch Script (Optional)

Create `start-all.bat` in the `distributed-sharding` folder:

```batch
@echo off
echo Building binaries...
if not exist bin mkdir bin
go build -o bin\node.exe .\cmd\node
go build -o bin\gateway.exe .\cmd\gateway

echo Creating data directory...
if not exist data mkdir data

echo Starting shard nodes...
start "Node 1" cmd /k ".\bin\node.exe -shard=1 -port=8001 -data=.\data"
start "Node 2" cmd /k ".\bin\node.exe -shard=2 -port=8002 -data=.\data"
start "Node 3" cmd /k ".\bin\node.exe -shard=3 -port=8003 -data=.\data"
start "Node 4" cmd /k ".\bin\node.exe -shard=4 -port=8004 -data=.\data"

echo Waiting for nodes to start...
timeout /t 3 /nobreak >nul

echo Starting API Gateway...
.\bin\gateway.exe -port=8000 -node1=localhost:8001 -node2=localhost:8002 -node3=localhost:8003 -node4=localhost:8004
```

Run with:

```powershell
.\start-all.bat
```

---

## Need Help?

1. Use Git Bash for a Unix-like terminal experience
2. Ensure all prerequisites have correct PATH settings
3. Try rebuilding: delete `bin\` folder and rebuild
4. Check Windows Firewall settings
