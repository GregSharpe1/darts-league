package postgres

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"
)

func TestImportMigrationPreservesLegacySummary(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	s, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	external := "legacy-" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "-")
	var id int64
	err = s.pool.QueryRow(ctx, `INSERT INTO pending_results(external_match_id,player_one_name,player_one_legs,player_one_average,player_two_name,player_two_legs,status,confirmed_by)
		VALUES($1,'Original A',3,62.5,'Original B',1,'confirmed','existing-admin') RETURNING id`, external).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := s.migrateImports(ctx); err != nil {
			t.Fatal(err)
		}
	}
	record, err := s.GetImport(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if record.Pending.Status != "confirmed" || record.Pending.ConfirmedBy != "existing-admin" || record.Import.SettingsEvidence != "legacy_stored_summary" || record.Import.PlayedAt != nil || record.Import.Detail != nil || record.Import.Players[1].Stats != nil || record.FixtureID != nil || record.Active {
		t.Fatalf("invented or lost legacy evidence: %+v", record)
	}
	if record.Import.Digest == "" || record.Import.Players[0].Stats.MatchAverage == nil || *record.Import.Players[0].Stats.MatchAverage != 62.5 {
		t.Fatal("lost stored average")
	}
	var players int
	if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM import_players WHERE pending_result_id=$1`, id).Scan(&players); err != nil {
		t.Fatal(err)
	}
	if players != 2 {
		t.Fatal("migration duplicated/lost players")
	}
}

func TestImportSchemaIsIdempotent(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	s, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `CREATE SCHEMA import_schema_test;SET LOCAL search_path TO import_schema_test`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, initialSchema); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if _, err := tx.Exec(ctx, importSchema); err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM pg_indexes WHERE schemaname='import_schema_test' AND indexname='pending_results_content_key'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("missing content identity constraint")
	}
}
