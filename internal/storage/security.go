package storage

import (
	"context"
	"errors"
	"time"
)

type StaffActionRecord struct {
	ActorID   string
	Action    string
	Details   string
	CreatedAt time.Time
}

type SanctionAppeal struct {
	ID        int64
	GuildID   string
	TargetID  string
	Sanction  string
	Reason    string
	Status    string
	CreatedAt time.Time
}

func (s *Store) RecentStaffActionsForUser(ctx context.Context, guildID, userID string, limit int) ([]StaffActionRecord, error) {
	if limit < 1 || limit > 25 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
		SELECT actor_id, action, details, created_at
		FROM staff_actions
		WHERE guild_id = $1 AND target_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, guildID, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	actions := make([]StaffActionRecord, 0, limit)
	for rows.Next() {
		var action StaffActionRecord
		if err := rows.Scan(&action.ActorID, &action.Action, &action.Details, &action.CreatedAt); err != nil {
			return nil, err
		}
		actions = append(actions, action)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return actions, nil
}

func (s *Store) CreateSanctionAppeal(ctx context.Context, guildID, targetID, sanction, reason string) (int64, error) {
	if guildID == "" || targetID == "" || sanction == "" || reason == "" {
		return 0, errors.New("storage: apelación incompleta")
	}
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO sanction_appeals (guild_id, target_id, sanction, reason)
		VALUES ($1, $2, $3, $4)
		RETURNING id
	`, guildID, targetID, sanction, reason).Scan(&id)
	return id, err
}

func (s *Store) ListSanctionAppeals(ctx context.Context, guildID, targetID string, limit int) ([]SanctionAppeal, error) {
	if limit < 1 || limit > 25 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, guild_id, target_id, sanction, reason, status, created_at
		FROM sanction_appeals
		WHERE guild_id = $1 AND ($2 = '' OR target_id = $2)
		ORDER BY CASE status WHEN 'pending' THEN 0 ELSE 1 END, created_at DESC
		LIMIT $3
	`, guildID, targetID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	appeals := make([]SanctionAppeal, 0, limit)
	for rows.Next() {
		var appeal SanctionAppeal
		if err := rows.Scan(
			&appeal.ID, &appeal.GuildID, &appeal.TargetID, &appeal.Sanction,
			&appeal.Reason, &appeal.Status, &appeal.CreatedAt,
		); err != nil {
			return nil, err
		}
		appeals = append(appeals, appeal)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return appeals, nil
}

func (s *Store) ResolveSanctionAppeal(ctx context.Context, guildID string, appealID int64, reviewerID, status string) error {
	if status != "accepted" && status != "denied" {
		return errors.New("storage: resolución de apelación no válida")
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE sanction_appeals
		SET status = $3, reviewed_by = $4, reviewed_at = NOW()
		WHERE guild_id = $1 AND id = $2 AND status = 'pending'
	`, guildID, appealID, status, reviewerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("storage: apelación inexistente o ya resuelta")
	}
	return nil
}
