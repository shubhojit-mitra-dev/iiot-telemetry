package api

import (
	"net/http"
)

// NewServer configures an http.Handler with all routing and standard middlewares applied.
func NewServer(h *Handler) http.Handler {
	mux := http.NewServeMux()

	// Ingestion Route
	mux.HandleFunc("/api/v1/telemetry", h.HandleIngest)

	// Analytical & Query Routes
	mux.HandleFunc("/api/v1/devices", h.HandleGetDevices)
	mux.HandleFunc("/api/v1/health", h.HandleHealth)

	// WebSocket Streaming Route
	mux.HandleFunc("/ws/telemetry", h.HandleWebSocket)

	// Apply Middlewares: Panic Recovery -> Structured Logging -> CORS -> Mux
	return Chain(mux,
		RecoveryMiddleware,
		LoggingMiddleware,
		CORSMiddleware,
	)
}
