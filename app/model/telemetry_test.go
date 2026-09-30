package model

import (
	"errors"
	"testing"
)

func TestTelemetryPayload_Validate(t *testing.T) {
	tests := []struct {
		name        string
		payload     TelemetryPayload
		expectedErr error
	}{
		{
			name: "valid payload",
			payload: TelemetryPayload{
				DeviceID:    "TURB-001",
				Timestamp:   1700000000000,
				Temperature: 85.5,
				Vibration:   6.2,
				RPM:         3200,
			},
			expectedErr: nil,
		},
		{
			name: "empty device id",
			payload: TelemetryPayload{
				DeviceID:    "",
				Timestamp:   1700000000000,
				Temperature: 85.5,
				Vibration:   6.2,
				RPM:         3200,
			},
			expectedErr: ErrEmptyDeviceID,
		},
		{
			name: "zero timestamp",
			payload: TelemetryPayload{
				DeviceID:    "TURB-001",
				Timestamp:   0,
				Temperature: 85.5,
				Vibration:   6.2,
				RPM:         3200,
			},
			expectedErr: ErrInvalidTimestamp,
		},
		{
			name: "negative timestamp",
			payload: TelemetryPayload{
				DeviceID:    "TURB-001",
				Timestamp:   -100,
				Temperature: 85.5,
				Vibration:   6.2,
				RPM:         3200,
			},
			expectedErr: ErrInvalidTimestamp,
		},
		{
			name: "minimum positive timestamp boundary",
			payload: TelemetryPayload{
				DeviceID:    "PUMP-001",
				Timestamp:   1,
				Temperature: 0.0,
				Vibration:   0.0,
				RPM:         0,
			},
			expectedErr: nil,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.payload.Validate()
			if !errors.Is(err, tc.expectedErr) {
				t.Fatalf("expected error %v, got %v", tc.expectedErr, err)
			}
		})
	}
}

func TestTelemetryPayload_Clone(t *testing.T) {
	original := TelemetryPayload{
		DeviceID:    "COMP-001",
		Timestamp:   1700000001000,
		Temperature: 78.4,
		Vibration:   5.9,
		RPM:         2850,
	}

	clone := original.Clone()

	if clone != original {
		t.Fatalf("expected clone to equal original, got %+v vs %+v", clone, original)
	}

	// Mutate original and assert clone is unaffected
	original.DeviceID = "MUTATED"
	original.Temperature = 999.9

	if clone.DeviceID == "MUTATED" || clone.Temperature == 999.9 {
		t.Fatalf("clone was mutated when original changed: %+v", clone)
	}
}

func TestPayloadPool_Recycling(t *testing.T) {
	// Test nil safety
	PutPayload(nil)

	// Acquire from pool
	p1 := GetPayload()
	if p1 == nil {
		t.Fatal("expected non-nil payload from pool")
	}

	// Populate values
	p1.DeviceID = "WELD-001"
	p1.Timestamp = 1700000002000
	p1.Temperature = 95.2
	p1.Vibration = 8.1
	p1.RPM = 2400

	// Release back to pool
	PutPayload(p1)

	// Ensure fields were zeroed out to prevent dirty memory reuse
	if p1.DeviceID != "" || p1.Timestamp != 0 || p1.Temperature != 0 || p1.Vibration != 0 || p1.RPM != 0 {
		t.Fatalf("expected payload fields to be zeroed upon PutPayload, got %+v", p1)
	}
}

func BenchmarkSyncPoolAllocation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := GetPayload()
		p.DeviceID = "TURB-001"
		p.Timestamp = 1700000000000
		p.Temperature = 82.5
		p.Vibration = 6.1
		p.RPM = 3200
		PutPayload(p)
	}
}

func BenchmarkStandardHeapAllocation(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		p := &TelemetryPayload{
			DeviceID:    "TURB-001",
			Timestamp:   1700000000000,
			Temperature: 82.5,
			Vibration:   6.1,
			RPM:         3200,
		}
		_ = p
	}
}
