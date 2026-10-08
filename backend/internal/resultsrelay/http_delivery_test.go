package resultsrelay

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

type failingStore struct {
	league.Store
	fail bool
}

func (s *failingStore) CreatePendingResult(ctx context.Context, pending league.PendingResult) (league.PendingResult, error) {
	if s.fail && pending.ExternalMatchID == "retry" {
		return league.PendingResult{}, errors.New("database write failed")
	}
	return s.Store.CreatePendingResult(ctx, pending)
}

func TestHTTPDeliveryRetriesStorageAndLostAckWithoutDuplicateImports(t *testing.T) {
	// Given a real ingestion service over a test store and an HTTP receive/ack queue.
	store := &failingStore{Store: league.NewMemoryStore(), fail: true}
	service := league.NewPendingResultService(store, league.NewResultService(store))
	var mu sync.Mutex
	queue := []Message{delivery("good"), delivery("retry")}
	loseAck := true
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		defer mu.Unlock()
		if r.Method == http.MethodGet {
			if err := json.NewEncoder(w).Encode(httpResponse{Messages: queue}); err != nil {
				t.Error(err)
			}
			return
		}
		var ack httpResponse
		if err := json.NewDecoder(r.Body).Decode(&ack); err != nil {
			t.Error(err)
			w.WriteHeader(400)
			return
		}
		for _, message := range ack.Messages {
			exists, err := store.PendingResultExistsForExternalMatch(r.Context(), message.MessageID)
			if err != nil || !exists {
				t.Errorf("ack before persistence: %v", err)
			}
		}
		if loseAck {
			w.WriteHeader(503)
			return
		}
		ids := []string{}
		for _, message := range ack.Messages {
			ids = append(ids, message.MessageID)
			for i, queued := range queue {
				if queued.ReceiptHandle == message.ReceiptHandle {
					queue = append(queue[:i], queue[i+1:]...)
					break
				}
			}
		}
		if err := json.NewEncoder(w).Encode(struct {
			Acknowledged []string `json:"acknowledged"`
			Failed       []string `json:"failed"`
		}{ids, []string{}}); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL + "/results")
	if err != nil {
		t.Fatal(err)
	}
	legacy := NewPoller(client, service, time.UTC, time.Minute, log.New(io.Discard, "", 0))
	// The memory store models the durable callback only for this offline test.
	poller := legacy.WithDurableIngest(func(ctx context.Context, pending league.PendingResult) error {
		_, err := service.Ingest(ctx, pending.ExternalMatchID, pending.PlayerOneName, pending.PlayerOneLegs, pending.PlayerOneAverage, pending.PlayerTwoName, pending.PlayerTwoLegs, pending.PlayerTwoAverage)
		return err
	})
	// When the first poll loses its acknowledgement and one database write fails.
	if err := poller.PollNow(context.Background()); err == nil {
		t.Fatal("partial failures must surface")
	}
	mu.Lock()
	if len(queue) != 2 {
		t.Error("failed messages disappeared")
	}
	store.fail, loseAck = false, false
	for i := range queue {
		queue[i].ReceiptHandle += "-redelivery"
	}
	mu.Unlock()
	if err := poller.PollNow(context.Background()); err != nil {
		t.Fatal(err)
	}
	// Then redelivery uses new receipts and each external match exists once.
	pending, err := service.List(context.Background())
	if err != nil || len(pending) != 2 {
		t.Fatalf("imports=%d err=%v", len(pending), err)
	}
	mu.Lock()
	defer mu.Unlock()
	if len(queue) != 0 {
		t.Fatalf("unacknowledged imports: %d", len(queue))
	}
}
