package ratelimit

import (
	"context"
	"sync"
	"time"
)

type memoryStore struct {
	mu     sync.Mutex
	values map[string]int64
	ttl    time.Duration
}

func newMemoryStore(ttl time.Duration) *memoryStore {
	return &memoryStore{
		values: make(map[string]int64),
		ttl:    ttl,
	}
}

func (s *memoryStore) Add(ctx context.Context, key string, delta int64, window time.Duration) (CounterState, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.values[key] += delta
	if s.values[key] < 0 {
		s.values[key] = 0
	}
	return CounterState{
		Value: s.values[key],
		TTL:   s.ttl,
	}, nil
}

func (s *memoryStore) value(key string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.values[key]
}
