# ⚡ GoLedgerly

> A staged, from-scratch distributed key-value store built in Go, featuring active/lazy TTL expiration, $O(1)$ LRU eviction, Write-Ahead Log (WAL) crash recovery, and atomic operational metrics.

![Go Version](https://img.shields.io/badge/Go-1.26%2B-00ADD8?style=for-the-badge&logo=go)
![Architecture](https://img.shields.io/badge/Architecture-Decoupled_KV_Engine-blueviolet?style=for-the-badge)
![Dependencies](https://img.shields.io/badge/Dependencies-Zero_External-success?style=for-the-badge)
![Testing](https://img.shields.io/badge/Tests-100%25_Passing-brightgreen?style=for-the-badge)

---

## 📌 Project Milestones & Status

| Stage | Feature | Status | Details |
| :--- | :--- | :---: | :--- |
| **Stage 0** | **Single-Node In-Memory Store** | ✅ Complete | Thread-safe `sync.RWMutex`, HTTP API (`GET`/`PUT`/`DELETE`) |
| **Stage 1** | **TTL Expiration & LRU Eviction** | ✅ Complete | Dual-phase TTL (lazy + 250ms active sweeper), $O(1)$ `container/list` LRU |
| **Stage 2** | **WAL Persistence & Recovery** | ✅ Complete | Append-only logging with `fsync`, automatic startup replay |
| **Stage 3** | **Production Readiness** | 🚧 In Progress | Atomic metrics complete; rate limiting, tests, and benchmarks planned |
| **Stage 4** | **Leader-Follower Replication** | ⏳ Planned | Primary-secondary streaming replication & read-scaling |
| **Stage 5** | **Raft Consensus Protocol** | ⏳ Optional | Dynamic leader election & distributed strong consistency |

---

## 📐 System Architecture

```
                      ┌─────────────────────────────────────────┐
                      │           HTTP Client / cURL            │
                      └────────────────────┬────────────────────┘
                                           │ Request
                                           ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│  cmd/api (HTTP Transport Layer)                                                 │
│                                                                                 │
│   ┌─────────────────────┐      ┌──────────────────┐     ┌────────────────────┐  │
│   │ HTTP Server         ├─────►│ ServeMux Router  ├────►│ Request Handlers   │  │
│   │                     │      │ (Go 1.22+ Mux)   │     │ (application*)     │  │
│   └─────────────────────┘      └──────────────────┘     └─────────┬──────────┘  │
└───────────────────────────────────────────────────────────────────┼─────────────┘
                                                                    │ Method Calls
                                                                    ▼
┌─────────────────────────────────────────────────────────────────────────────────┐
│  internal/store (Storage Engine)                                                │
│                                                                                 │
│   ┌─────────────────────────────────────────────────────────────────────────┐   │
│   │ Store (RWMutex Protected)                                               │   │
│   │  ├── data: map[string]entry                                             │   │
│   │  ├── lruList: container/list  ──► O(1) Eviction                         │   │
│   │  └── metrics: sync/atomic     ──► Lock-Free Counter                     │   │
│   └────────────────────┬──────────────────────────────────┬─────────────────┘   │
│                        │                                  │                     │
│                        ▼                                  ▼                     │
│         ┌─────────────────────────────┐    ┌─────────────────────────────┐      │
│         │ Background Expiry Loop      │    │ Write-Ahead Log (WAL)       │      │
│         │ (250ms Active Ticker)       │    │ (Append-Only + fsync)       │      │
│         └─────────────────────────────┘    └─────────────────────────────┘      │
└─────────────────────────────────────────────────────────────────────────────────┘
```

---

## 🛠️ Key Design Decisions & Engineering Tradeoffs

### 1. Storage / Transport Separation
- The `internal/store` engine has zero dependencies on `net/http` or JSON decoding. 
- **Benefit**: Storage logic remains 100% testable in isolation and ready to adopt gRPC or TCP transport layers without rewriting core storage mechanisms.

### 2. Lock-Optimized Concurrency Model
- Reads acquire `sync.RWMutex.RLock()`, enabling parallel read execution across goroutines.
- Operational metrics (Hits, Misses, Evictions, Expirations) use `sync/atomic` primitives, eliminating mutex contention during telemetry gathering.

### 3. Dual-Phase TTL Expiration
- **Lazy Expiration**: Checked on `Get()`. Expired keys are purged immediately upon access to guarantee zero stale reads.
- **Active Sweeper Goroutine**: A background ticker sweeps the map every 250ms to evict non-accessed expired keys, maintaining zero memory leakage.

### 4. $O(1)$ Double-Linked List LRU Eviction
- Combines Go's standard `container/list` doubly-linked list with a map pointer index. Accessing a key moves it to the front; exceeding capacity evicts the tail element in $O(1)$ time complexity.

### 5. Crash Resilience via Write-Ahead Log (WAL)
- Every mutation (`SET`, `SET_TTL`, `DEL`) writes to an append-only log file (`store.wal`) and invokes `file.Sync()` for physical disk persistence.
- On node startup, `ReplayWAL()` restores state into memory while safely discarding keys that expired while the server was down.

---

## 📁 Repository Structure

```
GoLedgerly/
├── cmd/
│   └── api/
│       ├── main.go          # Application wiring, env configuration, server boot
│       ├── routes.go        # HTTP route registration
│       └── handlers.go      # HTTP handlers (*application receiver methods)
└── internal/
    └── store/
        ├── store.go         # Core KV engine, RWMutex, LRU, & TTL loop
        ├── store_test.go    # Unit tests for store operations
        ├── ttl_test.go      # Active & passive TTL expiration tests
        ├── wal.go           # Write-Ahead Log logger, flush, & replay engine
        ├── metrics.go       # Atomic counter telemetry
        └── metrics_test.go
```

---

## ⚡ Quickstart Guide

### 1. Run the Server
```bash
go run ./cmd/api
```
*Server starts listening on `:4000` with active WAL file `./store.wal`.*

---

## 🧪 Interactive API Examples

### 1. Health Probe (Public)
```bash
curl -i http://localhost:4000/health
```
**Response (`200 OK`)**:
```json
{"status":"ok"}
```

### 2. Set a Key with a 30-Second TTL
```bash
curl -i -X PUT "http://localhost:4000/store/session_99?ttl=30s" \
  -d '{"user_id": 42, "role": "admin"}'
```
**Response**: `201 Created`

### 3. Retrieve Key
```bash
curl -i http://localhost:4000/store/session_99
```
**Response (`200 OK`)**:
```json
{"user_id": 42, "role": "admin"}
```

### 4. Inspect Live Operational Metrics
```bash
curl -s http://localhost:4000/metrics
```
**Response (`200 OK`)**:
```json
{
  "evictions": 0,
  "expirations": 1,
  "hits": 4,
  "keys": 1,
  "misses": 0
}
```

---

## 🔬 Testing & Verification

Run all package test suites:
```bash
go test -v ./...
```

Run race detector to verify zero data races under concurrent execution:
```bash
go test -race ./...
```
