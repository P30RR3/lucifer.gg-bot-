package storage

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
)

type CommunityFeatureSettings struct {
	HelpChannelID       string
	ConfessionChannelID string
}

type ShopItem struct {
	Key         string
	Name        string
	Description string
	Price       int64
}

type TrialCase struct {
	GuildID   string
	ThreadID  string
	PlannedBy string
	Defendant string
	Reason    string
	Deadline  time.Time
}

type InfernalCase struct {
	GuildID   string
	TargetID  string
	ThreadID  string
	ActorID   string
	Reason    string
	Challenge string
	Answer    string
	ExpiresAt time.Time
	Active    bool
	Note      string
}

var defaultShopItems = []ShopItem{
	{Key: "pentacle-coin", Name: "Moneda de pentáculo", Description: "Una moneda oscura para tu colección.", Price: 500},
	{Key: "maze-daggers", Name: "Daga ceremonial del Lux", Description: "Réplica decorativa para el inventario del club.", Price: 1200},
	{Key: "lux-vip-pass", Name: "Pase VIP del Lux", Description: "Un recuerdo exclusivo del club.", Price: 1800},
	{Key: "hellfire-cocktail", Name: "Cóctel Hellfire", Description: "Una bebida legendaria, solo de colección.", Price: 250},
	{Key: "silver-record", Name: "Disco de plata", Description: "Un disco raro para coleccionistas.", Price: 750},
	{Key: "maze-rose", Name: "Rosa nocturna", Description: "Una rosa negra encantada.", Price: 400},
	{Key: "lucky-dice", Name: "Dados de la suerte", Description: "Un par de dados de colección.", Price: 600},
	{Key: "lucifer-crown", Name: "Corona de Lucifer", Description: "La pieza más codiciada del catálogo.", Price: 5000},
}

func (s *Store) GetCommunityFeatureSettings(ctx context.Context, guildID string) (CommunityFeatureSettings, error) {
	var settings CommunityFeatureSettings
	err := s.pool.QueryRow(ctx, `
		SELECT help_channel_id, confession_channel_id
		FROM community_feature_settings
		WHERE guild_id = $1
	`, guildID).Scan(&settings.HelpChannelID, &settings.ConfessionChannelID)
	if errors.Is(err, pgx.ErrNoRows) {
		return CommunityFeatureSettings{}, nil
	}
	if err != nil {
		return CommunityFeatureSettings{}, fmt.Errorf("storage: consultar ajustes de comunidad: %w", err)
	}
	return settings, nil
}

func (s *Store) HelpThreadRequester(ctx context.Context, threadID string) (string, error) {
	var requesterID string
	err := s.pool.QueryRow(ctx, `
		SELECT requester_id FROM help_threads
		WHERE thread_id = $1 AND status = 'open'
	`, threadID).Scan(&requesterID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("storage: hilo de ayuda no encontrado o cerrado")
	}
	if err != nil {
		return "", fmt.Errorf("storage: consultar solicitante del hilo: %w", err)
	}
	return requesterID, nil
}

func (s *Store) SetCommunityChannel(ctx context.Context, guildID, setting, channelID string) error {
	var column string
	switch setting {
	case "help":
		column = "help_channel_id"
	case "confession":
		column = "confession_channel_id"
	default:
		return fmt.Errorf("storage: tipo de canal de comunidad inválido: %q", setting)
	}
	query := `INSERT INTO community_feature_settings (guild_id, ` + column + `)
		VALUES ($1, $2)
		ON CONFLICT (guild_id) DO UPDATE SET ` + column + ` = EXCLUDED.` + column + `, updated_at = NOW()`
	if _, err := s.pool.Exec(ctx, query, guildID, channelID); err != nil {
		return fmt.Errorf("storage: guardar canal de comunidad: %w", err)
	}
	return nil
}

func (s *Store) CreateHelpThread(ctx context.Context, guildID, threadID, requesterID string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO help_threads (guild_id, thread_id, requester_id)
		VALUES ($1, $2, $3)
	`, guildID, threadID, requesterID)
	if err != nil {
		return fmt.Errorf("storage: registrar hilo de ayuda: %w", err)
	}
	return nil
}

func (s *Store) CreateTrialCase(ctx context.Context, record TrialCase) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO trial_cases (guild_id, thread_id, plaintiff_id, defendant_id, reason, deadline)
		VALUES ($1, $2, $3, $4, $5, $6)
	`, record.GuildID, record.ThreadID, record.PlannedBy, record.Defendant, record.Reason, record.Deadline)
	if err != nil {
		return fmt.Errorf("storage: crear expediente de juicio: %w", err)
	}
	return nil
}

