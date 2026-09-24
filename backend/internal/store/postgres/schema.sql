CREATE TABLE IF NOT EXISTS seasons (
    id BIGSERIAL PRIMARY KEY,
    name TEXT NOT NULL,
    status TEXT NOT NULL,
    timezone TEXT NOT NULL DEFAULT 'Europe/London',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    started_at TIMESTAMPTZ,
    game_variant TEXT NOT NULL DEFAULT '501',
    legs_to_win INTEGER NOT NULL DEFAULT 3,
    games_per_week INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS divisions (
    id BIGSERIAL PRIMARY KEY,
    season_id BIGINT NOT NULL REFERENCES seasons (id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    slug TEXT NOT NULL,
    position INTEGER NOT NULL DEFAULT 1,
    slack_public_channel_id TEXT NOT NULL DEFAULT '',
    UNIQUE (season_id, slug)
);

ALTER TABLE seasons ADD COLUMN IF NOT EXISTS game_variant TEXT NOT NULL DEFAULT '501';
ALTER TABLE seasons ADD COLUMN IF NOT EXISTS legs_to_win INTEGER NOT NULL DEFAULT 3;
ALTER TABLE seasons ADD COLUMN IF NOT EXISTS games_per_week INTEGER NOT NULL DEFAULT 1;

CREATE TABLE IF NOT EXISTS players (
    id BIGSERIAL PRIMARY KEY,
    season_id BIGINT NOT NULL REFERENCES seasons (id) ON DELETE CASCADE,
    division_id BIGINT REFERENCES divisions (id) ON DELETE SET NULL,
    display_name TEXT NOT NULL,
    display_name_normalized TEXT NOT NULL,
    nickname TEXT,
    status TEXT NOT NULL DEFAULT 'waitlist',
    registered_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (display_name_normalized)
);

ALTER TABLE players ADD COLUMN IF NOT EXISTS division_id BIGINT REFERENCES divisions (id) ON DELETE SET NULL;
ALTER TABLE players ADD COLUMN IF NOT EXISTS status TEXT NOT NULL DEFAULT 'waitlist';

CREATE TABLE IF NOT EXISTS fixtures (
    id BIGSERIAL PRIMARY KEY,
    season_id BIGINT NOT NULL REFERENCES seasons (id) ON DELETE CASCADE,
    division_id BIGINT NOT NULL REFERENCES divisions (id) ON DELETE CASCADE,
    week_number INTEGER NOT NULL,
    scheduled_at TIMESTAMPTZ NOT NULL,
    player_one_id BIGINT NOT NULL REFERENCES players (id),
    player_two_id BIGINT NOT NULL REFERENCES players (id),
    game_variant TEXT NOT NULL DEFAULT '501',
    legs_to_win INTEGER NOT NULL DEFAULT 3,
    status TEXT NOT NULL DEFAULT 'scheduled'
);

ALTER TABLE fixtures ADD COLUMN IF NOT EXISTS division_id BIGINT REFERENCES divisions (id) ON DELETE CASCADE;

INSERT INTO divisions (season_id, name, slug)
SELECT DISTINCT season_id, 'Division 1', 'division-1'
FROM fixtures WHERE division_id IS NULL
ON CONFLICT (season_id, slug) DO NOTHING;

UPDATE players p SET division_id = d.id, status = 'assigned'
FROM divisions d
WHERE p.season_id = d.season_id AND d.slug = 'division-1' AND p.division_id IS NULL
  AND EXISTS (SELECT 1 FROM fixtures f WHERE f.season_id = p.season_id AND f.division_id IS NULL);

UPDATE fixtures f SET division_id = d.id
FROM divisions d
WHERE f.season_id = d.season_id AND d.slug = 'division-1' AND f.division_id IS NULL;

ALTER TABLE fixtures ALTER COLUMN division_id SET NOT NULL;

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
