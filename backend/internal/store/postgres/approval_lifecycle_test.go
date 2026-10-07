package postgres

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

func TestPostgresReplacementRollbackKeepsPreviousSource(t *testing.T) {
	store := approvalPG(t)
	service, req := approvalFixture(t, store)
	ctx := context.Background()
	original, err := service.Approve(ctx, req)
	if err != nil {
		t.Fatal(err)
	}
	record, err := store.GetImport(ctx, req.PendingID)
	if err != nil {
		t.Fatal(err)
	}
	changed, err := service.IngestPayload(ctx, []byte(strings.ReplaceAll(string(record.Import.Payload), "60.12", "61.12")))
	if err != nil {
		t.Fatal(err)
	}
	req.PendingID, req.ExpectedResult = changed.Pending.ID, league.ExpectedFromResult(original)
	if _, err := service.Approve(ctx, req); !errors.Is(err, league.ErrReplacementRequired) {
		t.Fatalf("implicit replacement: %v", err)
	}
	req.Replace = true
	if _, err := store.pool.Exec(ctx, `ALTER TABLE admin_audit_log ADD CONSTRAINT fail_replace CHECK (false) NOT VALID`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Approve(ctx, req); err == nil {
		t.Fatal("expected audit failure")
	}
	linked, err := store.ImportByResult(ctx, original.ID)
	if err != nil || linked.Pending.ID != record.Pending.ID {
		t.Fatalf("previous source not restored: %+v %v", linked, err)
	}
	after, err := store.GetResultByFixture(ctx, req.FixtureID)
	if err != nil || *after.PlayerTwoAverage != *original.PlayerTwoAverage {
		t.Fatalf("score/average changed: %+v %v", after, err)
	}
}

func TestPostgresLegacyApprovalRejectAndReadonly(t *testing.T) {
	store := approvalPG(t)
	service, req := approvalFixture(t, store)
	ctx := context.Background()
	pending, err := store.CreatePendingResult(ctx, league.PendingResult{ExternalMatchID: "legacy-stored", PlayerOneName: "Old A", PlayerOneLegs: 3, PlayerTwoName: "Old B", PlayerTwoLegs: 1, Status: league.PendingResultStatusPending, ReceivedAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	req.PendingID = pending.ID
	req.Mapping = map[string]int64{"legacy-1": req.Mapping["seat-a"], "legacy-2": req.Mapping["seat-b"]}
	if _, err := service.Approve(ctx, req); !errors.Is(err, league.ErrImportAttestation) {
		t.Fatalf("legacy bypass: %v", err)
	}
	req.AttestFormat = true
	if _, err := service.Approve(ctx, req); err != nil {
		t.Fatal(err)
	}
	record, err := store.GetImport(ctx, pending.ID)
	if err != nil || record.Import.PlayedAt != nil || record.Import.Digest == "" || !record.Active {
		t.Fatalf("legacy evidence: %+v %v", record, err)
	}
	var mapped int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM import_players WHERE pending_result_id=$1 AND league_player_id IS NOT NULL`, pending.ID).Scan(&mapped); err != nil || mapped != 2 {
		t.Fatalf("legacy mapping=%d %v", mapped, err)
	}
	rejected, err := service.IngestPayload(ctx, []byte(`{"matchId":"reject-pg","player1":{"name":"A","legsWon":3},"player2":{"name":"B","legsWon":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	if err := service.RejectWithReason(ctx, rejected.Pending.ID, "rejector", "not a league match"); err != nil {
		t.Fatal(err)
	}
	logs, err := store.ListAuditLogsBySeason(ctx, req.SeasonID)
	if err != nil || len(logs) != 2 || logs[0].Import.Reason != "not a league match" || logs[0].Actor != "rejector" {
		t.Fatalf("reject audit: %+v %v", logs, err)
	}
	if err := store.CloseSeason(ctx, req.SeasonID); err != nil {
		t.Fatal(err)
	}
	if _, err := league.NewResultService(store).EditResult(ctx, req.FixtureID, 3, 0, nil, nil, "manual"); !errors.Is(err, league.ErrSeasonCompleted) {
		t.Fatalf("closed edit: %v", err)
	}
	if err := store.CreateNextSeason(ctx, req.SeasonID, league.NewSeason("Next")); err != nil {
		t.Fatal(err)
	}
	if err := league.NewResultService(store).DeleteResult(ctx, req.FixtureID, "manual"); !errors.Is(err, league.ErrSeasonCompleted) {
		t.Fatalf("historical undo: %v", err)
	}
	retained, err := store.GetImport(ctx, pending.ID)
	if err != nil || !retained.Active || *retained.FixtureID != req.FixtureID {
		t.Fatalf("historical source changed: %+v %v", retained, err)
	}
}

func TestPostgresApprovalSerializesWithClose(t *testing.T) {
	store := approvalPG(t)
	service, req := approvalFixture(t, store)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	result, err := league.NewResultService(store).RecordResult(ctx, req.FixtureID, 3, 0, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Replace, req.ExpectedResult = true, league.ExpectedFromResult(result)
	start := make(chan struct{})
	approval := make(chan error, 1)
	closed := make(chan error, 1)
	go func() { <-start; _, err := service.Approve(ctx, req); approval <- err }()
	go func() { <-start; closed <- store.CloseSeason(ctx, req.SeasonID) }()
	close(start)
	if err := <-closed; err != nil {
		t.Fatal(err)
	}
	if err := <-approval; err != nil && !errors.Is(err, league.ErrSeasonCompleted) {
		t.Fatal(err)
	}
	if _, err := league.NewResultService(store).EditResult(ctx, req.FixtureID, 3, 1, nil, nil, "late"); !errors.Is(err, league.ErrSeasonCompleted) {
		t.Fatalf("post-close edit: %v", err)
	}
}
