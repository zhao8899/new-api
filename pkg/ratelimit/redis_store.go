package ratelimit

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type RedisStore struct {
	client *redis.Client
}

func NewRedisStore(client *redis.Client) RedisStore {
	return RedisStore{client: client}
}

func (s RedisStore) Add(ctx context.Context, key string, delta int64, window time.Duration) (CounterState, error) {
	pipe := s.client.TxPipeline()
	valueCmd := pipe.IncrBy(ctx, key, delta)
	ttlCmd := pipe.TTL(ctx, key)
	if window > 0 {
		pipe.ExpireNX(ctx, key, window)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return CounterState{}, err
	}

	ttl := ttlCmd.Val()
	if ttl <= 0 && window > 0 {
		ttl = window
	}
	return CounterState{
		Value: valueCmd.Val(),
		TTL:   ttl,
	}, nil
}
