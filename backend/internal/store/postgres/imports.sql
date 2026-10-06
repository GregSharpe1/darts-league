ALTER TABLE pending_results DROP CONSTRAINT IF EXISTS pending_results_external_match_id_key;
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
CREATE UNIQUE INDEX IF NOT EXISTS pending_results_content_key ON pending_results(source,external_match_id,digest);
CREATE UNIQUE INDEX IF NOT EXISTS pending_results_active_result_key ON pending_results(result_id) WHERE source_active;

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

CREATE OR REPLACE FUNCTION protect_import_original() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.digest IS NOT NULL AND
       ROW(OLD.source,OLD.external_match_id,OLD.digest,OLD.source_payload,OLD.played_at_original,
           OLD.settings_evidence,OLD.received_at,OLD.player_one_name,OLD.player_one_legs,OLD.player_one_average,
           OLD.player_two_name,OLD.player_two_legs,OLD.player_two_average,OLD.review_reason)
       IS DISTINCT FROM
       ROW(NEW.source,NEW.external_match_id,NEW.digest,NEW.source_payload,NEW.played_at_original,
           NEW.settings_evidence,NEW.received_at,NEW.player_one_name,NEW.player_one_legs,NEW.player_one_average,
           NEW.player_two_name,NEW.player_two_legs,NEW.player_two_average,NEW.review_reason) THEN
        RAISE EXCEPTION 'import originals are immutable';
    END IF;
    RETURN NEW;
END $$;
CREATE OR REPLACE TRIGGER pending_results_immutable_original
BEFORE UPDATE ON pending_results FOR EACH ROW EXECUTE FUNCTION protect_import_original();
