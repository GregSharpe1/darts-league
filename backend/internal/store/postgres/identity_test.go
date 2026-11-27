package postgres

import (
	"context"
	"errors"
	"os"
	"slices"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/playerstats"
	"github.com/greg/darts-league/backend/internal/tournament"
)

func TestIdentityMigrationAndReview(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	store, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	scoped := &Store{pool: tx}
	if _, err := tx.Exec(ctx, `CREATE SCHEMA identity_migration_test; SET LOCAL search_path TO identity_migration_test`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, initialSchema); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO seasons (id, name, status) VALUES (1, 'Spring', 'completed');
		INSERT INTO players (id, season_id, display_name, display_name_normalized, nickname)
		VALUES (1, 1, 'Alex Morgan', 'alex morgan', 'Arrow'), (2, 1, 'Sam Patel', 'sam patel', 'Arrow');
		INSERT INTO tournaments (slug, name, starts_on) VALUES ('cup', 'City Cup', '2026-05-01');
		INSERT INTO tournament_entries (id, tournament_slug, display_name, display_name_normalized, nickname, registered_at)
		VALUES (1, 'cup', 'Alex Morgan', 'alex morgan', 'New nickname', NOW());
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, identitySchema); err != nil {
		t.Fatal(err)
	}
	snapshot, err := scoped.IdentitySnapshot(ctx)
	if err != nil || len(snapshot.Entries) != 3 {
		t.Fatalf("backfill: %+v %v", snapshot, err)
	}
	alex, sam, cup := snapshot.Entries[0].IdentityID, snapshot.Entries[1].IdentityID, snapshot.Entries[2].IdentityID
	if alex == sam || alex == cup || sam == cup || alex <= 0 || sam <= 0 || cup <= 0 {
		t.Fatalf("legacy entries need distinct permanent identities: %+v", snapshot.Entries)
	}
	if got := snapshot.Review().Suggestions; len(got) != 2 || got[0].CanLink || !got[1].CanLink {
		t.Fatalf("expected name match and blocked shared nickname in one competition: %+v", got)
	}
	decide := func(a, b int64, action string) error {
		return scoped.DecideIdentity(ctx, playerstats.IdentityDecision{
			IdentityPair: playerstats.IdentityPair{Left: a, Right: b}, Action: action, Actor: "league-admin",
		})
	}
	if err := decide(alex, sam, "link"); !errors.Is(err, playerstats.ErrIdentityOverlap) {
		t.Fatalf("expected same-season overlap, got %v", err)
	}
	if err := decide(sam, cup, "separate"); err != nil {
		t.Fatal(err)
	}
	if err := decide(alex, cup, "link"); err != nil {
		t.Fatal(err)
	}
	if _, err := tx.Exec(ctx, identitySchema); err != nil {
		t.Fatal(err)
	}
	snapshot, err = scoped.IdentitySnapshot(ctx)
	if err != nil || len(snapshot.Entries) != 3 || snapshot.Entries[0].IdentityID != alex || snapshot.Entries[1].Kind != "tournament" || snapshot.Entries[1].IdentityID != alex {
		t.Fatalf("persisted identities after migration rerun: %+v, %v", snapshot, err)
	}
	if len(snapshot.Review().Suggestions) != 0 {
		t.Fatalf("separation should carry across a link: %+v", snapshot.Review())
	}
	if _, err := tx.Exec(ctx, `INSERT INTO tournaments (slug, name, starts_on, registration_open) VALUES ('next-cup', 'Next Cup', '2027-05-01', TRUE)`); err != nil {
		t.Fatal(err)
	}
	// The legacy tournament entry used an explicit ID.
	if _, err := tx.Exec(ctx, `SELECT setval(pg_get_serial_sequence('tournament_entries', 'id'), 1)`); err != nil {
		t.Fatal(err)
	}
	entry, err := scoped.CreateTournamentEntry(ctx, "next-cup", tournament.Entry{
		DisplayName: "A different name", RegisteredAt: time.Now(), RequestedIdentityID: alex,
	})
	if err != nil {
		t.Fatal(err)
	}
	snapshot, err = scoped.IdentitySnapshot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	var newID int64
	for _, e := range snapshot.Entries {
		if e.Kind == "tournament" && e.EntryID == entry.ID {
			newID = e.IdentityID
		}
	}
	if newID <= 0 || newID == alex {
		t.Fatalf("returning registration must have its own ID pending review: %+v", snapshot.Entries)
	}
	suggestions := snapshot.Review().Suggestions
	if len(suggestions) != 1 || suggestions[0].Left != alex || suggestions[0].Right != newID ||
		!slices.Contains(suggestions[0].Signals, "Selected existing profile at sign-up") {
		t.Fatalf("the requested link must be available in identity review: %+v", suggestions)
	}
	var actions int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM player_identity_decisions WHERE actor = 'league-admin'`).Scan(&actions); err != nil || actions != 2 {
		t.Fatalf("expected two audited decisions, got %d: %v", actions, err)
	}
}
