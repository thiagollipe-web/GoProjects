// Package ratelimit implements a per-key token bucket.
package ratelimit

import (
	"sync"
	"time"
)

type bucket struct {
	tokens float64
	last   time.Time
}

// Limiter allows `perMinute` requests per key with a burst of the same size.
type Limiter struct {
	mu        sync.Mutex
	perMinute float64
	buckets   map[string]*bucket
	now       func() time.Time
}

// New returns a limiter. perMinute <= 0 disables limiting.
func New(perMinute int) *Limiter {
	return &Limiter{perMinute: float64(perMinute), buckets: map[string]*bucket{}, now: time.Now}
}

// Allow reports whether the key may proceed, and how long to wait if not.
func (l *Limiter) Allow(key string) (bool, time.Duration) {
	if l.perMinute <= 0 {
		return true, 0
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.perMinute, last: now}
		l.buckets[key] = b
	}
	b.tokens += now.Sub(b.last).Seconds() * l.perMinute / 60
	if b.tokens > l.perMinute {
		b.tokens = l.perMinute
	}
	b.last = now
	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	need := (1 - b.tokens) * 60 / l.perMinute
	return false, time.Duration(need * float64(time.Second))
}
