package storage

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

var (
	ErrLuxEventNotFound  = errors.New("storage: evento no encontrado")
	ErrLuxEventClosed    = errors.New("storage: evento cancelado")
	ErrLuxEventFull      = errors.New("storage: evento completo")
	ErrLuxEventJoined    = errors.New("storage: ya inscrito en el evento")
	ErrLuxEventNotJoined = errors.New("storage: no inscrito en el evento")
	ErrLuxEventNotOwner  = errors.New("storage: solo quien creó el evento puede cancelarlo")
)

type LuxEvent struct {
	ID          int64
	GuildID     string
	ChannelID   string
	CreatorID   string
	Title       string
	Description string
	StartsAt    time.Time
	Capacity    int
	Attendees   int
	Status      string
}

func (s *Store) CreateLuxEvent(ctx context.Context, event *LuxEvent) error {
	err := s.pool.QueryRow(ctx, `
		INSERT INTO lux_events (
			guild_id, channel_id, creator_id, title, description, starts_at, capacity
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, status
	`, event.GuildID, event.ChannelID, event.CreatorID, event.Title, event.Description,
		event.StartsAt, event.Capacity).Scan(&event.ID, &event.Status)
	if err != nil {
		return fmt.Errorf("storage: crear evento del Lux: %w", err)
	}
	return nil
}

func (s *Store) GetLuxEvent(ctx context.Context, guildID string, id int64) (*LuxEvent, error) {
	event := &LuxEvent{}
	err := s.pool.QueryRow(ctx, `
		SELECT e.id, e.guild_id, e.channel_id, e.creator_id, e.title, e.description,
		       e.starts_at, e.capacity, e.status, COUNT(a.user_id)::INT
		FROM lux_events e
		LEFT JOIN lux_event_attendees a ON a.event_id = e.id
		WHERE e.guild_id = $1 AND e.id = $2
		GROUP BY e.id
	`, guildID, id).Scan(
		&event.ID, &event.GuildID, &event.ChannelID, &event.CreatorID,
		&event.Title, &event.Description, &event.StartsAt, &event.Capacity,
		&event.Status, &event.Attendees,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrLuxEventNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("storage: consultar evento del Lux: %w", err)
	}
	return event, nil
}

func (s *Store) ListLuxEvents(ctx context.Context, guildID string, limit int) ([]LuxEvent, error) {
	if limit < 1 || limit > 20 {
		return nil, errors.New("storage: límite de eventos fuera de rango")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT e.id, e.guild_id, e.channel_id, e.creator_id, e.title, e.description,
		       e.starts_at, e.capacity, e.status, COUNT(a.user_id)::INT
		FROM lux_events e
		LEFT JOIN lux_event_attendees a ON a.event_id = e.id
		WHERE e.guild_id = $1 AND e.status = 'open' AND e.starts_at > NOW()
		GROUP BY e.id
		ORDER BY e.starts_at
		LIMIT $2
	`, guildID, limit)
	if err != nil {
		return nil, fmt.Errorf("storage: listar eventos del Lux: %w", err)
	}
	defer rows.Close()

	events := make([]LuxEvent, 0)
	for rows.Next() {
		var event LuxEvent
		if err := rows.Scan(
			&event.ID, &event.GuildID, &event.ChannelID, &event.CreatorID,
			&event.Title, &event.Description, &event.StartsAt, &event.Capacity,
			&event.Status, &event.Attendees,
		); err != nil {
			return nil, fmt.Errorf("storage: leer evento del Lux: %w", err)
		}
		events = append(events, event)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: recorrer eventos del Lux: %w", err)
	}
	return events, nil
}

func (s *Store) JoinLuxEvent(ctx context.Context, guildID string, id int64, userID string) (int, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("storage: iniciar inscripción al evento: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	var capacity int
	err = tx.QueryRow(ctx, `
		SELECT status, capacity FROM lux_events
		WHERE guild_id = $1 AND id = $2 AND starts_at > NOW()
		FOR UPDATE
	`, guildID, id).Scan(&status, &capacity)
	if errors.Is(err, pgx.ErrNoRows) {
		event, getErr := s.GetLuxEvent(ctx, guildID, id)
		if errors.Is(getErr, ErrLuxEventNotFound) {
			return 0, ErrLuxEventNotFound
		}
		if getErr != nil {
			return 0, getErr
		}
		if event.Status != "open" || !event.StartsAt.After(time.Now()) {
			return 0, ErrLuxEventClosed
		}
		return 0, ErrLuxEventNotFound
	}
	if err != nil {
		return 0, fmt.Errorf("storage: consultar evento para inscripción: %w", err)
	}
	if status != "open" {
		return 0, ErrLuxEventClosed
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO lux_event_attendees (event_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (event_id, user_id) DO NOTHING
	`, id, userID)
	if err != nil {
		return 0, fmt.Errorf("storage: registrar inscripción al evento: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return 0, ErrLuxEventJoined
	}

	var attendees int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::INT FROM lux_event_attendees WHERE event_id = $1
	`, id).Scan(&attendees); err != nil {
		return 0, fmt.Errorf("storage: contar asistentes: %w", err)
	}
	if capacity > 0 && attendees > capacity {
		return 0, ErrLuxEventFull
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("storage: confirmar inscripción al evento: %w", err)
	}
	return attendees, nil
}

func (s *Store) LeaveLuxEvent(ctx context.Context, guildID string, id int64, userID string) (int, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("storage: iniciar baja del evento: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	err = tx.QueryRow(ctx, `
		SELECT TRUE FROM lux_events WHERE guild_id = $1 AND id = $2 AND status = 'open'
		FOR UPDATE
	`, guildID, id).Scan(&exists)
	if errors.Is(err, pgx.ErrNoRows) {
		event, getErr := s.GetLuxEvent(ctx, guildID, id)
		if errors.Is(getErr, ErrLuxEventNotFound) {
			return 0, ErrLuxEventNotFound
		}
		if getErr != nil {
			return 0, getErr
		}
		if event.Status != "open" {
			return 0, ErrLuxEventClosed
		}
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("storage: consultar evento para baja: %w", err)
	}

	tag, err := tx.Exec(ctx, `
		DELETE FROM lux_event_attendees WHERE event_id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return 0, fmt.Errorf("storage: retirar inscripción: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return 0, ErrLuxEventNotJoined
	}

	var attendees int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::INT FROM lux_event_attendees WHERE event_id = $1
	`, id).Scan(&attendees); err != nil {
		return 0, fmt.Errorf("storage: contar asistentes: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("storage: confirmar baja del evento: %w", err)
	}
	return attendees, nil
}

func (s *Store) CancelLuxEvent(ctx context.Context, guildID string, id int64, actorID string, moderator bool) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE lux_events
		SET status = 'cancelled', updated_at = NOW()
		WHERE guild_id = $1 AND id = $2 AND status = 'open'
		  AND (creator_id = $3 OR $4)
	`, guildID, id, actorID, moderator)
	if err != nil {
		return fmt.Errorf("storage: cancelar evento del Lux: %w", err)
	}
	if tag.RowsAffected() > 0 {
		return nil
	}

	event, getErr := s.GetLuxEvent(ctx, guildID, id)
	if errors.Is(getErr, ErrLuxEventNotFound) {
		return ErrLuxEventNotFound
	}
	if getErr != nil {
		return getErr
	}
	if event.Status != "open" {
		return ErrLuxEventClosed
	}
	return ErrLuxEventNotOwner
}
