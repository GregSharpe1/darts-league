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

func TestUpdateConfigAllowedAfterStartBeforeFirstReveal(t *testing.T) {
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
	assignAllPlayersInSeasonTest(t, ctx, registration, seasonService)

	if _, err := seasonService.StartSeason(ctx); err != nil {
		t.Fatalf("expected season start to succeed, got %v", err)
	}

	summary, err := seasonService.UpdateConfig(ctx, GameVariant301, 5, 1)
	if err != nil {
		t.Fatalf("expected config update after start to succeed, got %v", err)
	}
	if summary.GameVariant != GameVariant301 || summary.LegsToWin != 5 {
		t.Fatalf("expected config to update after start before release, got %+v", summary)
	}
}

func TestUpdateConfigLockedAfterFirstReveal(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	startNow := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	registration := NewRegistrationServiceWithNow(store, func() time.Time { return startNow })
	seasonService := NewSeasonServiceWithNow(store, func() time.Time { return startNow })
	ctx := context.Background()

	for _, player := range []Player{{DisplayName: "Alice"}, {DisplayName: "Bob"}} {
		if _, err := registration.RegisterPlayer(ctx, player); err != nil {
			t.Fatalf("expected registration to succeed, got %v", err)
		}
	}
	assignAllPlayersInSeasonTest(t, ctx, registration, seasonService)

	if _, err := seasonService.StartSeason(ctx); err != nil {
		t.Fatalf("expected season start to succeed, got %v", err)
	}

	seasonService = NewSeasonServiceWithNow(store, func() time.Time {
		return time.Date(2026, time.March, 23, 10, 0, 0, 0, time.UTC)
	})

	_, err := seasonService.UpdateConfig(ctx, GameVariant301, 5, 1)
	if err != ErrSeasonConfigLocked {
		t.Fatalf("expected config locked error after first reveal, got %v", err)
	}
}

func TestAssignPlayerAfterStartBeforeFirstRevealRegeneratesFixtures(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	now := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	registration := NewRegistrationServiceWithNow(store, func() time.Time { return now })
	seasonService := NewSeasonServiceWithNow(store, func() time.Time { return now })
	ctx := context.Background()

	for _, player := range []Player{{DisplayName: "Alice"}, {DisplayName: "Bob"}, {DisplayName: "Charlie"}, {DisplayName: "Deb"}} {
		if _, err := registration.RegisterPlayer(ctx, player); err != nil {
			t.Fatalf("expected registration to succeed, got %v", err)
		}
	}
	divisions, err := seasonService.ProvisionDivisions(ctx, 2)
	if err != nil {
		t.Fatalf("expected divisions to be provisioned, got %v", err)
	}
	players, err := registration.ListPlayers(ctx)
	if err != nil {
		t.Fatalf("expected players to be listed, got %v", err)
	}
	for index, player := range players {
		divisionID := divisions[0].ID
		if index >= 2 {
			divisionID = divisions[1].ID
		}
		if _, err := registration.AssignPlayer(ctx, player.ID, &divisionID); err != nil {
			t.Fatalf("expected assignment to succeed, got %v", err)
		}
	}
	if _, err := seasonService.StartSeason(ctx); err != nil {
		t.Fatalf("expected season start to succeed, got %v", err)
	}
	beforeFixtures, err := store.ListFixturesBySeason(ctx, 1)
	if err != nil {
		t.Fatalf("expected fixtures to be listed, got %v", err)
	}
	if len(beforeFixtures) != 2 {
		t.Fatalf("expected one fixture per division before reassignment, got %d", len(beforeFixtures))
	}
	if _, err := registration.AssignPlayer(ctx, players[0].ID, &divisions[1].ID); err != nil {
		t.Fatalf("expected reassignment after start before reveal to succeed, got %v", err)
	}
	afterFixtures, err := store.ListFixturesBySeason(ctx, 1)
	if err != nil {
		t.Fatalf("expected fixtures to be listed after reassignment, got %v", err)
	}
	if len(afterFixtures) != 3 {
		t.Fatalf("expected fixtures to be regenerated for the three-player division, got %d", len(afterFixtures))
	}
	for _, fixture := range afterFixtures {
		if fixture.DivisionID != divisions[1].ID {
			t.Fatalf("expected all regenerated fixtures in second division, got division %d", fixture.DivisionID)
		}
	}
}

func TestProvisionDivisionsAfterStartBeforeFirstRevealResetsAssignments(t *testing.T) {
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
	assignAllPlayersInSeasonTest(t, ctx, registration, seasonService)
	if _, err := seasonService.StartSeason(ctx); err != nil {
		t.Fatalf("expected season start to succeed, got %v", err)
	}
	divisions, err := seasonService.ProvisionDivisions(ctx, 3)
	if err != nil {
		t.Fatalf("expected division reprovision to succeed before first reveal, got %v", err)
	}
	if len(divisions) != 3 {
		t.Fatalf("expected 3 divisions after reprovision, got %d", len(divisions))
	}
	players, err := registration.ListPlayers(ctx)
	if err != nil {
		t.Fatalf("expected players to be listed, got %v", err)
	}
	for _, player := range players {
		if player.DivisionID != nil || player.Status != PlayerStatusWaitlist {
			t.Fatalf("expected player to reset to waitlist after reprovision, got %+v", player)
		}
	}
}

func TestUpdateDivisionNameAfterStartBeforeFirstReveal(t *testing.T) {
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
	divisions, err := seasonService.ProvisionDivisions(ctx, 1)
	if err != nil {
		t.Fatalf("expected division provisioning to succeed, got %v", err)
	}
	players, err := registration.ListPlayers(ctx)
	if err != nil {
		t.Fatalf("expected players to be listed, got %v", err)
	}
	for _, player := range players {
		if _, err := registration.AssignPlayer(ctx, player.ID, &divisions[0].ID); err != nil {
			t.Fatalf("expected assignment to succeed, got %v", err)
		}
	}
	if _, err := seasonService.StartSeason(ctx); err != nil {
		t.Fatalf("expected season start to succeed, got %v", err)
	}
	updated, err := seasonService.UpdateDivision(ctx, divisions[0].ID, "Premier Division", divisions[0].Slug, "CHAN")
	if err != nil {
		t.Fatalf("expected division rename to succeed before first reveal, got %v", err)
	}
	if updated.Name != "Premier Division" {
		t.Fatalf("expected updated division name, got %q", updated.Name)
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
	assignAllPlayersInSeasonTest(t, ctx, registration, seasonService)

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
	assignAllPlayersInSeasonTest(t, ctx, registration, seasonService)

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

func assignAllPlayersInSeasonTest(t *testing.T, ctx context.Context, registration RegistrationService, seasonService SeasonService) {
	t.Helper()
	divisions, err := seasonService.ProvisionDivisions(ctx, 1)
	if err != nil {
		t.Fatalf("expected division provisioning to succeed, got %v", err)
	}
	players, err := registration.ListPlayers(ctx)
	if err != nil {
		t.Fatalf("expected players to be listed, got %v", err)
	}
	for _, player := range players {
		if _, err := registration.AssignPlayer(ctx, player.ID, &divisions[0].ID); err != nil {
			t.Fatalf("expected assignment to succeed, got %v", err)
		}
	}
}
