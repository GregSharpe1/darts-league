package autodarts

import (
	"encoding/json"
	"os"
	"sort"
	"strconv"
	"testing"
)

// These are sanitized design projections, not ingestion DTOs. Adapt explicitly,
// joining identities and preserving coordinates without guessing source geometry.
func referencePayload(t *testing.T, name string) Payload {
	t.Helper()
	b, err := os.ReadFile("../../../docs/autodarts-fixtures/" + name + ".json")
	if err != nil {
		t.Fatal(err)
	}
	var ref struct {
		MatchID  string `json:"matchId"`
		WinnerID string `json:"winnerId"`
		Players  []struct {
			ID, Name string
			LegsWon  int
		} `json:"players"`
		MatchStats []struct {
			PlayerID                             string
			Average                              float64
			Checkouts, CheckoutsHit, DartsThrown int
		} `json:"matchStats"`
		Darts []struct {
			PlayerID, Segment string
			Leg, Visit, Dart  int
			X, Y              *float64
		} `json:"darts"`
	}
	if err := json.Unmarshal(b, &ref); err != nil {
		t.Fatal(err)
	}
	p := Payload{SchemaVersion: "autodarts.import.v1", Source: "autodarts", ExternalMatchID: ref.MatchID, Completed: true, Settings: Settings{501, 3, "double"}}
	for _, r := range ref.Players {
		player := Player{ID: r.ID, DisplayName: r.Name, LegsWon: r.LegsWon}
		for _, s := range ref.MatchStats {
			if s.PlayerID == r.ID {
				player.Stats = &Stats{MatchAverage: &s.Average, DartsThrown: &s.DartsThrown, CheckoutHits: &s.CheckoutsHit, CheckoutAttempts: &s.Checkouts}
			}
		}
		p.Players = append(p.Players, player)
	}
	if len(ref.Darts) == 0 {
		return p
	}
	sort.Slice(ref.Darts, func(i, j int) bool {
		a, b := ref.Darts[i], ref.Darts[j]
		if a.Leg != b.Leg {
			return a.Leg < b.Leg
		}
		if a.Visit != b.Visit {
			return a.Visit < b.Visit
		}
		return a.Dart < b.Dart
	})
	p.Detail = &Detail{Coverage: "complete", Legs: []Leg{}}
	for _, d := range ref.Darts {
		if len(p.Detail.Legs) == 0 || p.Detail.Legs[len(p.Detail.Legs)-1].Number != d.Leg {
			p.Detail.Legs = append(p.Detail.Legs, Leg{Number: d.Leg, Completed: true, WinnerID: &ref.WinnerID, Visits: []Visit{}})
		}
		leg := &p.Detail.Legs[len(p.Detail.Legs)-1]
		if len(leg.Visits) == 0 || leg.Visits[len(leg.Visits)-1].Number != d.Visit+1 {
			remaining := 501
			for _, v := range leg.Visits {
				if v.PlayerID == d.PlayerID {
					remaining = v.EndRemaining
				}
			}
			leg.Visits = append(leg.Visits, Visit{Number: d.Visit + 1, PlayerID: d.PlayerID, StartRemaining: remaining, EndRemaining: remaining, Throws: []Throw{}})
		}
		visit := &leg.Visits[len(leg.Visits)-1]
		n, err := strconv.Atoi(d.Segment[1:])
		if err != nil {
			t.Fatal(err)
		}
		segment := Segment{Number: n}
		switch d.Segment[0] {
		case 'M':
			segment = Segment{Bed: "miss", Number: 0}
		case 'T':
			segment.Bed = "triple"
		case 'D':
			segment.Bed = "double"
		case 'S':
			segment.Bed = "single"
		default:
			t.Fatal("unknown reference segment")
		}
		dart := Throw{Number: d.Dart + 1, Segment: segment, EntryType: "manual"}
		if d.X != nil && d.Y != nil {
			dart.Position = &Position{X: *d.X, Y: *d.Y, Provenance: "manual"}
		}
		visit.Throws = append(visit.Throws, dart)
		score, _ := segmentScore(segment)
		visit.EndRemaining -= score
	}
	return p
}

func TestSanitizedReferencesAdaptToVersionedImports(t *testing.T) {
	for _, name := range []string{"manual", "randomized", "missing-coordinate", "score-only", "reversed-order"} {
		t.Run(name, func(t *testing.T) {
			p := referencePayload(t, name)
			b, err := json.Marshal(p)
			if err != nil {
				t.Fatal(err)
			}
			got, err := Parse(b)
			if err != nil {
				t.Fatal(err)
			}
			if got.ReviewReason != "" {
				t.Fatal(got.ReviewReason)
			}
			if name == "score-only" {
				if got.Detail != nil || got.Players[0].Stats != nil {
					t.Fatal("invented detail")
				}
				return
			}
			for _, leg := range got.Detail.Legs {
				for _, v := range leg.Visits {
					for _, d := range v.Throws {
						if name == "missing-coordinate" && d.Position != nil {
							t.Fatal("invented position")
						}
					}
				}
			}
		})
	}
}

func TestCompleteTotalsAndContinuity(t *testing.T) {
	for _, tc := range []struct {
		reason string
		change func(*Payload)
	}{
		{"remaining_continuity", func(p *Payload) { p.Detail.Legs[0].Visits[2].StartRemaining = 320 }},
		{"summary_totals_conflict", func(p *Payload) { n := 26; p.Players[0].Stats.DartsThrown = &n }},
		{"turn_continuity", func(p *Payload) { p.Detail.Legs[0].Visits[1].PlayerID = p.Players[0].ID }},
		{"summary_legs_conflict", func(p *Payload) { p.Detail.Legs = p.Detail.Legs[:2] }},
	} {
		p := referencePayload(t, "manual")
		tc.change(&p)
		b, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		got, err := Parse(b)
		if err != nil {
			t.Fatal(err)
		}
		if got.ReviewReason != tc.reason {
			t.Fatalf("got %q want %q", got.ReviewReason, tc.reason)
		}
	}
}
