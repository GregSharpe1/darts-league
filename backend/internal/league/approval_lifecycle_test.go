package league

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestClosedAndNextSeasonCannotRetargetSource(t *testing.T) {
	store, service, req := approvalSetup(t)
	ctx := context.Background()
	if _, err := service.Approve(ctx, req); err != nil {
		t.Fatal(err)
	}
	changed := changedImport(t, service, req)
	if err := store.CloseSeason(ctx, req.SeasonID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Approve(ctx, changed); !errors.Is(err, ErrSeasonCompleted) {
		t.Fatalf("closed approval: %v", err)
	}
	if err := service.RejectWithReason(ctx, changed.PendingID, "admin", "closed"); !errors.Is(err, ErrSeasonCompleted) {
		t.Fatalf("closed reject: %v", err)
	}
	if err := NewResultService(store).DeleteResult(ctx, 1, "admin"); !errors.Is(err, ErrSeasonCompleted) {
		t.Fatalf("closed undo: %v", err)
	}
	if err := store.CreateNextSeason(ctx, req.SeasonID, NewSeason("Next")); err != nil {
		t.Fatal(err)
	}
	season, err := store.GetActiveSeason(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fixture, err := store.GetFixture(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	fixture.ID = 0
	fixture.SeasonID = season.ID
	fixtures, err := store.CreateFixtures(ctx, []Fixture{fixture})
	if err != nil {
		t.Fatal(err)
	}
	changed.SeasonID, changed.FixtureID, changed.Replace = season.ID, fixtures[0].ID, true
	if _, err := service.Approve(ctx, changed); !errors.Is(err, ErrApprovalConflict) {
		t.Fatalf("cross-season retarget: %v", err)
	}
	original, err := store.GetImport(ctx, req.PendingID)
	if err != nil || !original.Active || *original.SeasonID != req.SeasonID {
		t.Fatalf("historical source changed: %+v %v", original, err)
	}
}

func TestLegacyStoredApprovalRequiresAttestationAndMissingDateReason(t *testing.T) {
	store, service, req := approvalSetup(t)
	ctx := context.Background()
	pending, err := store.CreatePendingResult(ctx, PendingResult{ExternalMatchID: "legacy-old", PlayerOneName: "A", PlayerOneLegs: 3, PlayerTwoName: "B", PlayerTwoLegs: 1, Status: PendingResultStatusPending})
	if err != nil {
		t.Fatal(err)
	}
	req.PendingID = pending.ID
	req.Mapping = map[string]int64{"legacy-1": 11, "legacy-2": 22}
	req.MissingDateReason = ""
	if _, err := service.Approve(ctx, req); !errors.Is(err, ErrImportAttestation) {
		t.Fatal(err)
	}
	req.AttestFormat = true
	if _, err := service.Approve(ctx, req); !errors.Is(err, ErrImportAttestation) {
		t.Fatal(err)
	}
	req.MissingDateReason = "source did not store a date"
	if _, err := service.Approve(ctx, req); err != nil {
		t.Fatal(err)
	}
	r, err := store.GetImport(ctx, pending.ID)
	if err != nil || r.Import.PlayedAt != nil || r.Import.SettingsEvidence != "legacy_stored_summary" || !r.Approval.AttestFormat || r.Approval.MissingDateReason != req.MissingDateReason {
		t.Fatalf("fabricated source evidence: %+v %v", r, err)
	}
}

func TestBlockedDetailCannotConfirmAndRejectionIsAudited(t *testing.T) {
	store, service, req := approvalSetup(t)
	ctx := context.Background()
	r, err := store.GetImport(ctx, req.PendingID)
	if err != nil {
		t.Fatal(err)
	}
	source := r.Import
	source.Digest = "blocked-digest"
	source.ReviewReason = "conflicting detail"
	outcome, err := store.InsertImport(ctx, source, r.Pending.ReceivedAt)
	if err != nil {
		t.Fatal(err)
	}
	req.PendingID = outcome.Pending.ID
	req.Replace = true
	if _, err := service.Approve(ctx, req); !errors.Is(err, ErrImportReviewBlocked) {
		t.Fatalf("bypassed review block: %v", err)
	}
	if err := service.RejectWithReason(ctx, req.PendingID, "reviewer", "conflicting totals"); err != nil {
		t.Fatal(err)
	}
	after, err := store.GetImport(ctx, req.PendingID)
	if err != nil || !reflect.DeepEqual(after.Import, source) || after.Pending.Status != PendingResultStatusRejected {
		t.Fatalf("rejection changed original: %+v %v", after, err)
	}
	logs, err := store.ListAuditLogsBySeason(ctx, req.SeasonID)
	if err != nil || len(logs) != 1 || logs[0].Action != "import_rejected" || logs[0].Import.Reason != "conflicting totals" {
		t.Fatalf("rejection audit: %+v %v", logs, err)
	}
}

type auditFailureStore struct{ *MemoryStore }
type auditFailureTx struct {
	Store
	ApprovalStore
}

var errAuditFailure = errors.New("injected audit failure")

func (s auditFailureStore) Transaction(ctx context.Context, fn func(Store) error) error {
	return s.MemoryStore.Transaction(ctx, func(tx Store) error { return fn(auditFailureTx{Store: tx, ApprovalStore: tx.(ApprovalStore)}) })
}
func (auditFailureTx) CreateAuditLog(context.Context, AuditLogEntry) (AuditLogEntry, error) {
	return AuditLogEntry{}, errAuditFailure
}

func TestMemoryApprovalAndManualChangesRollbackAuditFailure(t *testing.T) {
	for _, action := range []string{"approve", "reject", "edit", "delete"} {
		t.Run(action, func(t *testing.T) {
			store, service, req := approvalSetup(t)
			ctx := context.Background()
			failing := auditFailureStore{store}
			failedService := NewPendingResultService(failing, NewResultService(failing))
			var err error
			if action == "approve" {
				_, err = failedService.Approve(ctx, req)
			} else if action == "reject" {
				err = failedService.Reject(ctx, req.PendingID, "admin")
			} else {
				if _, err := service.Approve(ctx, req); err != nil {
					t.Fatal(err)
				}
				if action == "edit" {
					_, err = NewResultService(failing).EditResult(ctx, 1, 3, 0, nil, nil, "manual")
				} else {
					err = NewResultService(failing).DeleteResult(ctx, 1, "manual")
				}
			}
			if !errors.Is(err, errAuditFailure) {
				t.Fatal(err)
			}
			r, err := store.GetImport(ctx, req.PendingID)
			if err != nil {
				t.Fatal(err)
			}
			if action == "approve" || action == "reject" {
				if r.Active || r.Pending.Status != PendingResultStatusPending {
					t.Fatalf("partial decision: %+v", r)
				}
				if _, err := store.GetResultByFixture(ctx, 1); !errors.Is(err, ErrResultNotFound) {
					t.Fatal(err)
				}
			} else {
				result, err := store.GetResultByFixture(ctx, 1)
				if err != nil || result.PlayerTwoLegs != 3 || !r.Active {
					t.Fatalf("partial mutation: %+v %+v %v", result, r, err)
				}
			}
		})
	}
}
