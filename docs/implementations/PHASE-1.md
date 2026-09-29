# Phase 1: Core Ingestion Backend Implementation Details

**Target Audience:** Junior Engineers / Implementing Agents
**Objective:** Build a high-performance Go-based API that captures IIoT telemetry data, buffers it in Redis, and broadcasts it over WebSockets.

## 1. Project Initialization
- Initialize a new Go module in a `backend/` directory: `go mod init github.com/mitra/iiot-telemetry/backend`
- Use **Go Fiber** (`github.com/gofiber/fiber/v2`) for the web framework due to its zero-allocation routing and high performance.
- Use **go-redis** (`github.com/redis/go-redis/v9`) for Redis connectivity.
- Use Fiber's WebSocket middleware (`github.com/gofiber/websocket/v2`).

## 2. Project Structure
Create the following strict directory structure within `backend/`:
```text
backend/
├── cmd/
│   └── server/
│       └── main.go         # Application entry point
├── internal/
│   ├── api/
│   │   ├── handlers/       # HTTP and WebSocket handlers
│   │   └── routes.go       # Route definitions
│   ├── models/             # Data structures (Telemetry Payload)
│   ├── repository/         # Redis connection and operations
│   └── service/            # Core business logic (anomaly detection, broadcasting)
```

## 3. Data Models
Create the core telemetry struct in `internal/models/telemetry.go`:
```go
type TelemetryPayload struct {
    DeviceID    string  `json:"device_id"`
    Timestamp   int64   `json:"timestamp"`
    Temperature float64 `json:"temperature"`
    Vibration   float64 `json:"vibration"`
    RPM         int64   `json:"rpm"`
}
```

## 4. Implementation Requirements

### A. Redis Repository (`internal/repository/redis.go`)
- Create a connection pool connecting to a Redis instance (assume `localhost:6379` for local dev).
- Implement a method `SaveLatestTelemetry(ctx context.Context, payload TelemetryPayload) error`.
- The Redis key must be formatted as: `device:latest:<DeviceID>`.
- Use a Redis Hash (`HSET`) to store the fields, allowing quick O(1) retrieval of individual metrics later.

### B. Ingestion Handler (`internal/api/handlers/ingest.go`)
- Implement `POST /api/v1/telemetry`.
- Validate the incoming JSON against the `TelemetryPayload` struct. If invalid, return `400 Bad Request`.
- Asynchronously (via a goroutine) pass the valid payload to the Service layer to prevent blocking the HTTP response.
- Return `202 Accepted` immediately upon successful validation.

### C. Service Layer & WebSocket Hub (`internal/service/hub.go`)
- The service layer must act as a central Hub for WebSocket clients.
- Maintain a thread-safe map of active WebSocket connections (`map[*websocket.Conn]bool`) protected by a `sync.RWMutex`.
- Provide a `Broadcast(payload TelemetryPayload)` method that iterates through active connections and writes the JSON payload.
- Implement basic anomaly detection: If `Temperature > 120`, log a high-priority warning (Phase 2 will integrate AI here).

### D. WebSocket Handler (`internal/api/handlers/ws.go`)
- Implement `GET /ws/telemetry`.
- Upgrade the HTTP connection to a WebSocket.
- Register the connection with the Service Hub.
- Handle client disconnections gracefully by unregistering them from the Hub and closing the socket.

## 5. Execution & Testing
- Ensure the server starts gracefully and listens on port `8080`.
- Implement graceful shutdown on `SIGINT`/`SIGTERM`, ensuring all Redis and WebSocket connections are closed cleanly.
- (Reminder: Follow the strict commit strategy defined in `AGENTS.md`).
