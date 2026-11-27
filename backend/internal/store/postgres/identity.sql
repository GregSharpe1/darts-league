CREATE SEQUENCE IF NOT EXISTS player_identity_id_seq AS BIGINT;

ALTER TABLE players ADD COLUMN IF NOT EXISTS identity_id BIGINT NOT NULL DEFAULT nextval('player_identity_id_seq');
ALTER TABLE tournament_entries ADD COLUMN IF NOT EXISTS identity_id BIGINT NOT NULL DEFAULT nextval('player_identity_id_seq');

ALTER TABLE players ADD COLUMN IF NOT EXISTS requested_identity_id BIGINT;
ALTER TABLE tournament_entries ADD COLUMN IF NOT EXISTS requested_identity_id BIGINT;

CREATE UNIQUE INDEX IF NOT EXISTS players_season_identity_key ON players (season_id, identity_id);
CREATE UNIQUE INDEX IF NOT EXISTS tournament_entries_event_identity_key ON tournament_entries (tournament_slug, identity_id);

CREATE TABLE IF NOT EXISTS player_identity_separations (
    left_id BIGINT NOT NULL,
    right_id BIGINT NOT NULL CHECK (left_id < right_id),
    PRIMARY KEY (left_id, right_id)
);

CREATE TABLE IF NOT EXISTS player_identity_decisions (
    id BIGSERIAL PRIMARY KEY,
    left_id BIGINT NOT NULL,
    right_id BIGINT NOT NULL,
    action TEXT NOT NULL CHECK (action IN ('link', 'separate')),
    actor TEXT NOT NULL,
    decided_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
