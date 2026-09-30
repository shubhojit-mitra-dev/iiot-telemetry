# Phase 1: High-Performance Vanilla Go Ingestion Backend

**Target Audience:** Senior/Staff Implementing Agents
**Objective:** Build an ultra-low latency, highly optimized Go HTTP ingestion server using *only* the Go Standard Library for routing. No Fiber, no Gin, no web frameworks.

## 1. Architectural Philosophy & Constraints
- **Zero Web Frameworks:** Use standard `net/http`. 
- **Memory Optimization:** Pre-allocate all slices. Use `sync.Pool` for struct allocation to prevent garbage collection (GC) thrashing during high-frequency ingestion spikes.
- **Asynchronous Processing:** The HTTP handler must *never* block for database I/O. Incoming requests are validated and immediately passed into a buffered Go channel. The HTTP handler immediately returns `202 Accepted`. A background Worker Pool consumes the channel to write to Redis.
- **Graceful Shutdown:** The server must intercept `SIGINT`/`SIGTERM`, stop accepting new requests, flush all channels, close Redis connections, and close WebSocket hubs cleanly.

## 2. Directory Structure (Domain-Driven Design)
```text
backend/
├── cmd/
│   └── server/
│       └── main.go         # Bootstrapper, Dependency Injection, Signal handling
├── internal/
│   ├── api/
│   │   ├── http.go         # net/http ServeMux and middleware (CORS, Logging, Recover)
│   │   ├── ingest.go       # POST /api/v1/telemetry handler
│   │   └── ws.go           # GET /ws/telemetry handler
│   ├── models/
│   │   └── telemetry.go    # Data structures with struct tags and sync.Pool definitions
│   ├── repository/
│   │   └── redis.go        # Redis connection pool and HSET logic (using go-redis/redis/v9)
│   └── service/
│       ├── pool.go         # Worker pool to process ingestion channels concurrently
│       └── hub.go          # WebSocket broadcasting hub using select and channels
```

## 3. Detailed Component Implementations

### A. Data Models & `sync.Pool` (`internal/models/telemetry.go`)
- Define `TelemetryPayload`: DeviceID (string), Timestamp (int64), Temperature (float64), Vibration (float64), RPM (int64).
- **Crucial:** Implement a `sync.Pool` for `TelemetryPayload`. During high throughput (10,000+ req/sec), instantiating a new struct for every request will kill performance via GC pauses.
- Create `GetPayload() *TelemetryPayload` and `PutPayload(p *TelemetryPayload)` functions.

### B. The Ingestion Handler (`internal/api/ingest.go`)
- **Route:** `POST /api/v1/telemetry`
- **Logic:**
  1. Acquire a `TelemetryPayload` from the `sync.Pool`.
  2. Use `json.NewDecoder(r.Body).Decode(payload)`. Do not use `ioutil.ReadAll` (it allocates memory dynamically).
  3. Validate fields (DeviceID cannot be empty, Timestamp > 0).
  4. If invalid, release to pool and return `400 Bad Request`.
  5. If valid, send a **copy** of the data to the `IngestChannel` and immediately release the struct back to the `sync.Pool`.
  6. Return `202 Accepted` and cleanly close the request body.

### C. Worker Pool & Redis Repository (`internal/service/pool.go` & `internal/repository/redis.go`)
- **Worker Pool:** Initialize a buffered channel `IngestChannel = make(chan TelemetryPayload, 10000)`.
- Spawn a fixed number of goroutines (e.g., `runtime.NumCPU() * 2`) that constantly read from `IngestChannel`.
- **Redis Writes:** For every payload received from the channel, call `redis.SaveLatestTelemetry`.
- **Redis Details:** Use `go-redis/v9`. Store data as `HSET device:latest:<DeviceID>`. Utilize Redis Pipelining if the worker batch-reads from the channel, otherwise direct `HSET` is fine since workers run concurrently.

### D. WebSocket Hub (`internal/service/hub.go`)
- Use `github.com/gorilla/websocket` (de-facto standard for WS, highly optimized).
- The Hub must maintain `Clients map[*Client]bool`.
- Use a `broadcast` channel. The Hub runs in a single goroutine using the strict `select` pattern:
  ```go
  select {
  case client := <-hub.register:
      hub.clients[client] = true
  case client := <-hub.unregister:
      if _, ok := hub.clients[client]; ok {
          delete(hub.clients, client)
          close(client.send)
      }
  case message := <-hub.broadcast:
      for client := range hub.clients {
          // non-blocking send
          select {
          case client.send <- message:
          default:
              close(client.send)
              delete(hub.clients, client)
          }
      }
  }
  ```
- **Zero-Block Broadcasting:** The `default` case above is critical. If a client's network is slow, its channel buffer fills up. We must drop the client immediately rather than blocking the entire broadcast loop for thousands of other devices.

## 4. Error Handling & Logging
- Do not use `panic`. Handle all errors gracefully.
- Use `log/slog` (introduced in Go 1.21) for highly optimized, zero-allocation structured JSON logging.

## 5. Next Steps for Implementation
1. Initialize module `github.com/mitra/iiot-telemetry/backend`.
2. Build the `TelemetryPayload` pool.
3. Build the HTTP server, routing, and Worker Pool.
4. Implement Redis connection and asynchronous HSET operations.
5. Benchmark locally to ensure minimal GC allocation on the hot path.
