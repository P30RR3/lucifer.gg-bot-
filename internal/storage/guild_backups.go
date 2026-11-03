package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrGuildBackupNotFound = errors.New("storage: respaldo del servidor no encontrado")
	ErrGuildBackupExists   = errors.New("storage: ya existe un respaldo para este servidor")
)

type GuildBackup struct {
	GuildID   string
	CreatedBy string
	Snapshot  []byte
	CreatedAt time.Time
}

type GuildBackupChannelSettings struct {
	GuildID               string `json:"guild_id"`
	LogChannelID          string `json:"log_channel_id"`
	WelcomeChannel        string `json:"welcome_channel"`
	AuditLogChannelID     string `json:"audit_log_channel_id"`
	SupportChannelID      string `json:"support_channel_id"`
	IncludeDeletedContent bool   `json:"include_deleted_content"`
	HelpChannelID         string `json:"help_channel_id"`
	ConfessionChannelID   string `json:"confession_channel_id"`
}

func (s *Store) GetGuildBackupChannelSettings(ctx context.Context, guildID string) (GuildBackupChannelSettings, error) {
	guild, err := s.GetOrCreateGuild(ctx, guildID)
	if err != nil {
		return GuildBackupChannelSettings{}, fmt.Errorf("storage: get backup guild settings: %w", err)
	}
	audit, err := s.GetAuditSettings(ctx, guildID)
	if err != nil {
		return GuildBackupChannelSettings{}, fmt.Errorf("storage: get backup audit settings: %w", err)
	}
	community, err := s.GetCommunityFeatureSettings(ctx, guildID)
	if err != nil {
		return GuildBackupChannelSettings{}, fmt.Errorf("storage: get backup community settings: %w", err)
	}
	return GuildBackupChannelSettings{
		GuildID: guildID, LogChannelID: guild.LogChannelID, WelcomeChannel: guild.WelcomeChannel,
		AuditLogChannelID: audit.LogChannelID, SupportChannelID: audit.SupportChannelID,
		IncludeDeletedContent: audit.IncludeDeletedContent,
		HelpChannelID:         community.HelpChannelID, ConfessionChannelID: community.ConfessionChannelID,
	}, nil
}

func (s *Store) RestoreGuildBackupChannelSettings(ctx context.Context, settings GuildBackupChannelSettings) error {
	if settings.GuildID == "" {
		return errors.New("storage: guild ID vacío en configuración del respaldo")
	}
	err := s.WithTx(ctx, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx, `
			INSERT INTO guilds (guild_id, log_channel_id, welcome_channel)
			VALUES ($1, $2, $3)
			ON CONFLICT (guild_id) DO UPDATE SET
				log_channel_id = EXCLUDED.log_channel_id,
				welcome_channel = EXCLUDED.welcome_channel,
				updated_at = NOW()
		`, settings.GuildID, settings.LogChannelID, settings.WelcomeChannel); err != nil {
			return fmt.Errorf("storage: restore guild channel settings: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO audit_settings
				(guild_id, log_channel_id, support_channel_id, include_deleted_content, updated_at)
			VALUES ($1, $2, $3, $4, NOW())
			ON CONFLICT (guild_id) DO UPDATE SET
				log_channel_id = EXCLUDED.log_channel_id,
				support_channel_id = EXCLUDED.support_channel_id,
				include_deleted_content = EXCLUDED.include_deleted_content,
				updated_at = NOW()
		`, settings.GuildID, settings.AuditLogChannelID, settings.SupportChannelID, settings.IncludeDeletedContent); err != nil {
			return fmt.Errorf("storage: restore audit channel settings: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO community_feature_settings
				(guild_id, help_channel_id, confession_channel_id, updated_at)
			VALUES ($1, $2, $3, NOW())
			ON CONFLICT (guild_id) DO UPDATE SET
				help_channel_id = EXCLUDED.help_channel_id,
				confession_channel_id = EXCLUDED.confession_channel_id,
				updated_at = NOW()
		`, settings.GuildID, settings.HelpChannelID, settings.ConfessionChannelID); err != nil {
			return fmt.Errorf("storage: restore community channel settings: %w", err)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("storage: restore guild backup channel settings: %w", err)
	}
	return nil
}

func (s *Store) CreateGuildBackup(ctx context.Context, backup GuildBackup) error {
	tag, err := s.pool.Exec(ctx,
		`INSERT INTO guild_backups (guild_id, created_by, snapshot)
		 VALUES ($1, $2, $3::jsonb)
		 ON CONFLICT (guild_id) DO NOTHING`,
		backup.GuildID, backup.CreatedBy, backup.Snapshot,
	)
	if err != nil {
		return fmt.Errorf("storage: create guild backup: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrGuildBackupExists
	}
	return nil
}

func (s *Store) GetGuildBackup(ctx context.Context, guildID string) (GuildBackup, error) {
	var backup GuildBackup
	err := s.pool.QueryRow(ctx,
		`SELECT guild_id, created_by, snapshot, created_at
		 FROM guild_backups WHERE guild_id = $1`,
		guildID,
	).Scan(&backup.GuildID, &backup.CreatedBy, &backup.Snapshot, &backup.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return GuildBackup{}, ErrGuildBackupNotFound
	}
	if err != nil {
		return GuildBackup{}, fmt.Errorf("storage: get guild backup: %w", err)
	}
	return backup, nil
}

func (s *Store) DeleteGuildBackup(ctx context.Context, guildID string) error {
	tag, err := s.pool.Exec(ctx, `DELETE FROM guild_backups WHERE guild_id = $1`, guildID)
	if err != nil {
		return fmt.Errorf("storage: delete guild backup: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrGuildBackupNotFound
	}
	return nil
}
