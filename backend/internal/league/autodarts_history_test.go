package league

import (
	"context"
	"testing"
)

func TestSuggestionDoesNotRetargetUnlinkedConfirmedHistory(t *testing.T) {
	ctx := context.Background()
	store, s, p := mappedSetup(t)
	out := ingestTestPayload(t, s.WithAutoSubmit(false), p)
	old := store.pendingByID[out.Pending.ID]
	old.Status = PendingResultStatusConfirmed
	store.pendingByID[old.ID] = old
	p.Players[1].LegsWon = 2
	changed := ingestTestPayload(t, s, p)
	detail, err := s.ImportDetail(ctx, changed.Pending.ID)
	if err != nil || detail.Suggestion.FixtureID != nil || len(detail.Suggestion.Mapping) != 0 {
		t.Fatalf("unsafe suggestion: %+v %v", detail.Suggestion, err)
	}
}
