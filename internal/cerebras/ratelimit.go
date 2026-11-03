package cerebras

import (
	"context"
	"sync"
	"time"
)

type RateLimiter struct {
	mu       sync.Mutex
	tokens   float64
	capacity float64
	refill   float64
	last     time.Time
}

func NewRateLimiter(rpm, burst int) *RateLimiter {
	if rpm <= 0 {
		rpm = 30
	}
	if burst <= 0 {
		burst = 1
	}
	if burst > rpm {
		burst = rpm
	}
	return &RateLimiter{
		tokens:   float64(burst),
		capacity: float64(burst),
		refill:   float64(rpm) / 60.0,
		last:     time.Now(),
	}
}

func (r *RateLimiter) Wait(ctx context.Context) error {
	for {
		if r.tryTake() {
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(50 * time.Millisecond):
		}
	}
}

func (r *RateLimiter) tryTake() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.last).Seconds()
	r.last = now
	r.tokens += elapsed * r.refill
	if r.tokens > r.capacity {
		r.tokens = r.capacity
	}
	if r.tokens >= 1.0 {
		r.tokens -= 1.0
		return true
	}
	return false
}

func (r *RateLimiter) RemainingTokens() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.tokens
}

func (r *RateLimiter) Capacity() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.capacity
}

func (r *RateLimiter) RefillPerSecond() float64 {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.refill
}

func (r *RateLimiter) Reset() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.tokens = r.capacity
	r.last = time.Now()
}