func (s *Store) LoadTrialCase(ctx context.Context, threadID string) (TrialCase, error) {
	record := TrialCase{}
	err := s.pool.QueryRow(ctx, `
		SELECT guild_id, thread_id, plaintiff_id, defendant_id, reason, deadline
		FROM trial_cases
		WHERE thread_id = $1 AND status = 'open'
	`, threadID).Scan(
		&record.GuildID, &record.ThreadID, &record.PlannedBy,
		&record.Defendant, &record.Reason, &record.Deadline,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return TrialCase{}, errors.New("storage: juicio no encontrado o ya resuelto")
	}
	if err != nil {
		return TrialCase{}, fmt.Errorf("storage: consultar expediente de juicio: %w", err)
	}
	return record, nil
}

func (s *Store) FinishTrialCase(ctx context.Context, threadID, verdict string) (TrialCase, error) {
	record := TrialCase{}
	err := s.pool.QueryRow(ctx, `
		UPDATE trial_cases
		SET status = 'resolved', verdict = $2
		WHERE thread_id = $1 AND status = 'open'
		RETURNING guild_id, thread_id, plaintiff_id, defendant_id, reason, deadline
	`, threadID, verdict).Scan(
		&record.GuildID, &record.ThreadID, &record.PlannedBy,
		&record.Defendant, &record.Reason, &record.Deadline,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return TrialCase{}, errors.New("storage: juicio no encontrado o ya resuelto")
	}
	if err != nil {
		return TrialCase{}, fmt.Errorf("storage: resolver expediente de juicio: %w", err)
	}
	return record, nil
}

func (s *Store) RecordHelpStaffReply(ctx context.Context, threadID, staffID string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO help_thread_staff_replies (thread_id, staff_id, reply_count, last_reply_at)
		SELECT thread_id, $2, 1, NOW()
		FROM help_threads
		WHERE thread_id = $1 AND status = 'open'
		ON CONFLICT (thread_id, staff_id) DO UPDATE
		SET reply_count = help_thread_staff_replies.reply_count + 1,
		    last_reply_at = NOW()
	`, threadID, staffID)
	if err != nil {
		return fmt.Errorf("storage: registrar respuesta del staff en ayuda: %w", err)
	}
	return nil
}

func (s *Store) CloseHelpThread(ctx context.Context, threadID, actorID, status string) (string, error) {
	if status != "resolved" && status != "more_help" {
		return "", errors.New("storage: resultado de ayuda inválido")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("storage: iniciar cierre de ayuda: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var guildID string
	err = tx.QueryRow(ctx, `
		UPDATE help_threads
		SET status = $3, closed_at = NOW(), closed_by = $2
		WHERE thread_id = $1 AND status = 'open'
		RETURNING guild_id
	`, threadID, actorID, status).Scan(&guildID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("storage: hilo de ayuda inexistente o ya cerrado")
	}
	if err != nil {
		return "", fmt.Errorf("storage: cerrar hilo de ayuda: %w", err)
	}

	var awarded string
	if status == "resolved" {
		err = tx.QueryRow(ctx, `
			SELECT staff_id
			FROM help_thread_staff_replies
			WHERE thread_id = $1
			ORDER BY reply_count DESC, last_reply_at DESC
			LIMIT 1
		`, threadID).Scan(&awarded)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return "", fmt.Errorf("storage: elegir staff destacado: %w", err)
		}
		if awarded != "" {
			if _, err := tx.Exec(ctx, `
				INSERT INTO staff_actions (guild_id, actor_id, target_id, action, details)
				VALUES ($1, $2, '', 'help_resolved', $3)
			`, guildID, awarded, threadID); err != nil {
				return "", fmt.Errorf("storage: reconocer resolución de ayuda: %w", err)
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO economy_accounts (guild_id, user_id, balance, experience)
				VALUES ($1, $2, $3, 10)
				ON CONFLICT (guild_id, user_id) DO UPDATE
				SET balance = economy_accounts.balance + 1,
				    experience = economy_accounts.experience + 10,
				    updated_at = NOW()
			`, guildID, awarded, initialEconomyBalance); err != nil {
				return "", fmt.Errorf("storage: sumar mérito al staff de ayuda: %w", err)
			}
		}
	}
	if _, err := tx.Exec(ctx, `UPDATE help_threads SET awarded_staff_id = $2 WHERE thread_id = $1`, threadID, awarded); err != nil {
		return "", fmt.Errorf("storage: guardar mérito del hilo: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("storage: confirmar cierre de ayuda: %w", err)
	}
	return awarded, nil
}

