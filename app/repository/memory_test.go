package repository

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/model"
)

func TestMemoryRepository_BasicCRUD(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	defer func() {
		if err := repo.Close(); err != nil {
			t.Fatalf("unexpected error closing memory repo: %v", err)
		}
	}()

	// 1. Assert Not Found for non-existent device
	_, err := repo.GetLatest(ctx, "NON_EXISTENT")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// 2. Save payload
	payload1 := model.TelemetryPayload{
		DeviceID:    "TURB-001",
		Timestamp:   1700000000000,
		Temperature: 85.0,
		Vibration:   6.0,
		RPM:         3200,
	}

	if err := repo.SaveLatest(ctx, payload1); err != nil {
		t.Fatalf("failed to save telemetry: %v", err)
	}

	// 3. Retrieve and assert match
	got, err := repo.GetLatest(ctx, "TURB-001")
	if err != nil {
		t.Fatalf("unexpected error fetching TURB-001: %v", err)
	}
	if *got != payload1 {
		t.Fatalf("expected %+v, got %+v", payload1, *got)
	}

	// 4. Update existing device with newer frame
	payload1Updated := model.TelemetryPayload{
		DeviceID:    "TURB-001",
		Timestamp:   1700000001000,
		Temperature: 86.5,
		Vibration:   6.2,
		RPM:         3210,
	}
	if err := repo.SaveLatest(ctx, payload1Updated); err != nil {
		t.Fatalf("failed to update telemetry: %v", err)
	}

	gotUpdated, err := repo.GetLatest(ctx, "TURB-001")
	if err != nil {
		t.Fatalf("unexpected error fetching updated TURB-001: %v", err)
	}
	if *gotUpdated != payload1Updated {
		t.Fatalf("expected %+v, got %+v", payload1Updated, *gotUpdated)
	}

	// 5. Save second device and assert GetAllLatest
	payload2 := model.TelemetryPayload{
		DeviceID:    "PUMP-001",
		Timestamp:   1700000000000,
		Temperature: 65.0,
		Vibration:   4.1,
		RPM:         1800,
	}
	if err := repo.SaveLatest(ctx, payload2); err != nil {
		t.Fatalf("failed to save second device: %v", err)
	}

	all, err := repo.GetAllLatest(ctx)
	if err != nil {
		t.Fatalf("unexpected error fetching all devices: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 devices in snapshot, got %d", len(all))
	}
	if all["TURB-001"] != payload1Updated || all["PUMP-001"] != payload2 {
		t.Fatalf("snapshot contents mismatch: %+v", all)
	}
}

func TestMemoryRepository_ConcurrentThreadSafety(t *testing.T) {
	ctx := context.Background()
	repo := NewMemoryRepository()
	defer repo.Close()

	var wg sync.WaitGroup
	workers := 50
	iterations := 100

	// Concurrent writers
	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			devID := fmt.Sprintf("DEV-%02d", workerID%5)
			for i := 0; i < iterations; i++ {
				_ = repo.SaveLatest(ctx, model.TelemetryPayload{
					DeviceID:    devID,
					Timestamp:   int64(i),
					Temperature: float64(workerID),
					Vibration:   float64(i),
					RPM:         int64(workerID * 100),
				})
			}
		}(w)
	}

	// Concurrent readers
	for r := 0; r < workers; r++ {
		wg.Add(1)
		go func(readerID int) {
			defer wg.Done()
			devID := fmt.Sprintf("DEV-%02d", readerID%5)
			for i := 0; i < iterations; i++ {
				_, _ = repo.GetLatest(ctx, devID)
				_, _ = repo.GetAllLatest(ctx)
			}
		}(r)
	}

	wg.Wait()
}
