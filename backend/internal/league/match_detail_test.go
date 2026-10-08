package league

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

func matchDetailSetup(t *testing.T) (*MemoryStore, PendingResultService, ApprovalRequest) {
	t.Helper()
	store, pending, req := approvalSetup(t)
	store.playersByID[11] = Player{ID: 11, SeasonID: req.SeasonID, DisplayName: "League One", Nickname: "The Arrow"}
	store.playersByID[22] = Player{ID: 22, SeasonID: req.SeasonID, DisplayName: "League Two"}
	f := store.fixturesByID[1]
	f.ScheduledAt = time.Date(2026, 3, 30, 18, 30, 0, 0, time.UTC)
	store.fixturesByID[1] = f
	return store, pending, req
}

func TestMatchDetailProjection(t *testing.T) {
	store, pending, req := matchDetailSetup(t)
	if _, err := pending.Approve(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	s := NewFixtureServiceWithNow(store, func() time.Time { return time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC) })
	detail, err := s.MatchDetail(context.Background(), 1, false)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Players[0].LeaguePlayerID != 11 || detail.Players[0].PreferredName != "The Arrow" || detail.Players[1].PreferredName != "League Two" || detail.Players[1].LegsWon != 3 {
		t.Fatalf("wrong fixture order: %+v", detail.Players)
	}
	if detail.Detail == nil || detail.Detail.Legs[0].Visits[0].PlayerID != "22" {
		t.Fatalf("unmapped detail: %+v", detail)
	}
	encoded, err := json.Marshal(detail)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"seat-a", "seat-b", "account_id", "external_match_id", "digest", "Payload", "display_name"} {
		if strings.Contains(string(encoded), secret) {
			t.Fatalf("leaked %s: %s", secret, encoded)
		}
	}
	if detail.Coverage.RecordedThrows == 0 || detail.Coverage.PositionFraction == nil {
		t.Fatalf("missing coverage: %+v", detail.Coverage)
	}
}

func TestMatchDetailRevealBoundaries(t *testing.T) {
	for _, monday := range []string{"2026-03-23T09:00:00Z", "2026-03-30T08:00:00Z", "2026-10-19T08:00:00Z", "2026-10-26T09:00:00Z"} {
		t.Run(monday, func(t *testing.T) {
			store, _, _ := matchDetailSetup(t)
			reveal, err := time.Parse(time.RFC3339, monday)
			if err != nil {
				t.Fatal(err)
			}
			f := store.fixturesByID[1]
			f.ScheduledAt = reveal.Add(10 * time.Hour)
			store.fixturesByID[1] = f
			if _, err := store.CreateResult(context.Background(), Result{FixtureID: 1, PlayerOneLegs: 3, PlayerTwoLegs: 1}); err != nil {
				t.Fatal(err)
			}
			for _, offset := range []time.Duration{-time.Nanosecond, 0, time.Nanosecond} {
				s := NewFixtureServiceWithNow(store, func() time.Time { return reveal.Add(offset) })
				_, err := s.MatchDetail(context.Background(), 1, false)
				if offset < 0 && !errors.Is(err, ErrFixtureNotFound) {
					t.Fatalf("future visible: %v", err)
				}
				if offset >= 0 && err != nil {
					t.Fatal(err)
				}
				if _, err := s.MatchDetail(context.Background(), 1, true); err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}
