.PHONY: build build-node build-gateway run-all run-node1 run-node2 run-node3 run-node4 run-gateway stop clean test

# Build all binaries
build: build-node build-gateway

build-node:
	@echo "Building shard node..."
	go build -o bin/node ./cmd/node

build-gateway:
	@echo "Building API gateway..."
	go build -o bin/gateway ./cmd/gateway

# Run individual components
run-node1: build-node
	./bin/node -shard=1 -port=8001 -data=./data

run-node2: build-node
	./bin/node -shard=2 -port=8002 -data=./data

run-node3: build-node
	./bin/node -shard=3 -port=8003 -data=./data

run-node4: build-node
	./bin/node -shard=4 -port=8004 -data=./data

run-gateway: build-gateway
	./bin/gateway -port=8000 -node1=localhost:8001 -node2=localhost:8002 -node3=localhost:8003 -node4=localhost:8004

# Run all nodes in background and gateway in foreground
run-all: build
	@echo "Starting all shard nodes..."
	@./bin/node -shard=1 -port=8001 -data=./data &
	@./bin/node -shard=2 -port=8002 -data=./data &
	@./bin/node -shard=3 -port=8003 -data=./data &
	@./bin/node -shard=4 -port=8004 -data=./data &
	@sleep 2
	@echo "Starting API gateway..."
	./bin/gateway -port=8000

# Stop all running processes
stop:
	@echo "Stopping all processes..."
	@pkill -f "bin/node" || true
	@pkill -f "bin/gateway" || true
	@echo "All processes stopped."

# Clean build artifacts and data
clean:
	@echo "Cleaning..."
	@rm -rf bin/
	@rm -rf data/
	@echo "Cleaned."

# Test the distributed system
test:
	@echo "Testing distributed system..."
	@echo "\n=== Creating users 1-4 ==="
	@curl -s -X POST http://localhost:8000/users -H "Content-Type: application/json" -d '{"id":1,"name":"User 1","email":"user1@example.com"}' | jq -c
	@curl -s -X POST http://localhost:8000/users -H "Content-Type: application/json" -d '{"id":2,"name":"User 2","email":"user2@example.com"}' | jq -c
	@curl -s -X POST http://localhost:8000/users -H "Content-Type: application/json" -d '{"id":3,"name":"User 3","email":"user3@example.com"}' | jq -c
	@curl -s -X POST http://localhost:8000/users -H "Content-Type: application/json" -d '{"id":4,"name":"User 4","email":"user4@example.com"}' | jq -c
	@echo "\n=== Getting shard status ==="
	@curl -s http://localhost:8000/shards | jq
	@echo "\n=== Getting all users ==="
	@curl -s http://localhost:8000/users | jq

# Help
help:
	@echo "Distributed Sharding System"
	@echo ""
	@echo "Usage:"
	@echo "  make build        - Build all binaries"
	@echo "  make run-all      - Run all nodes and gateway"
	@echo "  make run-node1    - Run shard node 1 on port 8001"
	@echo "  make run-node2    - Run shard node 2 on port 8002"
	@echo "  make run-node3    - Run shard node 3 on port 8003"
	@echo "  make run-node4    - Run shard node 4 on port 8004"
	@echo "  make run-gateway  - Run API gateway on port 8000"
	@echo "  make stop         - Stop all processes"
	@echo "  make clean        - Clean build artifacts and data"
	@echo "  make test         - Test the distributed system"
