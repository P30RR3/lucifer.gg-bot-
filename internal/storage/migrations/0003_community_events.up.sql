CREATE TABLE IF NOT EXISTS community_ideas (
    id          BIGSERIAL   PRIMARY KEY,
    guild_id    TEXT        NOT NULL,
    author_id   TEXT        NOT NULL,
    anonymous   BOOLEAN     NOT NULL DEFAULT FALSE,
    title       TEXT        NOT NULL,
    description TEXT        NOT NULL,
    status      TEXT        NOT NULL DEFAULT 'open',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT community_ideas_status_valid
        CHECK (status IN ('open', 'accepted', 'declined'))
);

CREATE INDEX IF NOT EXISTS idx_community_ideas_guild_status
    ON community_ideas (guild_id, status, created_at DESC);

CREATE TABLE IF NOT EXISTS community_idea_votes (
    idea_id     BIGINT      NOT NULL REFERENCES community_ideas(id) ON DELETE CASCADE,
    user_id     TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (idea_id, user_id)
);

CREATE TABLE IF NOT EXISTS lux_events (
    id          BIGSERIAL   PRIMARY KEY,
    guild_id    TEXT        NOT NULL,
    channel_id  TEXT        NOT NULL,
    creator_id  TEXT        NOT NULL,
    title       TEXT        NOT NULL,
    description TEXT        NOT NULL,
    starts_at   TIMESTAMPTZ NOT NULL,
    capacity    INT         NOT NULL DEFAULT 0,
    status      TEXT        NOT NULL DEFAULT 'open',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT lux_events_capacity_nonnegative CHECK (capacity >= 0),
    CONSTRAINT lux_events_status_valid CHECK (status IN ('open', 'cancelled'))
);

CREATE INDEX IF NOT EXISTS idx_lux_events_upcoming
    ON lux_events (guild_id, starts_at)
    WHERE status = 'open';

CREATE TABLE IF NOT EXISTS lux_event_attendees (
    event_id    BIGINT      NOT NULL REFERENCES lux_events(id) ON DELETE CASCADE,
    user_id     TEXT        NOT NULL,
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    PRIMARY KEY (event_id, user_id)
);
