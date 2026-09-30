package api

import (
	"encoding/json"
	"net/http"

	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/model"
	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/repository"
	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/service"
)

// Handler exposes HTTP and WebSocket endpoints for the telemetry platform.
type Handler struct {
	service *service.IngestionService
	repo    repository.TelemetryRepository
	hub     *service.Hub
}

// NewHandler constructs an initialized API handler.
func NewHandler(svc *service.IngestionService, repo repository.TelemetryRepository, hub *service.Hub) *Handler {
	return &Handler{
		service: svc,
		repo:    repo,
		hub:     hub,
	}
}

// HandleIngest captures inbound telemetry JSON with zero-allocation pooling.
// Route: POST /api/v1/telemetry
func (h *Handler) HandleIngest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	// 1. Acquire pooled struct to eliminate heap alloc on hot path
	payload := model.GetPayload()
	defer model.PutPayload(payload)

	// 2. Decode stream directly without ioutil.ReadAll
	if err := json.NewDecoder(r.Body).Decode(payload); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid json payload: " + err.Error()})
		return
	}

	// 3. Invariant validation
	if err := payload.Validate(); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// 4. Submit clone to asynchronous worker queue (non-blocking)
	if !h.service.Submit(payload.Clone()) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "ingestion queue saturated"})
		return
	}

	// 5. Microsecond response back to edge node
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_, _ = w.Write([]byte(`{"status":"accepted"}`))
}

// HandleGetDevices returns the latest telemetry snapshot for all observed machinery.
// Route: GET /api/v1/devices
func (h *Handler) HandleGetDevices(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, `{"error":"method not allowed"}`, http.StatusMethodNotAllowed)
		return
	}

	devices, err := h.repo.GetAllLatest(r.Context())
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(devices)
}

// HandleHealth returns application uptime and worker pipeline statistics.
// Route: GET /api/v1/health
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	processed, anomalies, queueDepth := h.service.Stats()

	resp := map[string]interface{}{
		"status":          "healthy",
		"active_clients":  h.hub.ActiveClients(),
		"total_processed": processed,
		"anomalies":       anomalies,
		"queue_depth":     queueDepth,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleWebSocket upgrades connection to real-time telemetry stream.
// Route: GET /ws/telemetry
func (h *Handler) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	h.hub.ServeWebSocket(w, r)
}
