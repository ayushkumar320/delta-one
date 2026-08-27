// Package service holds the gateway's rate limiter.
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Limiter is a fixed-window rate limiter backed by Redis, so the limit is
// shared by every gateway replica rather than counted per process.
//
// ponytail: a fixed window lets a client spend a full window's budget either
// side of a boundary. Move to a sliding window if that burst ever matters.
type Limiter struct {
	rdb    *redis.Client
	limit  int
	window time.Duration
}

// NewLimiter returns a Limiter allowing limit requests per window per key.
func NewLimiter(rdb *redis.Client, limit int, window time.Duration) *Limiter {
	return &Limiter{rdb: rdb, limit: limit, window: window}
}

// Allow reports whether the key may make another request, along with how many
// requests remain in the current window.
//
// A Redis failure allows the request: the limiter protects the services behind
// the gateway, and losing it should not take the platform down with it.
func (l *Limiter) Allow(ctx context.Context, key string) (allowed bool, remaining int) {
	window := time.Now().UnixMilli() / l.window.Milliseconds()
	counter := fmt.Sprintf("ratelimit:%s:%d", key, window)

	pipe := l.rdb.Pipeline()
	count := pipe.Incr(ctx, counter)
	// The expiry is set on every request rather than only the first, which
	// costs one command and avoids a key that never expires if the process
	// dies between INCR and EXPIRE.
	pipe.Expire(ctx, counter, l.window)
	if _, err := pipe.Exec(ctx); err != nil {
		return true, l.limit
	}

	used := int(count.Val())
	if used > l.limit {
		return false, 0
	}
	return true, l.limit - used
}

// Limit returns the configured request ceiling per window.
func (l *Limiter) Limit() int { return l.limit }
