package repository

import (
	"context"
	"testing"
	"time"
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
