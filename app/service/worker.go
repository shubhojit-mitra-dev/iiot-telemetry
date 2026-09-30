package service

import (
	"context"
	"log/slog"
	"sync"
	"sync/atomic"

	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/model"
	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/repository"
)

// IngestionService coordinates the buffered worker pool, state persistence, and real-time fan-out.
type IngestionService struct {
	repo         repository.TelemetryRepository
	hub          *Hub
	queue        chan model.TelemetryPayload
	workerCount  int
	wg           sync.WaitGroup
	processedCnt atomic.Uint64
	anomalyCnt   atomic.Uint64
	mu           sync.RWMutex
	closed       bool
}

// NewIngestionService initializes the ingestion pipeline with buffered channel and worker routines.
func NewIngestionService(repo repository.TelemetryRepository, hub *Hub, workerCount int, queueCapacity int) *IngestionService {
	if workerCount <= 0 {
		workerCount = 4
	}
	if queueCapacity <= 0 {
		queueCapacity = 10000
	}

	return &IngestionService{
		repo:        repo,
		hub:         hub,
		queue:       make(chan model.TelemetryPayload, queueCapacity),
		workerCount: workerCount,
	}
}

// Start launches the worker goroutines.
func (s *IngestionService) Start(ctx context.Context) {
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(ctx, i)
	}
	slog.Info("ingestion worker pool started", "workers", s.workerCount, "queue_capacity", cap(s.queue))
}

// Submit enqueues a payload for asynchronous processing without blocking the HTTP request thread.
// It returns false if the queue is saturated or the service is shutting down.
func (s *IngestionService) Submit(payload model.TelemetryPayload) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.closed {
		slog.Warn("ingestion service stopping, rejecting payload", "device_id", payload.DeviceID)
		return false
	}

	select {
	case s.queue <- payload:
		return true
	default:
		slog.Warn("ingestion queue saturated, backpressure drop triggered", "device_id", payload.DeviceID)
		return false
	}
}

func (s *IngestionService) worker(ctx context.Context, id int) {
	defer s.wg.Done()

	for payload := range s.queue {
		s.processPayload(ctx, payload)
	}
}

func (s *IngestionService) processPayload(ctx context.Context, payload model.TelemetryPayload) {
	// 1. Hot Path: Persist latest state to Redis
	if err := s.repo.SaveLatest(ctx, payload); err != nil {
		slog.Error("failed to persist latest telemetry", "device_id", payload.DeviceID, "error", err)
	}

	s.processedCnt.Add(1)

	// 2. Anomaly Check
	if payload.Temperature > 120.0 || payload.Vibration > 12.0 {
		s.anomalyCnt.Add(1)
		slog.Warn("critical telemetry threshold exceeded",
			"device_id", payload.DeviceID,
			"temperature", payload.Temperature,
			"vibration", payload.Vibration,
			"rpm", payload.RPM,
		)
	}

	// 3. Real-Time Broadcast: Fan out over WebSockets to connected dashboards
	if s.hub != nil {
		s.hub.BroadcastJSON(payload)
	}
}

// Stop closes the queue and waits for all in-flight jobs to be flushed.
func (s *IngestionService) Stop() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	close(s.queue)
	s.mu.Unlock()

	s.wg.Wait()
	slog.Info("ingestion worker pool gracefully stopped", "total_processed", s.processedCnt.Load())
}

// Stats returns runtime statistics for monitoring.
func (s *IngestionService) Stats() (processed uint64, anomalies uint64, queueDepth int) {
	return s.processedCnt.Load(), s.anomalyCnt.Load(), len(s.queue)
}
