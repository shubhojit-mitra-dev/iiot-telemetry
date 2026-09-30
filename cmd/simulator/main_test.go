package main

import (
	"context"
	"encoding/json"
	"math"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/model"
)

func TestDefaultDeviceConfigs(t *testing.T) {
	devices := DefaultDeviceConfigs()
	if len(devices) != 10 {
		t.Fatalf("expected 10 default devices, got %d", len(devices))
	}

	expectedIDs := []string{
		"TURB-001", "TURB-002", "COMP-001", "PUMP-001", "PUMP-002",
		"CONV-001", "WELD-001", "WELD-002", "CNC-001", "CNC-002",
	}

	for i, expected := range expectedIDs {
		if devices[i].ID != expected {
			t.Errorf("device[%d] expected ID %s, got %s", i, expected, devices[i].ID)
		}
		if devices[i].BaseTemp <= 0 || devices[i].BaseVib <= 0 || devices[i].BaseRPM <= 0 {
			t.Errorf("device %s has non-positive base metrics: %+v", expected, devices[i])
		}
	}
}

func TestMachineActor_PhysicsAndMeanReversion(t *testing.T) {
	cfg := DeviceConfig{
		ID:       "TEST-001",
		Name:     "Test Turbine",
		Type:     "Turbine",
		BaseTemp: 80.0,
		BaseVib:  5.0,
		BaseRPM:  3000,
	}

	rng := rand.New(rand.NewSource(42))
	actor := NewMachineActor(cfg, rng)

	// Artificially elevate temperature to test mean reversion physics
	actor.CurrentTemp = 150.0

	now := time.Now()
	for i := 0; i < 40; i++ {
		p := actor.Tick(now.Add(time.Duration(i)*time.Second), 0.0)
		if p.DeviceID != "TEST-001" {
			t.Errorf("expected DeviceID TEST-001, got %s", p.DeviceID)
		}
		if p.Timestamp <= 0 {
			t.Errorf("expected positive timestamp, got %d", p.Timestamp)
		}
	}

	// Mean reversion should pull temperature back near operational bounds (80 ± 5)
	if actor.CurrentTemp >= 105.0 {
		t.Errorf("expected temperature to mean-revert toward 80, but remains high: %f", actor.CurrentTemp)
	}
}

func TestMachineActor_MachineTypeDynamics(t *testing.T) {
	// Test Welder thermal pulse dynamics
	welderCfg := DeviceConfig{
		ID:       "WELD-001",
		Name:     "Robotic Welder",
		Type:     "Welder",
		BaseTemp: 95.0,
		BaseVib:  8.2,
		BaseRPM:  2400,
	}
	welder := NewMachineActor(welderCfg, rand.New(rand.NewSource(1)))
	welder.Cycle = 0.0

	var maxTemp float64
	now := time.Now()
	for i := 0; i < 50; i++ {
		p := welder.Tick(now.Add(time.Duration(i)*time.Second), 0.0)
		if p.Temperature > maxTemp {
			maxTemp = p.Temperature
		}
	}

	// Welder duty cycle should push temperature above base (95°C) during arc passes
	if maxTemp <= welderCfg.BaseTemp {
		t.Errorf("expected welder duty cycle to generate thermal pulses above base, max was: %f", maxTemp)
	}
}

func TestMachineActor_AnomalyInjection(t *testing.T) {
	cfg := DeviceConfig{
		ID:       "PUMP-001",
		Name:     "Coolant Pump A",
		Type:     "Pump",
		BaseTemp: 65.0,
		BaseVib:  4.2,
		BaseRPM:  1800,
	}

	rng := rand.New(rand.NewSource(100))
	actor := NewMachineActor(cfg, rng)

	actor.InjectAnomaly(3, 130.0, 14.5)

	now := time.Now()

	p1 := actor.Tick(now, 0.0)
	if p1.Temperature < 120.0 {
		t.Errorf("expected anomaly temp > 120, got %f", p1.Temperature)
	}
	if p1.Vibration < 12.0 {
		t.Errorf("expected anomaly vibration > 12, got %f", p1.Vibration)
	}

	p2 := actor.Tick(now.Add(time.Second), 0.0)
	if p2.Temperature < 120.0 {
		t.Errorf("expected tick 2 temp > 120, got %f", p2.Temperature)
	}

	p3 := actor.Tick(now.Add(2*time.Second), 0.0)
	if p3.Temperature < 120.0 {
		t.Errorf("expected tick 3 temp > 120, got %f", p3.Temperature)
	}

	// Tick 4: Anomaly expired
	p4 := actor.Tick(now.Add(3*time.Second), 0.0)
	if p4.Temperature > 128.0 {
		t.Errorf("expected temperature to drop after anomaly expired, got %f", p4.Temperature)
	}
}

