package league

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/greg/darts-league/backend/internal/autodarts"
	"testing"
	"time"
)

func mappedSetup(t *testing.T) (*MemoryStore, PendingResultService, autodarts.Payload) {
	t.Helper()
	ctx := context.Background()
	store := NewMemoryStore()
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	start := now.Add(-24 * time.Hour)
	season, _ := store.GetActiveSeason(ctx)
	season.Status, season.StartedAt = SeasonStatusStarted, &start
	if _, err := store.UpsertSeason(ctx, season); err != nil {
		t.Fatal(err)
	}
	s := NewPendingResultServiceWithNow(store, NewResultServiceWithNow(store, func() time.Time { return now }), func() time.Time { return now }).WithAutoSubmit(true)
	division := int64(1)
	for _, name := range []string{"One", "Two"} {
		p, err := store.CreatePlayer(ctx, Player{SeasonID: 1, DivisionID: &division, Status: PlayerStatusAssigned, DisplayName: name})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := s.SaveAutodartsMapping(ctx, 1, p.ID, "source "+name); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.CreateFixtures(ctx, []Fixture{{SeasonID: 1, DivisionID: 1, PlayerOneID: 1, PlayerTwoID: 2, GameVariant: "501", LegsToWin: 3, ScheduledAt: now.Add(7 * 24 * time.Hour)}}); err != nil {
		t.Fatal(err)
	}
	played := now.Add(-time.Hour).Format(time.RFC3339)
	p := autodarts.Payload{SchemaVersion: "autodarts.import.v1", Source: "autodarts", ExternalMatchID: "new", PlayedAt: &played, Completed: true, Settings: autodarts.Settings{BaseScore: 501, LegsToWin: 3, Out: "double"}, Players: []autodarts.Player{{ID: "a", DisplayName: " source Two ", LegsWon: 3}, {ID: "b", DisplayName: "source One", LegsWon: 1}}}
	return store, s, p
}

func ingestTestPayload(t *testing.T, s PendingResultService, p autodarts.Payload) ImportOutcome {
	t.Helper()
	body, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	out, err := s.IngestPayload(context.Background(), body)
	if err != nil {
		t.Fatal(err)
	}
	return out
}

func TestAutoSubmitOnlyNewEligibleImports(t *testing.T) {
	ctx := context.Background()
	store, s, p := mappedSetup(t)
	notifications := 0
	var submitted []Result
	s = s.WithNotifier(func(context.Context, PendingResult) { notifications++ }).WithAutoSubmitNotifier(func(ctx context.Context, r Result) {
		if _, err := store.GetResultByFixture(ctx, r.FixtureID); err != nil {
			t.Errorf("auto-submit notified before commit: %v", err)
		}
		submitted = append(submitted, r)
	})
	out := ingestTestPayload(t, s, p)
	if out.Pending.Status != PendingResultStatusConfirmed || notifications != 0 || len(submitted) != 1 || submitted[0].FixtureID != 1 {
		t.Fatalf("outcome %+v notifications %d submitted %+v", out, notifications, submitted)
	}
	r, err := store.GetResultByFixture(ctx, 1)
	if err != nil || r.PlayerOneLegs != 1 || r.PlayerTwoLegs != 3 || r.WinnerID != 2 {
		t.Fatalf("source order: %+v %v", r, err)
	}
	out = ingestTestPayload(t, s, p)
	if !out.Duplicate || out.Pending.Status != PendingResultStatusConfirmed || out.AutoSubmitted != nil || len(submitted) != 1 {
		t.Fatal(out, submitted)
	}
	audits, _ := store.ListAuditLogsBySeason(ctx, 1)
	if len(audits) != 1 || audits[0].Actor != "autodarts:auto-submit" {
		t.Fatal(audits)
	}
}

func TestAutoSubmitFallsBackToPending(t *testing.T) {
	for _, name := range []string{"disabled", "missing_mapping", "same_names", "case_mismatch", "mixed_divisions", "unassigned", "missing_date", "future_date", "old_date", "completed", "not_started", "scored", "format", "ambiguous", "changed"} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store, s, p := mappedSetup(t)
			switch name {
			case "disabled":
				s = s.WithAutoSubmit(false)
			case "missing_mapping":
				if err := s.ClearAutodartsMapping(ctx, 1, 1); err != nil {
					t.Fatal(err)
				}
			case "same_names":
				p.Players[1].DisplayName = p.Players[0].DisplayName
			case "case_mismatch":
				p.Players[1].DisplayName = "Source One"
			case "mixed_divisions":
				q := store.playersByID[1]
				d := int64(2)
				q.DivisionID = &d
				store.playersByID[1] = q
			case "unassigned":
				q := store.playersByID[1]
				q.DivisionID = nil
				store.playersByID[1] = q
			case "missing_date":
				p.PlayedAt = nil
			case "future_date":
				d := "2027-01-01T00:00:00Z"
				p.PlayedAt = &d
			case "old_date":
				d := "2025-01-01T00:00:00Z"
				p.PlayedAt = &d
			case "completed":
				store.activeSeason.Status = SeasonStatusCompleted
			case "not_started":
				store.activeSeason.Status = SeasonStatusRegistrationOpen
			case "scored":
				if _, err := NewResultService(store).RecordResult(ctx, 1, 3, 0, nil, nil); err != nil {
					t.Fatal(err)
				}
			case "format":
				f := store.fixturesByID[1]
				f.LegsToWin = 2
				store.fixturesByID[1] = f
			case "ambiguous":
				f := store.fixturesByID[1]
				f.ID = 2
				store.fixturesByID[2] = f
			case "changed":
				ingestTestPayload(t, s.WithAutoSubmit(false), p)
				p.Players[1].LegsWon = 2
			}
			notifications, submitted := 0, 0
			s = s.WithNotifier(func(context.Context, PendingResult) { notifications++ }).WithAutoSubmitNotifier(func(context.Context, Result) { submitted++ })
			out := ingestTestPayload(t, s, p)
			if out.Pending.Status != PendingResultStatusPending || notifications != 1 || submitted != 0 {
				t.Fatalf("%+v notifications %d submitted %d", out, notifications, submitted)
			}
		})
	}
}