func (s *Store) CreateDeepConfession(ctx context.Context, guildID, authorID, content string) (int64, error) {
	if _, err := s.pool.Exec(ctx, `DELETE FROM deep_confessions WHERE expires_at <= NOW()`); err != nil {
		return 0, fmt.Errorf("storage: limpiar confesiones vencidas: %w", err)
	}
	var id int64
	err := s.pool.QueryRow(ctx, `
		INSERT INTO deep_confessions (guild_id, author_id, content)
		VALUES ($1, $2, $3)
		RETURNING id
	`, guildID, authorID, content).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("storage: guardar confesión profunda: %w", err)
	}
	return id, nil
}

func (s *Store) RecordDeepConfessionOffer(ctx context.Context, guildID, offererID string, confessionID, price int64, evaluation string) error {
	if price < 1 {
		return errors.New("storage: oferta no válida")
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("storage: iniciar oferta por confesión: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var exists bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM deep_confessions
			WHERE id = $1 AND guild_id = $2 AND expires_at > NOW()
		)
	`, confessionID, guildID).Scan(&exists); err != nil {
		return fmt.Errorf("storage: verificar confesión ofertada: %w", err)
	}
	if !exists {
		return errors.New("storage: confesión no encontrada o expirada")
	}
	var alreadyOffered bool
	if err := tx.QueryRow(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM deep_confession_offers
			WHERE confession_id = $1 AND offerer_id = $2
		)
	`, confessionID, offererID).Scan(&alreadyOffered); err != nil {
		return fmt.Errorf("storage: comprobar oferta previa: %w", err)
	}
	if alreadyOffered {
		return errors.New("storage: ya presentaste una oferta por esta confesión")
	}
	if err := ensureEconomyAccount(ctx, tx, guildID, offererID, initialEconomyBalance); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		UPDATE economy_accounts
		SET balance = balance - $3, updated_at = NOW()
		WHERE guild_id = $1 AND user_id = $2 AND balance >= $3
	`, guildID, offererID, price)
	if err != nil {
		return fmt.Errorf("storage: cobrar oferta por confesión: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInsufficientChips
	}
	_, err = tx.Exec(ctx, `
		INSERT INTO deep_confession_offers (confession_id, offerer_id, price, evaluation)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (confession_id, offerer_id) DO UPDATE
		SET price = EXCLUDED.price, evaluation = EXCLUDED.evaluation, created_at = NOW()
	`, confessionID, offererID, price, evaluation)
	if err != nil {
		return fmt.Errorf("storage: guardar oferta por confesión: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("storage: confirmar oferta por confesión: %w", err)
	}
	return nil
}

func (s *Store) AddInvestigationEvidence(ctx context.Context, guildID, targetID, actorID, evidence string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO investigation_evidence (guild_id, target_id, actor_id, evidence)
		VALUES ($1, $2, $3, $4)
	`, guildID, targetID, actorID, evidence)
	if err != nil {
		return fmt.Errorf("storage: guardar evidencia: %w", err)
	}
	return nil
}

func (s *Store) ListInvestigationEvidence(ctx context.Context, guildID, targetID string, limit int) ([]string, error) {
	if limit < 1 || limit > 20 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
		SELECT evidence
		FROM investigation_evidence
		WHERE guild_id = $1 AND target_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, guildID, targetID, limit)
	if err != nil {
		return nil, fmt.Errorf("storage: listar evidencias: %w", err)
	}
	defer rows.Close()
	evidence := make([]string, 0)
	for rows.Next() {
		var item string
		if err := rows.Scan(&item); err != nil {
			return nil, fmt.Errorf("storage: leer evidencia: %w", err)
		}
		evidence = append(evidence, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: recorrer evidencias: %w", err)
	}
	return evidence, nil
}

