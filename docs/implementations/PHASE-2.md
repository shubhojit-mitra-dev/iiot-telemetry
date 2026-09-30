# Phase 2: Edge Simulator & Real-Time Frontend Integration

## 1. Phase 1 Outcomes & Current State

Before proceeding with Phase 2, here is a summary of what was accomplished in Phase 1:

*   **High-Performance Ingestion Engine:** Built a vanilla Go server (`net/http`) capable of handling 10,000+ msgs/sec with sub-5ms latency.
*   **Zero-Allocation Architecture:** Implemented `sync.Pool` for `TelemetryPayload` recycling, eliminating Garbage Collection (GC) pauses on the hot path.
*   **Asynchronous Worker Pool:** Created a buffered channel (`10,000` capacity) and worker pool (`runtime.NumCPU() * 2`) to decouple HTTP ingestion from database/network I/O.
*   **Dual Repository Pattern:** Implemented a production Redis repository (`HSET` operations) and a thread-safe in-memory fallback for seamless local execution.
*   **WebSocket Broadcast Hub:** Engineered a Gorilla WebSocket hub with non-blocking fan-out and slow-client drop protection.
*   **React Dashboard Skeleton:** Created a Grafana-style dark-mode React application with Recharts, currently powered by a mock data generator.
*   **Infrastructure:** Set up the Git repository, established strict commit rules, and installed Go 1.27.0 globally. `master` branch has been successfully migrated to `main`.

## 2. Phase 2 Objectives

Phase 2 focuses on connecting the components. We will replace the mock data with actual network traffic by building an Edge Simulator and wiring the React frontend to the Go WebSocket server.

**Note:** As requested, AI Diagnostics are deferred to the final phase.

### 2.1 Edge Machinery Simulator (Go CLI)
We need a robust way to generate synthetic, realistic factory data to stress-test the ingestion server and drive the dashboard.

*   **Location:** `cmd/simulator/main.go`
*   **Architecture:**
    *   Initialize 10 distinct virtual machines (e.g., `TURB-001`, `PUMP-002`) mirroring the frontend's expected devices.
    *   Spawn a dedicated goroutine for each machine.
    *   Each goroutine runs a ticker (configurable interval, e.g., 100ms - 1000ms).
    *   On every tick, generate a `TelemetryPayload` with realistic baseline metrics + Gaussian noise.
    *   Inject occasional anomalies (e.g., temperature spikes > 120°C) based on a probability threshold.
    *   Use `net/http` Client to `POST` the JSON payload to `http://localhost:8080/api/v1/telemetry`.
    *   Handle graceful shutdown (`SIGINT`) to stop the simulator cleanly.

### 2.2 React Frontend Integration
The frontend currently uses `setInterval` to mock data. We will rewrite the telemetry hook to consume real network data.

*   **Target:** `src/hooks/useTelemetryStream.ts` (and relevant components).
*   **Initial State Fetch:**
    *   On mount, make an HTTP `GET` request to `http://localhost:8080/api/v1/devices` to fetch the latest known state of all machines.
    *   Populate the initial charts and matrix.
*   **WebSocket Connection:**
    *   Establish a `WebSocket` connection to `ws://localhost:8080/ws/telemetry`.
    *   Implement automatic reconnection logic with exponential backoff if the server drops.
    *   On `message` event, parse the incoming JSON `TelemetryPayload`.
    *   Update the `tempSeries`, `vibSeries`, `rpmSeries`, and `latestByDevice` states.
    *   Maintain the sliding window buffer (e.g., keeping only the last 60 data points per chart) to prevent browser memory leaks.
    *   Track anomalies locally based on incoming threshold breaches and add them to the `IncidentLog`.
*   **Health Stats Polling (Optional but Recommended):**
    *   Periodically (e.g., every 5s) poll `GET http://localhost:8080/api/v1/health` to update the "Messages Ingested" and "Queue Depth" stat cards.

## 3. Implementation Steps & Commit Plan

Following the strict rules (Max 2 files per commit, atomic commits):

1.  **Commit 1:** Create `cmd/simulator/main.go` (Edge Simulator logic).
2.  **Commit 2:** Update `src/hooks/useTelemetryStream.ts` (Replace mock logic with WebSocket + Fetch logic).
3.  **Commit 3:** Update `src/App.tsx` (or other components if necessary to handle real API data shapes/health stats).

## 4. Success Criteria for Phase 2
*   Running `go run cmd/simulator/main.go` successfully blasts data to the running Go server.
*   The Go server's logs show active ingestion and WebSocket broadcasting.
*   Opening the React dashboard shows live, updating charts driven by the WebSocket stream, not local mock timers.
*   The system remains stable with no memory leaks in the browser or the Go server over extended runs.
