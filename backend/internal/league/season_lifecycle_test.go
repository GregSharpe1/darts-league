package league

import (
	"context"
	"errors"
	"reflect"
	"testing"
	"time"
)

func lifecycleSeason(t *testing.T) (*MemoryStore, SeasonService, []Fixture) {
	t.Helper()
	ctx := context.Background()
	store := NewMemoryStore()
	clock := func() time.Time { return time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC) }
	service := NewSeasonServiceWithNow(store, clock)
	registration := NewRegistrationServiceWithNow(store, clock)
	divisions, err := service.ProvisionDivisions(ctx, 3)
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"Alice", "Bob", "Charlie", "Deb"} {
		player, err := registration.RegisterPlayer(ctx, Player{DisplayName: name})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := registration.AssignPlayer(ctx, player.ID, &divisions[i/2].ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := service.StartSeason(ctx); err != nil {
		t.Fatal(err)
	}
	fixtures, err := store.ListFixturesBySeason(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	return store, service, fixtures
}

func TestCloseRequiresEveryDivisionResult(t *testing.T) {
	store, service, fixtures := lifecycleSeason(t)
	ctx := context.Background()
	results := NewResultService(store)
	if _, err := results.RecordResult(ctx, fixtures[0].ID, 3, 1, nil, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CloseSeason(ctx, 1); !errors.Is(err, ErrSeasonIncomplete) {
		t.Fatalf("got %v", err)
	}
	summary, err := service.Summary(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if summary.CanCloseSeason || summary.RemainingFixtures != 1 {
		t.Fatalf("%+v", summary)
	}
}

func TestClosedSeasonRetainsAndFreezesResultsAcrossRollover(t *testing.T) {
	store, service, fixtures := lifecycleSeason(t)
	ctx := context.Background()
	results := NewResultService(store)
	for _, fixture := range fixtures {
		if _, err := results.RecordResult(ctx, fixture.ID, 3, 1, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := results.EditResult(ctx, fixtures[0].ID, 3, 2, nil, nil, "admin"); err != nil {
		t.Fatal(err)
	}
	before, err := results.Standings(ctx, "division-1")
	if err != nil {
		t.Fatal(err)
	}
	summary, err := service.CloseSeason(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	if summary.Status != SeasonStatusCompleted || !summary.AdminLocked || !summary.CanCreateNextSeason || summary.CanCloseSeason {
		t.Fatalf("%+v", summary)
	}
	after, err := results.Standings(ctx, "division-1")
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatalf("standings changed: %v", err)
	}
	if _, err := service.CloseSeason(ctx, 1); !errors.Is(err, ErrSeasonTransition) {
		t.Fatalf("repeat close: %v", err)
	}
	if _, err := service.UpdateName(ctx, "Changed"); !errors.Is(err, ErrSeasonRenameLocked) {
		t.Fatalf("rename: %v", err)
	}
	if _, err := service.ProvisionDivisions(ctx, 1); !errors.Is(err, ErrDivisionSlugLocked) {
		t.Fatalf("divisions: %v", err)
	}
	if _, err := service.StartSeason(ctx); !errors.Is(err, ErrSeasonAlreadyStarted) {
		t.Fatalf("start: %v", err)
	}
	if _, err := NewRegistrationService(store).RegisterPlayer(ctx, Player{DisplayName: "New"}); !errors.Is(err, ErrRegistrationClosed) {
		t.Fatalf("registration: %v", err)
	}
	for _, rollover := range []bool{false, true} {
		if rollover {
			next, err := service.CreateNextSeason(ctx, 1, "Next season")
			if err != nil {
				t.Fatal(err)
			}
			if next.ID == 1 || !next.RegistrationOpen || next.TotalFixtures != 0 || next.PlayerCount != 0 || next.DivisionCount != 0 {
				t.Fatalf("%+v", next)
			}
		}
		if _, err := results.EditResult(ctx, fixtures[0].ID, 3, 0, nil, nil, "admin"); !errors.Is(err, ErrSeasonCompleted) {
			t.Fatalf("edit: %v", err)
		}
		if err := results.DeleteResult(ctx, fixtures[0].ID, "admin"); !errors.Is(err, ErrSeasonCompleted) {
			t.Fatalf("delete: %v", err)
		}
		if _, err := store.CreateResult(ctx, Result{FixtureID: fixtures[0].ID}); !errors.Is(err, ErrSeasonCompleted) {
			t.Fatalf("create: %v", err)
		}
	}
	if _, err := service.CreateNextSeason(ctx, 1, "Duplicate"); !errors.Is(err, ErrSeasonTransition) {
		t.Fatalf("stale next: %v", err)
	}
	if _, err := NewRegistrationService(store).RegisterPlayer(ctx, Player{DisplayName: "Alice"}); err != nil {
		t.Fatal(err)
	}
	audits, err := store.ListAuditLogsBySeason(ctx, 1)
	if err != nil || len(audits) != 1 {
		t.Fatalf("audits changed: %v %+v", err, audits)
	}
	retained, err := store.ListResultsBySeason(ctx, 1)
	if err != nil || len(retained) != 2 {
		t.Fatalf("results lost: %v", err)
	}
}

func TestCloseRejectsPrestartEmptyAndStaleSeasons(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	service := NewSeasonService(store)
	if _, err := service.CloseSeason(ctx, 1); !errors.Is(err, ErrSeasonTransition) {
		t.Fatalf("prestart: %v", err)
	}
	if _, err := store.UpsertSeason(ctx, store.activeSeason.Start(time.Now())); err != nil {
		t.Fatal(err)
	}
	if _, err := service.CloseSeason(ctx, 1); !errors.Is(err, ErrSeasonIncomplete) {
		t.Fatalf("empty: %v", err)
	}
	if _, err := service.CloseSeason(ctx, 99); !errors.Is(err, ErrSeasonTransition) {
		t.Fatalf("stale: %v", err)
	}
}
