package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

var (
	ErrCommunityIdeaNotFound = errors.New("storage: propuesta no encontrada")
	ErrCommunityIdeaClosed   = errors.New("storage: propuesta cerrada")
	ErrCommunityIdeaCooldown = errors.New("storage: espera antes de publicar otra propuesta")
)

type CommunityIdea struct {
	ID          int64
	GuildID     string
	AuthorID    string
	Anonymous   bool
	Title       string
	Description string
	Status      string
	Votes       int
}

func (s *Store) CreateCommunityIdea(ctx context.Context, guildID, authorID, title, description string, anonymous bool) (*CommunityIdea, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return nil, fmt.Errorf("storage: iniciar propuesta comunitaria: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtext($1), hashtext($2))`, guildID, authorID); err != nil {
		return nil, fmt.Errorf("storage: bloquear publicación de propuesta: %w", err)
	}
	var recent bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM community_ideas
			WHERE guild_id = $1 AND author_id = $2
			  AND created_at > NOW() - INTERVAL '10 minutes'
		)
	`, guildID, authorID).Scan(&recent); err != nil {
		return nil, fmt.Errorf("storage: comprobar frecuencia de propuestas: %w", err)
	}
	if recent {
		return nil, ErrCommunityIdeaCooldown
	}

	idea := &CommunityIdea{}
	err = tx.QueryRow(ctx, `
		INSERT INTO community_ideas (guild_id, author_id, anonymous, title, description)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, guild_id, author_id, anonymous, title, description, status
	`, guildID, authorID, anonymous, title, description).Scan(
		&idea.ID, &idea.GuildID, &idea.AuthorID, &idea.Anonymous,
		&idea.Title, &idea.Description, &idea.Status,
	)
	if err != nil {
		return nil, fmt.Errorf("storage: crear propuesta comunitaria: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("storage: confirmar propuesta comunitaria: %w", err)
	}
	return idea, nil
}

func (s *Store) GetCommunityIdea(ctx context.Context, guildID string, id int64) (*CommunityIdea, error) {
	idea := &CommunityIdea{}
	err := s.pool.QueryRow(ctx, `
		SELECT i.id, i.guild_id, i.author_id, i.anonymous, i.title, i.description, i.status,
		       COUNT(v.user_id)::INT
		FROM community_ideas i
		LEFT JOIN community_idea_votes v ON v.idea_id = i.id
		WHERE i.guild_id = $1 AND i.id = $2
		GROUP BY i.id
	`, guildID, id).Scan(
		&idea.ID, &idea.GuildID, &idea.AuthorID, &idea.Anonymous,
		&idea.Title, &idea.Description, &idea.Status, &idea.Votes,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrCommunityIdeaNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("storage: consultar propuesta comunitaria: %w", err)
	}
	return idea, nil
}

func (s *Store) ListCommunityIdeas(ctx context.Context, guildID string, limit int) ([]CommunityIdea, error) {
	if limit < 1 || limit > 20 {
		return nil, errors.New("storage: límite de propuestas fuera de rango")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT i.id, i.guild_id, i.author_id, i.anonymous, i.title, i.description, i.status,
		       COUNT(v.user_id)::INT
		FROM community_ideas i
		LEFT JOIN community_idea_votes v ON v.idea_id = i.id
		WHERE i.guild_id = $1
		GROUP BY i.id
		ORDER BY (i.status = 'open') DESC, COUNT(v.user_id) DESC, i.created_at DESC
		LIMIT $2
	`, guildID, limit)
	if err != nil {
		return nil, fmt.Errorf("storage: listar propuestas comunitarias: %w", err)
	}
	defer rows.Close()

	ideas := make([]CommunityIdea, 0)
	for rows.Next() {
		var idea CommunityIdea
		if err := rows.Scan(
			&idea.ID, &idea.GuildID, &idea.AuthorID, &idea.Anonymous,
			&idea.Title, &idea.Description, &idea.Status, &idea.Votes,
		); err != nil {
			return nil, fmt.Errorf("storage: leer propuesta comunitaria: %w", err)
		}
		ideas = append(ideas, idea)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: recorrer propuestas comunitarias: %w", err)
	}
	return ideas, nil
}

func (s *Store) ToggleCommunityIdeaVote(ctx context.Context, guildID string, id int64, userID string) (bool, int, error) {
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return false, 0, fmt.Errorf("storage: iniciar voto comunitario: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var status string
	err = tx.QueryRow(ctx, `
		SELECT status FROM community_ideas
		WHERE guild_id = $1 AND id = $2
		FOR UPDATE
	`, guildID, id).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, 0, ErrCommunityIdeaNotFound
	}
	if err != nil {
		return false, 0, fmt.Errorf("storage: consultar estado de propuesta: %w", err)
	}
	if status != "open" {
		return false, 0, ErrCommunityIdeaClosed
	}

	tag, err := tx.Exec(ctx, `
		INSERT INTO community_idea_votes (idea_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (idea_id, user_id) DO NOTHING
	`, id, userID)
	if err != nil {
		return false, 0, fmt.Errorf("storage: registrar voto comunitario: %w", err)
	}
	voted := tag.RowsAffected() == 1
	if !voted {
		if _, err := tx.Exec(ctx, `
			DELETE FROM community_idea_votes WHERE idea_id = $1 AND user_id = $2
		`, id, userID); err != nil {
			return false, 0, fmt.Errorf("storage: retirar voto comunitario: %w", err)
		}
	}

	var votes int
	if err := tx.QueryRow(ctx, `
		SELECT COUNT(*)::INT FROM community_idea_votes WHERE idea_id = $1
	`, id).Scan(&votes); err != nil {
		return false, 0, fmt.Errorf("storage: contar votos comunitarios: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return false, 0, fmt.Errorf("storage: confirmar voto comunitario: %w", err)
	}
	return voted, votes, nil
}

func (s *Store) SetCommunityIdeaStatus(ctx context.Context, guildID string, id int64, status string) error {
	if status != "accepted" && status != "declined" {
		return errors.New("storage: estado de propuesta inválido")
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE community_ideas
		SET status = $3, updated_at = NOW()
		WHERE guild_id = $1 AND id = $2 AND status = 'open'
	`, guildID, id, status)
	if err != nil {
		return fmt.Errorf("storage: cerrar propuesta comunitaria: %w", err)
	}
	if tag.RowsAffected() == 0 {
		idea, getErr := s.GetCommunityIdea(ctx, guildID, id)
		if errors.Is(getErr, ErrCommunityIdeaNotFound) {
			return ErrCommunityIdeaNotFound
		}
		if getErr != nil {
			return getErr
		}
		if idea.Status != "open" {
			return ErrCommunityIdeaClosed
		}
	}
	return nil
}
