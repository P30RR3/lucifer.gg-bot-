package storage

import (
	"context"
	"fmt"
)

type LuxPreferences struct {
	Language             string
	NotificationsEnabled bool
	HideInventory        bool
	LookingForGroup      bool
	ResponsesPublic      bool
}

type LuxInventoryItem struct {
	Name     string
	Quantity int
}

type LuxPanelData struct {
	Balance         int64
	Experience      int64
	Rank            string
	Preferences     LuxPreferences
	Inventory       []LuxInventoryItem
	LookingForGroup []string
}

func (s *Store) LoadLuxPanel(ctx context.Context, guildID, userID string) (*LuxPanelData, error) {
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO economy_accounts (guild_id, user_id, balance)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, user_id) DO NOTHING
	`, guildID, userID, initialEconomyBalance); err != nil {
		return nil, fmt.Errorf("storage: crear cuenta del panel Lux: %w", err)
	}
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO lux_member_preferences (guild_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id, user_id) DO NOTHING
	`, guildID, userID); err != nil {
		return nil, fmt.Errorf("storage: crear preferencias del panel Lux: %w", err)
	}

	panel := &LuxPanelData{Inventory: make([]LuxInventoryItem, 0), LookingForGroup: make([]string, 0)}
	if err := s.pool.QueryRow(ctx, `
		SELECT balance, experience
		FROM economy_accounts
		WHERE guild_id = $1 AND user_id = $2
	`, guildID, userID).Scan(&panel.Balance, &panel.Experience); err != nil {
		return nil, fmt.Errorf("storage: consultar economía del panel Lux: %w", err)
	}
	rank, err := s.LuxRank(ctx, guildID, panel.Experience/100+1)
	if err != nil {
		return nil, err
	}
	panel.Rank = rank
	if err := s.pool.QueryRow(ctx, `
		SELECT language, notifications_enabled, hide_inventory, looking_for_group, responses_public
		FROM lux_member_preferences
		WHERE guild_id = $1 AND user_id = $2
	`, guildID, userID).Scan(
		&panel.Preferences.Language,
		&panel.Preferences.NotificationsEnabled,
		&panel.Preferences.HideInventory,
		&panel.Preferences.LookingForGroup,
		&panel.Preferences.ResponsesPublic,
	); err != nil {
		return nil, fmt.Errorf("storage: consultar preferencias del panel Lux: %w", err)
	}

	rows, err := s.pool.Query(ctx, `
		SELECT item_name, quantity
		FROM lux_inventory
		WHERE guild_id = $1 AND user_id = $2
		ORDER BY item_name
		LIMIT 30
	`, guildID, userID)
	if err != nil {
		return nil, fmt.Errorf("storage: consultar inventario del panel Lux: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var item LuxInventoryItem
		if err := rows.Scan(&item.Name, &item.Quantity); err != nil {
			return nil, fmt.Errorf("storage: leer inventario del panel Lux: %w", err)
		}
		panel.Inventory = append(panel.Inventory, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: recorrer inventario del panel Lux: %w", err)
	}

	rows, err = s.pool.Query(ctx, `
		SELECT user_id
		FROM lux_member_preferences
		WHERE guild_id = $1 AND user_id <> $2 AND looking_for_group = TRUE
		ORDER BY updated_at DESC
		LIMIT 10
	`, guildID, userID)
	if err != nil {
		return nil, fmt.Errorf("storage: consultar búsqueda de grupo del panel Lux: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var user string
		if err := rows.Scan(&user); err != nil {
			return nil, fmt.Errorf("storage: leer búsqueda de grupo del panel Lux: %w", err)
		}
		panel.LookingForGroup = append(panel.LookingForGroup, user)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("storage: recorrer búsqueda de grupo del panel Lux: %w", err)
	}

	return panel, nil
}

func (s *Store) ToggleLuxPreference(ctx context.Context, guildID, userID, preference string) error {
	if _, err := s.pool.Exec(ctx, `
		INSERT INTO lux_member_preferences (guild_id, user_id)
		VALUES ($1, $2)
		ON CONFLICT (guild_id, user_id) DO NOTHING
	`, guildID, userID); err != nil {
		return fmt.Errorf("storage: preparar preferencias del panel Lux: %w", err)
	}

	var query string
	switch preference {
	case "notifications":
		query = `UPDATE lux_member_preferences SET notifications_enabled = NOT notifications_enabled, updated_at = NOW() WHERE guild_id = $1 AND user_id = $2`
	case "hide_inventory":
		query = `UPDATE lux_member_preferences SET hide_inventory = NOT hide_inventory, updated_at = NOW() WHERE guild_id = $1 AND user_id = $2`
	case "looking_for_group":
		query = `UPDATE lux_member_preferences SET looking_for_group = NOT looking_for_group, updated_at = NOW() WHERE guild_id = $1 AND user_id = $2`
	case "language":
		query = `UPDATE lux_member_preferences SET language = CASE language WHEN 'es' THEN 'en' ELSE 'es' END, updated_at = NOW() WHERE guild_id = $1 AND user_id = $2`
	case "responses_public":
		query = `UPDATE lux_member_preferences SET responses_public = NOT responses_public, updated_at = NOW() WHERE guild_id = $1 AND user_id = $2`
	default:
		return fmt.Errorf("storage: preferencia del panel Lux desconocida: %q", preference)
	}
	if _, err := s.pool.Exec(ctx, query, guildID, userID); err != nil {
		return fmt.Errorf("storage: actualizar preferencia del panel Lux: %w", err)
	}
	return nil
}

func (s *Store) LuxResponsesPublic(ctx context.Context, guildID, userID string) (bool, error) {
	var visible bool
	if err := s.pool.QueryRow(ctx, `
		SELECT COALESCE((
			SELECT responses_public
			FROM lux_member_preferences
			WHERE guild_id = $1 AND user_id = $2
		), TRUE)
	`, guildID, userID).Scan(&visible); err != nil {
		return false, fmt.Errorf("storage: consultar visibilidad de respuestas: %w", err)
	}
	return visible, nil
}
