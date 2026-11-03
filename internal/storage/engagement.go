package storage

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"
)

var (
	ErrMissionNotReady             = errors.New("storage: misión incompleta")
	ErrMissionAlreadyClaimed       = errors.New("storage: misión ya reclamada")
	ErrInsufficientCelestialPoints = errors.New("storage: puntos celestiales insuficientes")
	ErrRewardAlreadyOwned          = errors.New("storage: recompensa ya canjeada")
)

type ActivityPreferences struct {
	TrackingEnabled bool
	StatsPublic     bool
	ResponseMode    string
}

type ActivityWord struct {
	Word        string
	Occurrences int64
}

type MemberActivityStats struct {
	Messages          int64
	VoiceSeconds      int64
	ActiveHourUTC     int
	FavoriteChannelID string
	TopWords          []ActivityWord
	Preferences       ActivityPreferences
}

type EngagementLeader struct {
	UserID string
	Score  int64
}

type CelestialMission struct {
	ID          string
	Name        string
	Description string
	Current     int64
	Target      int64
	Points      int64
	Claimed     bool
	PeriodStart time.Time
}

type CelestialReward struct {
	Key         string
	RoleID      string
	Name        string
	Description string
	Price       int64
}

type celestialMissionDefinition struct {
	id          string
	name        string
	description string
	target      int64
	points      int64
	period      string
}

var celestialMissions = []celestialMissionDefinition{
	{id: "daily-messages", name: "Presencia en el club", description: "Envía 50 mensajes hoy.", target: 50, points: 15, period: "day"},
	{id: "daily-voice", name: "Recorre las salas", description: "Visita 3 canales de voz hoy.", target: 3, points: 25, period: "day"},
	{id: "weekly-messages", name: "Una semana memorable", description: "Envía 250 mensajes esta semana.", target: 250, points: 100, period: "week"},
}

var activityWordPattern = regexp.MustCompile(`[\p{L}\p{N}]{2,32}`)
var activityExcludedPattern = regexp.MustCompile(`(?i)(https?://|www\.)[^\s]+|<[@#][!&]?\d+>|<a?:[a-z0-9_]+:\d+>|[\p{L}\p{N}._%+-]+@[\p{L}\p{N}.-]+\.[\p{L}]{2,}`)

var activityStopWords = map[string]struct{}{
	"ahi": {}, "algo": {}, "alguien": {}, "ante": {}, "aqui": {}, "como": {}, "con": {}, "cual": {},
	"cuando": {}, "donde": {}, "desde": {}, "ella": {}, "ellos": {}, "entre": {}, "esta": {}, "este": {},
	"esto": {}, "haber": {}, "hace": {}, "hasta": {}, "hay": {}, "la": {}, "las": {}, "le": {}, "les": {},
	"los": {}, "mas": {}, "me": {}, "mi": {}, "mis": {}, "muy": {}, "no": {}, "nos": {}, "nuestro": {},
	"para": {}, "pero": {}, "por": {}, "porque": {}, "que": {}, "quien": {}, "se": {}, "ser": {}, "si": {},
	"sin": {}, "sobre": {}, "son": {}, "su": {}, "sus": {}, "tambien": {}, "te": {}, "tiene": {}, "todo": {},
	"tu": {}, "tus": {}, "un": {}, "una": {}, "uno": {}, "unos": {}, "ya": {}, "and": {}, "are": {}, "but": {},
	"for": {}, "from": {}, "have": {}, "not": {}, "the": {}, "this": {}, "that": {}, "was": {}, "with": {},
	"you": {}, "everyone": {}, "here": {},
}

