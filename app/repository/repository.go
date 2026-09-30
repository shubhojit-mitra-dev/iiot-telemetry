package repository

import (
	"context"
	"errors"

	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/model"
)

// ErrNotFound is returned when telemetry for a requested device cannot be found.
var ErrNotFound = errors.New("device telemetry not found")

// TelemetryRepository defines persistent or in-memory access patterns for latest device telemetry.
type TelemetryRepository interface {
	// SaveLatest saves the latest telemetry frame for a given device.
	SaveLatest(ctx context.Context, payload model.TelemetryPayload) error

	// GetLatest retrieves the most recent telemetry frame recorded for a device.
	GetLatest(ctx context.Context, deviceID string) (*model.TelemetryPayload, error)

	// GetAllLatest retrieves the latest state across all observed edge devices.
	GetAllLatest(ctx context.Context) (map[string]model.TelemetryPayload, error)

	// Close terminates any underlying connection pools.
	Close() error
}
