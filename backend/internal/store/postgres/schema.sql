CREATE TABLE IF NOT EXISTS seasons (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'Europe/London',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS players (
    id BIGSERIAL PRIMARY KEY,
    season_id BIGINT NOT NULL REFERENCES seasons (id) ON DELETE CASCADE,
    display_name TEXT NOT NULL,
    display_name_normalized TEXT NOT NULL,
    nickname TEXT,
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (season_id, display_name_normalized)
);

CREATE TABLE IF NOT EXISTS fixtures (
    id BIGSERIAL PRIMARY KEY,
    season_id BIGINT NOT NULL REFERENCES seasons (id) ON DELETE CASCADE,
    week_number INTEGER NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    player_one_id BIGINT NOT NULL REFERENCES players (id),
    player_two_id BIGINT NOT NULL REFERENCES players (id),
    game_variant TEXT NOT NULL DEFAULT '501',
    legs_to_win INTEGER NOT NULL DEFAULT 3,
    status TEXT NOT NULL DEFAULT 'scheduled'
);

CREATE TABLE IF NOT EXISTS results (
    id BIGSERIAL PRIMARY KEY,
    fixture_id BIGINT NOT NULL UNIQUE REFERENCES fixtures (id) ON DELETE CASCADE,
    player_one_legs INTEGER NOT NULL,
    player_two_legs INTEGER NOT NULL,
    player_one_average DOUBLE PRECISION,
    player_two_average DOUBLE PRECISION,
    winner_id BIGINT NOT NULL REFERENCES players (id),
    entered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

ALTER TABLE results ADD COLUMN IF NOT EXISTS player_one_average DOUBLE PRECISION;
ALTER TABLE results ADD COLUMN IF NOT EXISTS player_two_average DOUBLE PRECISION;

CREATE TABLE IF NOT EXISTS admin_audit_log (
    id BIGSERIAL PRIMARY KEY,
    fixture_id BIGINT NOT NULL REFERENCES fixtures (id) ON DELETE CASCADE,
    action TEXT NOT NULL,
    actor TEXT NOT NULL,
    old_payload JSONB,
    new_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
