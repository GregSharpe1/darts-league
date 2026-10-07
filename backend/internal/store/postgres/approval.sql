ALTER TABLE pending_results ADD COLUMN IF NOT EXISTS approval_metadata JSONB;
ALTER TABLE admin_audit_log ALTER COLUMN fixture_id DROP NOT NULL;
ALTER TABLE admin_audit_log ADD COLUMN IF NOT EXISTS season_id BIGINT REFERENCES seasons(id);
ALTER TABLE admin_audit_log ADD COLUMN IF NOT EXISTS import_metadata JSONB;
CREATE INDEX IF NOT EXISTS pending_results_fixture_lookup ON pending_results(fixture_id) WHERE source_active;

CREATE OR REPLACE FUNCTION protect_import_target() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
    IF OLD.fixture_id IS NOT NULL AND ROW(OLD.season_id,OLD.fixture_id) IS DISTINCT FROM ROW(NEW.season_id,NEW.fixture_id) THEN
        RAISE EXCEPTION 'confirmed import target is immutable';
    END IF;
    RETURN NEW;
END $$;
CREATE OR REPLACE TRIGGER pending_results_immutable_target
BEFORE UPDATE ON pending_results FOR EACH ROW EXECUTE FUNCTION protect_import_target();
