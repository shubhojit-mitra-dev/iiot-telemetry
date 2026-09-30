package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log/slog"
	"math"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"sync/atomic"
	"syscall"
	"time"

	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/model"
)

// DeviceConfig defines the baseline operating profile for an industrial machine.
type DeviceConfig struct {
	ID       string  `json:"id"`
	Name     string  `json:"name"`
	Type     string  `json:"type"`
	BaseTemp float64 `json:"base_temp"`
	BaseVib  float64 `json:"base_vib"`
	BaseRPM  int64   `json:"base_rpm"`
}

// DefaultDeviceConfigs returns the 10 industrial machines mirroring the frontend fleet.
func DefaultDeviceConfigs() []DeviceConfig {
	return []DeviceConfig{
		{ID: "TURB-001", Name: "Gas Turbine A", Type: "Turbine", BaseTemp: 82.0, BaseVib: 6.5, BaseRPM: 3200},
		{ID: "TURB-002", Name: "Gas Turbine B", Type: "Turbine", BaseTemp: 85.0, BaseVib: 7.0, BaseRPM: 3150},
		{ID: "COMP-001", Name: "Air Compressor", Type: "Compressor", BaseTemp: 78.0, BaseVib: 5.8, BaseRPM: 2800},
		{ID: "PUMP-001", Name: "Coolant Pump A", Type: "Pump", BaseTemp: 65.0, BaseVib: 4.2, BaseRPM: 1800},
		{ID: "PUMP-002", Name: "Coolant Pump B", Type: "Pump", BaseTemp: 67.0, BaseVib: 4.5, BaseRPM: 1820},
		{ID: "CONV-001", Name: "Main Conveyor", Type: "Conveyor", BaseTemp: 55.0, BaseVib: 3.8, BaseRPM: 900},
		{ID: "WELD-001", Name: "Robotic Welder 1", Type: "Welder", BaseTemp: 95.0, BaseVib: 8.2, BaseRPM: 2400},
		{ID: "WELD-002", Name: "Robotic Welder 2", Type: "Welder", BaseTemp: 92.0, BaseVib: 7.9, BaseRPM: 2380},
		{ID: "CNC-001", Name: "CNC Lathe Alpha", Type: "CNC", BaseTemp: 72.0, BaseVib: 5.5, BaseRPM: 4500},
		{ID: "CNC-002", Name: "CNC Mill Beta", Type: "CNC", BaseTemp: 74.0, BaseVib: 5.8, BaseRPM: 4200},
	}
}

// MachineActor maintains the persistent physical telemetry state of a single machine.
type MachineActor struct {
	mu                     sync.Mutex
	Config                 DeviceConfig
	CurrentTemp            float64
	CurrentVib             float64
	CurrentRPM             int64
	InAnomaly              bool
	AnomalyTicksRemaining  int
	TargetAnomalyTemp      float64
	TargetAnomalyVib       float64
	rng                    *rand.Rand
}

// NewMachineActor constructs a stateful actor initialized with baseline metrics.
func NewMachineActor(cfg DeviceConfig, rng *rand.Rand) *MachineActor {
	if rng == nil {
		rng = rand.New(rand.NewSource(time.Now().UnixNano()))
	}
	return &MachineActor{
		Config:      cfg,
		CurrentTemp: cfg.BaseTemp,
		CurrentVib:  cfg.BaseVib,
		CurrentRPM:  cfg.BaseRPM,
		rng:         rng,
	}
}

// InjectAnomaly forces the actor into an anomaly state for a specified number of ticks.
func (a *MachineActor) InjectAnomaly(ticks int, temp float64, vib float64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.InAnomaly = true
	a.AnomalyTicksRemaining = ticks
	a.TargetAnomalyTemp = temp
	a.TargetAnomalyVib = vib
}

// Tick calculates the next telemetry state incorporating Gaussian noise, mean reversion, and anomalies.
func (a *MachineActor) Tick(now time.Time, anomalyRate float64) model.TelemetryPayload {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Check for probabilistic anomaly trigger if not already undergoing an incident
	if !a.InAnomaly && anomalyRate > 0 && a.rng.Float64() < anomalyRate {
		a.InAnomaly = true
		a.AnomalyTicksRemaining = 3 + a.rng.Intn(3) // 3-5 consecutive ticks of anomaly
		a.TargetAnomalyTemp = 122.0 + a.rng.Float64()*18.0 // 122°C - 140°C
		a.TargetAnomalyVib = a.Config.BaseVib + 6.0 + a.rng.Float64()*4.0 // 10 - 15 mm/s
	}

	if a.InAnomaly {
		// Elevate metrics directly into critical anomaly zone
		a.CurrentTemp = a.TargetAnomalyTemp + (a.rng.NormFloat64() * 0.5)
		a.CurrentVib = a.TargetAnomalyVib + (a.rng.NormFloat64() * 0.3)
		a.CurrentRPM = a.Config.BaseRPM + int64(a.rng.NormFloat64()*120.0)

		a.AnomalyTicksRemaining--
		if a.AnomalyTicksRemaining <= 0 {
			a.InAnomaly = false
		}
	} else {
		// Normal physics: Mean-reverting Ornstein-Uhlenbeck-style drift towards base metrics
		tempMeanReversion := (a.Config.BaseTemp - a.CurrentTemp) * 0.10
		tempNoise := a.rng.NormFloat64() * 0.4
		a.CurrentTemp += tempMeanReversion + tempNoise

		vibMeanReversion := (a.Config.BaseVib - a.CurrentVib) * 0.12
		vibNoise := a.rng.NormFloat64() * 0.15
		a.CurrentVib += vibMeanReversion + vibNoise

		rpmMeanReversion := float64(a.Config.BaseRPM-a.CurrentRPM) * 0.15
		rpmNoise := a.rng.NormFloat64() * 15.0
		a.CurrentRPM += int64(rpmMeanReversion + rpmNoise)
	}

	return model.TelemetryPayload{
		DeviceID:    a.Config.ID,
		Timestamp:   now.UnixMilli(),
		Temperature: math.Round(a.CurrentTemp*10) / 10,
		Vibration:   math.Round(a.CurrentVib*10) / 10,
		RPM:         a.CurrentRPM,
	}
}

