package league

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sync"
	"testing"
)

func TestMemoryStoreCannotClaimDurableIngestion(t *testing.T) {
	store := NewMemoryStore()
	service := NewPendingResultService(store, NewResultService(store))
	_, err := service.IngestDurablePayload(context.Background(), []byte(`{"matchId":"memory","player1":{"name":"A","legsWon":3},"player2":{"name":"B","legsWon":1}}`))
	if !errors.Is(err, ErrDurableStoreRequired) {
		t.Fatalf("got %v", err)
	}
	pending, err := service.List(context.Background())
	if err != nil || len(pending) != 0 {
		t.Fatal("memory fallback ingested durable message")
	}
}

func TestMemoryConcurrentDetailedImports(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	service := NewPendingResultService(store, NewResultService(store))
	b, err := os.ReadFile("../../../docs/autodarts/examples-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var examples map[string]json.RawMessage
	if err := json.Unmarshal(b, &examples); err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	ids := make(chan int64, 12)
	for range 12 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			outcome, err := service.IngestPayload(ctx, examples["producer"])
			if err != nil {
				t.Error(err)
				return
			}
			ids <- outcome.Pending.ID
		}()
	}
	wg.Wait()
	close(ids)
	var id int64
	for next := range ids {
		if id != 0 && id != next {
			t.Fatal("concurrent duplicate")
		}
		id = next
	}
	record, err := service.ImportDetail(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if record.Import.Detail.Legs[0].Visits[0].PlayerID != "seat-a" {
		t.Fatal("lost linkage")
	}
	record.Import.Detail.Legs[0].Visits[0].PlayerID = "mutated"
	pending, err := store.GetPendingResult(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	*pending.PlayerOneAverage = 0
	record, err = service.ImportDetail(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if record.Import.Detail.Legs[0].Visits[0].PlayerID != "seat-a" || *record.Pending.PlayerOneAverage == 0 {
		t.Fatal("read mutated original")
	}
}

func TestMemoryDetailedImportReplayAndOriginal(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	service := NewPendingResultService(store, NewResultService(store))
	body := []byte(`{"matchId":"offline","player1":{"name":"A","legsWon":3,"matchAverage":60},"player2":{"name":"B","legsWon":1}}`)
	first, err := service.IngestPayload(ctx, body)
	if err != nil {
		t.Fatal(err)
	}
	*first.Pending.PlayerOneAverage = 1
	record, err := service.ImportDetail(ctx, first.Pending.ID)
	if err != nil {
		t.Fatal(err)
	}
	if *record.Import.Players[0].Average() != 60 || *record.Pending.PlayerOneAverage != 60 {
		t.Fatal("mutable original")
	}
	if err := service.Reject(ctx, first.Pending.ID, "admin"); err != nil {
		t.Fatal(err)
	}
	replay, err := service.IngestPayload(ctx, body)
	if err != nil {
		t.Fatal(err)
	}
	if !replay.Duplicate || replay.Pending.ID != first.Pending.ID || replay.Pending.Status != PendingResultStatusRejected {
		t.Fatal("reopened replay")
	}
	changed, err := service.IngestPayload(ctx, []byte(`{"matchId":"offline","player1":{"name":"A","legsWon":3},"player2":{"name":"B","legsWon":2}}`))
	if err != nil {
		t.Fatal(err)
	}
	if !changed.Changed || changed.Duplicate || changed.Pending.ID == first.Pending.ID {
		t.Fatal("changed import overwrote original")
	}
}
