ALTER TABLE players DROP CONSTRAINT IF EXISTS players_display_name_normalized_key;
CREATE UNIQUE INDEX IF NOT EXISTS players_season_id_display_name_normalized_key
    ON players (season_id, display_name_normalized);
