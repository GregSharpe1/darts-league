CREATE TABLE IF NOT EXISTS pending_results (
    id BIGSERIAL PRIMARY KEY,
    external_match_id TEXT UNIQUE,
    player_one_name TEXT NOT NULL,
    player_one_legs INTEGER NOT NULL,
    player_one_average DOUBLE PRECISION,
    player_two_name TEXT NOT NULL,
    player_two_legs INTEGER NOT NULL,
    player_two_average DOUBLE PRECISION,
    status TEXT NOT NULL DEFAULT 'pending',
    received_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    confirmed_at TIMESTAMPTZ,
    confirmed_by TEXT
);

CREATE INDEX IF NOT EXISTS pending_results_status_idx ON pending_results (status, received_at);
