CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS guilds (
    guild_id        TEXT PRIMARY KEY,
    locale          TEXT        NOT NULL DEFAULT 'es',
    prefix          TEXT        NOT NULL DEFAULT '!',
    antiraid_on     BOOLEAN     NOT NULL DEFAULT TRUE,
    score_threshold INT         NOT NULL DEFAULT 70,
    default_action  TEXT        NOT NULL DEFAULT 'timeout',
    log_channel_id  TEXT        NOT NULL DEFAULT '',
    welcome_channel TEXT        NOT NULL DEFAULT '',
    welcome_message TEXT        NOT NULL DEFAULT '',
    ai_moderation   BOOLEAN     NOT NULL DEFAULT TRUE,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT guilds_score_range CHECK (score_threshold BETWEEN 0 AND 100),
    CONSTRAINT guilds_action_valid CHECK (
        default_action IN ('timeout','kick','ban','warn','lockdown')
    )
);

CREATE TABLE IF NOT EXISTS incidents (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    guild_id    TEXT        NOT NULL REFERENCES guilds(guild_id) ON DELETE CASCADE,
    detector    TEXT        NOT NULL,
    user_id     TEXT        NOT NULL DEFAULT '',
    channel_id  TEXT        NOT NULL DEFAULT '',
    score       INT         NOT NULL DEFAULT 0,
    action      TEXT        NOT NULL DEFAULT 'none',
    reason      TEXT        NOT NULL DEFAULT '',
    evidence    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by TEXT        NOT NULL DEFAULT '',

    CONSTRAINT incidents_score_range CHECK (score BETWEEN 0 AND 100)
);

CREATE INDEX IF NOT EXISTS idx_incidents_guild_created
    ON incidents (guild_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_incidents_user
    ON incidents (guild_id, user_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_incidents_detector
    ON incidents (guild_id, detector, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_incidents_unresolved
    ON incidents (guild_id, created_at DESC)
    WHERE resolved_at IS NULL;