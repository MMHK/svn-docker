package auth

import (
	"sync"
	"time"
)

// RateLimiter is an in-memory sliding window. It is used for login attempts
// (5 per minute per client IP by default) and never grows beyond the clients
// seen inside the current window.
type RateLimiter struct {
	limit  int
	window time.Duration
	now    func() time.Time

	mu   sync.Mutex
	hits map[string][]time.Time
}

// NewRateLimiter builds a limiter. A non-positive limit disables limiting.
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	if window <= 0 {
		window = time.Minute
	}

	return &RateLimiter{
		limit:  limit,
		window: window,
		now:    time.Now,
		hits:   make(map[string][]time.Time),
	}
}

// Allow records one attempt for key. When the attempt would exceed the limit
// it is rejected and the suggested retry delay is returned.
func (r *RateLimiter) Allow(key string) (bool, time.Duration) {
	if r.limit <= 0 {
		return true, 0
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	now := r.now()
	cutoff := now.Add(-r.window)

	attempts := r.hits[key]
	kept := attempts[:0]

	for _, attempt := range attempts {
		if attempt.After(cutoff) {
			kept = append(kept, attempt)
		}
	}

	if len(kept) >= r.limit {
		retry := kept[0].Add(r.window).Sub(now)
		if retry < 0 {
			retry = 0
		}

		r.hits[key] = kept

		return false, retry
	}

	kept = append(kept, now)
	r.hits[key] = kept

	// Drop clients whose window expired so the map cannot grow without bound.
	for other, values := range r.hits {
		if len(values) == 0 || !values[len(values)-1].After(cutoff) {
			delete(r.hits, other)
		}
	}

	return true, 0
}
