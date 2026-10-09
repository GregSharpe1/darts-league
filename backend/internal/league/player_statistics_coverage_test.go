package league

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

func TestPlayerStatisticsUnknownAndZero(t *testing.T) {
	for _, tc := range []struct {
		name                     string
		hits, attempts, maximums *int
		percentage               *float64
	}{
		{name: "unknown"},
		{name: "zero attempts", hits: statisticsInt(0), attempts: statisticsInt(0), maximums: statisticsInt(0)},
		{name: "known zero hits", hits: statisticsInt(0), attempts: statisticsInt(4), maximums: statisticsInt(0), percentage: statisticsRatio(0, 1)},
		{name: "missing denominator", hits: statisticsInt(0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			// Given a confirmed summary with explicitly known or unknown source values.
			store, service := statisticsSetup(t)
			payload := statisticsPayload(false)
			payload.Detail = nil
			payload.Players[0].Stats = &autodarts.Stats{CheckoutHits: tc.hits, CheckoutAttempts: tc.attempts, Total180: tc.maximums}
			statisticsApprove(t, store, payload)
			// When the losing player's statistics are read.
			got, err := service.PublicPlayerStatistics(context.Background(), 1, 2)
			// Then absence stays distinct from a known zero and zero attempts has no ratio.
			if err != nil {
				t.Fatal(err)
			}
			if (got.CheckoutPercentage == nil) != (tc.percentage == nil) || (got.CheckoutPercentage != nil && *got.CheckoutPercentage != *tc.percentage) {
				t.Fatalf("percentage: %+v", got)
			}
			if (got.Total180 == nil) != (tc.maximums == nil) || (got.Total180 != nil && *got.Total180 != 0) {
				t.Fatalf("180: %+v", got)
			}
			known := 0
			if tc.hits != nil && tc.attempts != nil {
				known = 1
			}
			if got.Coverage.Checkout != known || got.Coverage.DartWeightedAverage != 0 || got.Coverage.MatchesWithDetail != 0 {
				t.Fatalf("coverage: %+v", got.Coverage)
			}
		})
	}
}

func statisticsInt(n int) *int { return &n }

func TestPlayerStatisticsPartialAndManual(t *testing.T) {
	// Given partial recorded detail, unknown geometry, and a manual summary with a known average.
	store, service := statisticsSetup(t)
	payload := statisticsPayload(false)
	payload.Detail.Coverage = "partial"
	payload.Detail.Legs = payload.Detail.Legs[:1]
	payload.Detail.Legs[0].Visits = payload.Detail.Legs[0].Visits[4:]
	visit := &payload.Detail.Legs[0].Visits[0]
	visit.Throws[0].EntryType = "manual"
	visit.Throws[0].Position = &autodarts.Position{X: 123, Y: -12, Provenance: "automatic"}
	statisticsApprove(t, store, payload)
	manual := statisticsApprove(t, store, statisticsPayload(false))
	average := 42.0
	if _, err := service.EditResult(context.Background(), manual.FixtureID, 3, 1, &average, nil, "manual"); err != nil {
		t.Fatal(err)
	}
	// When the public projection is read and serialized.
	got, err := service.PublicPlayerStatistics(context.Background(), 1, 1)
	if err != nil {
		t.Fatal(err)
	}
	body, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	// Then partial visits are not full totals, while real manual positions remain labelled and unplottable.
	if got.Played != 2 || got.Points != 4 || got.LegsFor != 6 || got.LegsAgainst != 1 || got.MatchAverageMean == nil || *got.MatchAverageMean != 42 {
		t.Fatalf("manual: %+v", got)
	}
	if got.DartWeightedAverage != nil || got.Total180 != nil || got.BestLegDarts != nil || got.HighestFinish == nil || *got.HighestFinish != 141 {
		t.Fatalf("partial: %+v", got)
	}
	if got.Coverage.MatchesWithDetail != 1 || got.Coverage.RecordedThrows != 3 || got.Coverage.KnownPositions != 1 || got.Coverage.PlottablePositions != 0 || *got.Coverage.PositionFraction != 1.0/3 {
		t.Fatalf("positions: %+v", got.Coverage)
	}
	if got.Throws[0].Position.Provenance != "manual" || got.Throws[0].Position.X != 123 || got.Throws[0].Plottable {
		t.Fatalf("position: %+v", got.Throws[0])
	}
	for _, forbidden := range []string{"Private A", "Private B", "account_id", "match_player_id", "external_match_id", "raw_payload"} {
		if strings.Contains(string(body), forbidden) {
			t.Fatalf("private evidence leaked: %s", forbidden)
		}
	}
	if got.History[0].OpponentName != "League Two" {
		t.Fatalf("league labels: %+v", got.History)
	}
}

func TestPlayerStatisticsBustDartsCountWithZeroPoints(t *testing.T) {
	// Given a complete leg with a bust that consumes three actual darts.
	store, service := statisticsSetup(t)
	payload := statisticsPayload(false)
	leg := &payload.Detail.Legs[0]
	last := leg.Visits[4]
	last.Number = 7
	leg.Visits = append(leg.Visits[:4], autodarts.Visit{Number: 5, PlayerID: "a", StartRemaining: 141, EndRemaining: 141, Bust: true, Throws: []autodarts.Throw{{Number: 1, Segment: autodarts.Segment{Bed: "triple", Number: 20}, EntryType: "automatic"}, {Number: 2, Segment: autodarts.Segment{Bed: "triple", Number: 20}, EntryType: "automatic"}, {Number: 3, Segment: autodarts.Segment{Bed: "single", Number: 20}, EntryType: "automatic"}}}, leg.Visits[3], last)
	leg.Visits[5].Number = 6
	statisticsApprove(t, store, payload)
	// When aggregating complete scoring evidence.
	got, err := service.PublicPlayerStatistics(context.Background(), 1, 1)
	// Then the bust's three darts count but its 140 apparent points do not.
	if err != nil {
		t.Fatal(err)
	}
	if got.DartWeightedAverage == nil || *got.DartWeightedAverage != 4509.0/30 {
		t.Fatalf("bust weighting: %+v", got)
	}
}

func TestPlayerStatisticsUnfinishedDetailDoesNotEstablishTotals(t *testing.T) {
	// Given an unfinished partial leg and unverified summary points/darts.
	store, service := statisticsSetup(t)
	payload := statisticsPayload(false)
	payload.Detail.Coverage = "partial"
	payload.Detail.Legs = payload.Detail.Legs[:1]
	leg := &payload.Detail.Legs[0]
	leg.Completed, leg.WinnerID = false, nil
	leg.Visits = leg.Visits[:2]
	payload.Players[1].Stats = &autodarts.Stats{PointsScored: statisticsInt(1503), DartsThrown: statisticsInt(27)}
	statisticsApprove(t, store, payload)
	// When reading the partial recording.
	got, err := service.PublicPlayerStatistics(context.Background(), 1, 1)
	// Then it supplies recorded throws but no complete totals or completed-leg extrema.
	if err != nil {
		t.Fatal(err)
	}
	if got.DartWeightedAverage != nil || got.BestLegDarts != nil || got.HighestFinish != nil || got.Total180 != nil || got.Coverage.RecordedThrows != 3 {
		t.Fatalf("unfinished: %+v", got)
	}
}
