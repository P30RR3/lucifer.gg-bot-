ALTER TABLE economy_accounts
    ADD COLUMN IF NOT EXISTS experience BIGINT NOT NULL DEFAULT 0;

CREATE TABLE IF NOT EXISTS lux_member_preferences (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    language TEXT NOT NULL DEFAULT 'es',
    notifications_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    hide_inventory BOOLEAN NOT NULL DEFAULT FALSE,
    looking_for_group BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (guild_id, user_id),
    CONSTRAINT lux_member_language_valid CHECK (language IN ('es', 'en'))
);

CREATE TABLE IF NOT EXISTS lux_inventory (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    item_key TEXT NOT NULL,
    item_name TEXT NOT NULL,
    quantity INTEGER NOT NULL DEFAULT 1,
    acquired_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (guild_id, user_id, item_key),
    CONSTRAINT lux_inventory_quantity_positive CHECK (quantity > 0)
);

CREATE INDEX IF NOT EXISTS idx_lux_group_search
    ON lux_member_preferences (guild_id, updated_at DESC)
    WHERE looking_for_group = TRUE;
