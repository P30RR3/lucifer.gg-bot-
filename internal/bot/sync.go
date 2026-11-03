package bot

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/cenkalti/backoff/v4"
)

type Syncer struct {
	session *discordgo.Session
	appID   string
	log     *slog.Logger
	timeout time.Duration
}

type SyncResult struct {
	Global    bool
	GuildID   string
	Count     int
	Duration  time.Duration
	Retries   int
	Timestamp time.Time
}

var (
	ErrNoCommands  = errors.New("sync: sin comandos para sincronizar")
	ErrSyncTimeout = errors.New("sync: timeout sincronizando comandos")
	ErrSyncFailed  = errors.New("sync: command synchronization failed")
)

func NewSyncer(s *discordgo.Session, appID string, log *slog.Logger, timeout time.Duration) *Syncer {
	if log == nil {
		log = slog.Default()
	}
	if timeout <= 0 {
		timeout = 120 * time.Second
	}
	return &Syncer{
		session: s,
		appID:   appID,
		log:     log.With("component", "syncer"),
		timeout: timeout,
	}
}

func (s *Syncer) SyncAll(ctx context.Context, reg *Registry, devGuildID string) ([]SyncResult, error) {
	cmds := reg.ToApplicationCommands()
	if len(cmds) == 0 {
		return nil, ErrNoCommands
	}

	var results []SyncResult

	if devGuildID != "" {
		res, err := s.SyncGuild(ctx, cmds, devGuildID)
		if err != nil {
			return nil, err
		}
		results = append(results, res)
		return results, nil
	}

	res, err := s.SyncGlobal(ctx, cmds)
	if err != nil {
		return nil, err
	}
	results = append(results, res)
	return results, nil
}

func (s *Syncer) SyncGlobal(ctx context.Context, cmds []*discordgo.ApplicationCommand) (SyncResult, error) {
	return s.sync(ctx, cmds, "", true)
}

func (s *Syncer) SyncGuild(ctx context.Context, cmds []*discordgo.ApplicationCommand, guildID string) (SyncResult, error) {
	if guildID == "" {
		return SyncResult{}, fmt.Errorf("sync: guild_id is empty")
	}
	return s.sync(ctx, cmds, guildID, false)
}

func (s *Syncer) sync(ctx context.Context, cmds []*discordgo.ApplicationCommand, guildID string, global bool) (SyncResult, error) {
	start := time.Now()
	scope := "global"
	if !global {
		scope = "guild:" + guildID
	}

	s.log.Info("sincronizando slash commands", "scope", scope, "count", len(cmds))

	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	var result SyncResult
	result.Global = global
	result.GuildID = guildID
	result.Count = len(cmds)

	retries := 0
	op := func() error {
		_, err := s.session.ApplicationCommandBulkOverwrite(s.appID, guildID, cmds)
		if err != nil {
			retries++
			s.log.Warn("command synchronization failed", "guild_id", guildID, "err", err)
			return err
		}
		return nil
	}

	bo := backoff.NewExponentialBackOff()
	bo.MaxElapsedTime = s.timeout
	bo.InitialInterval = 2 * time.Second
	bo.MaxInterval = 30 * time.Second
	bo.Multiplier = 1.8
	bo.RandomizationFactor = 0.3

	if err := backoff.Retry(op, backoff.WithContext(bo, ctx)); err != nil {
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
			return SyncResult{}, ErrSyncTimeout
		}
		return SyncResult{}, fmt.Errorf("%w: %v", ErrSyncFailed, err)
	}

	result.Retries = retries
	result.Duration = time.Since(start)
	result.Timestamp = time.Now()

	s.log.Info("slash commands sincronizados",
		"scope", scope,
		"count", len(cmds),
		"duration", result.Duration.String(),
		"retries", retries,
	)
	return result, nil
}

func (s *Syncer) DeleteAllGuild(ctx context.Context, guildID string) error {
	if guildID == "" {
		return fmt.Errorf("sync: guild_id is empty")
	}
	_, err := s.session.ApplicationCommandBulkOverwrite(s.appID, guildID, []*discordgo.ApplicationCommand{})
	if err != nil {
		return fmt.Errorf("sync: delete guild commands: %w", err)
	}
	return nil
}

func (s *Syncer) DeleteAllGlobal(ctx context.Context) error {
	_, err := s.session.ApplicationCommandBulkOverwrite(s.appID, "", []*discordgo.ApplicationCommand{})
	if err != nil {
		return fmt.Errorf("sync: delete global commands: %w", err)
	}
	return nil
}

func (s *Syncer) ListRegistered(ctx context.Context, guildID string) ([]*discordgo.ApplicationCommand, error) {
	if guildID == "" {
		return s.session.ApplicationCommands(s.appID, "")
	}
	return s.session.ApplicationCommands(s.appID, guildID)
}

func (s *Syncer) Diff(ctx context.Context, reg *Registry, guildID string) (added, removed, changed int) {
	existing, err := s.ListRegistered(ctx, guildID)
	if err != nil {
		return 0, 0, 0
	}
	desired := reg.ToApplicationCommands()

	existingMap := make(map[string]*discordgo.ApplicationCommand)
	for _, e := range existing {
		existingMap[e.Name] = e
	}
	desiredMap := make(map[string]*discordgo.ApplicationCommand)
	for _, d := range desired {
		desiredMap[d.Name] = d
	}

	for name := range desiredMap {
		if _, ok := existingMap[name]; !ok {
			added++
		}
	}
	for name := range existingMap {
		if _, ok := desiredMap[name]; !ok {
			removed++
		}
	}
	for name, d := range desiredMap {
		if e, ok := existingMap[name]; ok {
			if e.Description != d.Description {
				changed++
			}
		}
	}
	return added, removed, changed
}
