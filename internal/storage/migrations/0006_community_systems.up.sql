CREATE TABLE IF NOT EXISTS community_feature_settings (
    guild_id TEXT PRIMARY KEY,
    help_channel_id TEXT NOT NULL DEFAULT '',
    confession_channel_id TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE IF NOT EXISTS help_threads (
    guild_id TEXT NOT NULL,
    thread_id TEXT PRIMARY KEY,
    requester_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    closed_at TIMESTAMPTZ,
    closed_by TEXT NOT NULL DEFAULT '',
    awarded_staff_id TEXT NOT NULL DEFAULT '',
    CONSTRAINT help_threads_status_valid CHECK (status IN ('open', 'resolved', 'more_help'))
);

CREATE TABLE IF NOT EXISTS help_thread_staff_replies (
    thread_id TEXT NOT NULL REFERENCES help_threads(thread_id) ON DELETE CASCADE,
    staff_id TEXT NOT NULL,
    reply_count INTEGER NOT NULL DEFAULT 0,
    last_reply_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (thread_id, staff_id),
    CONSTRAINT help_thread_reply_count_positive CHECK (reply_count > 0)
);

CREATE TABLE IF NOT EXISTS trial_cases (
    guild_id TEXT NOT NULL,
    thread_id TEXT PRIMARY KEY,
    plaintiff_id TEXT NOT NULL,
    defendant_id TEXT NOT NULL,
    reason TEXT NOT NULL,
    deadline TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    verdict TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT trial_cases_status_valid CHECK (status IN ('open', 'resolved', 'cancelled'))
);

CREATE TABLE IF NOT EXISTS deep_confessions (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    author_id TEXT NOT NULL,
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    expires_at TIMESTAMPTZ NOT NULL DEFAULT (NOW() + INTERVAL '30 days')
);

CREATE INDEX IF NOT EXISTS idx_deep_confessions_guild_created
    ON deep_confessions (guild_id, created_at DESC);

CREATE TABLE IF NOT EXISTS deep_confession_offers (
    confession_id BIGINT NOT NULL REFERENCES deep_confessions(id) ON DELETE CASCADE,
    offerer_id TEXT NOT NULL,
    price BIGINT NOT NULL,
    evaluation TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (confession_id, offerer_id),
    CONSTRAINT deep_confession_offer_price_positive CHECK (price > 0)
);

CREATE TABLE IF NOT EXISTS investigation_evidence (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    actor_id TEXT NOT NULL,
    evidence TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_investigation_evidence_target
    ON investigation_evidence (guild_id, target_id, created_at DESC);

CREATE TABLE IF NOT EXISTS investigation_responses (
    id BIGSERIAL PRIMARY KEY,
    guild_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    response TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_investigation_responses_target
    ON investigation_responses (guild_id, target_id, created_at DESC);

CREATE TABLE IF NOT EXISTS infernal_cases (
    guild_id TEXT NOT NULL,
    target_id TEXT NOT NULL,
    thread_id TEXT NOT NULL DEFAULT '',
    actor_id TEXT NOT NULL,
    reason TEXT NOT NULL,
    challenge TEXT NOT NULL DEFAULT '',
    challenge_answer TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    note TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    resolved_at TIMESTAMPTZ,
    resolved_by TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (guild_id, target_id)
);

CREATE TABLE IF NOT EXISTS lux_shop_items (
    guild_id TEXT NOT NULL,
    item_key TEXT NOT NULL,
    name TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    price BIGINT NOT NULL,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (guild_id, item_key),
    CONSTRAINT lux_shop_item_price_nonnegative CHECK (price >= 0)
);

CREATE TABLE IF NOT EXISTS lux_ranks (
    guild_id TEXT NOT NULL,
    min_level BIGINT NOT NULL,
    name TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (guild_id, min_level),
    CONSTRAINT lux_rank_level_positive CHECK (min_level > 0)
);
