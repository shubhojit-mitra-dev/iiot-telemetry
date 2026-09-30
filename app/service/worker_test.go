package service

import (
	"context"
	"testing"
	"time"

	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/model"
	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/repository"
)

func TestIngestionService_Defaults(t *testing.T) {
	repo := repository.NewMemoryRepository()
	defer repo.Close()

	svc := NewIngestionService(repo, nil, 0, 0)
	if svc.workerCount != 4 {
		t.Fatalf("expected default worker count 4, got %d", svc.workerCount)
	}
	if cap(svc.queue) != 10000 {
		t.Fatalf("expected default queue capacity 10000, got %d", cap(svc.queue))
	}
}

func TestIngestionService_ProcessAndStats(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := repository.NewMemoryRepository()
	defer repo.Close()
	hub := NewHub()

	svc := NewIngestionService(repo, hub, 2, 100)
	svc.Start(ctx)

	payload := model.TelemetryPayload{
		DeviceID:    "TURB-001",
		Timestamp:   1700000000000,
		Temperature: 85.0,
		Vibration:   6.0,
		RPM:         3200,
	}

	if ok := svc.Submit(payload); !ok {
		t.Fatal("failed to submit payload to worker queue")
	}

	// Wait briefly for worker to consume
	time.Sleep(50 * time.Millisecond)

	processed, anomalies, queueDepth := svc.Stats()
	if processed != 1 {
		t.Fatalf("expected 1 processed, got %d", processed)
	}
	if anomalies != 0 {
		t.Fatalf("expected 0 anomalies, got %d", anomalies)
	}
	if queueDepth != 0 {
		t.Fatalf("expected 0 queue depth, got %d", queueDepth)
	}

	// Verify state in repository
	latest, err := repo.GetLatest(ctx, "TURB-001")
	if err != nil {
		t.Fatalf("unexpected error fetching from repo: %v", err)
	}
	if latest.Temperature != 85.0 {
		t.Fatalf("expected temperature 85.0, got %f", latest.Temperature)
	}

	svc.Stop()
}

func TestIngestionService_AnomalyDetection(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	repo := repository.NewMemoryRepository()
	defer repo.Close()

	svc := NewIngestionService(repo, nil, 2, 100)
	svc.Start(ctx)

	// Anomaly 1: Temp > 120
	svc.Submit(model.TelemetryPayload{
		DeviceID:    "PUMP-001",
		Timestamp:   1700000001000,
		Temperature: 125.0,
		Vibration:   5.0,
		RPM:         1800,
	})

	// Anomaly 2: Vibration > 12
	svc.Submit(model.TelemetryPayload{
		DeviceID:    "PUMP-002",
		Timestamp:   1700000002000,
		Temperature: 80.0,
		Vibration:   15.5,
		RPM:         1800,
	})

	// Normal payload
	svc.Submit(model.TelemetryPayload{
		DeviceID:    "PUMP-003",
		Timestamp:   1700000003000,
		Temperature: 75.0,
		Vibration:   4.5,
		RPM:         1800,
	})

	time.Sleep(50 * time.Millisecond)

	processed, anomalies, _ := svc.Stats()
	if processed != 3 {
		t.Fatalf("expected 3 processed, got %d", processed)
	}
	if anomalies != 2 {
		t.Fatalf("expected 2 anomalies, got %d", anomalies)
	}

	svc.Stop()
}

func TestIngestionService_QueueSaturationBackpressure(t *testing.T) {
	repo := repository.NewMemoryRepository()
	defer repo.Close()

	// Initialize service with tiny queue capacity and DO NOT start workers
	svc := NewIngestionService(repo, nil, 1, 2)

	p := model.TelemetryPayload{DeviceID: "DEV-1", Timestamp: 1}

	if ok := svc.Submit(p); !ok {
		t.Fatal("first submit should succeed")
	}
	if ok := svc.Submit(p); !ok {
		t.Fatal("second submit should succeed")
	}

	// Queue is now full (capacity 2)
	if ok := svc.Submit(p); ok {
		t.Fatal("third submit should fail due to queue saturation")
	}
}

func TestIngestionService_GracefulDraining(t *testing.T) {
	repo := repository.NewMemoryRepository()
	defer repo.Close()

	// Capacity 50, start 2 workers
	ctx, cancel := context.WithCancel(context.Background())
	svc := NewIngestionService(repo, nil, 2, 50)
	svc.Start(ctx)

	for i := 0; i < 20; i++ {
		svc.Submit(model.TelemetryPayload{
			DeviceID:    "DEV",
			Timestamp:   int64(i + 1),
			Temperature: float64(i),
		})
	}

	// Cancel context to trigger draining mode
	cancel()
	svc.Stop()

	processed, _, _ := svc.Stats()
	if processed != 20 {
		t.Fatalf("expected all 20 payloads to be drained and processed, got %d", processed)
	}
}
