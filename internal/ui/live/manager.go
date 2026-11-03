package live

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

var ErrManagerClosed = errors.New("live: manager cerrado")
var ErrMessageNotFound = errors.New("live: mensaje no encontrado")

type Manager struct {
	session    *discordgo.Session
	debounce   time.Duration
	fps        int
	maxUpdates int

	mu       sync.Mutex
	messages map[string]*Message
	updates  map[string]int
	closed   bool
}

func NewManager(s *discordgo.Session, fps, maxUpdates int) *Manager {
	if fps <= 0 {
		fps = 4
	}
	if fps > 10 {
		fps = 10
	}
	if maxUpdates <= 0 {
		maxUpdates = 60
	}
	return &Manager{
		session:    s,
		debounce:   time.Duration(1000/fps) * time.Millisecond,
		fps:        fps,
		maxUpdates: maxUpdates,
		messages:   make(map[string]*Message),
		updates:    make(map[string]int),
	}
}

func (m *Manager) Start(ctx context.Context, key, channelID string, initial *discordgo.MessageSend) (*Message, error) {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return nil, ErrManagerClosed
	}
	if existing, ok := m.messages[key]; ok {
		m.mu.Unlock()
		return existing, nil
	}
	m.mu.Unlock()

	msg, err := NewMessage(ctx, m.session, channelID, initial, m.debounce)
	if err != nil {
		return nil, err
	}

	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		msg.Close()
		return nil, ErrManagerClosed
	}
	m.messages[key] = msg
	m.updates[key] = 0
	m.mu.Unlock()
	return msg, nil
}

func (m *Manager) Update(ctx context.Context, key string, payload *discordgo.MessageSend) error {
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return ErrManagerClosed
	}
	msg, ok := m.messages[key]
	count := m.updates[key]
	m.mu.Unlock()

	if !ok {
		return ErrMessageNotFound
	}
	if count >= m.maxUpdates {
		return nil
	}

	if err := msg.Update(ctx, payload); err != nil {
		return err
	}

	m.mu.Lock()
	m.updates[key]++
	m.mu.Unlock()
	return nil
}

func (m *Manager) Finish(key string, payload *discordgo.MessageSend) error {
	m.mu.Lock()
	msg, ok := m.messages[key]
	delete(m.messages, key)
	delete(m.updates, key)
	m.mu.Unlock()

	if !ok {
		return ErrMessageNotFound
	}
	return msg.Finalize(payload)
}

func (m *Manager) Delete(key string) error {
	m.mu.Lock()
	msg, ok := m.messages[key]
	delete(m.messages, key)
	delete(m.updates, key)
	m.mu.Unlock()

	if !ok {
		return ErrMessageNotFound
	}
	return msg.Delete()
}

func (m *Manager) Get(key string) (*Message, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	msg, ok := m.messages[key]
	return msg, ok
}

func (m *Manager) UpdatesFor(key string) int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updates[key]
}

func (m *Manager) ActiveCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.messages)
}

func (m *Manager) Keys() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]string, 0, len(m.messages))
	for k := range m.messages {
		out = append(out, k)
	}
	return out
}

func (m *Manager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	m.closed = true
	for _, msg := range m.messages {
		msg.Close()
	}
	m.messages = make(map[string]*Message)
	m.updates = make(map[string]int)
}

func (m *Manager) FPS() int { return m.fps }

func (m *Manager) Debounce() time.Duration { return m.debounce }
