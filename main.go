package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"
	"time"

	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/api"
	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/repository"
	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/service"
)

func main() {
	// 1. Initialize High-Performance Structured JSON Logger
	logLevel := slog.LevelInfo
	if os.Getenv("LOG_LEVEL") == "WARN" {
		logLevel = slog.LevelWarn
	}
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: logLevel,
	}))
	slog.SetDefault(logger)

	slog.Info("starting iiot telemetry platform",
		"go_version", runtime.Version(),
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"num_cpu", runtime.NumCPU(),
		"bench_mode", os.Getenv("BENCH_MODE") == "1",
	)

	// 2. Setup Context with Signal Notification for Graceful Shutdown
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	// 3. Environment & Configuration
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}
	redisPassword := os.Getenv("REDIS_PASSWORD")

	// 4. Initialize Repository (Redis with graceful fallback to In-Memory store)
	var repo repository.TelemetryRepository
	pingCtx, pingCancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer pingCancel()

	redisRepo, err := repository.NewRedisRepository(pingCtx, redisAddr, redisPassword, 0)
	if err != nil {
		slog.Warn("redis connection unavailable, activating high-performance in-memory repository",
			"target_addr", redisAddr,
			"reason", err.Error(),
		)
		repo = repository.NewMemoryRepository()
	} else {
		slog.Info("successfully connected to redis cluster", "addr", redisAddr)
		repo = redisRepo
	}
	defer func() {
		_ = repo.Close()
	}()

	// 5. Initialize Real-Time WebSocket Hub
	hub := service.NewHub()
	go hub.Run(ctx)

	// 6. Initialize Worker Pool for Ingestion Pipeline
	workerCount := runtime.NumCPU() * 2
	if envWorkers := os.Getenv("WORKER_COUNT"); envWorkers != "" {
		if w, err := strconv.Atoi(envWorkers); err == nil && w > 0 {
			workerCount = w
		}
	}
	queueCapacity := 10000
	if envCap := os.Getenv("QUEUE_CAPACITY"); envCap != "" {
		if c, err := strconv.Atoi(envCap); err == nil && c > 0 {
			queueCapacity = c
		}
	}

	ingestService := service.NewIngestionService(repo, hub, workerCount, queueCapacity)
	ingestService.Start(ctx)

	// 7. Initialize HTTP Server
	handler := api.NewHandler(ingestService, repo, hub)
	router := api.NewServer(handler)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%s", port),
		Handler:      router,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 10 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// 8. Launch Server Asynchronously
	serverErr := make(chan error, 1)
	go func() {
		slog.Info("telemetry server listening",
			"addr", server.Addr,
			"http_endpoint", fmt.Sprintf("http://localhost:%s/api/v1/telemetry", port),
			"ws_endpoint", fmt.Sprintf("ws://localhost:%s/ws/telemetry", port),
		)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	// 9. Block until OS signal or Server Error
	select {
	case err := <-serverErr:
		slog.Error("server listener encountered fatal error", "error", err)
	case <-ctx.Done():
		slog.Info("shutdown signal received, initiating graceful teardown...")
	}

	// 10. Clean Graceful Shutdown with 10s Deadline
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("server forced to shutdown due to timeout", "error", err)
	}

	// 11. Gracefully drain and stop worker pool AFTER HTTP listener stops accepting requests
	ingestService.Stop()

	slog.Info("telemetry ingestion platform successfully stopped")
}
