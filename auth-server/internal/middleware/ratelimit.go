package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type RateLimiter struct {
	client *redis.Client
}

func NewRateLimiter(client *redis.Client) *RateLimiter {
	return &RateLimiter{client: client}
}

func (rl *RateLimiter) Check(ctx context.Context, key string, limit int, window time.Duration) (allowed bool, remaining int, err error) {
	count, err := rl.client.Incr(ctx, key).Result()
	if err != nil {
		return false, 0, fmt.Errorf("incr rate limit: %w", err)
	}

	if count == 1 {
		rl.client.Expire(ctx, key, window)
	}

	rem := limit - int(count)
	if rem < 0 {
		rem = 0
	}

	return int(count) <= limit, rem, nil
}
