package league

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
)

func changedImport(t *testing.T, s PendingResultService, req ApprovalRequest) ApprovalRequest {
	t.Helper()
	r, err := s.ImportDetail(context.Background(), req.PendingID)
	if err != nil {
		t.Fatal(err)
	}
	body := strings.ReplaceAll(string(r.Import.Payload), "60.12", "61.12")
	outcome, err := s.IngestPayload(context.Background(), []byte(body))
	if err != nil || outcome.Duplicate {
		t.Fatalf("changed import: %+v %v", outcome, err)
	}
	req.PendingID = outcome.Pending.ID
	return req
}

func TestEarlierDigestCannotAutoReplaceAfterLaterApprovalAndUndo(t *testing.T) {
	store, service, first := approvalSetup(t)
	second := changedImport(t, service, first)
	second.Replace = true
	if _, err := service.Approve(context.Background(), second); err != nil {
		t.Fatal(err)
	}
	if err := NewResultService(store).DeleteResult(context.Background(), 1, "undo"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Approve(context.Background(), first); !errors.Is(err, ErrReplacementRequired) {
		t.Fatalf("earlier digest silently approved: %v", err)
	}
}

func TestMappingCannotChangeAfterSourceDetached(t *testing.T) {
	store, service, req := approvalSetup(t)
	if _, err := service.Approve(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	if err := NewResultService(store).DeleteResult(context.Background(), 1, "undo"); err != nil {
		t.Fatal(err)
	}
	req = changedImport(t, service, req)
	req.Replace = true
	req.Mapping = map[string]int64{"seat-a": 11, "seat-b": 22}
	if _, err := service.Approve(context.Background(), req); !errors.Is(err, ErrApprovalConflict) {
		t.Fatalf("remapped confirmed source: %v", err)
	}
}

func TestConcurrentApprovalHasOneWinner(t *testing.T) {
	store, service, req := approvalSetup(t)
	var wg sync.WaitGroup
	outcomes := make(chan error, 10)
	for range 10 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := service.Approve(context.Background(), req); outcomes <- err }()
	}
	wg.Wait()
	close(outcomes)
	successes := 0
	for err := range outcomes {
		if err == nil {
			successes++
		} else if !errors.Is(err, ErrPendingResultNotPending) {
			t.Error(err)
		}
	}
	logs, err := store.ListAuditLogsBySeason(context.Background(), req.SeasonID)
	if err != nil || successes != 1 || len(logs) != 1 {
		t.Fatalf("successes=%d logs=%d err=%v", successes, len(logs), err)
	}
}

func TestStaleExpectedResultCannotOverwriteManualEdit(t *testing.T) {
	store, service, req := approvalSetup(t)
	ctx := context.Background()
	existing, err := NewResultService(store).RecordResult(ctx, 1, 3, 1, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.ExpectedResult, req.Replace = ExpectedFromResult(existing), true
	if _, err := NewResultService(store).EditResult(ctx, 1, 0, 3, nil, nil, "manual"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Approve(ctx, req); !errors.Is(err, ErrApprovalConflict) {
		t.Fatalf("stale request succeeded: %v", err)
	}
}

func TestManualAverageChangeDetachesSource(t *testing.T) {
	store, service, req := approvalSetup(t)
	ctx := context.Background()
	result, err := service.Approve(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	average := 42.0
	if _, err := NewResultService(store).EditResult(ctx, 1, result.PlayerOneLegs, result.PlayerTwoLegs, &average, result.PlayerTwoAverage, "manual"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ImportByResult(ctx, result.ID); !errors.Is(err, ErrPendingResultNotFound) {
		t.Fatalf("stale source stats survived: %v", err)
	}
}
