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
		event, err := service.Create(ctx, slug, "Tournament", "2027-02-06")
		if err != nil || event.StartsOn != "2027-02-06" || event.RegistrationOpen {
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
	if err := store.DeleteTournamentEntry(ctx, "city-cup", first.ID); err != nil {
		t.Fatal(err)
	}
	entries, err = store.ListTournamentEntries(ctx, "city-cup")
	if err != nil || entries == nil || len(entries) != 0 {
		t.Fatalf("expected empty roster after removal, got %+v, %v", entries, err)
	}
}
