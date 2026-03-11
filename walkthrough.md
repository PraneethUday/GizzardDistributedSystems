# Distributed Systems Algorithms — Walkthrough

## Summary

Implemented **4 distributed systems algorithms** in the GizzardDistributedSystems project as a new `algorithms/` Go package, fully integrated into the node and gateway binaries with API endpoints.

## Files Created

### Algorithm Core (`algorithms/`)

| File | Algorithm | Key Types |
|------|-----------|-----------|
| [vector_clock.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/vector_clock.go) | Vector Clocks | [VectorClock](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/vector_clock.go#39-44), [EventLog](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/vector_clock.go#172-178), [Event](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/vector_clock.go#163-170) |
| [snapshot.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/snapshot.go) | Chandy-Lamport Snapshot | [SnapshotManager](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/snapshot.go#42-61), [SnapshotState](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/snapshot.go#10-19), [ChannelRecorder](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/snapshot.go#22-27) |
| [leader_election.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/leader_election.go) | Bully Leader Election | [BullyElection](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/leader_election.go#69-81), [ElectionMessage](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/leader_election.go#33-38), [ElectionState](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/leader_election.go#40-49) |
| [consistent_hashing.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/consistent_hashing.go) | Consistent Hashing | [ConsistentHashRing](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/consistent_hashing.go#47-54), [HashRingNode](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/consistent_hashing.go#12-17), [VirtualNode](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/consistent_hashing.go#20-25) |

### Unit Tests (`algorithms/`)

| File | Tests |
|------|-------|
| [vector_clock_test.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/vector_clock_test.go) | 10 tests: Tick, Send, Receive, Compare (BEFORE/AFTER/CONCURRENT/EQUAL), EventLog |
| [snapshot_test.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/snapshot_test.go) | 7 tests: Initiate, HandleMarker (first/duplicate), state recording, completion |
| [leader_election_test.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/leader_election_test.go) | 7 tests: Highest node wins, no response, higher responds, election/victory messages |
| [consistent_hashing_test.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/consistent_hashing_test.go) | 9 tests: Add/Remove, consistency, distribution (24-27%), redistribution (~27%) |

## Files Modified

| File | Changes |
|------|---------|
| [cmd/node/main.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/cmd/node/main.go) | Added algorithm state to [ShardNode](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/cmd/node/main.go#38-49), 8 new endpoints, vector clock ticks on CRUD |
| [cmd/gateway/main.go](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/cmd/gateway/main.go) | Added [HashRing](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/algorithms/consistent_hashing.go#55-68) to [Gateway](file:///Users/praneethudayakumar227/Documents/GitHub/GizzardDistributedSystems/distributed-sharding/cmd/gateway/main.go#73-79), 12 new endpoints incl. `/algorithms` overview |

## New API Endpoints

### Gateway (port 8000)

| Method | Endpoint | Algorithm |
|--------|----------|-----------|
| `GET` | `/algorithms` | List all algorithms |
| `GET` | `/clocks` | Vector Clocks — all node clocks |
| `GET` | `/events` | Vector Clocks — all event logs |
| `POST` | `/snapshot` | Chandy-Lamport — initiate |
| `GET` | `/snapshot` | Chandy-Lamport — results |
| `POST` | `/election/start` | Bully — trigger election |
| `GET` | `/election/leader` | Bully — leader status |
| `GET` | `/hash-ring/status` | Consistent Hashing — ring state |
| `POST` | `/hash-ring/lookup` | Consistent Hashing — key lookup |
| `POST` | `/hash-ring/add-node` | Consistent Hashing — add node |
| `DELETE` | `/hash-ring/remove-node/:id` | Consistent Hashing — remove node |

### Node (ports 8001-8004)

| Method | Endpoint | Algorithm |
|--------|----------|-----------|
| `GET` | `/clock` | Vector clock + event log |
| `POST` | `/clock/event` | Log custom event |
| `POST` | `/snapshot/initiate` | Initiate snapshot |
| `POST` | `/snapshot/marker` | Receive marker |
| `GET` | `/snapshot/state` | Get snapshot state |
| `POST` | `/election/start` | Start election |
| `POST` | `/election/message` | Handle election msg |
| `GET` | `/election/leader` | Get leader state |

## Testing & Validation

- **33/33 unit tests pass** (`go test ./algorithms/ -v`)
- **`make build` compiles** both `bin/node` and `bin/gateway` successfully
- **`go vet ./...`** reports zero issues

### Usage After Starting System

```bash
# Start all: make run-all

# List all algorithms
curl http://localhost:8000/algorithms | jq

# Vector Clocks — create some users first, then inspect clocks
curl http://localhost:8000/clocks | jq
curl http://localhost:8000/events | jq

# Chandy-Lamport Snapshot
curl -X POST http://localhost:8000/snapshot | jq
curl http://localhost:8000/snapshot | jq

# Bully Leader Election
curl -X POST http://localhost:8000/election/start | jq
curl http://localhost:8000/election/leader | jq

# Consistent Hashing
curl http://localhost:8000/hash-ring/status | jq
curl -X POST http://localhost:8000/hash-ring/lookup \
  -H "Content-Type: application/json" \
  -d '{"user_id": 42}' | jq
```
