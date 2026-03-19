package league

import (
	"context"
	"testing"
	"time"
)

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
