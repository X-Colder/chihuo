package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

var ErrUnavailable = errors.New("rate limiter unavailable")

type Limiter interface {
	Allow(ctx context.Context, key string) (bool, error)
	Close() error
}

type localBucket struct {
	windowStart time.Time
	count       int
}

type LocalLimiter struct {
	mu     sync.Mutex
	rps    int
	burst  int
	bucket map[string]localBucket
}

func NewLocal(rps, burst int) *LocalLimiter {
	return &LocalLimiter{rps: rps, burst: burst, bucket: make(map[string]localBucket)}
}

func (l *LocalLimiter) Allow(_ context.Context, key string) (bool, error) {
	now := time.Now()
	window := now.Truncate(time.Second)
	l.mu.Lock()
	defer l.mu.Unlock()
	entry := l.bucket[key]
	if entry.windowStart != window {
		entry = localBucket{windowStart: window}
	}
	limit := l.rps
	if limit > l.burst {
		limit = l.burst
	}
	entry.count++
	l.bucket[key] = entry
	if len(l.bucket) > 100_000 {
		for bucketKey, value := range l.bucket {
			if now.Sub(value.windowStart) > 2*time.Second {
				delete(l.bucket, bucketKey)
			}
		}
	}
	return entry.count <= limit, nil
}

func (l *LocalLimiter) Close() error { return nil }

type RedisLimiter struct {
	client *redis.Client
	rps    int
	burst  int
}

func NewRedis(rawURL, password string, rps, burst int) (*RedisLimiter, error) {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" {
		return nil, fmt.Errorf("invalid REDIS_URL")
	}
	options, err := redis.ParseURL(rawURL)
	if err != nil {
		return nil, err
	}
	if password != "" {
		options.Password = password
	}
	return &RedisLimiter{client: redis.NewClient(options), rps: rps, burst: burst}, nil
}

func (l *RedisLimiter) Allow(ctx context.Context, key string) (bool, error) {
	windowKey := "chihuo:rate:" + key + ":" + strconv.FormatInt(time.Now().Unix(), 10)
	count, err := l.client.Incr(ctx, windowKey).Result()
	if err != nil {
		return false, fmt.Errorf("%w: %v", ErrUnavailable, err)
	}
	if count == 1 {
		_ = l.client.Expire(ctx, windowKey, 2*time.Second).Err()
	}
	limit := int64(l.rps)
	if l.burst > l.rps {
		limit = int64(l.burst)
	}
	return count <= limit, nil
}

func (l *RedisLimiter) Close() error { return l.client.Close() }
