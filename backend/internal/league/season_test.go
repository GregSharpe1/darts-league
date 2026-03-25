package league

import (
	"context"
	"testing"
	"time"
)

func TestUpdateConfigBeforeSeasonStart(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	now := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	registration := NewRegistrationServiceWithNow(store, func() time.Time { return now })
	seasonService := NewSeasonServiceWithNow(store, func() time.Time { return now })
	ctx := context.Background()

	for _, player := range []Player{{DisplayName: "Alice"}, {DisplayName: "Bob"}, {DisplayName: "Charlie"}} {
		if _, err := registration.RegisterPlayer(ctx, player); err != nil {
			t.Fatalf("expected registration to succeed, got %v", err)
		}
	}

	summary, err := seasonService.UpdateConfig(ctx, GameVariant301, 5, 2)
	if err != nil {
		t.Fatalf("expected config update to succeed, got %v", err)
	}
	if summary.GameVariant != GameVariant301 || summary.LegsToWin != 5 || summary.GamesPerWeek != 2 {
		t.Fatalf("expected config to be updated, got %+v", summary)
	}
}

func TestUpdateConfigLockedAfterStart(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	now := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	registration := NewRegistrationServiceWithNow(store, func() time.Time { return now })
	seasonService := NewSeasonServiceWithNow(store, func() time.Time { return now })
	ctx := context.Background()

	for _, player := range []Player{{DisplayName: "Alice"}, {DisplayName: "Bob"}} {
		if _, err := registration.RegisterPlayer(ctx, player); err != nil {
			t.Fatalf("expected registration to succeed, got %v", err)
		}
	}

	if _, err := seasonService.StartSeason(ctx); err != nil {
		t.Fatalf("expected season start to succeed, got %v", err)
	}

	_, err := seasonService.UpdateConfig(ctx, GameVariant301, 5, 1)
	if err != ErrSeasonConfigLocked {
		t.Fatalf("expected config locked error, got %v", err)
	}
}

func TestStartSeasonValidatesConfigAtStartTime(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	now := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	registration := NewRegistrationServiceWithNow(store, func() time.Time { return now })
	seasonService := NewSeasonServiceWithNow(store, func() time.Time { return now })
	ctx := context.Background()

	for _, player := range []Player{{DisplayName: "Alice"}, {DisplayName: "Bob"}} {
		if _, err := registration.RegisterPlayer(ctx, player); err != nil {
			t.Fatalf("expected registration to succeed, got %v", err)
		}
	}

	// Manually set an invalid game variant on the season.
	season, _ := store.GetActiveSeason(ctx)
	season.GameVariant = "999"
	store.UpsertSeason(ctx, season)

	_, err := seasonService.StartSeason(ctx)
	if err != ErrInvalidGameVariant {
		t.Fatalf("expected invalid game variant error at start time, got %v", err)
	}
}

func TestStartSeasonDoesNotPartiallyUpdateWhenFixtureGenerationFails(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	now := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	registration := NewRegistrationServiceWithNow(store, func() time.Time { return now })
	seasonService := NewSeasonServiceWithNow(store, func() time.Time { return now })
	ctx := context.Background()

	season, err := store.GetActiveSeason(ctx)
	if err != nil {
		t.Fatalf("expected active season, got %v", err)
	}
	season.Timezone = "Mars/Phobos"
	if _, err := store.UpsertSeason(ctx, season); err != nil {
		t.Fatalf("expected season timezone update, got %v", err)
	}

	for _, player := range []Player{{DisplayName: "Luke Humphries"}, {DisplayName: "Michael Smith", Nickname: "Bully Boy"}} {
		if _, err := registration.RegisterPlayer(ctx, player); err != nil {
			t.Fatalf("expected player registration to succeed, got %v", err)
		}
	}

	if _, err := seasonService.StartSeason(ctx); err == nil {
		t.Fatal("expected season start to fail when timezone cannot be loaded")
	}

	seasonAfter, err := store.GetActiveSeason(ctx)
	if err != nil {
		t.Fatalf("expected active season after failed start, got %v", err)
	}
	if seasonAfter.Status != SeasonStatusRegistrationOpen {
		t.Fatalf("expected season to remain registration_open, got %q", seasonAfter.Status)
	}
	fixtures, err := store.ListFixturesBySeason(ctx, seasonAfter.ID)
	if err != nil {
		t.Fatalf("expected fixtures query to succeed, got %v", err)
	}
	if len(fixtures) != 0 {
		t.Fatalf("expected no fixtures after failed start, got %d", len(fixtures))
	}
}