func (s *Store) SaveInvestigationResponse(ctx context.Context, guildID, targetID, response string) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO investigation_responses (guild_id, target_id, response)
		VALUES ($1, $2, $3)
	`, guildID, targetID, response)
	if err != nil {
		return fmt.Errorf("storage: guardar respuesta de interrogatorio: %w", err)
	}
	return nil
}

func (s *Store) ListInvestigationResponses(ctx context.Context, guildID, targetID string, limit int) ([]string, error) {
	if limit < 1 || limit > 20 {
		limit = 10
	}
	rows, err := s.pool.Query(ctx, `
		SELECT response
		FROM investigation_responses
		WHERE guild_id = $1 AND target_id = $2
		ORDER BY created_at DESC
		LIMIT $3
	`, guildID, targetID, limit)
	if err != nil {
		return nil, fmt.Errorf("storage: listar respuestas de interrogatorio: %w", err)
	}
	defer rows.Close()
	responses := make([]string, 0)
	for rows.Next() {
		var response string
		if err := rows.Scan(&response); err != nil {
			return nil, fmt.Errorf("storage: leer respuesta de interrogatorio: %w", err)
		}
		responses = append(responses, response)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: recorrer respuestas de interrogatorio: %w", err)
	}
	return responses, nil
}

func (s *Store) StartInfernalCase(ctx context.Context, record InfernalCase) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO infernal_cases (guild_id, target_id, thread_id, actor_id, reason, challenge, challenge_answer, expires_at, active, note, created_at, resolved_at, resolved_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, TRUE, '', NOW(), NULL, '')
		ON CONFLICT (guild_id, target_id) DO UPDATE SET
			thread_id = EXCLUDED.thread_id,
			actor_id = EXCLUDED.actor_id,
			reason = EXCLUDED.reason,
			challenge = EXCLUDED.challenge,
			challenge_answer = EXCLUDED.challenge_answer,
			expires_at = EXCLUDED.expires_at,
			active = TRUE,
			note = '',
			created_at = NOW(),
			resolved_at = NULL,
			resolved_by = ''
	`, record.GuildID, record.TargetID, record.ThreadID, record.ActorID, record.Reason, record.Challenge, record.Answer, record.ExpiresAt)
	if err != nil {
		return fmt.Errorf("storage: iniciar expediente infernal: %w", err)
	}
	return nil
}

func (s *Store) GetInfernalCase(ctx context.Context, guildID, targetID string) (*InfernalCase, error) {
	record := &InfernalCase{}
	err := s.pool.QueryRow(ctx, `
		SELECT guild_id, target_id, thread_id, actor_id, reason, challenge, challenge_answer, expires_at, active, note
		FROM infernal_cases
		WHERE guild_id = $1 AND target_id = $2
	`, guildID, targetID).Scan(
		&record.GuildID, &record.TargetID, &record.ThreadID, &record.ActorID, &record.Reason,
		&record.Challenge, &record.Answer,
		&record.ExpiresAt, &record.Active, &record.Note,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, errors.New("storage: expediente infernal no encontrado")
	}
	if err != nil {
		return nil, fmt.Errorf("storage: consultar expediente infernal: %w", err)
	}
	return record, nil
}

