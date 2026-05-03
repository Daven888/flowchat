package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Manager handles Redis-based distributed locks for session generation.
type Manager struct {
	client *redis.Client
}

// NewManager creates a new lock Manager.
func NewManager(client *redis.Client) *Manager {
	return &Manager{client: client}
}

// Acquire tries to acquire the session generation lock via SETNX.
// Returns true if the lock was acquired, false if already held.
func (m *Manager) Acquire(ctx context.Context, sessionID int64, requestID string, ttlSeconds int) (bool, error) {
	key := keyForSession(sessionID)
	ok, err := m.client.SetNX(ctx, key, requestID, time.Duration(ttlSeconds)*time.Second).Result()
	if err != nil {
		return false, fmt.Errorf("redis lock acquire error: %w", err)
	}
	return ok, nil
}

// Release releases the lock only if it is still held by the given requestID.
// Uses a Lua script to ensure atomicity of compare-and-delete.
func (m *Manager) Release(ctx context.Context, sessionID int64, requestID string) error {
	key := keyForSession(sessionID)
	script := `
		if redis.call("GET", KEYS[1]) == ARGV[1] then
			return redis.call("DEL", KEYS[1])
		else
			return 0
		end
	`
	_, err := m.client.Eval(ctx, script, []string{key}, requestID).Result()
	if err != nil {
		return fmt.Errorf("redis lock release error: %w", err)
	}
	return nil
}

func keyForSession(sessionID int64) string {
	return fmt.Sprintf("chat:session:generating:%d", sessionID)
}
