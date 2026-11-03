CREATE TABLE IF NOT EXISTS audit_settings (
    guild_id TEXT PRIMARY KEY,
    log_channel_id TEXT NOT NULL DEFAULT '',
    support_channel_id TEXT NOT NULL DEFAULT '',
    include_deleted_content BOOLEAN NOT NULL DEFAULT FALSE,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS deleted_messages (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    message_id TEXT NOT NULL,
    author_id TEXT NOT NULL DEFAULT '',
    channel_id TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
    deleted_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (guild_id, message_id)
);

CREATE INDEX IF NOT EXISTS idx_deleted_messages_guild_time
    ON deleted_messages (guild_id, deleted_at DESC);

CREATE TABLE IF NOT EXISTS staff_actions (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    target_id TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    details TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_staff_actions_guild_time
    ON staff_actions (guild_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_staff_actions_created_at
    ON staff_actions (created_at);
