UPDATE lux_shop_items
SET name = 'Daga ceremonial del Lux'
WHERE item_key = 'maze-daggers';

UPDATE lux_shop_items
SET name = 'Rosa nocturna'
WHERE item_key = 'maze-rose';

UPDATE lux_inventory
SET item_name = 'Daga ceremonial del Lux'
WHERE item_key = 'maze-daggers';

UPDATE lux_inventory
SET item_name = 'Rosa nocturna'
WHERE item_key = 'maze-rose';

CREATE TABLE IF NOT EXISTS member_activity_preferences (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    tracking_enabled BOOLEAN NOT NULL DEFAULT FALSE,
    stats_public BOOLEAN NOT NULL DEFAULT FALSE,
    response_mode TEXT NOT NULL DEFAULT 'private',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (guild_id, user_id),
    CONSTRAINT member_activity_response_mode_valid CHECK (response_mode IN ('private', 'public', 'dm'))
);

CREATE TABLE IF NOT EXISTS member_activity_hourly (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    activity_date DATE NOT NULL,
    hour_utc SMALLINT NOT NULL,
    messages BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (guild_id, user_id, activity_date, hour_utc),
    CONSTRAINT member_activity_hour_valid CHECK (hour_utc BETWEEN 0 AND 23)
);

CREATE TABLE IF NOT EXISTS member_activity_channels (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    messages BIGINT NOT NULL DEFAULT 0,
    last_message_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (guild_id, user_id, channel_id)
);

CREATE TABLE IF NOT EXISTS member_activity_words (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    word TEXT NOT NULL,
    occurrences BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (guild_id, user_id, word)
);

CREATE TABLE IF NOT EXISTS member_voice_sessions (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    channel_id TEXT NOT NULL,
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (guild_id, user_id)
);

CREATE TABLE IF NOT EXISTS member_voice_daily (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    activity_date DATE NOT NULL,
    channel_id TEXT NOT NULL,
    visits INTEGER NOT NULL DEFAULT 0,
    seconds BIGINT NOT NULL DEFAULT 0,
    PRIMARY KEY (guild_id, user_id, activity_date, channel_id)
);

CREATE INDEX IF NOT EXISTS idx_member_activity_hourly_leaderboard
    ON member_activity_hourly (guild_id, user_id, activity_date DESC);
CREATE INDEX IF NOT EXISTS idx_member_voice_daily_leaderboard
    ON member_voice_daily (guild_id, user_id, activity_date DESC);
CREATE INDEX IF NOT EXISTS idx_member_activity_words_top
    ON member_activity_words (guild_id, user_id, occurrences DESC);

CREATE TABLE IF NOT EXISTS celestial_wallets (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    points BIGINT NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (guild_id, user_id),
    CONSTRAINT celestial_points_nonnegative CHECK (points >= 0)
);

CREATE TABLE IF NOT EXISTS celestial_mission_claims (
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    mission_id TEXT NOT NULL,
    period_start DATE NOT NULL,
    points BIGINT NOT NULL,
    claimed_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (guild_id, user_id, mission_id, period_start),
    CONSTRAINT celestial_mission_points_positive CHECK (points > 0)
);

CREATE TABLE IF NOT EXISTS celestial_rewards (
    guild_id TEXT NOT NULL,
    reward_key TEXT NOT NULL,
    role_id TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price BIGINT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (guild_id, reward_key),
    UNIQUE (guild_id, role_id),
    CONSTRAINT celestial_reward_price_positive CHECK (price > 0)
);

CREATE TABLE IF NOT EXISTS celestial_redemptions (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    user_id TEXT NOT NULL,
    reward_key TEXT NOT NULL,
    role_id TEXT NOT NULL,
    points BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ,
    CONSTRAINT celestial_redemption_status_valid CHECK (status IN ('pending', 'completed', 'cancelled')),
    CONSTRAINT celestial_redemption_points_positive CHECK (points > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_celestial_completed_reward_once
    ON celestial_redemptions (guild_id, user_id, reward_key)
    WHERE status IN ('pending', 'completed');
CREATE INDEX IF NOT EXISTS idx_celestial_rewards_active
    ON celestial_rewards (guild_id, active, price);
