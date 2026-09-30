package model

import (
	"errors"
	"sync"
)

var (
	// ErrEmptyDeviceID is returned when device_id is missing or blank.
	ErrEmptyDeviceID = errors.New("device_id cannot be empty")
	// ErrInvalidTimestamp is returned when timestamp is non-positive.
	ErrInvalidTimestamp = errors.New("timestamp must be positive")
)

// TelemetryPayload represents an inbound industrial sensor measurement frame.
type TelemetryPayload struct {
	DeviceID    string  `json:"device_id"`
	Timestamp   int64   `json:"timestamp"`
	Temperature float64 `json:"temperature"`
	Vibration   float64 `json:"vibration"`
	RPM         int64   `json:"rpm"`
}

// Validate checks essential payload invariants to ensure data integrity.
func (p *TelemetryPayload) Validate() error {
	if p.DeviceID == "" {
		return ErrEmptyDeviceID
	}
	if p.Timestamp <= 0 {
		return ErrInvalidTimestamp
	}
	return nil
}

// payloadPool recycles TelemetryPayload instances to eliminate heap allocations
// on the hot ingestion path (10,000+ msgs/sec).
var payloadPool = sync.Pool{
	New: func() any {
		return new(TelemetryPayload)
	},
}

// GetPayload retrieves a recycled TelemetryPayload pointer from the pool.
func GetPayload() *TelemetryPayload {
	return payloadPool.Get().(*TelemetryPayload)
}

// PutPayload resets the fields of a TelemetryPayload and returns it to the pool.
func PutPayload(p *TelemetryPayload) {
	if p == nil {
		return
	}
	p.DeviceID = ""
	p.Timestamp = 0
	p.Temperature = 0
	p.Vibration = 0
	p.RPM = 0
	payloadPool.Put(p)
}

// Clone creates an independent copy of the payload for downstream processing.
func (p *TelemetryPayload) Clone() TelemetryPayload {
	return TelemetryPayload{
		DeviceID:    p.DeviceID,
		Timestamp:   p.Timestamp,
		Temperature: p.Temperature,
		Vibration:   p.Vibration,
		RPM:         p.RPM,
	}
}
