package repository

import (
	"context"
	"fmt"
	"net"
	"strconv"

	"github.com/redis/go-redis/v9"
	"github.com/shubhojit-mitra-dev/iiot-telemetry/app/model"
)

// RedisRepository implements TelemetryRepository using a Redis client connection pool.
type RedisRepository struct {
	client *redis.Client
}

// NewRedisRepository establishes a connection pool to Redis and validates connectivity.
func NewRedisRepository(ctx context.Context, addr string, password string, db int) (*RedisRepository, error) {
	// Fast TCP probe to avoid connection pool retry storm if Redis is offline
	var dialer net.Dialer
	conn, err := dialer.DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("failed to reach redis at %s: %w", addr, err)
	}
	_ = conn.Close()

	rdb := redis.NewClient(&redis.Options{
		Addr:         addr,
		Password:     password,
		DB:           db,
		PoolSize:     100, // Handle high concurrency worker connections
		MinIdleConns: 5,
		MaxRetries:   1, // Minimize retry noise when testing connectivity
	})

	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to ping redis at %s: %w", addr, err)
	}

	return &RedisRepository{client: rdb}, nil
}

// SaveLatest stores the telemetry frame as a Redis Hash under key "device:latest:<device_id>".
func (r *RedisRepository) SaveLatest(ctx context.Context, payload model.TelemetryPayload) error {
	key := fmt.Sprintf("device:latest:%s", payload.DeviceID)
	fields := map[string]interface{}{
		"device_id":   payload.DeviceID,
		"timestamp":   payload.Timestamp,
		"temperature": payload.Temperature,
		"vibration":   payload.Vibration,
		"rpm":         payload.RPM,
	}

	if err := r.client.HSet(ctx, key, fields).Err(); err != nil {
		return fmt.Errorf("failed to HSet %s: %w", key, err)
	}
	return nil
}

// GetLatest retrieves the latest telemetry values from the device hash.
func (r *RedisRepository) GetLatest(ctx context.Context, deviceID string) (*model.TelemetryPayload, error) {
	key := fmt.Sprintf("device:latest:%s", deviceID)
	data, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to HGetAll %s: %w", key, err)
	}
	if len(data) == 0 {
		return nil, ErrNotFound
	}

	ts, _ := strconv.ParseInt(data["timestamp"], 10, 64)
	temp, _ := strconv.ParseFloat(data["temperature"], 64)
	vib, _ := strconv.ParseFloat(data["vibration"], 64)
	rpm, _ := strconv.ParseInt(data["rpm"], 10, 64)

	return &model.TelemetryPayload{
		DeviceID:    data["device_id"],
		Timestamp:   ts,
		Temperature: temp,
		Vibration:   vib,
		RPM:         rpm,
	}, nil
}

// GetAllLatest scans for all device keys and returns their current state.
func (r *RedisRepository) GetAllLatest(ctx context.Context) (map[string]model.TelemetryPayload, error) {
	var cursor uint64
	results := make(map[string]model.TelemetryPayload)

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, "device:latest:*", 50).Result()
		if err != nil {
			return nil, fmt.Errorf("redis scan failed: %w", err)
		}

		for _, key := range keys {
			data, err := r.client.HGetAll(ctx, key).Result()
			if err != nil || len(data) == 0 {
				continue
			}
			ts, _ := strconv.ParseInt(data["timestamp"], 10, 64)
			temp, _ := strconv.ParseFloat(data["temperature"], 64)
			vib, _ := strconv.ParseFloat(data["vibration"], 64)
			rpm, _ := strconv.ParseInt(data["rpm"], 10, 64)

			devID := data["device_id"]
			results[devID] = model.TelemetryPayload{
				DeviceID:    devID,
				Timestamp:   ts,
				Temperature: temp,
				Vibration:   vib,
				RPM:         rpm,
			}
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	return results, nil
}

// Close gracefully closes the Redis client connection pool.
func (r *RedisRepository) Close() error {
	return r.client.Close()
}
