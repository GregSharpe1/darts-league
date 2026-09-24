package postgres

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

func completedFixtures(t *testing.T) (*Store, int64, []league.Fixture) {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	store, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	resetTables(t, ctx, store)
	season, err := store.EnsureActiveSeason(ctx, league.NewSeason("Lifecycle"))
	if err != nil {
		t.Fatal(err)
	}
	service := league.NewSeasonService(store)
	registration := league.NewRegistrationService(store)
	divisions, err := service.ProvisionDivisions(ctx, 2)
	if err != nil {
		t.Fatal(err)
	}
	for i, name := range []string{"Alice", "Bob", "Charlie", "Deb"} {
		player, err := registration.RegisterPlayer(ctx, league.Player{DisplayName: name})
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
	fixtures, err := store.ListFixturesBySeason(ctx, season.ID)
	if err != nil {
		t.Fatal(err)
	}
	results := league.NewResultService(store)
	for _, fixture := range fixtures {
		if _, err := results.RecordResult(ctx, fixture.ID, 3, 1, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := results.EditResult(ctx, fixtures[0].ID, 3, 2, nil, nil, "admin"); err != nil {
		t.Fatal(err)
	}
	return store, season.ID, fixtures
}

func TestLifecyclePersistsAfterReopenAndRollover(t *testing.T) {
	store, id, fixtures := completedFixtures(t)
	ctx := context.Background()
	if err := store.CloseSeason(ctx, id); err != nil {
		t.Fatal(err)
	}
	reopened, err := Open(ctx, os.Getenv("TEST_DATABASE_URL"))
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	current, err := reopened.EnsureActiveSeason(ctx, league.NewSeason("Must not replace"))
	if err != nil || current.ID != id || current.Status != league.SeasonStatusCompleted {
		t.Fatalf("bootstrap: %+v %v", current, err)
	}
	for _, rollover := range []bool{false, true} {
		if rollover {
			if err := reopened.CreateNextSeason(ctx, id, league.NewSeason("Next")); err != nil {
				t.Fatal(err)
			}
		}
		result, err := reopened.GetResultByFixture(ctx, fixtures[0].ID)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := reopened.CreateResult(ctx, result); !errors.Is(err, league.ErrSeasonCompleted) {
			t.Fatalf("create: %v", err)
		}
		if _, err := reopened.UpdateResult(ctx, result); !errors.Is(err, league.ErrSeasonCompleted) {
			t.Fatalf("update: %v", err)
		}
		if err := reopened.DeleteResultByFixture(ctx, fixtures[0].ID); !errors.Is(err, league.ErrSeasonCompleted) {
			t.Fatalf("delete: %v", err)
		}
		if _, err := reopened.ReplaceDivisions(ctx, id, nil); !errors.Is(err, league.ErrSeasonCompleted) {
			t.Fatalf("divisions: %v", err)
		}
		if _, err := reopened.ReplaceFixturesBySeason(ctx, id, nil); !errors.Is(err, league.ErrSeasonCompleted) {
			t.Fatalf("fixtures: %v", err)
		}
		current.Status = league.SeasonStatusStarted
		if _, err := reopened.UpsertSeason(ctx, current); !errors.Is(err, league.ErrSeasonCompleted) {
			t.Fatalf("stale update: %v", err)
		}
	}
	registration := league.NewRegistrationService(reopened)
	if _, err := registration.RegisterPlayer(ctx, league.Player{DisplayName: "Alice"}); err != nil {
		t.Fatal(err)
	}
	if _, err := registration.RegisterPlayer(ctx, league.Player{DisplayName: "ALICE"}); !errors.Is(err, league.ErrDuplicatePlayerName) {
		t.Fatalf("duplicate: %v", err)
	}
	next, err := reopened.GetActiveSeason(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.CreatePlayer(ctx, league.Player{SeasonID: next.ID, DisplayName: "ALICE", RegisteredAt: time.Now()}); !errors.Is(err, league.ErrDuplicatePlayerName) {
		t.Fatalf("database duplicate constraint: %v", err)
	}
	results, err := reopened.ListResultsBySeason(ctx, id)
	if err != nil || len(results) != 2 {
		t.Fatalf("retention: %+v %v", results, err)
	}
	audits, err := reopened.ListAuditLogsBySeason(ctx, id)
	if err != nil || len(audits) != 1 {
		t.Fatalf("audits: %+v %v", audits, err)
	}
}

func TestConcurrentNextSeasonCreatesOneSuccessor(t *testing.T) {
	store, id, _ := completedFixtures(t)
	ctx := context.Background()
	if err := store.CloseSeason(ctx, id); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	done := make(chan error, 2)
	for range 2 {
		go func() { <-start; done <- store.CreateNextSeason(ctx, id, league.NewSeason("Next")) }()
	}
	close(start)
	succeeded := 0
	for range 2 {
		err := <-done
		if err == nil {
			succeeded++
		} else if !errors.Is(err, league.ErrSeasonTransition) {
			t.Fatal(err)
		}
	}
	if succeeded != 1 {
		t.Fatalf("created %d successors", succeeded)
	}
}

func TestCloseSerializesWithResultDeletion(t *testing.T) {
	store, id, fixtures := completedFixtures(t)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	tx, err := store.beginResultWrite(ctx, fixtures[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	done := make(chan error, 1)
	go func() { done <- store.CloseSeason(ctx, id) }()
	if _, err := tx.Exec(ctx, `DELETE FROM results WHERE fixture_id = $1`, fixtures[0].ID); err != nil {
		t.Fatal(err)
	}
	if err := tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err := <-done; !errors.Is(err, league.ErrSeasonIncomplete) {
		t.Fatalf("close with missing result: %v", err)
	}
}
