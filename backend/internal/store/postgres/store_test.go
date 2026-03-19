package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

func TestStoreSeasonPlayerAndResultFlow(t *testing.T) {
	databaseURL := os.Getenv("TEST_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}

	ctx := context.Background()
	store, err := Open(ctx, databaseURL)
	if err != nil {
		t.Fatalf("expected postgres store to open, got %v", err)
	}
	defer store.Close()
	resetTables(t, ctx, store)

	season, err := store.EnsureActiveSeason(ctx, league.NewSeason("Integration Season"))
	if err != nil {
		t.Fatalf("expected active season, got %v", err)
	}

	playerOne, err := store.CreatePlayer(ctx, league.Player{SeasonID: season.ID, DisplayName: "Luke Humphries", RegisteredAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("expected player one, got %v", err)
	}
	playerTwo, err := store.CreatePlayer(ctx, league.Player{SeasonID: season.ID, DisplayName: "Michael Smith", RegisteredAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("expected player two, got %v", err)
	}

	season = season.Start(time.Now().UTC())
	season, err = store.UpsertSeason(ctx, season)
	if err != nil {
		t.Fatalf("expected season update, got %v", err)
	}

	fixtures, err := store.CreateFixtures(ctx, []league.Fixture{{
		SeasonID: season.ID, WeekNumber: 1, ScheduledAt: time.Now().UTC(),
		PlayerOneID: playerOne.ID, PlayerTwoID: playerTwo.ID, GameVariant: league.GameVariant501, LegsToWin: league.LegsToWin, Status: "scheduled",
	}})
	if err != nil {
		t.Fatalf("expected fixtures insert, got %v", err)
	}

	result, err := store.CreateResult(ctx, league.Result{FixtureID: fixtures[0].ID, PlayerOneLegs: 3, PlayerTwoLegs: 1, WinnerID: playerOne.ID, EnteredAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatalf("expected result insert, got %v", err)
	}
	result.PlayerTwoLegs = 2
	result.UpdatedAt = time.Now().UTC()
	if _, err := store.UpdateResult(ctx, result); err != nil {
		t.Fatalf("expected result update, got %v", err)
	}
	if _, err := store.CreateAuditLog(ctx, league.AuditLogEntry{FixtureID: fixtures[0].ID, Action: "result_edited", Actor: "admin", OldResult: &league.ResultSnapshot{PlayerOneLegs: 3, PlayerTwoLegs: 1, WinnerID: playerOne.ID}, NewResult: &league.ResultSnapshot{PlayerOneLegs: 3, PlayerTwoLegs: 2, WinnerID: playerOne.ID}, CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("expected audit log insert, got %v", err)
	}

	entries, err := store.ListAuditLogsBySeason(ctx, season.ID)
	if err != nil {
		t.Fatalf("expected audit log list, got %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one audit entry, got %d", len(entries))
	}
}

func resetTables(t *testing.T, ctx context.Context, store *Store) {
	t.Helper()
	_, err := store.pool.Exec(ctx, `TRUNCATE admin_audit_log, results, fixtures, players, seasons RESTART IDENTITY CASCADE`)
	if err != nil {
		t.Fatalf("expected table reset, got %v", err)
	}
}
