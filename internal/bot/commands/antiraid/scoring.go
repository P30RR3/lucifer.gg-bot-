package antiraid

import (
	"sync"
	"time"
)

type ScoreEntry struct {
	Value    float64
	Updated  time.Time
	Detector string
	Reasons  []string
}

type Score struct {
	mu       sync.RWMutex
	values   map[string]*ScoreEntry
	weights  map[string]float64
	halfLife time.Duration
}

func NewScore() *Score {
	return &Score{
		values:   make(map[string]*ScoreEntry),
		weights:  make(map[string]float64),
		halfLife: 10 * time.Minute,
	}
}

func (s *Score) SetHalfLife(d time.Duration) {
	s.mu.Lock()
	s.halfLife = d
	s.mu.Unlock()
}

func (s *Score) Add(key string, delta float64, detector, reason string) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()

	entry, ok := s.values[key]
	if !ok {
		entry = &ScoreEntry{Value: 0}
		s.values[key] = entry
	}

	w := s.weights[detector]
	if w <= 0 {
		w = 1.0
	}

	entry.Value += delta * w
	if entry.Value > 100 {
		entry.Value = 100
	}
	if entry.Value < 0 {
		entry.Value = 0
	}
	entry.Updated = time.Now()
	entry.Detector = detector
	if reason != "" {
		entry.Reasons = append(entry.Reasons, reason)
		if len(entry.Reasons) > 10 {
			entry.Reasons = entry.Reasons[len(entry.Reasons)-10:]
		}
	}
	return entry.Value
}

func (s *Score) Get(key string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if e, ok := s.values[key]; ok {
		return e.Value
	}
	return 0
}

func (s *Score) GetEntry(key string) *ScoreEntry {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if e, ok := s.values[key]; ok {
		clone := *e
		return &clone
	}
	return nil
}

func (s *Score) Decay() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	removed := 0
	for k, e := range s.values {
		elapsed := now.Sub(e.Updated)
		halves := float64(elapsed) / float64(s.halfLife)
		factor := pow(0.5, halves)
		e.Value *= factor
		if e.Value < 1 {
			delete(s.values, k)
			removed++
		}
	}
	return removed
}

func (s *Score) Reset(key string) {
	s.mu.Lock()
	delete(s.values, key)
	s.mu.Unlock()
}

func (s *Score) ResetGuild(prefix string) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	count := 0
	for k := range s.values {
		if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
			delete(s.values, k)
			count++
		}
	}
	return count
}

func (s *Score) SetWeight(detector string, w float64) {
	s.mu.Lock()
	s.weights[detector] = w
	s.mu.Unlock()
}

func (s *Score) Weight(detector string) float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if w, ok := s.weights[detector]; ok {
		return w
	}
	return 1.0
}

func (s *Score) Top(n int) []struct {
	Key   string
	Value float64
} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	type kv struct {
		Key   string
		Value float64
	}
	var list []kv
	for k, e := range s.values {
		list = append(list, kv{k, e.Value})
	}

	for i := 0; i < len(list); i++ {
		for j := i + 1; j < len(list); j++ {
			if list[j].Value > list[i].Value {
				list[i], list[j] = list[j], list[i]
			}
		}
	}

	if n > 0 && len(list) > n {
		list = list[:n]
	}

	out := make([]struct {
		Key   string
		Value float64
	}, len(list))
	for i, item := range list {
		out[i].Key = item.Key
		out[i].Value = item.Value
	}
	return out
}

func pow(base, exp float64) float64 {
	if exp == 0 {
		return 1
	}
	if exp < 0 {
		return 1 / pow(base, -exp)
	}
	result := 1.0
	for i := 0; i < int(exp); i++ {
		result *= base
	}
	frac := exp - float64(int(exp))
	if frac > 0 {
		result *= 1 - (1-base)*frac
	}
	return result
}
