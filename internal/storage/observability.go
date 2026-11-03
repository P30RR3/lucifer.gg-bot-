package storage

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type AuditSettings struct {
	GuildID               string
	LogChannelID          string
	SupportChannelID      string
	IncludeDeletedContent bool
}

type DeletedMessage struct {
	MessageID string
	AuthorID  string
	ChannelID string
	Content   string
	DeletedAt time.Time
}

type StaffRankingEntry struct {
	ActorID  string
	Bans     int
	Warnings int
	Kicks    int
	Timeouts int
	Cleans   int
	Total    int
}

func (s *Store) GetAuditSettings(ctx context.Context, guildID string) (AuditSettings, error) {
	var settings AuditSettings
	err := s.pool.QueryRow(ctx, `
		SELECT guild_id, log_channel_id, support_channel_id, include_deleted_content
		FROM audit_settings
		WHERE guild_id = $1
	`, guildID).Scan(
		&settings.GuildID,
		&settings.LogChannelID,
		&settings.SupportChannelID,
		&settings.IncludeDeletedContent,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return AuditSettings{GuildID: guildID}, nil
	}
	return settings, err
}

func (s *Store) SaveAuditSettings(ctx context.Context, settings AuditSettings) error {
	settings.GuildID = strings.TrimSpace(settings.GuildID)
	settings.LogChannelID = strings.TrimSpace(settings.LogChannelID)
	settings.SupportChannelID = strings.TrimSpace(settings.SupportChannelID)
	if settings.GuildID == "" {
		return errors.New("storage: guild ID vacío en configuración de auditoría")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO audit_settings
			(guild_id, log_channel_id, support_channel_id, include_deleted_content, updated_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (guild_id) DO UPDATE SET
			log_channel_id = EXCLUDED.log_channel_id,
			support_channel_id = EXCLUDED.support_channel_id,
			include_deleted_content = EXCLUDED.include_deleted_content,
			updated_at = NOW()
	`, settings.GuildID, settings.LogChannelID, settings.SupportChannelID, settings.IncludeDeletedContent)
	return err
}

func (s *Store) RecordDeletedMessage(ctx context.Context, guildID string, message DeletedMessage) error {
	return s.RecordDeletedMessages(ctx, guildID, []DeletedMessage{message})
}

func (s *Store) RecordDeletedMessages(ctx context.Context, guildID string, messages []DeletedMessage) error {
	if len(messages) == 0 {
		return nil
	}
	batch := &pgx.Batch{}
	for _, message := range messages {
		batch.Queue(`
		INSERT INTO deleted_messages (guild_id, message_id, author_id, channel_id, content, deleted_at)
		VALUES ($1, $2, $3, $4, $5, COALESCE($6, NOW()))
		ON CONFLICT (guild_id, message_id) DO NOTHING
	`, guildID, message.MessageID, message.AuthorID, message.ChannelID, message.Content, nullableTime(message.DeletedAt))
	}
	results := s.pool.SendBatch(ctx, batch)
	for range messages {
		if _, err := results.Exec(); err != nil {
			_ = results.Close()
			return err
		}
	}
	if err := results.Close(); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, `
		DELETE FROM deleted_messages
		WHERE guild_id = $1 AND deleted_at < NOW() - INTERVAL '7 days'
	`, guildID)
	return err
}

func (s *Store) RecentDeletedMessages(ctx context.Context, guildID string, since time.Time, limit int) ([]DeletedMessage, error) {
	if limit < 1 || limit > 5 {
		limit = 5
	}
	rows, err := s.pool.Query(ctx, `
		SELECT message_id, author_id, channel_id, content, deleted_at
		FROM deleted_messages
		WHERE guild_id = $1 AND deleted_at >= $2
		ORDER BY deleted_at DESC
		LIMIT $3
	`, guildID, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	messages := make([]DeletedMessage, 0, limit)
	for rows.Next() {
		var message DeletedMessage
		if err := rows.Scan(&message.MessageID, &message.AuthorID, &message.ChannelID, &message.Content, &message.DeletedAt); err != nil {
			return nil, err
		}
		messages = append(messages, message)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return messages, nil
}

func (s *Store) RecordStaffAction(ctx context.Context, guildID, actorID, targetID, action, details string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO staff_actions (guild_id, actor_id, target_id, action, details)
		VALUES ($1, $2, $3, $4, $5)
	`, guildID, actorID, targetID, action, details)
	if err != nil {
		return err
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM staff_actions WHERE created_at < NOW() - INTERVAL '365 days'`)
	return err
}

func (s *Store) StaffRanking(ctx context.Context, guildID string, since time.Time, limit int) ([]StaffRankingEntry, error) {
	if limit < 1 || limit > 10 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
		SELECT actor_id,
			COUNT(*) FILTER (WHERE action = 'ban')::INT,
			COUNT(*) FILTER (WHERE action = 'warn')::INT,
			COUNT(*) FILTER (WHERE action = 'kick')::INT,
			COUNT(*) FILTER (WHERE action = 'timeout')::INT,
			COUNT(*) FILTER (WHERE action = 'purge')::INT,
			COUNT(*)::INT AS total
		FROM staff_actions
		WHERE guild_id = $1 AND created_at >= $2
		GROUP BY actor_id
		ORDER BY total DESC, actor_id ASC
		LIMIT $3
	`, guildID, since, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := make([]StaffRankingEntry, 0, limit)
	for rows.Next() {
		var entry StaffRankingEntry
		if err := rows.Scan(&entry.ActorID, &entry.Bans, &entry.Warnings, &entry.Kicks, &entry.Timeouts, &entry.Cleans, &entry.Total); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}

func nullableTime(value time.Time) any {
	if value.IsZero() {
		return nil
	}
	return value
}
