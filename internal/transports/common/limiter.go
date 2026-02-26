package common

import (
	"sync"
	"time"
)

// RateLimiter реализует простой sliding-window limit на key.
type RateLimiter struct {
	mu     sync.Mutex
	limit  int
	window time.Duration
	events map[string][]time.Time
}

// NewRateLimiter создает limiter с лимитом событий в окне.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if limit <= 0 {
		limit = 1
	}
	if window <= 0 {
		window = time.Second
	}
	return &RateLimiter{
		limit:  limit,
		window: window,
		events: make(map[string][]time.Time),
	}
}

// Allow возвращает true, если запрос укладывается в лимит.
func (l *RateLimiter) Allow(key string, now time.Time) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	cutoff := now.Add(-l.window)
	items := l.events[key]
	kept := items[:0]
	for _, ts := range items {
		if ts.After(cutoff) {
			kept = append(kept, ts)
		}
	}
	if len(kept) >= l.limit {
		l.events[key] = kept
		return false
	}
	kept = append(kept, now)
	l.events[key] = kept
	return true
}
