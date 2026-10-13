package league

import (
	"context"
	"testing"
	"time"
)

func TestSuggestionAmbiguityAndManualScoredFixture(t *testing.T) {
	for _, tc := range []struct {
		name              string
		count             int
		division, fixture bool
	}{
		{"none", 0, false, false}, {"identical", 0, false, false}, {"mixed", 2, false, false},
		{"unassigned", 2, false, false}, {"no_fixture", 2, true, false}, {"ambiguous", 2, true, false},
		{"scored", 2, true, true}, {"normal", 2, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			store, s, p := mappedSetup(t)
			s = s.WithAutoSubmit(false)
			switch tc.name {
			case "none":
				store.autodartsUsernames = map[int64]string{}
			case "identical":
				p.Players[1].DisplayName = p.Players[0].DisplayName
			case "mixed":
				q := store.playersByID[1]
				d := int64(2)
				q.DivisionID = &d
				store.playersByID[1] = q
			case "unassigned":
				q := store.playersByID[1]
				q.DivisionID = nil
				store.playersByID[1] = q
			case "no_fixture":
				store.fixturesByID = map[int64]Fixture{}
			case "ambiguous":
				f := store.fixturesByID[1]
				f.ID = 2
				store.fixturesByID[2] = f
			case "scored":
				if _, err := NewResultService(store).RecordResult(ctx, 1, 3, 0, nil, nil); err != nil {
					t.Fatal(err)
				}
			}
			out := ingestTestPayload(t, s, p)
			detail, err := s.ImportDetail(ctx, out.Pending.ID)
			if err != nil {
				t.Fatal(err)
			}
			got := detail.Suggestion
			if len(got.Mapping) != tc.count || (got.DivisionID != nil) != tc.division || (got.FixtureID != nil) != tc.fixture {
				t.Fatalf("suggestion %+v", got)
			}
		})
	}
}

func TestMappingEditsPreserveApprovedHistory(t *testing.T) {
	ctx := context.Background()
	store, s, p := mappedSetup(t)
	out := ingestTestPayload(t, s, p)
	if _, err := s.SaveAutodartsMapping(ctx, 1, 1, "replacement"); err != nil {
		t.Fatal(err)
	}
	record, err := store.GetImport(ctx, out.Pending.ID)
	if err != nil || record.Mapping["b"] != 1 || record.Pending.ConfirmedBy != "autodarts:auto-submit" {
		t.Fatalf("history: %+v %v", record, err)
	}
	p.Players[1].LegsWon = 2
	changed := ingestTestPayload(t, s, p)
	detail, err := s.ImportDetail(ctx, changed.Pending.ID)
	if err != nil || detail.Suggestion.FixtureID != nil || len(detail.Suggestion.Mapping) != 0 {
		t.Fatalf("changed history: %+v %v", detail, err)
	}
	audits, err := store.ListAuditLogsBySeason(ctx, 1)
	if err != nil || len(audits) != 1 || audits[0].Import.Mapping["b"] != 1 {
		t.Fatalf("audit altered %+v %v", audits, err)
	}
}

func TestAutoSubmitDateBoundaries(t *testing.T) {
	for _, atStart := range []bool{true, false} {
		store, s, p := mappedSetup(t)
		date := s.now().Format(time.RFC3339)
		if atStart {
			date = store.activeSeason.StartedAt.Format(time.RFC3339)
		}
		p.PlayedAt = &date
		if out := ingestTestPayload(t, s, p); out.Pending.Status != PendingResultStatusConfirmed {
			t.Fatal(out)
		}
	}
}

func TestMappingsEditableAtEverySeasonStage(t *testing.T) {
	for _, status := range []SeasonStatus{SeasonStatusRegistrationOpen, SeasonStatusStarted, SeasonStatusCompleted} {
		t.Run(string(status), func(t *testing.T) {
			store, s, _ := mappedSetup(t)
			store.activeSeason.Status = status
			store.seasonsByID[1] = store.activeSeason
			if _, err := s.SaveAutodartsMapping(context.Background(), 1, 1, "changed"); err != nil {
				t.Fatal(err)
			}
			if err := s.ClearAutodartsMapping(context.Background(), 1, 1); err != nil {
				t.Fatal(err)
			}
		})
	}
}
