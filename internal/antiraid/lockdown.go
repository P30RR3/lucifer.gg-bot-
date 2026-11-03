package antiraid

import (
	"context"
	"errors"
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

	mu         sync.Mutex
	active     map[string]time.Time
	affected   map[string]map[string]permissionSnapshot
	timers     map[string]*time.Timer
	activating map[string]bool
}

type permissionSnapshot struct {
	existed bool
	allow   int64
	deny    int64
}

const lockdownDeniedPermissions = discordgo.PermissionSendMessages |
	discordgo.PermissionAddReactions |
	discordgo.PermissionSendMessagesInThreads

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
		session:    s,
		cooldown:   cooldown,
		log:        log.With("component", "lockdown"),
		active:     make(map[string]time.Time),
		affected:   make(map[string]map[string]permissionSnapshot),
		timers:     make(map[string]*time.Timer),
		activating: make(map[string]bool),
	}
}

func (l *LockdownManager) IsActive(guildID string) (bool, time.Time) {
	l.mu.Lock()
	defer l.mu.Unlock()
	until, ok := l.active[guildID]
	if !ok {
		return false, time.Time{}
	}
	return time.Now().Before(until), until
}

func (l *LockdownManager) Info(guildID string) *LockdownInfo {
	l.mu.Lock()
	defer l.mu.Unlock()
	until, ok := l.active[guildID]
	channels := make([]string, 0, len(l.affected[guildID]))
	for channelID := range l.affected[guildID] {
		channels = append(channels, channelID)
	}
	return &LockdownInfo{
		GuildID:  guildID,
		Until:    until,
		Channels: channels,
		Active:   ok && time.Now().Before(until),
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
	if l.activating[guildID] {
		l.mu.Unlock()
		return errors.New("lockdown: ya hay una activación en curso")
	}
	hasPreviousSnapshot := len(l.affected[guildID]) > 0
	if timer, ok := l.timers[guildID]; ok {
		timer.Stop()
		delete(l.timers, guildID)
	}
	l.activating[guildID] = true
	l.mu.Unlock()

	if hasPreviousSnapshot {
		if err := l.Deactivate(ctx, guildID); err != nil {
			l.mu.Lock()
			delete(l.activating, guildID)
			l.mu.Unlock()
			return fmt.Errorf("lockdown: no se pudo restaurar el bloqueo anterior: %w", err)
		}
		l.mu.Lock()
		l.activating[guildID] = true
		l.mu.Unlock()
	}

	channels, err := l.session.GuildChannels(guildID)
	if err != nil {
		l.mu.Lock()
		delete(l.activating, guildID)
		l.mu.Unlock()
		return fmt.Errorf("lockdown: guild channels: %w", err)
	}

	everyone := guildID
	affected := make(map[string]permissionSnapshot, len(channels))
	for _, ch := range channels {
		if ch.Type != discordgo.ChannelTypeGuildText && ch.Type != discordgo.ChannelTypeGuildNews {
			continue
		}
		snapshot := permissionSnapshot{}
		for _, overwrite := range ch.PermissionOverwrites {
			if overwrite.ID == everyone && overwrite.Type == discordgo.PermissionOverwriteTypeRole {
				snapshot = permissionSnapshot{existed: true, allow: overwrite.Allow, deny: overwrite.Deny}
				break
			}
		}
		err := l.session.ChannelPermissionSet(
			ch.ID, everyone, discordgo.PermissionOverwriteTypeRole,
			snapshot.allow, snapshot.deny|lockdownDeniedPermissions,
		)
		if err != nil {
			rollbackErr := restorePermissions(l.session, everyone, affected)
			l.mu.Lock()
			delete(l.activating, guildID)
			if rollbackErr != nil {
				l.active[guildID] = time.Now().Add(d)
				l.affected[guildID] = affected
			}
			l.mu.Unlock()
			if rollbackErr != nil {
				return fmt.Errorf("lockdown: canal %s falló: %w; también falló la reversión: %v", ch.ID, err, rollbackErr)
			}
			return fmt.Errorf("lockdown: canal %s: %w", ch.ID, err)
		}
		affected[ch.ID] = snapshot
	}
	if len(affected) == 0 {
		l.mu.Lock()
		delete(l.activating, guildID)
		l.mu.Unlock()
		return errors.New("lockdown: no hay canales de texto que se puedan bloquear")
	}

	l.mu.Lock()
	delete(l.activating, guildID)
	l.active[guildID] = time.Now().Add(d)
	l.affected[guildID] = affected
	timer := time.AfterFunc(d, func() {
		if err := l.Deactivate(context.Background(), guildID); err != nil {
			l.log.Error("no se pudieron restaurar todos los permisos del bloqueo", "guild", guildID, "err", err)
		}
	})
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
	affected := make(map[string]permissionSnapshot, len(l.affected[guildID]))
	for channelID, snapshot := range l.affected[guildID] {
		affected[channelID] = snapshot
	}
	if t, ok := l.timers[guildID]; ok {
		t.Stop()
		delete(l.timers, guildID)
	}
	l.mu.Unlock()

	failed := make(map[string]permissionSnapshot)
	var errs []error
	for channelID, snapshot := range affected {
		if err := restorePermission(l.session, guildID, channelID, snapshot); err != nil {
			failed[channelID] = snapshot
			errs = append(errs, fmt.Errorf("canal %s: %w", channelID, err))
		}
	}
	l.mu.Lock()
	if len(failed) == 0 {
		delete(l.active, guildID)
		delete(l.affected, guildID)
	} else {
		l.affected[guildID] = failed
	}
	l.mu.Unlock()

	if len(errs) != 0 {
		return errors.Join(errs...)
	}
	l.log.Info("lockdown desactivado", "guild", guildID, "channels", len(affected))
	return nil
}

func (l *LockdownManager) LockChannel(channelID, guildID string) error {
	l.mu.Lock()
	if snapshot, tracked := l.affected[guildID][channelID]; tracked {
		l.mu.Unlock()
		return l.session.ChannelPermissionSet(
			channelID, guildID, discordgo.PermissionOverwriteTypeRole,
			snapshot.allow, snapshot.deny|lockdownDeniedPermissions,
		)
	}
	l.mu.Unlock()

	channel, err := l.session.Channel(channelID)
	if err != nil {
		return err
	}
	snapshot := permissionSnapshot{}
	for _, overwrite := range channel.PermissionOverwrites {
		if overwrite.ID == guildID && overwrite.Type == discordgo.PermissionOverwriteTypeRole {
			snapshot = permissionSnapshot{existed: true, allow: overwrite.Allow, deny: overwrite.Deny}
			break
		}
	}
	if err := l.session.ChannelPermissionSet(
		channelID, guildID, discordgo.PermissionOverwriteTypeRole,
		snapshot.allow, snapshot.deny|lockdownDeniedPermissions,
	); err != nil {
		return err
	}
	l.mu.Lock()
	if l.affected[guildID] == nil {
		l.affected[guildID] = make(map[string]permissionSnapshot)
	}
	l.affected[guildID][channelID] = snapshot
	l.mu.Unlock()
	return nil
}

func (l *LockdownManager) UnlockChannel(channelID, guildID string) error {
	l.mu.Lock()
	snapshot, tracked := l.affected[guildID][channelID]
	l.mu.Unlock()
	if tracked {
		if err := restorePermission(l.session, guildID, channelID, snapshot); err != nil {
			return err
		}
		l.mu.Lock()
		delete(l.affected[guildID], channelID)
		if len(l.affected[guildID]) == 0 {
			delete(l.affected, guildID)
		}
		l.mu.Unlock()
		return nil
	}
	return fmt.Errorf("lockdown: no existe una copia de permisos para restaurar el canal %s", channelID)
}

func restorePermissions(session *discordgo.Session, guildID string, affected map[string]permissionSnapshot) error {
	var errs []error
	for channelID, snapshot := range affected {
		if err := restorePermission(session, guildID, channelID, snapshot); err != nil {
			errs = append(errs, fmt.Errorf("canal %s: %w", channelID, err))
		}
	}
	return errors.Join(errs...)
}

func restorePermission(session *discordgo.Session, guildID, channelID string, snapshot permissionSnapshot) error {
	if !snapshot.existed {
		return session.ChannelPermissionDelete(channelID, guildID)
	}
	return session.ChannelPermissionSet(
		channelID, guildID, discordgo.PermissionOverwriteTypeRole,
		snapshot.allow, snapshot.deny,
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
	guildSet := make(map[string]struct{}, len(l.affected)+len(l.timers))
	for guildID := range l.affected {
		guildSet[guildID] = struct{}{}
	}
	for guildID, t := range l.timers {
		t.Stop()
		guildSet[guildID] = struct{}{}
	}
	l.timers = make(map[string]*time.Timer)
	l.mu.Unlock()
	for guildID := range guildSet {
		if err := l.Deactivate(context.Background(), guildID); err != nil {
			l.log.Error("no se pudo restaurar un bloqueo durante el apagado", "guild", guildID, "err", err)
		}
	}
}
