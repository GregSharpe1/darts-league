ALTER TABLE pending_results DROP CONSTRAINT IF EXISTS pending_results_external_match_id_key;
CREATE UNIQUE INDEX IF NOT EXISTS pending_results_content_key ON pending_results(source,external_match_id,digest);
CREATE UNIQUE INDEX IF NOT EXISTS pending_results_active_result_key ON pending_results(result_id) WHERE source_active;

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
