package live

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"krypton.dev/internal/ui"
)

var ErrMessageClosed = errors.New("live: mensaje cerrado")

type Message struct {
	session   *discordgo.Session
	channelID string
	messageID string

	mu       sync.Mutex
	content  *discordgo.MessageSend
	lastEdit time.Time
	debounce time.Duration
	closed   bool
	updates  int
	maxEdits int
}

func NewMessage(
	ctx context.Context,
	s *discordgo.Session,
	channelID string,
	initial *discordgo.MessageSend,
	debounce time.Duration,
) (*Message, error) {
	if debounce <= 0 {
		debounce = 220 * time.Millisecond
	}
	if initial == nil {
		return nil, errors.New("live: payload inicial nil")
	}
	ui.BrandMessageSend(initial)

	type sendResult struct {
		msg *discordgo.Message
		err error
	}
	ch := make(chan sendResult, 1)
	go func() {
		sent, err := s.ChannelMessageSendComplex(channelID, initial)
		ch <- sendResult{msg: sent, err: err}
	}()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case r := <-ch:
		if r.err != nil {
			return nil, r.err
		}
		return &Message{
			session:   s,
			channelID: channelID,
			messageID: r.msg.ID,
			content:   initial,
			lastEdit:  time.Now(),
			debounce:  debounce,
			maxEdits:  500,
		}, nil
	}
}

func (m *Message) Update(ctx context.Context, payload *discordgo.MessageSend) error {
	ui.BrandMessageSend(payload)
	m.mu.Lock()
	if m.closed {
		m.mu.Unlock()
		return ErrMessageClosed
	}
	if m.updates >= m.maxEdits {
		m.mu.Unlock()
		return nil
	}
	m.content = payload
	wait := m.debounce - time.Since(m.lastEdit)
	m.mu.Unlock()

	if wait > 0 {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(wait):
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	content := payload.Content
	edit := &discordgo.MessageEdit{
		ID:      m.messageID,
		Channel: m.channelID,
	}
	if payload.Content != "" || payload.Embeds != nil || payload.Components != nil {
		edit.Content = &content
		edit.Embeds = &payload.Embeds
		edit.Components = &payload.Components
	}

	if _, err := m.session.ChannelMessageEditComplex(edit); err != nil {
		return err
	}
	m.lastEdit = time.Now()
	m.updates++
	return nil
}

func (m *Message) Finalize(payload *discordgo.MessageSend) error {
	ui.BrandMessageSend(payload)
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()

	content := ""
	if payload != nil {
		content = payload.Content
	}

	edit := &discordgo.MessageEdit{
		ID:      m.messageID,
		Channel: m.channelID,
	}
	if payload != nil {
		edit.Content = &content
		edit.Embeds = &payload.Embeds
		edit.Components = &payload.Components
	}
	ui.BrandMessageEdit(edit)

	_, err := m.session.ChannelMessageEditComplex(edit)
	return err
}

func (m *Message) Delete() error {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
	return m.session.ChannelMessageDelete(m.channelID, m.messageID)
}

func (m *Message) AddReaction(emoji string) error {
	m.mu.Lock()
	id, ch := m.messageID, m.channelID
	m.mu.Unlock()
	return m.session.MessageReactionAdd(ch, id, emoji)
}

func (m *Message) ID() string { return m.messageID }

func (m *Message) Channel() string { return m.channelID }

func (m *Message) Updates() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.updates
}

func (m *Message) Closed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func (m *Message) Close() {
	m.mu.Lock()
	m.closed = true
	m.mu.Unlock()
}
