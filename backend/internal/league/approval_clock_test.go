package league

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestExpectedResultChangesWhenClockDoesNotAdvance(t *testing.T) {
	store, service, req := approvalSetup(t)
	ctx := context.Background()
	original, err := service.Approve(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	frozen := NewPendingResultServiceWithNow(store, NewResultService(store), func() time.Time { return original.UpdatedAt })
	record, err := store.GetImport(ctx, req.PendingID)
	if err != nil {
		t.Fatal(err)
	}
	requests := []ApprovalRequest{}
	for _, label := range []string{"Replacement A", "Replacement B"} {
		source := record.Import
		source.Players = append(source.Players[:0:0], source.Players...)
		source.Players[0].DisplayName = label
		source.Digest = label
		outcome, err := store.InsertImport(ctx, source, original.UpdatedAt)
		if err != nil {
			t.Fatal(err)
		}
		candidate := req
		candidate.PendingID, candidate.Replace, candidate.ExpectedResult = outcome.Pending.ID, true, ExpectedFromResult(original)
		requests = append(requests, candidate)
	}
	if _, err := frozen.Approve(ctx, requests[0]); err != nil {
		t.Fatal(err)
	}
	if _, err := frozen.Approve(ctx, requests[1]); !errors.Is(err, ErrApprovalConflict) {
		t.Fatalf("identical summary bypassed stale comparison: %v", err)
	}
}
