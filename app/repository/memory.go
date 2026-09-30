package repository

import (
	"context"
	"sync"

	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/model"
)

// MemoryRepository provides a thread-safe in-memory store for latest device telemetry.
// Useful for local development and integration tests when Redis is not running.
type MemoryRepository struct {
	mu    sync.RWMutex
	store map[string]model.TelemetryPayload
}

// NewMemoryRepository initializes an empty in-memory repository.
func NewMemoryRepository() *MemoryRepository {
	return &MemoryRepository{
		store: make(map[string]model.TelemetryPayload),
	}
}

// SaveLatest stores the latest telemetry payload for a device.
func (m *MemoryRepository) SaveLatest(ctx context.Context, payload model.TelemetryPayload) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.store[payload.DeviceID] = payload
	return nil
}

// GetLatest retrieves the latest telemetry frame for a single device.
func (m *MemoryRepository) GetLatest(ctx context.Context, deviceID string) (*model.TelemetryPayload, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	p, ok := m.store[deviceID]
	if !ok {
		return nil, ErrNotFound
	}
	res := p
	return &res, nil
}

// GetAllLatest returns a copy of all current device telemetry records.
func (m *MemoryRepository) GetAllLatest(ctx context.Context) (map[string]model.TelemetryPayload, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := make(map[string]model.TelemetryPayload, len(m.store))
	for k, v := range m.store {
		snapshot[k] = v
	}
	return snapshot, nil
}

// Close is a no-op for the memory repository.
func (m *MemoryRepository) Close() error {
	return nil
}
