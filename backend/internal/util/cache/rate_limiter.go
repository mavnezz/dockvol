package cache_utils

import (
	"fmt"
	"sync"
	"time"
)

// window counts requests seen since resetAt within a single fixed window.
type window struct {
	count   int
	resetAt time.Time
}

// RateLimiter is a process-local fixed-window limiter. Single-process DockVol has
// no cross-instance traffic to coordinate, so a per-key counter guarded by a mutex
// enforces the limit without any network round-trips.
type RateLimiter struct {
	mu      sync.Mutex
	windows map[string]window
}

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		windows: make(map[string]window),
	}
}

// CheckLimit records one request against identifier+endpoint and reports whether
// it stays within maxRequests over windowDuration. The window resets on the first
// request after it elapses.
func (r *RateLimiter) CheckLimit(
	identifier string,
	endpoint string,
	maxRequests int,
	windowDuration time.Duration,
) (bool, error) {
	key := fmt.Sprintf("%s:%s", endpoint, identifier)
	now := time.Now().UTC()

	r.mu.Lock()
	defer r.mu.Unlock()

	current, exists := r.windows[key]
	if !exists || now.After(current.resetAt) {
		r.windows[key] = window{count: 1, resetAt: now.Add(windowDuration)}

		return 1 <= maxRequests, nil
	}

	current.count++
	r.windows[key] = current

	return current.count <= maxRequests, nil
}
