package postgres

import (
	"context"
	"encoding/json"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

func TestDurableImportConcurrentReplay(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	store, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	b, err := os.ReadFile("../../../../docs/autodarts/examples-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var examples map[string]json.RawMessage
	if err := json.Unmarshal(b, &examples); err != nil {
		t.Fatal(err)
	}
	body := strings.ReplaceAll(string(examples["producer"]), "fictional-match-001", "concurrent-"+strings.ReplaceAll(time.Now().Format("20060102150405.000000000"), ".", "-"))
	parsed, err := autodarts.Parse([]byte(body))
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	ids := make(chan int64, 12)
	created := make(chan bool, 12)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := store.InsertImport(ctx, parsed, time.Now())
			if err != nil {
				t.Error(err)
				return
			}
			ids <- result.Pending.ID
			created <- !result.Duplicate
		}()
	}
	wg.Wait()
	close(ids)
	close(created)
	var id int64
	for got := range ids {
		if id != 0 && got != id {
			t.Fatal("duplicate rows")
		}
		id = got
	}
	count := 0
	for fresh := range created {
		if fresh {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("created %d times", count)
	}
	detail, err := store.GetImport(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Import.Players[0].ID != "seat-a" || detail.Import.Detail.Legs[0].Visits[0].PlayerID != "seat-a" {
		t.Fatal("lost player/leg linkage")
	}
	if _, err := store.pool.Exec(ctx, `UPDATE pending_results SET source_payload='{}' WHERE id=$1`, id); err == nil {
		t.Fatal("original is mutable")
	}
	changedPayload, err := autodarts.Parse([]byte(strings.ReplaceAll(body, "60.12", "61.12")))
	if err != nil {
		t.Fatal(err)
	}
	changed, err := store.InsertImport(ctx, changedPayload, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if changed.Duplicate || changed.Pending.ID == id || !changed.Changed {
		t.Fatal("changed content replaced original")
	}
	if _, err := store.pool.Exec(ctx, `UPDATE pending_results SET status='rejected' WHERE id=$1`, id); err != nil {
		t.Fatal(err)
	}
	replay, err := store.InsertImport(ctx, parsed, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Duplicate || replay.Pending.ID != id || replay.Pending.Status != "rejected" {
		t.Fatal("reopened rejected import")
	}
}