func (s *Store) ListActiveInfernalCases(ctx context.Context, guildID string, limit int) ([]InfernalCase, error) {
	if limit < 1 || limit > 25 {
		return nil, errors.New("storage: límite de expedientes infernales fuera de rango")
	}
	rows, err := s.pool.Query(ctx, `
		SELECT guild_id, target_id, thread_id, actor_id, reason, challenge,
		       challenge_answer, expires_at, active, note
		FROM infernal_cases
		WHERE guild_id = $1 AND active = TRUE AND expires_at > NOW()
		ORDER BY expires_at
		LIMIT $2
	`, guildID, limit)
	if err != nil {
		return nil, fmt.Errorf("storage: listar expedientes infernales activos: %w", err)
	}
	defer rows.Close()
	cases := make([]InfernalCase, 0)
	for rows.Next() {
		var record InfernalCase
		if err := rows.Scan(
			&record.GuildID, &record.TargetID, &record.ThreadID, &record.ActorID,
			&record.Reason, &record.Challenge, &record.Answer, &record.ExpiresAt,
			&record.Active, &record.Note,
		); err != nil {
			return nil, fmt.Errorf("storage: leer expediente infernal: %w", err)
		}
		cases = append(cases, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: recorrer expedientes infernales: %w", err)
	}
	return cases, nil
}

func (s *Store) CompleteInfernalTask(ctx context.Context, guildID, targetID, answer string) (string, error) {
	var threadID string
	err := s.pool.QueryRow(ctx, `
		UPDATE infernal_cases
		SET active = FALSE, resolved_at = NOW(), resolved_by = 'redemption-task'
		WHERE guild_id = $1 AND target_id = $2 AND active = TRUE
		  AND expires_at > NOW()
		  AND LOWER(TRIM(challenge_answer)) = LOWER(TRIM($3))
		RETURNING thread_id
	`, guildID, targetID, answer).Scan(&threadID)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", errors.New("storage: respuesta incorrecta o no hay castigo activo")
	}
	if err != nil {
		return "", fmt.Errorf("storage: completar tarea infernal: %w", err)
	}
	return threadID, nil
}

func (s *Store) UpdateInfernalNote(ctx context.Context, guildID, targetID, note string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE infernal_cases SET note = $3
		WHERE guild_id = $1 AND target_id = $2 AND active = TRUE
	`, guildID, targetID, note)
	if err != nil {
		return fmt.Errorf("storage: actualizar nota infernal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("storage: no hay castigo activo para ese usuario")
	}
	return nil
}

func (s *Store) ResolveInfernalCase(ctx context.Context, guildID, targetID, moderatorID string) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE infernal_cases
		SET active = FALSE, resolved_at = NOW(), resolved_by = $3
		WHERE guild_id = $1 AND target_id = $2 AND active = TRUE
	`, guildID, targetID, moderatorID)
	if err != nil {
		return fmt.Errorf("storage: absolver expediente infernal: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("storage: no hay castigo activo para ese usuario")
	}
	return nil
}

