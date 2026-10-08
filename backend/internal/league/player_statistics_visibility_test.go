package league

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestPlayerStatisticsRevealDST(t *testing.T) {
	for _, tc := range []struct{ name, date, before, after string }{
		{"spring", "2026-03-30T09:00:00+01:00", "2026-03-30T07:59:59Z", "2026-03-30T08:00:00Z"},
		{"autumn", "2026-10-26T09:00:00Z", "2026-10-26T08:59:59Z", "2026-10-26T09:00:00Z"},
	} {
		for _, scope := range []string{"public before", "public at reveal", "admin before"} {
			t.Run(tc.name+"/"+scope, func(t *testing.T) {
				// Given one revealed manual result and one future imported result.
				ctx := context.Background()
				store, service := statisticsSetup(t)
				manual := statisticsApprove(t, store, statisticsPayload(false))
				if _, err := service.EditResult(ctx, manual.FixtureID, 3, 1, nil, nil, "manual"); err != nil {
					t.Fatal(err)
				}
				future := statisticsApprove(t, store, statisticsPayload(false))
				date, err := time.Parse(time.RFC3339, tc.date)
				if err != nil {
					t.Fatal(err)
				}
				store.mu.Lock()
				f := store.fixturesByID[future.FixtureID]
				f.WeekNumber, f.ScheduledAt = 2, date
				store.fixturesByID[f.ID] = f
				store.mu.Unlock()
				now, err := time.Parse(time.RFC3339, tc.before)
				if err != nil {
					t.Fatal(err)
				}
				if scope == "public at reveal" {
					now, err = time.Parse(time.RFC3339, tc.after)
					if err != nil {
						t.Fatal(err)
					}
				}
				service = NewResultServiceWithNow(store, func() time.Time { return now })
				// When using explicitly public or authenticated-admin intent.
				var got PlayerStatistics
				if scope == "admin before" {
					got, err = service.AdminPlayerStatistics(ctx, 1, 1)
				} else {
					got, err = service.PublicPlayerStatistics(ctx, 1, 1)
				}
				// Then every public metric, history entry and coordinate denominator obeys the same reveal boundary.
				if err != nil {
					t.Fatal(err)
				}
				wantPlayed, wantDetail, wantThrows := 2, 1, 27
				if scope == "public before" {
					wantPlayed, wantDetail, wantThrows = 1, 0, 0
				}
				if got.Coverage.EligibleMatches != wantPlayed || got.Played != wantPlayed || len(got.History) != wantPlayed || got.Coverage.MatchesWithDetail != wantDetail || got.Coverage.RecordedThrows != wantThrows || got.Coverage.DartWeightedAverage != wantDetail {
					t.Fatalf("reveal: %+v", got)
				}
			})
		}
	}
}

func TestPlayerStatisticsSeasonIsolation(t *testing.T) {
	// Given a completed old season and a returning name registered in the next season.
	ctx := context.Background()
	store, service := statisticsSetup(t)
	statisticsApprove(t, store, statisticsPayload(false))
	if err := store.CloseSeason(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateNextSeason(ctx, 1, NewSeason("Next")); err != nil {
		t.Fatal(err)
	}
	season, err := store.GetActiveSeason(ctx)
	if err != nil {
		t.Fatal(err)
	}
	returning, err := store.CreatePlayer(ctx, Player{SeasonID: season.ID, DisplayName: "League One"})
	if err != nil {
		t.Fatal(err)
	}
	// When querying each explicit season/player pair.
	old, oldErr := service.PublicPlayerStatistics(ctx, 1, 1)
	current, currentErr := service.PublicPlayerStatistics(ctx, season.ID, returning.ID)
	_, crossErr := service.PublicPlayerStatistics(ctx, season.ID, 1)
	// Then history stays readable without cross-season name/account matching.
	if oldErr != nil || currentErr != nil || !errors.Is(crossErr, ErrPlayerNotFound) {
		t.Fatalf("scope errors: %v %v %v", oldErr, currentErr, crossErr)
	}
	if old.Played != 1 || current.Played != 0 || current.Coverage.EligibleMatches != 0 {
		t.Fatalf("season isolation: old=%+v current=%+v", old, current)
	}
}

func TestPlayerStatisticsMissingPlayerAndCancellation(t *testing.T) {
	_, service := statisticsSetup(t)
	for _, ids := range [][2]int64{{0, 1}, {1, 0}, {999, 1}, {1, 999}} {
		if _, err := service.PublicPlayerStatistics(context.Background(), ids[0], ids[1]); !errors.Is(err, ErrPlayerNotFound) {
			t.Fatalf("missing: %v", err)
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := service.PublicPlayerStatistics(ctx, 1, 1); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled: %v", err)
	}
}
