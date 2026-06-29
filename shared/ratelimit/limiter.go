package ratelimit

import (
	"context"
	"time"

	"github.com/KoiralaSam/ZorbaHealth/shared/env"
	"github.com/redis/go-redis/v9"
)

// Limiter checks fixed-window rate limits.
type Limiter interface {
	Allow(ctx context.Context, key string, max int, window time.Duration) (allowed bool, retryAfter time.Duration, err error)
}

type redisLimiter struct {
	client *redis.Client
}

func NewFromEnv() Limiter {
	client := redis.NewClient(&redis.Options{
		Addr:     env.GetString("REDIS_ADDR", "localhost:6379"),
		Password: env.GetString("REDIS_PASSWORD", ""),
		DB:       0,
	})
	if err := client.Ping(context.Background()).Err(); err != nil {
		return noopLimiter{}
	}
	return &redisLimiter{client: client}
}

type noopLimiter struct{}

func (noopLimiter) Allow(ctx context.Context, key string, max int, window time.Duration) (bool, time.Duration, error) {
	return true, 0, nil
}

func (r *redisLimiter) Allow(ctx context.Context, key string, max int, window time.Duration) (bool, time.Duration, error) {
	if max <= 0 {
		return true, 0, nil
	}
	pipe := r.client.TxPipeline()
	incr := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	if _, err := pipe.Exec(ctx); err != nil {
		return true, 0, err
	}
	count, err := incr.Result()
	if err != nil {
		return true, 0, err
	}
	if int(count) <= max {
		return true, 0, nil
	}
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil || ttl <= 0 {
		ttl = window
	}
	return false, ttl, nil
}
