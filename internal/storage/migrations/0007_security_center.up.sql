CREATE TABLE IF NOT EXISTS sanction_appeals (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    sanction TEXT NOT NULL,
    reason TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    reviewed_by TEXT NOT NULL DEFAULT '',
    reviewed_at TIMESTAMPTZ,
    CONSTRAINT sanction_appeals_status_valid CHECK (status IN ('pending', 'accepted', 'denied'))
);

CREATE INDEX IF NOT EXISTS idx_sanction_appeals_target
    ON sanction_appeals (guild_id, target_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_sanction_appeals_pending
    ON sanction_appeals (guild_id, status, created_at DESC);