// SimulatorConfig holds parameters for the fleet simulation engine.
type SimulatorConfig struct {
	TargetURL   string
	Interval    time.Duration
	AnomalyRate float64
	HTTPClient  *http.Client
	Devices     []DeviceConfig
}

// Simulator manages concurrent machinery actor goroutines and transmits payloads to the ingestion server.
type Simulator struct {
	cfg        SimulatorConfig
	actors     []*MachineActor
	wg         sync.WaitGroup
	sentCount  atomic.Uint64
	errorCount atomic.Uint64
}

// NewSimulator constructs an initialized simulator instance.
func NewSimulator(cfg SimulatorConfig) *Simulator {
	if cfg.Interval <= 0 {
		cfg.Interval = 1 * time.Second
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{
			Timeout: 5 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 50,
				IdleConnTimeout:     90 * time.Second,
			},
		}
	}
	if len(cfg.Devices) == 0 {
		cfg.Devices = DefaultDeviceConfigs()
	}

	actors := make([]*MachineActor, len(cfg.Devices))
	for i, d := range cfg.Devices {
		// Use unique seeds per machine actor for independent deterministic entropy
		actors[i] = NewMachineActor(d, rand.New(rand.NewSource(time.Now().UnixNano()+int64(i*1000))))
	}

	return &Simulator{
		cfg:    cfg,
		actors: actors,
	}
}

// Start launches a dedicated background goroutine for each virtual machine actor.
func (s *Simulator) Start(ctx context.Context) {
	for _, actor := range s.actors {
		s.wg.Add(1)
		go s.runActor(ctx, actor)
	}
}

// runActor executes the periodic ticker loop for a single machine actor.
func (s *Simulator) runActor(ctx context.Context, actor *MachineActor) {
	defer s.wg.Done()

	ticker := time.NewTicker(s.cfg.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case now := <-ticker.C:
			payload := actor.Tick(now, s.cfg.AnomalyRate)
			if err := s.SendPayload(ctx, payload); err != nil {
				s.errorCount.Add(1)
				slog.Error("failed to dispatch telemetry frame",
					"device_id", payload.DeviceID,
					"error", err,
				)
			} else {
				s.sentCount.Add(1)
			}
		}
	}
}

// SendPayload serializes and POSTs a telemetry payload to the ingestion endpoint.
func (s *Simulator) SendPayload(ctx context.Context, payload model.TelemetryPayload) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal payload error: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, s.cfg.TargetURL, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request error: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.cfg.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("http dispatch error: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("unexpected http status: %d", resp.StatusCode)
	}
	return nil
}

// Wait blocks until all machine actor goroutines have completed.
func (s *Simulator) Wait() {
	s.wg.Wait()
}

// Stats returns runtime metrics for monitoring the simulator.
func (s *Simulator) Stats() (sent uint64, errors uint64) {
	return s.sentCount.Load(), s.errorCount.Load()
}

func main() {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	targetURL := flag.String("url", "http://localhost:8080/api/v1/telemetry", "Target HTTP Ingestion URL")
	interval := flag.Duration("interval", 1*time.Second, "Telemetry generation interval per machine")
	anomalyRate := flag.Float64("anomaly-rate", 0.02, "Probability of anomaly occurrence per tick (0.0 - 1.0)")
	duration := flag.Duration("duration", 0, "Execution duration (0 for indefinite until signal)")
	flag.Parse()

	slog.Info("starting iiot machinery edge simulator",
		"target_url", *targetURL,
		"interval", *interval,
		"anomaly_rate", *anomalyRate,
		"device_count", len(DefaultDeviceConfigs()),
	)

	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	if *duration > 0 {
		var timeCancel context.CancelFunc
		ctx, timeCancel = context.WithTimeout(ctx, *duration)
		defer timeCancel()
	}

	sim := NewSimulator(SimulatorConfig{
		TargetURL:   *targetURL,
		Interval:    *interval,
		AnomalyRate: *anomalyRate,
	})

	sim.Start(ctx)

	// Block until context cancellation (Ctrl+C or duration elapsed)
	<-ctx.Done()
	slog.Info("shutdown signal received, stopping machine actors...")

	sim.Wait()
	sent, errs := sim.Stats()
	slog.Info("edge simulator shutdown complete",
		"total_sent", sent,
		"total_errors", errs,
	)
}