func (s *Store) GetActivityPreferences(ctx context.Context, guildID, userID string) (ActivityPreferences, error) {
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO member_activity_preferences (guild_id, user_id)
		VALUES ($1, $2) ON CONFLICT (guild_id, user_id) DO NOTHING
	`, guildID, userID); err != nil {
		return ActivityPreferences{}, fmt.Errorf("storage: preparar preferencias de actividad: %w", err)
	}
	var preferences ActivityPreferences
	if err := s.pool.QueryRow(ctx, `
		SELECT tracking_enabled, stats_public, response_mode
		FROM member_activity_preferences WHERE guild_id = $1 AND user_id = $2
	`, guildID, userID).Scan(&preferences.TrackingEnabled, &preferences.StatsPublic, &preferences.ResponseMode); err != nil {
		return ActivityPreferences{}, fmt.Errorf("storage: consultar preferencias de actividad: %w", err)
	}
	return preferences, nil
}

func (s *Store) ToggleActivityPreference(ctx context.Context, guildID, userID, preference string) error {
	if preference == "tracking" {
		tx, err := s.pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("storage: iniciar cambio de seguimiento: %w", err)
		}
		defer func() { _ = tx.Rollback(ctx) }()
		if _, err := tx.Exec(ctx, `
			INSERT INTO member_activity_preferences (guild_id, user_id)
			VALUES ($1, $2) ON CONFLICT (guild_id, user_id) DO NOTHING
		`, guildID, userID); err != nil {
			return fmt.Errorf("storage: preparar seguimiento de actividad: %w", err)
		}
		var wasEnabled bool
		if err := tx.QueryRow(ctx, `
			SELECT tracking_enabled FROM member_activity_preferences
			WHERE guild_id = $1 AND user_id = $2 FOR UPDATE
		`, guildID, userID).Scan(&wasEnabled); err != nil {
			return fmt.Errorf("storage: consultar seguimiento de actividad: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			UPDATE member_activity_preferences
			SET tracking_enabled = NOT tracking_enabled, updated_at = NOW()
			WHERE guild_id = $1 AND user_id = $2
		`, guildID, userID); err != nil {
			return fmt.Errorf("storage: cambiar seguimiento de actividad: %w", err)
		}
		if wasEnabled {
			var joinedAt time.Time
			var channelID string
			err := tx.QueryRow(ctx, `
				SELECT channel_id, joined_at FROM member_voice_sessions
				WHERE guild_id = $1 AND user_id = $2 FOR UPDATE
			`, guildID, userID).Scan(&channelID, &joinedAt)
			if err != nil && !errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("storage: consultar sesión de voz al desactivar: %w", err)
			}
			if err == nil {
				seconds := int64(time.Since(joinedAt).Seconds())
				if seconds < 0 {
					seconds = 0
				}
				if _, err := tx.Exec(ctx, `
					INSERT INTO member_voice_daily (guild_id, user_id, activity_date, channel_id, seconds)
					VALUES ($1, $2, (NOW() AT TIME ZONE 'UTC')::DATE, $3, $4)
					ON CONFLICT (guild_id, user_id, activity_date, channel_id)
					DO UPDATE SET seconds = member_voice_daily.seconds + EXCLUDED.seconds
				`, guildID, userID, channelID, seconds); err != nil {
					return fmt.Errorf("storage: cerrar actividad de voz al desactivar: %w", err)
				}
			}
			if _, err := tx.Exec(ctx, `
				DELETE FROM member_voice_sessions WHERE guild_id = $1 AND user_id = $2
			`, guildID, userID); err != nil {
				return fmt.Errorf("storage: cerrar sesión de voz al desactivar: %w", err)
			}
		}
		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("storage: confirmar cambio de seguimiento: %w", err)
		}
		return nil
	}

	var query string
	switch preference {
	case "stats_public":
		query = `UPDATE member_activity_preferences SET stats_public = NOT stats_public, updated_at = NOW() WHERE guild_id = $1 AND user_id = $2`
	case "response_mode":
		query = `UPDATE member_activity_preferences SET response_mode = CASE response_mode WHEN 'private' THEN 'public' WHEN 'public' THEN 'dm' ELSE 'private' END, updated_at = NOW() WHERE guild_id = $1 AND user_id = $2`
	default:
		return fmt.Errorf("storage: preferencia de actividad desconocida: %q", preference)
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO member_activity_preferences (guild_id, user_id)
		VALUES ($1, $2) ON CONFLICT (guild_id, user_id) DO NOTHING
	`, guildID, userID); err != nil {
		return fmt.Errorf("storage: preparar preferencia de actividad: %w", err)
	}
	if _, err := s.pool.Exec(ctx, query, guildID, userID); err != nil {
		return fmt.Errorf("storage: cambiar preferencia de actividad: %w", err)
	}
	return nil
}

func (s *Store) RecordMemberMessage(ctx context.Context, guildID, userID, channelID, content string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("storage: iniciar registro de actividad: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO member_activity_preferences (guild_id, user_id)
		VALUES ($1, $2) ON CONFLICT (guild_id, user_id) DO NOTHING
	`, guildID, userID); err != nil {
		return fmt.Errorf("storage: preparar registro de actividad: %w", err)
	}
	var enabled bool
	if err := tx.QueryRow(ctx, `
		SELECT tracking_enabled FROM member_activity_preferences
		WHERE guild_id = $1 AND user_id = $2
	`, guildID, userID).Scan(&enabled); err != nil {
		return fmt.Errorf("storage: comprobar seguimiento de actividad: %w", err)
	}
	if !enabled {
		return tx.Commit(ctx)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO member_activity_hourly (guild_id, user_id, activity_date, hour_utc, messages)
		VALUES ($1, $2, (NOW() AT TIME ZONE 'UTC')::DATE, EXTRACT(HOUR FROM NOW() AT TIME ZONE 'UTC')::SMALLINT, 1)
		ON CONFLICT (guild_id, user_id, activity_date, hour_utc)
		DO UPDATE SET messages = member_activity_hourly.messages + 1
	`, guildID, userID); err != nil {
		return fmt.Errorf("storage: contar mensaje por hora: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO member_activity_channels (guild_id, user_id, channel_id, messages)
		VALUES ($1, $2, $3, 1)
		ON CONFLICT (guild_id, user_id, channel_id)
		DO UPDATE SET messages = member_activity_channels.messages + 1, last_message_at = NOW()
	`, guildID, userID, channelID); err != nil {
		return fmt.Errorf("storage: contar mensaje por canal: %w", err)
	}
	for word, count := range extractActivityWords(content) {
		if _, err := tx.Exec(ctx, `
			INSERT INTO member_activity_words (guild_id, user_id, word, occurrences)
			SELECT $1, $2, $3, $4
			WHERE EXISTS (
			    SELECT 1 FROM member_activity_words
			    WHERE guild_id = $1 AND user_id = $2 AND word = $3
			) OR (
			    SELECT COUNT(*) FROM member_activity_words
			    WHERE guild_id = $1 AND user_id = $2
			) < 100
			ON CONFLICT (guild_id, user_id, word)
			DO UPDATE SET occurrences = member_activity_words.occurrences + EXCLUDED.occurrences
		`, guildID, userID, word, count); err != nil {
			return fmt.Errorf("storage: contar palabra frecuente: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("storage: confirmar registro de actividad: %w", err)
	}
	return nil
}

func (s *Store) RecordMemberVoiceTransition(ctx context.Context, guildID, userID, oldChannelID, newChannelID string, at time.Time) error {
	if oldChannelID == newChannelID {
		return nil
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("storage: iniciar registro de voz: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `
		INSERT INTO member_activity_preferences (guild_id, user_id)
		VALUES ($1, $2) ON CONFLICT (guild_id, user_id) DO NOTHING
	`, guildID, userID); err != nil {
		return fmt.Errorf("storage: preparar registro de voz: %w", err)
	}
	var enabled bool
	if err := tx.QueryRow(ctx, `
		SELECT tracking_enabled FROM member_activity_preferences
		WHERE guild_id = $1 AND user_id = $2
	`, guildID, userID).Scan(&enabled); err != nil {
		return fmt.Errorf("storage: comprobar seguimiento de voz: %w", err)
	}
	var joinedAt time.Time
	var sessionChannel string
	err = tx.QueryRow(ctx, `
		SELECT channel_id, joined_at FROM member_voice_sessions
		WHERE guild_id = $1 AND user_id = $2 FOR UPDATE
	`, guildID, userID).Scan(&sessionChannel, &joinedAt)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("storage: consultar sesión de voz: %w", err)
	}
	if err == nil {
		if enabled {
			seconds := int64(at.Sub(joinedAt).Seconds())
			if seconds < 0 {
				seconds = 0
			}
			if _, err := tx.Exec(ctx, `
				INSERT INTO member_voice_daily (guild_id, user_id, activity_date, channel_id, seconds)
				VALUES ($1, $2, ($3::timestamptz AT TIME ZONE 'UTC')::DATE, $4, $5)
				ON CONFLICT (guild_id, user_id, activity_date, channel_id)
				DO UPDATE SET seconds = member_voice_daily.seconds + EXCLUDED.seconds
			`, guildID, userID, at, sessionChannel, seconds); err != nil {
				return fmt.Errorf("storage: acumular tiempo de voz: %w", err)
			}
		}
		if _, err := tx.Exec(ctx, `DELETE FROM member_voice_sessions WHERE guild_id = $1 AND user_id = $2`, guildID, userID); err != nil {
			return fmt.Errorf("storage: cerrar sesión de voz: %w", err)
		}
	}
	if newChannelID != "" && enabled {
		if _, err := tx.Exec(ctx, `
			INSERT INTO member_voice_sessions (guild_id, user_id, channel_id, joined_at)
			VALUES ($1, $2, $3, $4)
			ON CONFLICT (guild_id, user_id)
			DO UPDATE SET channel_id = EXCLUDED.channel_id, joined_at = EXCLUDED.joined_at
		`, guildID, userID, newChannelID, at); err != nil {
			return fmt.Errorf("storage: abrir sesión de voz: %w", err)
		}
		if _, err := tx.Exec(ctx, `
			INSERT INTO member_voice_daily (guild_id, user_id, activity_date, channel_id, visits)
			VALUES ($1, $2, ($3::timestamptz AT TIME ZONE 'UTC')::DATE, $4, 1)
			ON CONFLICT (guild_id, user_id, activity_date, channel_id)
			DO UPDATE SET visits = member_voice_daily.visits + 1
		`, guildID, userID, at, newChannelID); err != nil {
			return fmt.Errorf("storage: contar visita a canal de voz: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("storage: confirmar registro de voz: %w", err)
	}
	return nil
}

func (s *Store) LoadMemberActivityStats(ctx context.Context, guildID, userID string) (*MemberActivityStats, error) {
	preferences, err := s.GetActivityPreferences(ctx, guildID, userID)
	if err != nil {
		return nil, err
	}
	stats := &MemberActivityStats{Preferences: preferences, TopWords: make([]ActivityWord, 0, 5)}
	if err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(messages), 0)
		FROM member_activity_hourly WHERE guild_id = $1 AND user_id = $2
	`, guildID, userID).Scan(&stats.Messages); err != nil {
		return nil, fmt.Errorf("storage: consultar total de mensajes: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(MAX(hour_utc), 0)
		FROM member_activity_hourly
		WHERE guild_id = $1 AND user_id = $2
		  AND (activity_date, hour_utc) = (
		      SELECT activity_date, hour_utc FROM member_activity_hourly
		      WHERE guild_id = $1 AND user_id = $2
		      ORDER BY messages DESC, activity_date DESC, hour_utc DESC LIMIT 1
		  )
	`, guildID, userID).Scan(&stats.ActiveHourUTC); err != nil {
		return nil, fmt.Errorf("storage: consultar hora más activa: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(seconds), 0) +
		       COALESCE((SELECT GREATEST(0, EXTRACT(EPOCH FROM (NOW() - joined_at))::BIGINT)
		                 FROM member_voice_sessions WHERE guild_id = $1 AND user_id = $2), 0)
		FROM member_voice_daily WHERE guild_id = $1 AND user_id = $2
	`, guildID, userID).Scan(&stats.VoiceSeconds); err != nil {
		return nil, fmt.Errorf("storage: consultar tiempo de voz: %w", err)
	}
	if err := s.pool.QueryRow(ctx, `
		SELECT channel_id FROM member_activity_channels
		WHERE guild_id = $1 AND user_id = $2 ORDER BY messages DESC, last_message_at DESC LIMIT 1
	`, guildID, userID).Scan(&stats.FavoriteChannelID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("storage: consultar canal favorito: %w", err)
	}
	rows, err := s.pool.Query(ctx, `
		SELECT word, occurrences FROM member_activity_words
		WHERE guild_id = $1 AND user_id = $2
		ORDER BY occurrences DESC, word LIMIT 5
	`, guildID, userID)
	if err != nil {
		return nil, fmt.Errorf("storage: consultar palabras frecuentes: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var word ActivityWord
		if err := rows.Scan(&word.Word, &word.Occurrences); err != nil {
			return nil, fmt.Errorf("storage: leer palabras frecuentes: %w", err)
		}
		stats.TopWords = append(stats.TopWords, word)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: recorrer palabras frecuentes: %w", err)
	}
	return stats, nil
}

func (s *Store) ListEngagementLeaders(ctx context.Context, guildID, category string) ([]EngagementLeader, error) {
	var query string
	switch category {
	case "messages":
		query = `
			SELECT h.user_id, SUM(h.messages)::BIGINT AS score
			FROM member_activity_hourly h
			JOIN member_activity_preferences p USING (guild_id, user_id)
			WHERE h.guild_id = $1 AND p.stats_public = TRUE AND p.tracking_enabled = TRUE
			GROUP BY h.user_id ORDER BY score DESC, h.user_id LIMIT 10`
	case "voice":
		query = `
			SELECT v.user_id, SUM(v.seconds)::BIGINT AS score
			FROM member_voice_daily v
			JOIN member_activity_preferences p USING (guild_id, user_id)
			WHERE v.guild_id = $1 AND p.stats_public = TRUE AND p.tracking_enabled = TRUE
			GROUP BY v.user_id ORDER BY score DESC, v.user_id LIMIT 10`
	case "help":
		query = `
			SELECT r.staff_id, SUM(r.reply_count)::BIGINT AS score
			FROM help_thread_staff_replies r
			JOIN help_threads t USING (thread_id)
			JOIN member_activity_preferences p ON p.guild_id = t.guild_id AND p.user_id = r.staff_id
			WHERE t.guild_id = $1 AND p.stats_public = TRUE
			GROUP BY r.staff_id ORDER BY score DESC, r.staff_id LIMIT 10`
	default:
		return nil, fmt.Errorf("storage: categoría de clasificación desconocida: %q", category)
	}
	rows, err := s.pool.Query(ctx, query, guildID)
	if err != nil {
		return nil, fmt.Errorf("storage: consultar clasificación celestial: %w", err)
	}
	defer rows.Close()
	leaders := make([]EngagementLeader, 0, 10)
	for rows.Next() {
		var leader EngagementLeader
		if err := rows.Scan(&leader.UserID, &leader.Score); err != nil {
			return nil, fmt.Errorf("storage: leer clasificación celestial: %w", err)
		}
		leaders = append(leaders, leader)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: recorrer clasificación celestial: %w", err)
	}
	return leaders, nil
}

func (s *Store) CelestialMissions(ctx context.Context, guildID, userID string, now time.Time) ([]CelestialMission, error) {
	utcNow := now.UTC()
	dayStart := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day(), 0, 0, 0, 0, time.UTC)
	weekStart := dayStart.AddDate(0, 0, -int((dayStart.Weekday()+6)%7))
	missions := make([]CelestialMission, 0, len(celestialMissions))
	for _, definition := range celestialMissions {
		periodStart := dayStart
		var progress int64
		switch definition.id {
		case "daily-messages":
			err := s.pool.QueryRow(ctx, `
				SELECT COALESCE(SUM(messages), 0) FROM member_activity_hourly
				WHERE guild_id = $1 AND user_id = $2 AND activity_date = $3
			`, guildID, userID, dayStart).Scan(&progress)
			if err != nil {
				return nil, fmt.Errorf("storage: consultar progreso de misión diaria: %w", err)
			}
		case "daily-voice":
			err := s.pool.QueryRow(ctx, `
				SELECT COUNT(*) FROM member_voice_daily
				WHERE guild_id = $1 AND user_id = $2 AND activity_date = $3 AND visits > 0
			`, guildID, userID, dayStart).Scan(&progress)
			if err != nil {
				return nil, fmt.Errorf("storage: consultar progreso de misión de voz: %w", err)
			}
		case "weekly-messages":
			periodStart = weekStart
			err := s.pool.QueryRow(ctx, `
				SELECT COALESCE(SUM(messages), 0) FROM member_activity_hourly
				WHERE guild_id = $1 AND user_id = $2 AND activity_date >= $3 AND activity_date < $4
			`, guildID, userID, weekStart, weekStart.AddDate(0, 0, 7)).Scan(&progress)
			if err != nil {
				return nil, fmt.Errorf("storage: consultar progreso de misión semanal: %w", err)
			}
		}
		var claimed bool
		if err := s.pool.QueryRow(ctx, `
			SELECT EXISTS (
				SELECT 1 FROM celestial_mission_claims
				WHERE guild_id = $1 AND user_id = $2 AND mission_id = $3 AND period_start = $4
			)
		`, guildID, userID, definition.id, periodStart).Scan(&claimed); err != nil {
			return nil, fmt.Errorf("storage: consultar reclamo de misión: %w", err)
		}
		missions = append(missions, CelestialMission{
			ID: definition.id, Name: definition.name, Description: definition.description,
			Current: progress, Target: definition.target, Points: definition.points,
			Claimed: claimed, PeriodStart: periodStart,
		})
	}
	return missions, nil
}

func (s *Store) ClaimCelestialMission(ctx context.Context, guildID, userID, missionID string, now time.Time) (int64, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("storage: iniciar reclamo celestial: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	utcNow := now.UTC()
	periodStart := time.Date(utcNow.Year(), utcNow.Month(), utcNow.Day(), 0, 0, 0, 0, time.UTC)
	var progress int64
	var definition *celestialMissionDefinition
	for idx := range celestialMissions {
		if celestialMissions[idx].id == missionID {
			definition = &celestialMissions[idx]
			break
		}
	}
	if definition == nil {
		return 0, errors.New("storage: misión celestial desconocida")
	}
	switch definition.period {
	case "week":
		periodStart = periodStart.AddDate(0, 0, -int((periodStart.Weekday()+6)%7))
		err = tx.QueryRow(ctx, `
			SELECT COALESCE(SUM(messages), 0) FROM member_activity_hourly
			WHERE guild_id = $1 AND user_id = $2 AND activity_date >= $3 AND activity_date < $4
		`, guildID, userID, periodStart, periodStart.AddDate(0, 0, 7)).Scan(&progress)
	case "day":
		if missionID == "daily-voice" {
			err = tx.QueryRow(ctx, `
				SELECT COUNT(*) FROM member_voice_daily
				WHERE guild_id = $1 AND user_id = $2 AND activity_date = $3 AND visits > 0
			`, guildID, userID, periodStart).Scan(&progress)
		} else {
			err = tx.QueryRow(ctx, `
				SELECT COALESCE(SUM(messages), 0) FROM member_activity_hourly
				WHERE guild_id = $1 AND user_id = $2 AND activity_date = $3
			`, guildID, userID, periodStart).Scan(&progress)
		}
	}
	if err != nil {
		return 0, fmt.Errorf("storage: comprobar progreso de misión: %w", err)
	}
	if progress < definition.target {
		return 0, ErrMissionNotReady
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO celestial_mission_claims (guild_id, user_id, mission_id, period_start, points)
		VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING
	`, guildID, userID, missionID, periodStart, definition.points)
	if err != nil {
		return 0, fmt.Errorf("storage: registrar reclamo de misión: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return 0, ErrMissionAlreadyClaimed
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO celestial_wallets (guild_id, user_id, points)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, user_id)
		DO UPDATE SET points = celestial_wallets.points + EXCLUDED.points, updated_at = NOW()
	`, guildID, userID, definition.points); err != nil {
		return 0, fmt.Errorf("storage: acreditar puntos de misión: %w", err)
	}
	var balance int64
	if err := tx.QueryRow(ctx, `
		SELECT points FROM celestial_wallets WHERE guild_id = $1 AND user_id = $2
	`, guildID, userID).Scan(&balance); err != nil {
		return 0, fmt.Errorf("storage: consultar saldo de misión: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("storage: confirmar reclamo celestial: %w", err)
	}
	return balance, nil
}

func (s *Store) CelestialBalance(ctx context.Context, guildID, userID string) (int64, error) {
	var points int64
	err := s.pool.QueryRow(ctx, `
		SELECT COALESCE((SELECT points FROM celestial_wallets WHERE guild_id = $1 AND user_id = $2), 0)
	`, guildID, userID).Scan(&points)
	if err != nil {
		return 0, fmt.Errorf("storage: consultar saldo celestial: %w", err)
	}
	return points, nil
}

func (s *Store) ConfigureCelestialRole(ctx context.Context, guildID, roleID, name, description string, price int64) error {
	if guildID == "" || roleID == "" || strings.TrimSpace(name) == "" || price <= 0 {
		return errors.New("storage: recompensa de rol inválida")
	}
	key := "role-" + roleID
	_, err := s.pool.Exec(ctx, `
		INSERT INTO celestial_rewards (guild_id, reward_key, role_id, name, description, price)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (guild_id, role_id) DO UPDATE
		SET role_id = EXCLUDED.role_id, name = EXCLUDED.name,
		    reward_key = EXCLUDED.reward_key, description = EXCLUDED.description,
		    price = EXCLUDED.price, active = TRUE
	`, guildID, key, roleID, strings.TrimSpace(name), strings.TrimSpace(description), price)
	if err != nil {
		return fmt.Errorf("storage: configurar recompensa celestial: %w", err)
	}
	return nil
}

func (s *Store) ListCelestialRewards(ctx context.Context, guildID string) ([]CelestialReward, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT reward_key, role_id, name, description, price
		FROM celestial_rewards WHERE guild_id = $1 AND active = TRUE
		ORDER BY price, name LIMIT 25
	`, guildID)
	if err != nil {
		return nil, fmt.Errorf("storage: listar bendiciones: %w", err)
	}
	defer rows.Close()
	rewards := make([]CelestialReward, 0)
	for rows.Next() {
		var reward CelestialReward
		if err := rows.Scan(&reward.Key, &reward.RoleID, &reward.Name, &reward.Description, &reward.Price); err != nil {
			return nil, fmt.Errorf("storage: leer bendición: %w", err)
		}
		rewards = append(rewards, reward)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: recorrer bendiciones: %w", err)
	}
	return rewards, nil
}

func (s *Store) BeginCelestialRedemption(ctx context.Context, guildID, userID, rewardKey string) (int64, CelestialReward, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return 0, CelestialReward{}, fmt.Errorf("storage: iniciar canje celestial: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var reward CelestialReward
	err = tx.QueryRow(ctx, `
		SELECT reward_key, role_id, name, description, price
		FROM celestial_rewards WHERE guild_id = $1 AND reward_key = $2 AND active = TRUE
		FOR UPDATE
	`, guildID, rewardKey).Scan(&reward.Key, &reward.RoleID, &reward.Name, &reward.Description, &reward.Price)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, CelestialReward{}, errors.New("storage: recompensa no encontrada")
	}
	if err != nil {
		return 0, CelestialReward{}, fmt.Errorf("storage: consultar recompensa: %w", err)
	}
	var redemptionID int64
	err = tx.QueryRow(ctx, `
		INSERT INTO celestial_redemptions (guild_id, user_id, reward_key, role_id, points)
		VALUES ($1, $2, $3, $4, $5) RETURNING id
	`, guildID, userID, reward.Key, reward.RoleID, reward.Price).Scan(&redemptionID)
	if err != nil {
		if strings.Contains(err.Error(), "idx_celestial_completed_reward_once") {
			return 0, CelestialReward{}, ErrRewardAlreadyOwned
		}
		return 0, CelestialReward{}, fmt.Errorf("storage: registrar canje pendiente: %w", err)
	}
	tag, err := tx.Exec(ctx, `
		UPDATE celestial_wallets SET points = points - $3, updated_at = NOW()
		WHERE guild_id = $1 AND user_id = $2 AND points >= $3
	`, guildID, userID, reward.Price)
	if err != nil {
		return 0, CelestialReward{}, fmt.Errorf("storage: descontar puntos de canje: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return 0, CelestialReward{}, ErrInsufficientCelestialPoints
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, CelestialReward{}, fmt.Errorf("storage: confirmar canje pendiente: %w", err)
	}
	return redemptionID, reward, nil
}

func (s *Store) CompleteCelestialRedemption(ctx context.Context, redemptionID int64) error {
	tag, err := s.pool.Exec(ctx, `
		UPDATE celestial_redemptions
		SET status = 'completed', completed_at = NOW()
		WHERE id = $1 AND status = 'pending'
	`, redemptionID)
	if err != nil {
		return fmt.Errorf("storage: finalizar canje celestial: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return errors.New("storage: canje pendiente no encontrado")
	}
	return nil
}

func (s *Store) CancelCelestialRedemption(ctx context.Context, redemptionID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("storage: iniciar reembolso celestial: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var guildID, userID string
	var points int64
	err = tx.QueryRow(ctx, `
		UPDATE celestial_redemptions SET status = 'cancelled'
		WHERE id = $1 AND status = 'pending'
		RETURNING guild_id, user_id, points
	`, redemptionID).Scan(&guildID, &userID, &points)
	if err != nil {
		return fmt.Errorf("storage: cancelar canje celestial: %w", err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO celestial_wallets (guild_id, user_id, points)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, user_id)
		DO UPDATE SET points = celestial_wallets.points + EXCLUDED.points, updated_at = NOW()
	`, guildID, userID, points); err != nil {
		return fmt.Errorf("storage: reembolsar canje celestial: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("storage: confirmar reembolso celestial: %w", err)
	}
	return nil
}

func extractActivityWords(content string) map[string]int64 {
	words := make(map[string]int64)
	content = activityExcludedPattern.ReplaceAllString(content, " ")
	for _, word := range activityWordPattern.FindAllString(strings.ToLower(content), 100) {
		word = strings.Map(func(r rune) rune {
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		}, word)
		if len([]rune(word)) < 3 {
			continue
		}
		hasLetter := false
		for _, character := range word {
			if unicode.IsLetter(character) {
				hasLetter = true
				break
			}
		}
		if !hasLetter {
			continue
		}
		if _, stop := activityStopWords[word]; stop {
			continue
		}
		words[word]++
	}
	return words
}
