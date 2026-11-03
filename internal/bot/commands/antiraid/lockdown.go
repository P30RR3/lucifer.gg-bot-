package antiraid

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
)

type LockdownManager struct {
	session  *discordgo.Session
	cooldown time.Duration
	log      *slog.Logger

	mu       sync.Mutex
	active   map[string]time.Time
	affected map[string][]string
	timers   map[string]*time.Timer
}

type LockdownInfo struct {
	GuildID     string
	Until       time.Time
	Channels    []string
	Active      bool
	TriggeredBy string
}

func NewLockdownManager(s *discordgo.Session, cooldown time.Duration, log *slog.Logger) *LockdownManager {
	if cooldown <= 0 {
		cooldown = 5 * time.Minute
	}
	if log == nil {
		log = slog.Default()
	}
	return &LockdownManager{
		session:  s,
		cooldown: cooldown,
		log:      log.With("component", "lockdown"),
		active:   make(map[string]time.Time),
		affected: make(map[string][]string),
		timers:   make(map[string]*time.Timer),
	}
}

func (l *LockdownManager) IsActive(guildID string) (bool, time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	until, ok := l.active[guildID]
	if !ok {
		return false, time.Time{}
	}
	if time.Now().After(until) {
		delete(l.active, guildID)
		return false, time.Time{}
	}
	return true, until
}

func (l *LockdownManager) Info(guildID string) *LockdownInfo {
	l.mu.Lock()
	defer l.mu.Unlock()
	until, ok := l.active[guildID]
	return &LockdownInfo{
		GuildID:  guildID,
		Until:    until,
		Channels: append([]string{}, l.affected[guildID]...),
		Active:   ok,
	}
}

func (l *LockdownManager) Activate(ctx context.Context, guildID string, d time.Duration) error {
	if d <= 0 {
		d = l.cooldown
	}

	l.mu.Lock()
	if until, ok := l.active[guildID]; ok && time.Now().Before(until) {
		l.mu.Unlock()
		return fmt.Errorf("lockdown: ya activo hasta %s", until.Format(time.RFC3339))
	}
	l.active[guildID] = time.Now().Add(d)
	l.mu.Unlock()

	channels, err := l.session.GuildChannels(guildID)
	if err != nil {
		return fmt.Errorf("lockdown: guild channels: %w", err)
	}

	everyone := guildID
	affected := make([]string, 0, len(channels))
	for _, ch := range channels {
		if ch.Type != discordgo.ChannelTypeGuildText {
			continue
		}
		err := l.session.ChannelPermissionSet(
			ch.ID, everyone, discordgo.PermissionOverwriteTypeRole,
			0, discordgo.PermissionSendMessages|discordgo.PermissionAddReactions,
		)
		if err == nil {
			affected = append(affected, ch.ID)
		}
	}

	l.mu.Lock()
	l.affected[guildID] = affected
	l.mu.Unlock()

	timer := time.AfterFunc(d, func() {
		_ = l.Deactivate(context.Background(), guildID)
	})
	l.mu.Lock()
	if existing, ok := l.timers[guildID]; ok {
		existing.Stop()
	}
	l.timers[guildID] = timer
	l.mu.Unlock()

	l.log.Warn("lockdown activado",
		"guild", guildID,
		"duration", d.String(),
		"channels", len(affected),
	)
	return nil
}

func (l *LockdownManager) Deactivate(ctx context.Context, guildID string) error {
	l.mu.Lock()
	affected := l.affected[guildID]
	delete(l.active, guildID)
	delete(l.affected, guildID)
	if t, ok := l.timers[guildID]; ok {
		t.Stop()
		delete(l.timers, guildID)
	}
	l.mu.Unlock()

	everyone := guildID
	for _, chID := range affected {
		_ = l.session.ChannelPermissionSet(
			chID, everyone, discordgo.PermissionOverwriteTypeRole,
			0, 0,
		)
	}

	l.log.Info("lockdown desactivado", "guild", guildID, "channels", len(affected))
	return nil
}

func (l *LockdownManager) LockChannel(channelID, guildID string) error {
	return l.session.ChannelPermissionSet(
		channelID, guildID, discordgo.PermissionOverwriteTypeRole,
		0, discordgo.PermissionSendMessages|discordgo.PermissionAddReactions,
	)
}

func (l *LockdownManager) UnlockChannel(channelID, guildID string) error {
	return l.session.ChannelPermissionSet(
		channelID, guildID, discordgo.PermissionOverwriteTypeRole,
		0, 0,
	)
}

func (l *LockdownManager) ActiveGuilds() []string {
	l.mu.Lock()
	defer l.mu.Unlock()
	out := make([]string, 0, len(l.active))
	for g := range l.active {
		out = append(out, g)
	}
	return out
}

func (l *LockdownManager) Shutdown() {
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, t := range l.timers {
		t.Stop()
	}
	l.timers = make(map[string]*time.Timer)
}
