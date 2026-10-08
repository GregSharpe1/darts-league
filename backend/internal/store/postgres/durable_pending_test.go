package postgres

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/resultsrelay"
	"github.com/jackc/pgx/v5/pgxpool"
)

func isolatedLegacyStore(t *testing.T) *Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; real Postgres durability not exercised")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	var token [16]byte
	if _, err := rand.Read(token[:]); err != nil {
		t.Fatal(err)
	}
	schema := fmt.Sprintf("relay_test_%x", token)
	if _, err := admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	// This test drops only its own randomly named schema, never shared tables.
	t.Cleanup(func() {
		if _, err := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
			t.Error(err)
		}
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	store := &Store{pool: pool}
	t.Cleanup(store.Close)
	if err := store.pingAndMigrate(ctx); err != nil {
		t.Fatal(err)
	}
	return store
}

func TestDurableLegacyConcurrentReplayAndAdminRejection(t *testing.T) {
	store := isolatedLegacyStore(t)
	ctx := context.Background()
	var notices, fresh, duplicates atomic.Int32
	service := league.NewPendingResultService(store, league.NewResultService(store)).WithNotifier(func(context.Context, league.PendingResult) { notices.Add(1) })
	input := league.PendingResult{ExternalMatchID: "concurrent", PlayerOneName: "Alice", PlayerOneLegs: 3, PlayerTwoName: "Bob", PlayerTwoLegs: 2}
	var wg sync.WaitGroup
	for range 16 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			err := service.IngestDurable(ctx, input)
			switch {
			case err == nil:
				fresh.Add(1)
			case errors.Is(err, league.ErrDuplicateExternalMatch):
				duplicates.Add(1)
			default:
				t.Errorf("ingestion failed: %v", err)
			}
		}()
	}
	wg.Wait()
	if fresh.Load() != 1 || duplicates.Load() != 15 || notices.Load() != 1 {
		t.Fatalf("fresh=%d duplicate=%d notices=%d", fresh.Load(), duplicates.Load(), notices.Load())
	}
	items, err := service.List(ctx)
	if err != nil || len(items) != 1 {
		t.Fatalf("pending=%d: %v", len(items), err)
	}
	if items[0].PlayerOneAverage != nil {
		t.Fatal("null average changed")
	}
	if err := service.Reject(ctx, items[0].ID, "admin"); err != nil {
		t.Fatal(err)
	}
	if err := service.IngestDurable(ctx, input); !errors.Is(err, league.ErrDuplicateExternalMatch) {
		t.Fatalf("rejected replay: %v", err)
	}
	stored, err := store.GetPendingResult(ctx, items[0].ID)
	if err != nil || stored.Status != league.PendingResultStatusRejected || notices.Load() != 1 {
		t.Fatalf("rejected row changed: %v", err)
	}
	stored.Status = league.PendingResultStatusConfirmed
	if _, err := store.UpdatePendingResult(ctx, stored); err != nil {
		t.Fatal(err)
	}
	if err := service.IngestDurable(ctx, input); !errors.Is(err, league.ErrDuplicateExternalMatch) {
		t.Fatalf("confirmed replay: %v", err)
	}
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := service.IngestDurable(cancelled, input); err == nil || errors.Is(err, league.ErrDuplicateExternalMatch) {
		t.Fatalf("cancelled lookup treated as persisted: %v", err)
	}
}

type legacyDelivery struct{ acknowledged int }

func (d *legacyDelivery) ReceiveMessages(context.Context) ([]resultsrelay.Message, error) {
	return []resultsrelay.Message{{MessageID: "message", ReceiptHandle: "receipt", Body: `{"matchId":"real-db","player1":{"name":"Alice","legsWon":3,"matchAverage":null},"player2":{"name":"Bob","legsWon":1,"matchAverage":45.5}}`}}, nil
}
func (d *legacyDelivery) AcknowledgeMessages(_ context.Context, messages []resultsrelay.Message) error {
	d.acknowledged += len(messages)
	return nil
}

func TestDurableLegacyPollAcknowledgesCommittedRowsNotStorageFailure(t *testing.T) {
	store := isolatedLegacyStore(t)
	ctx := context.Background()
	service := league.NewPendingResultService(store, league.NewResultService(store))
	client := &legacyDelivery{}
	poller := resultsrelay.NewPoller(client, service, time.UTC, time.Minute, log.New(io.Discard, "", 0)).WithDurableIngest(service.IngestDurable)
	if err := poller.PollNow(ctx); err != nil {
		t.Fatal(err)
	}
	if client.acknowledged != 1 {
		t.Fatal("commit not acknowledged")
	}
	items, err := service.List(ctx)
	if err != nil || len(items) != 1 {
		t.Fatalf("missing committed row: %v", err)
	}
	if err := service.Reject(ctx, items[0].ID, "admin"); err != nil {
		t.Fatal(err)
	}
	if err := poller.PollNow(ctx); err != nil {
		t.Fatal(err)
	}
	if client.acknowledged != 2 {
		t.Fatal("persisted rejected duplicate not acknowledged")
	}
	store.Close()
	if err := poller.PollNow(ctx); err == nil {
		t.Fatal("storage failure hidden")
	}
	if client.acknowledged != 2 {
		t.Fatal("storage failure acknowledged")
	}
}
