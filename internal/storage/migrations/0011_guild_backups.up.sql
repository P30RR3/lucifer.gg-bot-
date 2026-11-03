CREATE TABLE IF NOT EXISTS guild_backups (
    guild_id TEXT PRIMARY KEY,
    created_by TEXT NOT NULL,
    snapshot JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
