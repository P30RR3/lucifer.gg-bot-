package antiraid

import (
	"sync"
	"time"
)

type Window struct {
	mu       sync.Mutex
	duration time.Duration
	events   map[string][]Event
}

func NewWindow(duration time.Duration) *Window {
	if duration <= 0 {
		duration = 10 * time.Second
	}
	return &Window{
		duration: duration,
		events:   make(map[string][]Event),
	}
}

func (w *Window) Add(key string, event Event) {
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	w.prune(key, time.Now())
	w.events[key] = append(w.events[key], event)
}

func (w *Window) GarbageCollect() int {
	w.mu.Lock()
	defer w.mu.Unlock()

	removed := 0
	now := time.Now()
	for key := range w.events {
		before := len(w.events[key])
		w.prune(key, now)
		if _, ok := w.events[key]; !ok {
			removed++
		} else if len(w.events[key]) < before {
			removed += before - len(w.events[key])
		}
	}
	return removed
}

func (w *Window) Reset(key string) {
	w.mu.Lock()
	delete(w.events, key)
	w.mu.Unlock()
}

func (w *Window) KeyCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.events)
}

func (w *Window) prune(key string, now time.Time) {
	events := w.events[key]
	cutoff := now.Add(-w.duration)
	first := 0
	for first < len(events) && events[first].Timestamp.Before(cutoff) {
		first++
	}
	if first == len(events) {
		delete(w.events, key)
		return
	}
	if first > 0 {
		w.events[key] = append([]Event(nil), events[first:]...)
	}
}
