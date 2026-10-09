package postgres

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestImportExpansionIsAdditive(t *testing.T) {
	for _, operation := range []string{"DROP", "CREATE TRIGGER", "CREATE OR REPLACE", "UPDATE PENDING_RESULTS", "DELETE FROM", "TRUNCATE"} {
		if strings.Contains(strings.ToUpper(importExpansionSchema), operation) {
			t.Fatalf("expand-only migration contains contract operation %q", operation)
		}
	}
}

func TestImportExpansionPreservesLegacyWritersAndRestart(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("requires disposable TEST_DATABASE_URL")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer admin.Close()
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatal(err)
	}
	name := "expand_" + hex.EncodeToString(nonce[:])
	schema := pgx.Identifier{name}.Sanitize()
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	}()
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	config.ConnConfig.RuntimeParams["search_path"] = name
	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if _, err := pool.Exec(ctx, initialSchema); err != nil {
		t.Fatal(err)
	}
	store := &Store{pool: pool}
	var oldID int64
	if err := pool.QueryRow(ctx, `INSERT INTO pending_results(external_match_id,player_one_name,player_one_legs,player_two_name,player_two_legs)
		VALUES('legacy-before','One',3,'Two',1) RETURNING id`).Scan(&oldID); err != nil {
		t.Fatal(err)
	}
	if err := store.migrateImportExpansion(ctx); err != nil {
		t.Fatal(err)
	}
	if err := store.migrateImportExpansion(ctx); err != nil {
		t.Fatal(err)
	}
	// The legacy conflict target and update semantics must remain available.
	_, err = pool.Exec(ctx, `INSERT INTO pending_results(external_match_id,player_one_name,player_one_legs,player_two_name,player_two_legs)
		VALUES('legacy-before','Corrected',3,'Two',2)
		ON CONFLICT(external_match_id) DO UPDATE SET player_one_name=EXCLUDED.player_one_name,player_two_legs=EXCLUDED.player_two_legs`)
	if err != nil {
		t.Fatal(err)
	}
	got, err := store.GetPendingResult(ctx, oldID)
	if err != nil || got.PlayerOneName != "Corrected" || got.PlayerTwoLegs != 2 {
		t.Fatalf("legacy write: %+v %v", got, err)
	}
	if _, err := pool.Exec(ctx, `INSERT INTO pending_results(external_match_id,player_one_name,player_one_legs,player_two_name,player_two_legs)
		VALUES('legacy-after','Three',3,'Four',0)`); err != nil {
		t.Fatal(err)
	}
	// Re-running the old schema simulates rolling the application back, not dropping new storage.
	if _, err := pool.Exec(ctx, initialSchema); err != nil {
		t.Fatal(err)
	}
	var summaries, inventedEvidence, triggers int
	if err := pool.QueryRow(ctx, `SELECT count(*),count(*) FILTER (WHERE digest IS NOT NULL OR source_payload IS NOT NULL OR source_active) FROM pending_results`).Scan(&summaries, &inventedEvidence); err != nil {
		t.Fatal(err)
	}
	if summaries != 2 || inventedEvidence != 0 {
		t.Fatalf("legacy data changed: rows=%d evidence=%d", summaries, inventedEvidence)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM pg_trigger WHERE tgrelid='pending_results'::regclass AND NOT tgisinternal`).Scan(&triggers); err != nil {
		t.Fatal(err)
	}
	if triggers != 0 {
		t.Fatalf("contract triggers installed early: %d", triggers)
	}
}