func TestMachineActor_ProbabilisticAnomaly(t *testing.T) {
	cfg := DeviceConfig{
		ID:       "CNC-001",
		Name:     "CNC Lathe",
		Type:     "CNC",
		BaseTemp: 72.0,
		BaseVib:  5.5,
		BaseRPM:  4500,
	}

	rng := rand.New(rand.NewSource(2026))
	actor := NewMachineActor(cfg, rng)

	// Anomaly rate 1.0 (100% chance)
	p := actor.Tick(time.Now(), 1.0)
	if p.Temperature < 120.0 && p.Vibration < 12.0 {
		t.Errorf("expected 100%% anomaly rate to trigger threshold breach, got temp=%f, vib=%f", p.Temperature, p.Vibration)
	}
}

func TestMachineActor_ConcurrentThreadSafety(t *testing.T) {
	cfg := DeviceConfig{
		ID:       "TURB-001",
		Name:     "Turbine",
		Type:     "Turbine",
		BaseTemp: 82.0,
		BaseVib:  6.5,
		BaseRPM:  3200,
	}

	actor := NewMachineActor(cfg, rand.New(rand.NewSource(time.Now().UnixNano())))

	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	wg.Add(workers * 2)
	for i := 0; i < workers; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				_ = actor.Tick(time.Now(), 0.05)
			}
		}()
		go func() {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				if j%20 == 0 {
					actor.InjectAnomaly(2, 125.0, 13.0)
				}
			}
		}()
	}

	wg.Wait()
}

func TestSimulator_SendPayload_Success(t *testing.T) {
	var receivedPayload model.TelemetryPayload
	var receivedHeader string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeader = r.Header.Get("Content-Type")
		if err := json.NewDecoder(r.Body).Decode(&receivedPayload); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"status":"accepted"}`))
	}))
	defer server.Close()

	sim := NewSimulator(SimulatorConfig{
		TargetURL:   server.URL,
		Interval:    10 * time.Millisecond,
		AnomalyRate: 0.0,
		HTTPClient:  server.Client(),
	})

	payload := model.TelemetryPayload{
		DeviceID:    "TURB-001",
		Timestamp:   1700000000000,
		Temperature: 84.5,
		Vibration:   6.8,
		RPM:         3210,
	}

	err := sim.SendPayload(context.Background(), payload)
	if err != nil {
		t.Fatalf("unexpected SendPayload error: %v", err)
	}

	if receivedHeader != "application/json" {
		t.Errorf("expected Content-Type application/json, got %s", receivedHeader)
	}
	if receivedPayload.DeviceID != "TURB-001" || math.Abs(receivedPayload.Temperature-84.5) > 0.001 {
		t.Errorf("unexpected received payload: %+v", receivedPayload)
	}
}

func TestSimulator_SendPayload_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server failure", http.StatusInternalServerError)
	}))
	defer server.Close()

	sim := NewSimulator(SimulatorConfig{
		TargetURL:   server.URL,
		Interval:    10 * time.Millisecond,
		AnomalyRate: 0.0,
		HTTPClient:  server.Client(),
	})

	payload := model.TelemetryPayload{
		DeviceID:  "DEV-001",
		Timestamp: time.Now().UnixMilli(),
	}

	err := sim.SendPayload(context.Background(), payload)
	if err == nil {
		t.Fatal("expected error on 500 response, got nil")
	}
}

func TestSimulator_Lifecycle(t *testing.T) {
	var count atomic.Int64

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	sim := NewSimulator(SimulatorConfig{
		TargetURL:   server.URL,
		Interval:    15 * time.Millisecond,
		AnomalyRate: 0.0,
		HTTPClient:  server.Client(),
		Devices: []DeviceConfig{
			{ID: "D1", BaseTemp: 70, BaseVib: 5, BaseRPM: 2000},
			{ID: "D2", BaseTemp: 80, BaseVib: 6, BaseRPM: 3000},
		},
	})

	ctx, cancel := context.WithCancel(context.Background())
	sim.Start(ctx)

	time.Sleep(100 * time.Millisecond)

	cancel()
	sim.Wait()

	sent, errs := sim.Stats()
	if sent == 0 {
		t.Errorf("expected sent > 0, got %d", sent)
	}
	if errs != 0 {
		t.Errorf("expected 0 errors, got %d", errs)
	}
	if count.Load() == 0 {
		t.Errorf("server received 0 messages")
	}
}
