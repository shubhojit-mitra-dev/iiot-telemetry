package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/model"
)

func TestNewRedisRepository_ConnectionFailure(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	// Attempt connection to non-existent unreachable port
	repo, err := NewRedisRepository(ctx, "127.0.0.1:59999", "", 0)
	if err == nil {
		if repo != nil {
			_ = repo.Close()
		}
		t.Fatal("expected connection error to unreachable redis address, got nil")
	}
}

func TestRedisRepository_Operations(t *testing.T) {
	s := miniredis.RunT(t)

	ctx := context.Background()
	repo, err := NewRedisRepository(ctx, s.Addr(), "", 0)
	if err != nil {
		t.Fatalf("failed to connect to miniredis: %v", err)
	}
	defer repo.Close()

	// 1. Get non-existent device
	_, err = repo.GetLatest(ctx, "NON_EXISTENT")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}

	// 2. Save payload
	payload1 := model.TelemetryPayload{
		DeviceID:    "TURB-001",
		Timestamp:   1700000000000,
		Temperature: 85.5,
		Vibration:   6.2,
		RPM:         3200,
	}

	if err := repo.SaveLatest(ctx, payload1); err != nil {
		t.Fatalf("failed to save latest to redis: %v", err)
	}

	// 3. Get payload
	got, err := repo.GetLatest(ctx, "TURB-001")
	if err != nil {
		t.Fatalf("failed to get latest from redis: %v", err)
	}
	if *got != payload1 {
		t.Fatalf("expected %+v, got %+v", payload1, *got)
	}

	// 4. Save second device and test GetAllLatest
	payload2 := model.TelemetryPayload{
		DeviceID:    "PUMP-001",
		Timestamp:   1700000001000,
		Temperature: 65.0,
		Vibration:   4.1,
		RPM:         1800,
	}
	if err := repo.SaveLatest(ctx, payload2); err != nil {
		t.Fatalf("failed to save second device: %v", err)
	}

	all, err := repo.GetAllLatest(ctx)
	if err != nil {
		t.Fatalf("failed to get all latest: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 devices, got %d", len(all))
	}
	if all["TURB-001"] != payload1 || all["PUMP-001"] != payload2 {
		t.Fatalf("unexpected GetAllLatest result: %+v", all)
	}
}
