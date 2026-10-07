package league

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"testing"
	"time"
)

func TestTransactionRollback(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	failure := errors.New("audit failed")
	err := store.Transaction(ctx, func(tx Store) error {
		_, err := tx.CreatePendingResult(ctx, PendingResult{Status: PendingResultStatusPending})
		if err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	rows, err := store.ListPendingResults(ctx, PendingResultStatusPending)
	if err != nil || len(rows) != 0 {
		t.Fatalf("rollback: %v %v", rows, err)
	}
}

func approvalSetup(t *testing.T) (*MemoryStore, PendingResultService, ApprovalRequest) {
	t.Helper()
	ctx := context.Background()
	store := NewMemoryStore()
	season, _ := store.GetActiveSeason(ctx)
	season.Status = SeasonStatusStarted
	if _, err := store.UpsertSeason(ctx, season); err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateFixtures(ctx, []Fixture{{SeasonID: season.ID, DivisionID: 1, PlayerOneID: 11, PlayerTwoID: 22, GameVariant: "501", LegsToWin: 3, ScheduledAt: time.Now()}}); err != nil {
		t.Fatal(err)
	}
	service := NewPendingResultService(store, NewResultService(store))
	b, err := os.ReadFile("../../../docs/autodarts/examples-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var examples map[string]json.RawMessage
	if err := json.Unmarshal(b, &examples); err != nil {
		t.Fatal(err)
	}
	outcome, err := service.IngestPayload(ctx, examples["producer"])
	if err != nil {
		t.Fatal(err)
	}
	return store, service, ApprovalRequest{PendingID: outcome.Pending.ID, SeasonID: season.ID, FixtureID: 1, Mapping: map[string]int64{"seat-a": 22, "seat-b": 11}, Actor: "reviewer", Reason: "verified", MissingDateReason: "not supplied"}
}

func TestApprovalMapsSourceAndDetachesAfterManualEdit(t *testing.T) {
	store, service, req := approvalSetup(t)
	ctx := context.Background()
	result, err := service.Approve(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	if result.PlayerTwoLegs != 3 || result.WinnerID != 22 {
		t.Fatalf("reversed mapping: %+v", result)
	}
	linked, err := store.ImportByFixture(ctx, 1)
	if err != nil || linked.Mapping["seat-a"] != 22 || !linked.Active {
		t.Fatalf("link: %+v %v", linked, err)
	}
	mapped, err := linked.MappedImport()
	if err != nil || mapped.Players[0].ID != "22" || mapped.Detail.Legs[0].Visits[0].PlayerID != "22" {
		t.Fatalf("mapped references: %+v %v", mapped, err)
	}
	if _, err := NewResultService(store).EditResult(ctx, 1, 3, 0, nil, nil, "manual"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ImportByFixture(ctx, 1); !errors.Is(err, ErrPendingResultNotFound) {
		t.Fatalf("stale stats: %v", err)
	}
	original, err := store.GetImport(ctx, req.PendingID)
	if err != nil || original.Import.Players[0].ID != "seat-a" || original.FixtureID == nil || original.Active {
		t.Fatalf("lost original/mapping: %+v %v", original, err)
	}
}

func TestApprovalRequiresExplicitReplacementAndExpectedResult(t *testing.T) {
	store, service, req := approvalSetup(t)
	ctx := context.Background()
	existing, err := NewResultService(store).RecordResult(ctx, 1, 3, 1, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Approve(ctx, req); !errors.Is(err, ErrApprovalConflict) {
		t.Fatalf("blind overwrite: %v", err)
	}
	req.ExpectedResult = ExpectedFromResult(existing)
	if _, err := service.Approve(ctx, req); !errors.Is(err, ErrReplacementRequired) {
		t.Fatalf("implicit replace: %v", err)
	}
	req.Replace = true
	if _, err := service.Approve(ctx, req); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Approve(ctx, req); !errors.Is(err, ErrPendingResultNotPending) {
		t.Fatalf("replay: %v", err)
	}
}
