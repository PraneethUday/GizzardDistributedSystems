# Distributed Systems Algorithms — Walkthrough

## Summary

Implemented **4 distributed systems algorithms** with a **React frontend dashboard** for triggering and visualizing them.

## Algorithm Implementations (`algorithms/`)

| File | Algorithm | Key Feature |
|------|-----------|-------------|
| [vector_clock.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/vector_clock.go) | Vector Clocks | Thread-safe causal ordering with event logging |
| [snapshot.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/snapshot.go) | Chandy-Lamport Snapshot | Consistent global state capture via markers |
| [leader_election.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/leader_election.go) | Bully Election | Highest-ID node wins with timeout handling |
| [consistent_hashing.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/consistent_hashing.go) | Consistent Hashing | SHA-256 hash ring with 150 virtual nodes |

## Backend Integration

| File | Changes |
|------|---------|
| [cmd/node/main.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/cmd/node/main.go) | 8 new endpoints, vector clock ticks on CRUD |
| [cmd/gateway/main.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/cmd/gateway/main.go) | 12 new endpoints, `/algorithms` listing, hash ring |

## Frontend Dashboard

| File | Changes |
|------|---------|
| [App.jsx](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/frontend/src/App.jsx) | Added tabbed Algorithms Dashboard with 4 panels |
| [index.css](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/frontend/src/index.css) | Added ~500 lines of algorithm-specific styling |

### Dashboard Features

- **⏱ Vector Clocks** — Fetch and display each node's clock vector + recent events
- **📸 Chandy-Lamport** — One-click snapshot trigger + view snapshot states from all nodes
- **👑 Bully Election** — Trigger election + real-time leader status with crown animation
- **🔗 Consistent Hashing** — View hash ring stats, key distribution bars, + key lookup

## Testing & Validation

- ✅ **33/33 unit tests pass**
- ✅ **`make build` compiles** both binaries
- ✅ **`go vet ./...`** — zero issues
- ✅ Frontend builds and runs with Vite HMR
