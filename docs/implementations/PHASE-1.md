# Phase 1: High-Performance Vanilla Go Ingestion Backend

**Target Audience:** Senior/Staff Implementing Agents  
**Objective:** Build an ultra-low latency, highly optimized Go HTTP ingestion server using *only* the Go Standard Library for routing. No Fiber, no Gin, no web frameworks.

## 1. Architectural Philosophy & Constraints
- **Zero Web Frameworks:** Use standard `net/http.ServeMux`.
- **Memory Optimization:** Pre-allocate slices where possible. Use `sync.Pool` for struct allocation to eliminate garbage collection (GC) thrashing during high-frequency ingestion spikes (10,000+ msgs/sec).
- **Asynchronous Processing:** The HTTP handler must *never* block for database I/O. Incoming requests are validated and immediately passed into a buffered Go channel (`make(chan TelemetryPayload, 10000)`). The HTTP handler immediately returns `202 Accepted`. A background Worker Pool consumes the channel to persist state and broadcast.
- **Graceful Shutdown:** The server must intercept `SIGINT`/`SIGTERM`, stop accepting new requests, flush all worker queues with a 10s deadline, close Redis connection pools, and terminate WebSocket client connections cleanly.
- **Resilient Fallback:** The repository uses an interface pattern with dual implementations: production Redis cluster pool and a concurrent in-memory store for standalone/testing environments.

## 2. Directory Structure (Flat / Fider-Inspired Domain Architecture)
```text
├── main.go                     # Bootstrapper, Dependency Injection, Signal handling
├── app/
│   ├── api/
│   │   ├── handler.go          # Handlers for Ingest, Health, Devices, WebSockets
│   │   ├── middleware.go       # CORS, Structured Slog Logging, Panic Recovery
│   │   └── server.go           # net/http ServeMux routing & middleware chaining
│   ├── model/
│   │   └── telemetry.go        # TelemetryPayload struct, invariant validation, sync.Pool
│   ├── repository/
│   │   ├── repository.go       # TelemetryRepository interface definition
│   │   ├── redis.go            # Production Redis HSET/HGetAll pool implementation
│   │   └── memory.go           # Thread-safe in-memory fallback with sync.RWMutex
│   └── service/
│       ├── hub.go              # Gorilla WebSocket broadcasting hub with non-blocking drop
│       └── worker.go           # Asynchronous worker pool consuming buffered ingestion queue
├── go.mod                      # Module: github.com/shubhojit-mitra-dev/iiot-telemetry
└── go.sum                      # Verified checksums for gorilla/websocket & go-redis
```

## 3. Detailed Component Implementations

### A. Data Models & `sync.Pool` (`app/model/telemetry.go`)
- `TelemetryPayload`: `DeviceID` (string), `Timestamp` (int64), `Temperature` (float64), `Vibration` (float64), `RPM` (int64).
- **sync.Pool:** `payloadPool` recycles instances. `GetPayload() *TelemetryPayload` retrieves from pool; `PutPayload(p)` zeroes fields and returns struct to pool.
- **Validation:** `Validate()` checks `DeviceID != ""` and `Timestamp > 0`.
- **Clone:** `Clone()` creates an independent stack/heap copy before recycling the original back to the pool.

### B. The Ingestion Handler (`app/api/handler.go`)
- **Route:** `POST /api/v1/telemetry`
- **Logic:**
  1. Acquire `TelemetryPayload` from `sync.Pool`.
  2. Stream via `json.NewDecoder(r.Body).Decode(payload)`. (Zero `ioutil.ReadAll` allocation).
  3. Validate fields. On error, return `400 Bad Request`.
  4. Submit `payload.Clone()` to worker queue via `Submit()`. If queue is saturated, return `503 Service Unavailable`.
  5. Return `202 Accepted` (`{"status":"accepted"}`). Defer `PutPayload(payload)` releases struct immediately.

### C. Repository Layer (`app/repository/`)
- `TelemetryRepository` interface defines `SaveLatest`, `GetLatest`, `GetAllLatest`, `Close`.
- **Redis (`redis.go`):** Stores latest telemetry frame as Redis Hash under key `device:latest:<device_id>`. Pool size: 100 connections.
- **Memory (`memory.go`):** In-memory map protected by `sync.RWMutex`. Used when Redis is unreachable, guaranteeing zero-downtime local development.

### D. Worker Pool (`app/service/worker.go`)
- Buffered channel `queue = make(chan TelemetryPayload, 10000)`.
- Spawns `runtime.NumCPU() * 2` worker goroutines.
- Workers persist to repository, check anomaly thresholds (`Temperature > 120°C` or `Vibration > 12.0 mm/s`), and broadcast via WebSocket Hub.
- `Stop()` cleanly closes queue and waits for all in-flight items with `sync.WaitGroup`.

### E. WebSocket Hub (`app/service/hub.go`)
- Manages active client map with Gorilla WebSocket.
- Non-blocking broadcast loop using `select` + `default`. If a client channel buffer (256 messages) is full, it is dropped immediately to protect the broadcast loop from stalling.
- Read/write pumps with automated ping/pong heartbeats to cleanly detect client disconnects.

## 4. Testing & Verification Standards
- Strict adherence to FIRST principles (Fast, Independent, Repeatable, Self-validating, Timely).
- Table-driven unit tests for validation, models, handlers, and repositories.
- Benchmark tests verifying `sync.Pool` zero-allocation performance against standard heap allocation.
- Concurrency race condition detection via `go test -race ./...`.
- Code coverage enforced at >= 80% across all packages.
