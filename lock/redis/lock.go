package redis

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

// Lock implements cronlib.DistLock using Redis.
type Lock struct {
	client *redis.Client
}

// New creates a new Redis lock.
func New(addr string) *Lock {
	rdb := redis.NewClient(&redis.Options{
		Addr: addr,
	})
	return &Lock{client: rdb}
}

// Lock attempts to acquire a lock.
func (l *Lock) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	// SetNX returns true if the key was set (lock acquired).
	return l.client.SetNX(ctx, key, "locked", ttl).Result()
}

// Unlock releases the lock.
func (l *Lock) Unlock(ctx context.Context, key string) error {
	return l.client.Del(ctx, key).Err()
}