func (s *Store) EnsureShopCatalog(ctx context.Context, guildID string) error {
	for _, item := range defaultShopItems {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO lux_shop_items (guild_id, item_key, name, description, price)
			VALUES ($1, $2, $3, $4, $5)
			ON CONFLICT (guild_id, item_key) DO NOTHING
		`, guildID, item.Key, item.Name, item.Description, item.Price); err != nil {
			return fmt.Errorf("storage: cargar catálogo del Lux: %w", err)
		}
	}
	defaultRanks := []struct {
		level int64
		name  string
	}{{1, "Alma nueva"}, {5, "Habitual"}, {10, "VIP"}, {20, "Leyenda del Lux"}}
	for _, rank := range defaultRanks {
		if _, err := s.pool.Exec(ctx, `
			INSERT INTO lux_ranks (guild_id, min_level, name)
			VALUES ($1, $2, $3)
			ON CONFLICT (guild_id, min_level) DO NOTHING
		`, guildID, rank.level, rank.name); err != nil {
			return fmt.Errorf("storage: cargar rangos del Lux: %w", err)
		}
	}
	return nil
}

func (s *Store) ListShopItems(ctx context.Context, guildID string) ([]ShopItem, error) {
	if err := s.EnsureShopCatalog(ctx, guildID); err != nil {
		return nil, err
	}
	rows, err := s.pool.Query(ctx, `
		SELECT item_key, name, description, price
		FROM lux_shop_items
		WHERE guild_id = $1 AND active = TRUE
		ORDER BY price, name
		LIMIT 25
	`, guildID)
	if err != nil {
		return nil, fmt.Errorf("storage: listar tienda del Lux: %w", err)
	}
	defer rows.Close()
	items := make([]ShopItem, 0)
	for rows.Next() {
		var item ShopItem
		if err := rows.Scan(&item.Key, &item.Name, &item.Description, &item.Price); err != nil {
			return nil, fmt.Errorf("storage: leer artículo de tienda: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: recorrer tienda del Lux: %w", err)
	}
	return items, nil
}

func (s *Store) BuyShopItem(ctx context.Context, guildID, userID, itemKey string) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("storage: iniciar compra Lux: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var item ShopItem
	err = tx.QueryRow(ctx, `
		SELECT item_key, name, description, price
		FROM lux_shop_items
		WHERE guild_id = $1 AND item_key = $2 AND active = TRUE
		FOR UPDATE
	`, guildID, itemKey).Scan(&item.Key, &item.Name, &item.Description, &item.Price)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, errors.New("storage: artículo no encontrado")
	}
	if err != nil {
		return 0, fmt.Errorf("storage: consultar artículo: %w", err)
	}
	if err := ensureEconomyAccount(ctx, tx, guildID, userID, initialEconomyBalance); err != nil {
		return 0, err
	}
	var balance int64
	err = tx.QueryRow(ctx, `
		UPDATE economy_accounts
		SET balance = balance - $3, updated_at = NOW()
		WHERE guild_id = $1 AND user_id = $2 AND balance >= $3
		RETURNING balance
	`, guildID, userID, item.Price).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrInsufficientChips
	}
	if err != nil {
		return 0, fmt.Errorf("storage: cobrar compra Lux: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO lux_inventory (guild_id, user_id, item_key, item_name, quantity)
		VALUES ($1, $2, $3, $4, 1)
		ON CONFLICT (guild_id, user_id, item_key) DO UPDATE
		SET quantity = lux_inventory.quantity + 1
	`, guildID, userID, item.Key, item.Name); err != nil {
		return 0, fmt.Errorf("storage: añadir artículo al inventario: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("storage: confirmar compra Lux: %w", err)
	}
	return balance, nil
}

func (s *Store) SetShopItemPrice(ctx context.Context, guildID, name string, price int64) error {
	if price < 0 {
		return errors.New("storage: el precio no puede ser negativo")
	}
	if err := s.EnsureShopCatalog(ctx, guildID); err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `
		UPDATE lux_shop_items SET price = $3
		WHERE guild_id = $1 AND LOWER(name) = LOWER($2) AND active = TRUE
	`, guildID, strings.TrimSpace(name), price)
	if err != nil {
		return fmt.Errorf("storage: modificar precio de tienda: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("storage: artículo de tienda no encontrado")
	}
	return nil
}

func (s *Store) AddShopItem(ctx context.Context, guildID, name, description string, price int64) error {
	if strings.TrimSpace(name) == "" || price < 0 {
		return errors.New("storage: nombre o precio de artículo inválido")
	}
	key := shopKey(name)
	_, err := s.pool.Exec(ctx, `
		INSERT INTO lux_shop_items (guild_id, item_key, name, description, price)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (guild_id, item_key) DO UPDATE
		SET name = EXCLUDED.name, description = EXCLUDED.description,
		    price = EXCLUDED.price, active = TRUE
	`, guildID, key, strings.TrimSpace(name), strings.TrimSpace(description), price)
	if err != nil {
		return fmt.Errorf("storage: añadir artículo a tienda: %w", err)
	}
	return nil
}

func (s *Store) SetLuxRank(ctx context.Context, guildID string, level int64, name string) error {
	if level < 1 || strings.TrimSpace(name) == "" {
		return errors.New("storage: nivel o nombre de rango inválido")
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO lux_ranks (guild_id, min_level, name)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, min_level) DO UPDATE
		SET name = EXCLUDED.name, updated_at = NOW()
	`, guildID, level, strings.TrimSpace(name))
	if err != nil {
		return fmt.Errorf("storage: personalizar rango Lux: %w", err)
	}
	return nil
}

func (s *Store) LuxRank(ctx context.Context, guildID string, level int64) (string, error) {
	if err := s.EnsureShopCatalog(ctx, guildID); err != nil {
		return "", err
	}
	var name string
	err := s.pool.QueryRow(ctx, `
		SELECT name FROM lux_ranks
		WHERE guild_id = $1 AND min_level <= $2
		ORDER BY min_level DESC
		LIMIT 1
	`, guildID, level).Scan(&name)
	if err != nil {
		return "", fmt.Errorf("storage: consultar rango Lux: %w", err)
	}
	return name, nil
}

func shopKey(name string) string {
	var builder strings.Builder
	lastDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(name)) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			builder.WriteRune(r)
			lastDash = false
		} else if !lastDash && builder.Len() > 0 {
			builder.WriteByte('-')
			lastDash = true
		}
	}
	key := strings.Trim(builder.String(), "-")
	if len(key) > 90 {
		key = key[:90]
	}
	return key
}
