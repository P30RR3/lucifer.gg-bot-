package detectors

import (
	"context"
	"fmt"
	"sync"
	"time"

	"krypton.dev/internal/antiraid"
)

type Nuke struct {
	mu       sync.Mutex
	limit    int
	duration time.Duration
	buckets  map[string][]time.Time
	weight   float64
}

func NewNuke(limit int, duration time.Duration) *Nuke {
	if limit <= 0 {
		limit = 3
	}
	if duration <= 0 {
		duration = 30 * time.Second
	}
	return &Nuke{
		limit:    limit,
		duration: duration,
		buckets:  make(map[string][]time.Time),
		weight:   2.0,
	}
}

func (n *Nuke) Name() string    { return "nuke" }
func (n *Nuke) Weight() float64 { return n.weight }

func (n *Nuke) Evaluate(ctx context.Context, ev *antiraid.Event) *antiraid.DetectorResult {
	if ev.UserID == "" {
		return &antiraid.DetectorResult{Triggered: false}
	}

	now := time.Now()
	cutoff := now.Add(-n.duration)
	key := ev.UserID

	n.mu.Lock()
	bucket := n.buckets[key]
	idx := 0
	for idx < len(bucket) && bucket[idx].Before(cutoff) {
		idx++
	}
	if idx > 0 {
		bucket = bucket[idx:]
	}
	bucket = append(bucket, now)
	n.buckets[key] = bucket
	count := len(bucket)
	n.mu.Unlock()

	if count < n.limit {
		return &antiraid.DetectorResult{Triggered: false}
	}

	ratio := float64(count) / float64(n.limit)
	score := 85.0 + (ratio-1.0)*10
	if score > 100 {
		score = 100
	}

	return &antiraid.DetectorResult{
		Triggered: true,
		Score:     score,
		Reason:    fmt.Sprintf("comportamiento destructivo: %d acciones en %s", count, n.duration),
		Evidence: map[string]any{
			"actions":  count,
			"limit":    n.limit,
			"duration": n.duration.String(),
		},
	}
}
