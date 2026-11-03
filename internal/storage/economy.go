package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

const initialEconomyBalance int64 = 1000

var (
	ErrInsufficientChips = errors.New("storage: fichas insuficientes")
	ErrDailyCooldown     = errors.New("storage: fichas diarias ya reclamadas")
)

func (s *Store) EconomyBalance(ctx context.Context, guildID, userID string) (int64, error) {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO economy_accounts (guild_id, user_id, balance)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, user_id) DO NOTHING
	`, guildID, userID, initialEconomyBalance)
	if err != nil {
		return 0, fmt.Errorf("storage: crear cuenta de economía: %w", err)
	}

	var balance int64
	if err := s.pool.QueryRow(ctx, `
		SELECT balance FROM economy_accounts
		WHERE guild_id = $1 AND user_id = $2
	`, guildID, userID).Scan(&balance); err != nil {
		return 0, fmt.Errorf("storage: consultar saldo: %w", err)
	}
	return balance, nil
}

func (s *Store) ClaimDailyChips(ctx context.Context, guildID, userID string) (int64, error) {
	const dailyReward int64 = 250

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("storage: iniciar reclamo diario: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := ensureEconomyAccount(ctx, tx, guildID, userID, initialEconomyBalance); err != nil {
		return 0, err
	}

	var balance int64
	err = tx.QueryRow(ctx, `
		UPDATE economy_accounts
		SET balance = balance + $3, experience = experience + 25,
		    daily_claimed_at = NOW(), updated_at = NOW()
		WHERE guild_id = $1 AND user_id = $2
		  AND (daily_claimed_at IS NULL OR daily_claimed_at <= NOW() - INTERVAL '24 hours')
		RETURNING balance
	`, guildID, userID, dailyReward).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrDailyCooldown
	}
	if err != nil {
		return 0, fmt.Errorf("storage: reclamar fichas diarias: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("storage: confirmar reclamo diario: %w", err)
	}
	return balance, nil
}

func (s *Store) PlayRoulette(ctx context.Context, guildID, userID string, bet, payout int64) (int64, error) {
	if bet <= 0 || payout < 0 {
		return 0, errors.New("storage: apuesta o premio inválido")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("storage: iniciar apuesta: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := ensureEconomyAccount(ctx, tx, guildID, userID, initialEconomyBalance); err != nil {
		return 0, err
	}

	var balance int64
	err = tx.QueryRow(ctx, `
		UPDATE economy_accounts
		SET balance = balance - $3, updated_at = NOW()
		WHERE guild_id = $1 AND user_id = $2 AND balance >= $3
		RETURNING balance
	`, guildID, userID, bet).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrInsufficientChips
	}
	if err != nil {
		return 0, fmt.Errorf("storage: debitar apuesta: %w", err)
	}

	if payout > 0 {
		err = tx.QueryRow(ctx, `
			UPDATE economy_accounts
			SET balance = balance + $3, updated_at = NOW()
			WHERE guild_id = $1 AND user_id = $2
			RETURNING balance
		`, guildID, userID, payout).Scan(&balance)
		if err != nil {
			return 0, fmt.Errorf("storage: acreditar premio: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("storage: confirmar apuesta: %w", err)
	}
	return balance, nil
}

func (s *Store) TipMaze(ctx context.Context, guildID, userID string, amount int64) (int64, error) {
	if amount <= 0 {
		return 0, errors.New("storage: propina inválida")
	}

	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return 0, fmt.Errorf("storage: iniciar propina: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if err := ensureEconomyAccount(ctx, tx, guildID, userID, initialEconomyBalance); err != nil {
		return 0, err
	}

	var balance int64
	err = tx.QueryRow(ctx, `
		UPDATE economy_accounts
		SET balance = balance - $3, updated_at = NOW()
		WHERE guild_id = $1 AND user_id = $2 AND balance >= $3
		RETURNING balance
	`, guildID, userID, amount).Scan(&balance)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrInsufficientChips
	}
	if err != nil {
		return 0, fmt.Errorf("storage: debitar propina: %w", err)
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO economy_accounts (guild_id, user_id, balance)
		VALUES ($1, 'maze', $2)
		ON CONFLICT (guild_id, user_id)
		DO UPDATE SET balance = economy_accounts.balance + EXCLUDED.balance, updated_at = NOW()
	`, guildID, amount)
	if err != nil {
		return 0, fmt.Errorf("storage: acreditar aporte a la caja del Lux: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("storage: confirmar propina: %w", err)
	}
	return balance, nil
}

func ensureEconomyAccount(ctx context.Context, tx pgx.Tx, guildID, userID string, initialBalance int64) error {
	_, err := tx.Exec(ctx, `
		INSERT INTO economy_accounts (guild_id, user_id, balance)
		VALUES ($1, $2, $3)
		ON CONFLICT (guild_id, user_id) DO NOTHING
	`, guildID, userID, initialBalance)
	if err != nil {
		return fmt.Errorf("storage: crear cuenta de economía: %w", err)
	}
	return nil
}
