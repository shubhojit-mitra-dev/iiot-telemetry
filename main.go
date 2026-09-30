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
	"syscall"
	"time"

	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/api"
	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/repository"
	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/service"
)

func main() {
	// 1. Initialize High-Performance Structured JSON Logger
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("starting iiot telemetry platform",
		"go_version", runtime.Version(),
		"os", runtime.GOOS,
		"arch", runtime.GOARCH,
		"num_cpu", runtime.NumCPU(),
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
	ingestService := service.NewIngestionService(repo, hub, workerCount, 10000)
	ingestService.Start(ctx)
	defer ingestService.Stop()

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
		slog.Info("telemetry ingestion server listening", "addr", server.Addr)
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

	slog.Info("telemetry ingestion platform successfully stopped")
}
