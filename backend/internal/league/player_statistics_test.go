package league

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

func statisticsSetup(t *testing.T) (*MemoryStore, ResultService) {
	t.Helper()
	ctx := context.Background()
	store := NewMemoryStore()
	season, err := store.GetActiveSeason(ctx)
	if err != nil {
		t.Fatal(err)
	}
	season.Status = SeasonStatusStarted
	if _, err = store.UpsertSeason(ctx, season); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"League One", "League Two"} {
		if _, err = store.CreatePlayer(ctx, Player{SeasonID: season.ID, DisplayName: name}); err != nil {
			t.Fatal(err)
		}
	}
	return store, NewResultServiceWithNow(store, func() time.Time { return time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC) })
}

func statisticsPayload(slow bool) autodarts.Payload {
	detail := &autodarts.Detail{Coverage: "complete"}
	for n := 1; n <= 3; n++ {
		winner := "a"
		leg := autodarts.Leg{Number: n, Completed: true, WinnerID: &winner}
		remaining := 501
		beds := [][]autodarts.Segment{
			{{Bed: "triple", Number: 20}, {Bed: "triple", Number: 20}, {Bed: "triple", Number: 20}},
			{{Bed: "triple", Number: 20}, {Bed: "triple", Number: 20}, {Bed: "triple", Number: 20}},
			{{Bed: "triple", Number: 20}, {Bed: "triple", Number: 19}, {Bed: "double", Number: 12}},
		}
		if slow {
			beds = append([][]autodarts.Segment{{{Bed: "miss"}, {Bed: "miss"}, {Bed: "miss"}}}, beds...)
		}
		for i, segments := range beds {
			end := remaining - 180
			if slow && i == 0 {
				end = remaining
			}
			if i == len(beds)-1 {
				end = 0
			}
			visit := autodarts.Visit{Number: 2*i + 1, PlayerID: "a", StartRemaining: remaining, EndRemaining: end}
			for j, segment := range segments {
				visit.Throws = append(visit.Throws, autodarts.Throw{Number: j + 1, Segment: segment, EntryType: "automatic"})
			}
			leg.Visits = append(leg.Visits, visit)
			remaining = end
			if end > 0 {
				miss := autodarts.Visit{Number: 2*i + 2, PlayerID: "b", StartRemaining: 501, EndRemaining: 501}
				for j := 1; j <= 3; j++ {
					miss.Throws = append(miss.Throws, autodarts.Throw{Number: j, Segment: autodarts.Segment{Bed: "miss"}, EntryType: "unknown"})
				}
				leg.Visits = append(leg.Visits, miss)
			}
		}
		detail.Legs = append(detail.Legs, leg)
	}
	return autodarts.Payload{SchemaVersion: "autodarts.import.v1", Source: "autodarts", ExternalMatchID: "stats", Settings: autodarts.Settings{BaseScore: 501, LegsToWin: 3, Out: "double"}, Completed: true,
		Players: []autodarts.Player{{ID: "b", DisplayName: "Private B"}, {ID: "a", DisplayName: "Private A", LegsWon: 3}}, Detail: detail}
}

func statisticsApprove(t *testing.T, store *MemoryStore, payload autodarts.Payload) Result {
	t.Helper()
	ctx := context.Background()
	fixtures, err := store.ListFixturesBySeason(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	payload.ExternalMatchID = fmt.Sprintf("stats-%d", len(fixtures)+1)
	created, err := store.CreateFixtures(ctx, []Fixture{{SeasonID: 1, DivisionID: 1, PlayerOneID: 1, PlayerTwoID: 2, WeekNumber: 1, ScheduledAt: time.Date(2026, 3, 23, 9, 0, 0, 0, time.UTC), GameVariant: "501", LegsToWin: 3}})
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	pending := NewPendingResultService(store, NewResultService(store))
	outcome, err := pending.IngestPayload(ctx, body)
	if err != nil {
		t.Fatal(err)
	}
	result, err := pending.Approve(ctx, ApprovalRequest{PendingID: outcome.Pending.ID, SeasonID: 1, FixtureID: created[0].ID, Mapping: map[string]int64{"a": 1, "b": 2}, Actor: "reviewer", MissingDateReason: "unknown"})
	if err != nil {
		t.Fatalf("approve: %v; pending %+v", err, outcome.Pending)
	}
	return result
}

func TestPlayerStatisticsWeightedAndMean(t *testing.T) {
	// Given two validated complete matches with unequal dart counts and reversed source order.
	store, service := statisticsSetup(t)
	for i, slow := range []bool{false, true} {
		payload := statisticsPayload(slow)
		average, first := float64(60+i*30), float64(90+i*30)
		hits, attempts, maximums := 3, 3+i*3, i
		payload.Players[1].Stats = &autodarts.Stats{MatchAverage: &average, FirstNineAverage: &first, CheckoutHits: &hits, CheckoutAttempts: &attempts, Total180: &maximums}
		statisticsApprove(t, store, payload)
	}
	// When statistics are read through the real service/store.
	got, err := service.PublicPlayerStatistics(context.Background(), 1, 1)
	// Then means and ratios have their own denominators and evidence coverage.
	if err != nil {
		t.Fatal(err)
	}
	if got.MatchAverageMean == nil || *got.MatchAverageMean != 75 || got.DartWeightedAverage == nil || math.Abs(*got.DartWeightedAverage-9018.0/63) > 1e-9 {
		t.Fatalf("averages: %+v", got)
	}
	if got.FirstNineMatchAverageMean == nil || *got.FirstNineMatchAverageMean != 105 || got.CheckoutPercentage == nil || math.Abs(*got.CheckoutPercentage-600.0/9) > 1e-9 {
		t.Fatalf("ratios: %+v", got)
	}
	if got.Total180 == nil || *got.Total180 != 1 || got.BestLegDarts == nil || *got.BestLegDarts != 9 || got.HighestFinish == nil || *got.HighestFinish != 141 {
		t.Fatalf("extrema: %+v", got)
	}
	if got.Coverage.EligibleMatches != 2 || got.Coverage.DartWeightedAverage != 2 || got.Coverage.MatchesWithDetail != 2 || got.Played != 2 || got.Points != 4 {
		t.Fatalf("coverage: %+v", got)
	}
}
