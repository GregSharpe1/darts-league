package postgres

import (
	"context"
	"os"
	"testing"
)

func TestMigrationPreservesLegacySeason(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `CREATE SCHEMA legacy_migration_test; SET LOCAL search_path TO legacy_migration_test`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, initialSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
		ALTER TABLE players DROP CONSTRAINT players_season_id_display_name_normalized_key;
		ALTER TABLE players ADD CONSTRAINT players_display_name_normalized_key UNIQUE (display_name_normalized);
		ALTER TABLE players DROP COLUMN division_id, DROP COLUMN status;
		ALTER TABLE fixtures DROP COLUMN division_id;
		DROP TABLE divisions;
		INSERT INTO seasons (id, name, status, started_at) VALUES (1, 'Legacy', 'started', NOW());
		INSERT INTO players (id, season_id, display_name, display_name_normalized)
		VALUES (1, 1, 'Alice', 'alice'), (2, 1, 'Bob', 'bob');
		INSERT INTO fixtures (id, season_id, week_number, scheduled_at, player_one_id, player_two_id)
		VALUES (1, 1, 1, NOW(), 1, 2);
		INSERT INTO results (fixture_id, player_one_legs, player_two_legs, winner_id) VALUES (1, 3, 1, 1);
		INSERT INTO admin_audit_log (fixture_id, action, actor) VALUES (1, 'result_created', 'admin');
	`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := tx.Exec(ctx, initialSchema); err != nil {
			t.Fatal(err)
		}
	}
	var divisions, players, fixtures, results, audits int
	if err := tx.QueryRow(ctx, `SELECT
		(SELECT COUNT(*) FROM divisions WHERE season_id = 1 AND slug = 'division-1'),
		(SELECT COUNT(*) FROM players p JOIN divisions d ON d.id = p.division_id WHERE p.status = 'assigned'),
		(SELECT COUNT(*) FROM fixtures f JOIN divisions d ON d.id = f.division_id),
		(SELECT COUNT(*) FROM results WHERE fixture_id = 1 AND player_one_legs = 3),
		(SELECT COUNT(*) FROM admin_audit_log WHERE fixture_id = 1)
	`).Scan(&divisions, &players, &fixtures, &results, &audits); err != nil {
		t.Fatal(err)
	}
	if divisions != 1 || players != 2 || fixtures != 1 || results != 1 || audits != 1 {
		t.Fatalf("migration lost legacy data: divisions=%d players=%d fixtures=%d results=%d audits=%d", divisions, players, fixtures, results, audits)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO seasons (id, name, status) VALUES (2, 'Next', 'registration_open');
		INSERT INTO players (id, season_id, display_name, display_name_normalized) VALUES (3, 2, 'Alice', 'alice');
	`); err != nil {
		t.Fatalf("returning player after migration: %v", err)
	}
}
