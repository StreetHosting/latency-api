package mtr

import (
	"sync"
	"time"
)

// Limiter enforces a minimum interval between MTR runs per key (e.g. client IP).
type Limiter struct {
	mu          sync.Mutex
	last        map[string]time.Time
	minInterval time.Duration
}

// NewLimiter creates a rate limiter.
func NewLimiter(minInterval time.Duration) *Limiter {
	return &Limiter{
		last:        make(map[string]time.Time),
		minInterval: minInterval,
	}
}

// Allow reports whether a new MTR run is permitted for key.
func (l *Limiter) Allow(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	if t, ok := l.last[key]; ok && now.Sub(t) < l.minInterval {
		return false
	}
	l.last[key] = now
	return true
}
