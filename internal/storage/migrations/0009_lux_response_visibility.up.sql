ALTER TABLE lux_member_preferences
    ADD COLUMN IF NOT EXISTS responses_public BOOLEAN NOT NULL DEFAULT TRUE;
