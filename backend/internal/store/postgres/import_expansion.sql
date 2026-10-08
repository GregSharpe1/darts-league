ALTER TABLE pending_results ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'autodarts';
ALTER TABLE pending_results ADD COLUMN IF NOT EXISTS digest TEXT;
ALTER TABLE pending_results ADD COLUMN IF NOT EXISTS source_payload JSONB;
ALTER TABLE pending_results ADD COLUMN IF NOT EXISTS played_at_original TEXT;
ALTER TABLE pending_results ADD COLUMN IF NOT EXISTS settings_evidence TEXT NOT NULL DEFAULT 'legacy_unverified';
ALTER TABLE pending_results ADD COLUMN IF NOT EXISTS review_reason TEXT NOT NULL DEFAULT '';
ALTER TABLE pending_results ADD COLUMN IF NOT EXISTS changed_import BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE pending_results ADD COLUMN IF NOT EXISTS season_id BIGINT REFERENCES seasons(id);
ALTER TABLE pending_results ADD COLUMN IF NOT EXISTS fixture_id BIGINT REFERENCES fixtures(id);
ALTER TABLE pending_results ADD COLUMN IF NOT EXISTS result_id BIGINT REFERENCES results(id) ON DELETE SET NULL;
ALTER TABLE pending_results ADD COLUMN IF NOT EXISTS source_active BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE pending_results ADD COLUMN IF NOT EXISTS approval_metadata JSONB;
ALTER TABLE admin_audit_log ADD COLUMN IF NOT EXISTS season_id BIGINT REFERENCES seasons(id);
ALTER TABLE admin_audit_log ADD COLUMN IF NOT EXISTS import_metadata JSONB;

CREATE TABLE IF NOT EXISTS import_players (
    pending_result_id BIGINT NOT NULL REFERENCES pending_results(id),
    source_player_id TEXT NOT NULL,
    source_order INTEGER NOT NULL CHECK (source_order IN (0,1)),
    account_id TEXT,
    display_name TEXT NOT NULL,
    legs_won INTEGER NOT NULL,
    match_average DOUBLE PRECISION,
    points_scored INTEGER,
    darts_thrown INTEGER,
    checkout_hits INTEGER,
    checkout_attempts INTEGER,
    league_player_id BIGINT REFERENCES players(id),
    fixture_slot INTEGER CHECK (fixture_slot IN (1,2)),
    PRIMARY KEY (pending_result_id,source_player_id),
    UNIQUE (pending_result_id,source_order),
    UNIQUE (pending_result_id,fixture_slot)
);
