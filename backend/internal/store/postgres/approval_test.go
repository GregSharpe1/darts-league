package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/jackc/pgx/v5"
)

func approvalPG(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := pgx.Identifier{"approval_" + strconv.FormatInt(time.Now().UnixNano(), 10)}.Sanitize()
	if _, err := conn.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// Only the unique schema created by this test is destroyed; never shared tables.
		if _, err := conn.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Error(err)
		}
		if err := conn.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	store, err := Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	return store
}

func approvalFixture(t *testing.T, store *Store) (league.PendingResultService, league.ApprovalRequest) {
	t.Helper()
	ctx := context.Background()
	season, err := store.EnsureActiveSeason(ctx, league.NewSeason("Approval test"))
	if err != nil {
		t.Fatal(err)
	}
	divisions, err := store.ReplaceDivisions(ctx, season.ID, []league.Division{{SeasonID: season.ID, Name: "One", Slug: "one", Position: 1}})
	if err != nil {
		t.Fatal(err)
	}
	a, err := store.CreatePlayer(ctx, league.Player{SeasonID: season.ID, DivisionID: &divisions[0].ID, DisplayName: "A", Status: league.PlayerStatusAssigned, RegisteredAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	b, err := store.CreatePlayer(ctx, league.Player{SeasonID: season.ID, DivisionID: &divisions[0].ID, DisplayName: "B", Status: league.PlayerStatusAssigned, RegisteredAt: time.Now()})
	if err != nil {
		t.Fatal(err)
	}
	season.Status = league.SeasonStatusStarted
	if _, err := store.UpsertSeason(ctx, season); err != nil {
		t.Fatal(err)
	}
	fixtures, err := store.CreateFixtures(ctx, []league.Fixture{{SeasonID: season.ID, DivisionID: divisions[0].ID, PlayerOneID: a.ID, PlayerTwoID: b.ID, GameVariant: "501", LegsToWin: 3, WeekNumber: 1, ScheduledAt: time.Now(), Status: "scheduled"}})
	if err != nil {
		t.Fatal(err)
	}
	service := league.NewPendingResultService(store, league.NewResultService(store))
	raw, err := os.ReadFile("../../../../docs/autodarts/examples-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var examples map[string]json.RawMessage
	if err := json.Unmarshal(raw, &examples); err != nil {
		t.Fatal(err)
	}
	outcome, err := service.IngestPayload(ctx, examples["producer"])
	if err != nil {
		t.Fatal(err)
	}
	return service, league.ApprovalRequest{PendingID: outcome.Pending.ID, SeasonID: season.ID, FixtureID: fixtures[0].ID, Mapping: map[string]int64{"seat-a": b.ID, "seat-b": a.ID}, Actor: "reviewer", Reason: "verified", MissingDateReason: "unavailable"}
}

func TestApprovalPostgresConcurrentAndUndo(t *testing.T) {
	store := approvalPG(t)
	service, req := approvalFixture(t, store)
	ctx := context.Background()
	var wg sync.WaitGroup
	outcomes := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() { defer wg.Done(); _, err := service.Approve(ctx, req); outcomes <- err }()
	}
	wg.Wait()
	close(outcomes)
	success := 0
	for err := range outcomes {
		if err == nil {
			success++
		} else if !errors.Is(err, league.ErrPendingResultNotPending) {
			t.Error(err)
		}
	}
	if success != 1 {
		t.Fatalf("successful approvals=%d", success)
	}
	result, err := store.GetResultByFixture(ctx, req.FixtureID)
	if err != nil {
		t.Fatal(err)
	}
	if result.WinnerID != req.Mapping["seat-a"] || result.PlayerTwoLegs != 3 {
		t.Fatalf("wrong mapping: %+v", result)
	}
	logs, err := store.ListAuditLogsBySeason(ctx, req.SeasonID)
	if err != nil || len(logs) != 1 || logs[0].Actor != "reviewer" || logs[0].Import.Mapping["seat-a"] != result.WinnerID {
		t.Fatalf("audit: %+v %v", logs, err)
	}
	if err := league.NewResultService(store).DeleteResult(ctx, req.FixtureID, "undo"); err != nil {
		t.Fatal(err)
	}
	original, err := store.GetImport(ctx, req.PendingID)
	if err != nil || original.Active || original.ResultID != nil || original.FixtureID == nil || original.Import.Detail == nil {
		t.Fatalf("undo/original: %+v %v", original, err)
	}
}

func TestApprovalPostgresRollsBackAuditFailure(t *testing.T) {
	store := approvalPG(t)
	service, req := approvalFixture(t, store)
	ctx := context.Background()
	if _, err := store.pool.Exec(ctx, `ALTER TABLE admin_audit_log ADD CONSTRAINT fail_audit CHECK (action <> 'import_confirmed')`); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Approve(ctx, req); err == nil {
		t.Fatal("expected audit failure")
	}
	pending, err := store.GetImport(ctx, req.PendingID)
	if err != nil || pending.Pending.Status != league.PendingResultStatusPending || pending.Active || pending.FixtureID != nil {
		t.Fatalf("partial approval: %+v %v", pending, err)
	}
	if _, err := store.GetResultByFixture(ctx, req.FixtureID); !errors.Is(err, league.ErrResultNotFound) {
		t.Fatalf("result escaped rollback: %v", err)
	}
}

func TestManualPostgresMutationRollsBackAuditFailure(t *testing.T) {
	for _, action := range []string{"result_edited", "result_deleted"} {
		t.Run(action, func(t *testing.T) {
			store := approvalPG(t)
			service, req := approvalFixture(t, store)
			ctx := context.Background()
			before, err := service.Approve(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := store.pool.Exec(ctx, `ALTER TABLE admin_audit_log ADD CONSTRAINT fail_manual_audit CHECK (action = 'import_confirmed')`); err != nil {
				t.Fatal(err)
			}
			if action == "result_edited" {
				_, err = league.NewResultService(store).EditResult(ctx, req.FixtureID, 3, 0, nil, nil, "manual")
			} else {
				err = league.NewResultService(store).DeleteResult(ctx, req.FixtureID, "manual")
			}
			if err == nil {
				t.Fatal("expected audit failure")
			}
			after, err := store.GetResultByFixture(ctx, req.FixtureID)
			if err != nil || after.ID != before.ID || after.PlayerTwoLegs != 3 {
				t.Fatalf("partial result: %+v %v", after, err)
			}
			if r, err := store.ImportByResult(ctx, before.ID); err != nil || !r.Active {
				t.Fatalf("detachment escaped rollback: %+v %v", r, err)
			}
		})
	}
}
