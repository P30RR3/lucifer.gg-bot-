CREATE TABLE IF NOT EXISTS economy_accounts (
    guild_id         TEXT        NOT NULL,
    user_id          TEXT        NOT NULL,
    balance          BIGINT      NOT NULL DEFAULT 1000,
    daily_claimed_at TIMESTAMPTZ,
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (guild_id, user_id),
    CONSTRAINT economy_balance_nonnegative CHECK (balance >= 0)
);
