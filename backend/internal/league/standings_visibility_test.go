package league

import (
	"context"
	"testing"
	"time"
)

func TestStandingsRevealBoundaryAcrossLondonDST(t *testing.T) {
	for _, scenario := range []struct {
		name   string
		start  time.Time
		reveal time.Time
	}{
		{"spring", time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC), time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)},
		{"autumn", time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC), time.Date(2026, 10, 26, 9, 0, 0, 0, time.UTC)},
	} {
		t.Run(scenario.name, func(t *testing.T) {
			ctx := context.Background()
			store := NewMemoryStore()
			now := scenario.start
			clock := func() time.Time { return now }
			seasons := NewSeasonServiceWithNow(store, clock)
			registration := NewRegistrationServiceWithNow(store, clock)
			divisions, err := seasons.ProvisionDivisions(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"Ember", "Vale"} {
				player, err := registration.RegisterPlayer(ctx, Player{DisplayName: name})
				if err != nil {
					t.Fatal(err)
				}
				if _, err := registration.AssignPlayer(ctx, player.ID, &divisions[0].ID); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := seasons.StartSeason(ctx); err != nil {
				t.Fatal(err)
			}
			fixtures, err := store.ListFixturesByDivision(ctx, divisions[0].ID)
			if err != nil || len(fixtures) != 1 {
				t.Fatalf("fixtures: %v %v", fixtures, err)
			}
			service := NewResultServiceWithNow(store, clock)
			average := 60.0
			if _, err := service.RecordResult(ctx, fixtures[0].ID, 3, 0, &average, nil); err != nil {
				t.Fatal(err)
			}
			now = scenario.reveal.Add(-time.Nanosecond)
			rows, err := service.Standings(ctx, divisions[0].Slug)
			if err != nil || len(rows) != 2 {
				t.Fatalf("standings: %v %v", rows, err)
			}
			for _, row := range rows {
				if row.Played != 0 || row.Points != 0 || row.Average != nil {
					t.Fatalf("locked result contributes: %+v", row)
				}
			}
			now = scenario.reveal
			rows, err = service.Standings(ctx, divisions[0].Slug)
			if err != nil || rows[0].Played != 1 || rows[0].Points != 2 || rows[0].Average == nil || *rows[0].Average != average {
				t.Fatalf("revealed result missing: %v %v", rows, err)
			}
		})
	}
}
