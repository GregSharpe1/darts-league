package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/greg/darts-league/backend/internal/autodarts"
	"github.com/greg/darts-league/backend/internal/league"
	"sync"
	"testing"
	"time"
)

func autoPG(t *testing.T) (*Store, league.PendingResultService, []byte, league.ApprovalRequest) {
	t.Helper()
	ctx := context.Background()
	store := approvalPG(t)
	_, req := approvalFixture(t, store)
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	start := now.Add(-48 * time.Hour)
	season, err := store.GetActiveSeason(ctx)
	if err != nil {
		t.Fatal(err)
	}
	season.StartedAt = &start
	if _, err := store.UpsertSeason(ctx, season); err != nil {
		t.Fatal(err)
	}
	service := league.NewPendingResultServiceWithNow(store, league.NewResultServiceWithNow(store, func() time.Time { return now }), func() time.Time { return now }).WithAutoSubmit(true)
	for source, id := range req.Mapping {
		name := "Birch Guest"
		if source == "seat-a" {
			name = "Aster Example"
		}
		if _, err := service.SaveAutodartsMapping(ctx, season.ID, id, name); err != nil {
			t.Fatal(err)
		}
	}
	record, err := store.GetImport(ctx, req.PendingID)
	if err != nil {
		t.Fatal(err)
	}
	var payload autodarts.Payload
	if err := json.Unmarshal(record.Import.Payload, &payload); err != nil {
		t.Fatal(err)
	}
	payload.ExternalMatchID = "new-auto"
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return store, service, body, req
}

func TestAutoSubmitPostgresConcurrentDeliveryAndManualApproval(t *testing.T) {
	ctx := context.Background()
	store, service, body, req := autoPG(t)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := service.IngestDurablePayload(ctx, body); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		_, err := service.Approve(ctx, req)
		if err != nil && !errors.Is(err, league.ErrApprovalConflict) {
			t.Error(err)
		}
	}()
	wg.Wait()
	results, err := store.ListResultsBySeason(ctx, req.SeasonID)
	if err != nil || len(results) != 1 {
		t.Fatalf("results: %+v %v", results, err)
	}
	audits, err := store.ListAuditLogsBySeason(ctx, req.SeasonID)
	if err != nil || len(audits) != 1 {
		t.Fatalf("audits: %+v %v", audits, err)
	}
	if results[0].PlayerOneLegs != 0 || results[0].PlayerTwoLegs != 3 {
		t.Fatal("source ordering lost")
	}
}

func TestAutoSubmitPostgresRollbackBeforeNotification(t *testing.T) {
	for _, table := range []string{"results", "admin_audit_log"} {
		t.Run(table, func(t *testing.T) {
			ctx := context.Background()
			store, service, body, req := autoPG(t)
			if _, err := store.pool.Exec(ctx, `CREATE FUNCTION fail_auto() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected failure'; END $$; CREATE TRIGGER fail_auto BEFORE INSERT ON `+table+` FOR EACH ROW EXECUTE FUNCTION fail_auto()`); err != nil {
				t.Fatal(err)
			}
			notifications := 0
			service = service.WithNotifier(func(context.Context, league.PendingResult) { notifications++ })
			if _, err := service.IngestDurablePayload(ctx, body); err == nil {
				t.Fatal("expected injected error")
			}
			rows, err := store.ImportsBySource(ctx, "autodarts", "new-auto")
			if err != nil || len(rows) != 0 || notifications != 0 {
				t.Fatalf("uncommitted import/notification: %+v %d %v", rows, notifications, err)
			}
			results, err := store.ListResultsBySeason(ctx, req.SeasonID)
			if err != nil || len(results) != 0 {
				t.Fatalf("result escaped rollback: %+v %v", results, err)
			}
			if _, err := store.pool.Exec(ctx, `DROP TRIGGER fail_auto ON `+table); err != nil {
				t.Fatal(err)
			}
			out, err := service.IngestDurablePayload(ctx, body)
			if err != nil || out.Duplicate || out.Pending.Status != league.PendingResultStatusConfirmed || notifications != 0 {
				t.Fatalf("retry: %+v %v", out, err)
			}
		})
	}
}

func TestAutoSubmitPostgresMappingTransactionOrder(t *testing.T) {
	for _, saveFirst := range []bool{true, false} {
		t.Run(map[bool]string{true: "mapping_first", false: "import_first"}[saveFirst], func(t *testing.T) {
			ctx := context.Background()
			store, service, body, req := autoPG(t)
			id := req.Mapping["seat-a"]
			if err := service.ClearAutodartsMapping(ctx, req.SeasonID, id); err != nil {
				t.Fatal(err)
			}
			locked, release := make(chan struct{}), make(chan struct{})
			firstDone := make(chan error, 1)
			go func() {
				firstDone <- store.Transaction(ctx, func(tx league.Store) error {
					close(locked)
					<-release
					if saveFirst {
						name := "Aster Example"
						return tx.(league.AutodartsMappingStore).SetAutodartsUsername(ctx, req.SeasonID, id, &name)
					}
					_, err := league.NewPendingResultService(tx, league.NewResultService(tx)).WithAutoSubmit(true).IngestPayload(ctx, body)
					return err
				})
			}()
			<-locked
			secondDone := make(chan error, 1)
			go func() {
				if saveFirst {
					_, err := service.IngestDurablePayload(ctx, body)
					secondDone <- err
				} else {
					_, err := service.SaveAutodartsMapping(ctx, req.SeasonID, id, "Aster Example")
					secondDone <- err
				}
			}()
			close(release)
			if err := <-firstDone; err != nil {
				t.Fatal(err)
			}
			if err := <-secondDone; err != nil {
				t.Fatal(err)
			}
			rows, err := store.ImportsBySource(ctx, "autodarts", "new-auto")
			if err != nil || len(rows) != 1 {
				t.Fatalf("rows: %+v %v", rows, err)
			}
			want := league.PendingResultStatusPending
			if saveFirst {
				want = league.PendingResultStatusConfirmed
			}
			if rows[0].Pending.Status != want {
				t.Fatalf("got %s want %s", rows[0].Pending.Status, want)
			}
			out, err := service.IngestDurablePayload(ctx, body)
			if err != nil || !out.Duplicate || out.Pending.Status != want {
				t.Fatalf("replay: %+v %v", out, err)
			}
		})
	}
}
