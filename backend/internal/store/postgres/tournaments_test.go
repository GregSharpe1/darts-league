package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/tournament"
)

func TestTournamentRegistrationPostgres(t *testing.T) {
	store := approvalPG(t)
	ctx := context.Background()
	service := tournament.Service{Store: store, Now: time.Now}

	for _, slug := range []string{"city-cup", "winter-cup"} {
		event, err := service.Create(ctx, slug, "Tournament", "2027-02-06", tournament.DefaultMatchFormat())
		if err != nil || event.StartsOn != "2027-02-06" || event.RegistrationOpen || event.MatchFormat != tournament.DefaultMatchFormat() {
			t.Fatalf("create tournament returned %+v, %v", event, err)
		}
		if _, err := store.SetTournamentRegistration(ctx, slug, true); err != nil {
			t.Fatal(err)
		}
	}
	events, err := store.ListTournaments(ctx)
	if err != nil || len(events) != 2 || events[0].Slug != "city-cup" {
		t.Fatalf("list tournaments returned %+v, %v", events, err)
	}

	first, err := service.Register(ctx, "city-cup", "Maya Evans", "Flight")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Register(ctx, "city-cup", "maya  EVANS", ""); !errors.Is(err, league.ErrDuplicatePlayerName) {
		t.Fatalf("duplicate registration returned %v", err)
	}
	if _, err := service.Register(ctx, "winter-cup", "maya  EVANS", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteTournamentEntry(ctx, "winter-cup", first.ID); !errors.Is(err, tournament.ErrEntryNotFound) {
		t.Fatalf("cross-event removal returned %v", err)
	}

	// Hold an admin closure open to check that a signup cannot read the old state
	// and commit independently while registration is being closed.
	tx, err := store.pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	closingStore := &Store{pool: tx}
	if _, err := closingStore.SetTournamentRegistration(ctx, "city-cup", false); err != nil {
		t.Fatal(err)
	}
	wait, cancel := context.WithTimeout(ctx, 500*time.Millisecond)
	_, err = service.Register(wait, "city-cup", "Late Entrant", "")
	cancel()
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("signup should wait for closure to commit, got %v", err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Register(ctx, "city-cup", "Late Entrant", ""); !errors.Is(err, league.ErrRegistrationClosed) {
		t.Fatalf("late signup returned %v", err)
	}

	entries, err := store.ListTournamentEntries(ctx, "city-cup")
	if err != nil || len(entries) != 1 || entries[0].ID != first.ID || entries[0].Nickname != "Flight" {
		t.Fatalf("persisted roster returned %+v, %v", entries, err)
	}
	if err := store.DeleteTournament(ctx, "city-cup"); !errors.Is(err, tournament.ErrDeleteLocked) {
		t.Fatalf("closed tournament deletion returned %v", err)
	}
	if err := store.DeleteTournamentEntry(ctx, "city-cup", first.ID); err != nil {
		t.Fatal(err)
	}
	entries, err = store.ListTournamentEntries(ctx, "city-cup")
	if err != nil || entries == nil || len(entries) != 0 {
		t.Fatalf("expected empty roster after removal, got %+v, %v", entries, err)
	}

	if _, err := service.Register(ctx, "winter-cup", "Second Player", ""); err != nil {
		t.Fatal(err)
	}
	if _, err := store.StartTournament(ctx, "winter-cup"); !errors.Is(err, tournament.ErrCloseBeforeStart) {
		t.Fatalf("start while registration is open returned %v", err)
	}
	if _, err := store.SetTournamentRegistration(ctx, "winter-cup", false); err != nil {
		t.Fatal(err)
	}
	format := tournament.MatchFormat{GameVariant: league.GameVariant301, LegsToWin: 5}
	if _, err := service.UpdateFormat(ctx, "winter-cup", format); err != nil {
		t.Fatal(err)
	}
	started, err := store.StartTournament(ctx, "winter-cup")
	if err != nil || len(started.Bracket) != 1 || len(started.Bracket[0]) != 1 || started.MatchFormat != format {
		t.Fatalf("start returned %+v, %v", started, err)
	}
	reloaded, err := store.StartTournament(ctx, "winter-cup")
	if err != nil || len(reloaded.Bracket) != 1 || reloaded.Bracket[0][0] != started.Bracket[0][0] {
		t.Fatalf("repeat start returned %+v, %v", reloaded, err)
	}
	if _, err := store.SetTournamentRegistration(ctx, "winter-cup", true); !errors.Is(err, tournament.ErrTournamentStarted) {
		t.Fatalf("reopen after start returned %v", err)
	}
	if _, err := service.UpdateFormat(ctx, "winter-cup", tournament.DefaultMatchFormat()); !errors.Is(err, tournament.ErrFormatLocked) {
		t.Fatalf("edit format after start returned %v", err)
	}
	if err := store.DeleteTournament(ctx, "winter-cup"); !errors.Is(err, tournament.ErrDeleteLocked) {
		t.Fatalf("started tournament deletion returned %v", err)
	}
	if _, err := store.SetTournamentRegistration(ctx, "city-cup", true); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Register(ctx, "city-cup", "Maya Evans", ""); err != nil {
		t.Fatal(err)
	}
	if err := store.DeleteTournament(ctx, "city-cup"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.GetTournament(ctx, "city-cup"); !errors.Is(err, tournament.ErrNotFound) {
		t.Fatalf("deleted tournament lookup returned %v", err)
	}
	if _, err := service.Create(ctx, "city-cup", "New City Cup", "2027-02-06", tournament.DefaultMatchFormat()); err != nil {
		t.Fatal(err)
	}
	entries, err = store.ListTournamentEntries(ctx, "city-cup")
	if err != nil || len(entries) != 0 {
		t.Fatalf("recreated tournament inherited entries: %+v, %v", entries, err)
	}
}

func TestTournamentFormatMigrationKeepsExistingBracket(t *testing.T) {
	store := approvalPG(t)
	ctx := context.Background()
	if _, err := store.pool.Exec(ctx, `
		ALTER TABLE tournaments DROP COLUMN game_variant, DROP COLUMN legs_to_win;
		INSERT INTO tournaments (slug, name, starts_on, bracket) VALUES (
			'old-cup', 'Old Cup', '2027-02-06',
			'[[{"number":1,"slots":[{"player":"Alice"},{"player":"Bob"}]}]]'
		);
		INSERT INTO tournament_entries (tournament_slug, display_name, display_name_normalized, nickname, registered_at)
		VALUES ('old-cup', 'Alice', 'alice', '', NOW());`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := store.migrateImports(ctx); err != nil {
			t.Fatal(err)
		}
	}
	event, err := store.GetTournament(ctx, "old-cup")
	if err != nil || event.MatchFormat != tournament.DefaultMatchFormat() || len(event.Bracket) != 1 ||
		len(event.Bracket[0]) != 1 || event.Bracket[0][0].Slots[0].Player != "Alice" {
		t.Fatalf("upgraded tournament returned %+v, %v", event, err)
	}
	entries, err := store.ListTournamentEntries(ctx, "old-cup")
	if err != nil || len(entries) != 1 || entries[0].DisplayName != "Alice" {
		t.Fatalf("upgraded roster returned %+v, %v", entries, err)
	}
}