func TestSuggestionsRefreshWithoutApprovingOldInbox(t *testing.T) {
	ctx := context.Background()
	store, s, p := mappedSetup(t)
	if err := s.ClearAutodartsMapping(ctx, 1, 1); err != nil {
		t.Fatal(err)
	}
	out := ingestTestPayload(t, s, p)
	detail, err := s.ImportDetail(ctx, out.Pending.ID)
	if err != nil || detail.Suggestion.FixtureID != nil || len(detail.Suggestion.Mapping) != 1 {
		t.Fatalf("partial: %+v %v", detail, err)
	}
	if _, err := s.SaveAutodartsMapping(ctx, 1, 1, "source One"); err != nil {
		t.Fatal(err)
	}
	detail, err = s.ImportDetail(ctx, out.Pending.ID)
	if err != nil || detail.Suggestion.FixtureID == nil || *detail.Suggestion.FixtureID != 1 || detail.Suggestion.Mapping["a"] != 2 {
		t.Fatalf("full: %+v %v", detail, err)
	}
	if detail.Pending.Status != PendingResultStatusPending || len(detail.Mapping) != 0 {
		t.Fatal("GET approved/changed history")
	}
	if replay := ingestTestPayload(t, s, p); !replay.Duplicate || replay.Pending.Status != PendingResultStatusPending {
		t.Fatal(replay)
	}
	rows, _ := store.ListAuditLogsBySeason(ctx, 1)
	if len(rows) != 0 {
		t.Fatal(fmt.Sprint(rows))
	}
}
