package detectors

import (
	"context"
	"fmt"
	"regexp"
	"sync"
	"time"

	"krypton.dev/internal/antiraid"
)

var mentionRe = regexp.MustCompile(`<@[!&]?\d+>`)

type MentionSpam struct {
	mu       sync.Mutex
	limit    int
	duration time.Duration
	buckets  map[string][]int
	weight   float64
}

func NewMentionSpam(limit int, duration time.Duration) *MentionSpam {
	if limit <= 0 {
		limit = 6
	}
	if duration <= 0 {
		duration = 10 * time.Second
	}
	return &MentionSpam{
		limit:    limit,
		duration: duration,
		buckets:  make(map[string][]int),
		weight:   1.3,
	}
}

func (m *MentionSpam) Name() string    { return "mentionspam" }
func (m *MentionSpam) Weight() float64 { return m.weight }

func (m *MentionSpam) Evaluate(ctx context.Context, ev *antiraid.Event) *antiraid.DetectorResult {
	content := ""
	if ev.Meta != nil {
		content = ev.Meta["content"]
	}
	if content == "" {
		return &antiraid.DetectorResult{Triggered: false}
	}

	mentions := len(mentionRe.FindAllString(content, -1))
	if mentions == 0 {
		return &antiraid.DetectorResult{Triggered: false}
	}

	if mentions >= 5 {
		score := 85.0 + float64(mentions-5)*3
		if score > 100 {
			score = 100
		}
		return &antiraid.DetectorResult{
			Triggered: true,
			Score:     score,
			Reason:    fmt.Sprintf("mensaje con %d menciones", mentions),
			DeleteMsg: true,
			Evidence: map[string]any{
				"mentions": mentions,
				"content":  content,
			},
		}
	}

	now := time.Now()
	cutoff := now.Add(-m.duration)
	key := ev.UserID

	m.mu.Lock()
	bucket := m.buckets[key]
	idx := 0
	for idx < len(bucket) && time.Unix(int64(bucket[idx]), 0).Before(cutoff) {
		idx++
	}
	if idx > 0 {
		bucket = bucket[idx:]
	}
	bucket = append(bucket, int(now.Unix()))
	m.buckets[key] = bucket

	total := 0
	for _, ts := range bucket {
		if time.Unix(int64(ts), 0).After(cutoff) {
			total++
		}
	}
	m.mu.Unlock()

	if total < m.limit {
		return &antiraid.DetectorResult{Triggered: false}
	}

	ratio := float64(total) / float64(m.limit)
	score := 60.0 + (ratio-1.0)*20
	if score > 100 {
		score = 100
	}

	return &antiraid.DetectorResult{
		Triggered: true,
		Score:     score,
		Reason:    fmt.Sprintf("%d mensajes con menciones en %s", total, m.duration),
		DeleteMsg: true,
		Evidence: map[string]any{
			"messages_with_mentions": total,
			"limit":                  m.limit,
			"duration":               m.duration.String(),
		},
	}
}
