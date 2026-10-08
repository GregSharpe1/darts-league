package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/config"
	"github.com/greg/darts-league/backend/internal/league"
)

func TestResultPollerDisabledWithoutEndpoint(t *testing.T) {
	stop, poll := startResultPoller(config.Config{}, nil, league.ResultService{}, time.Now)
	defer stop()
	if poll != nil {
		t.Fatal("empty endpoint enabled polling")
	}
}

func TestResultPollerMemoryFallbackCannotAcknowledge(t *testing.T) {
	var acks atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			acks.Add(1)
			t.Error("memory attempted ack")
			return
		}
		if _, err := w.Write([]byte(`{"messages":[{"messageId":"id","receiptHandle":"receipt","body":"{\"matchId\":\"id\",\"player1\":{\"name\":\"Alice\",\"legsWon\":3},\"player2\":{\"name\":\"Bob\",\"legsWon\":1}}"}]}`)); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	store := league.NewMemoryStore()
	stop, poll := startResultPoller(config.Config{ResultsEndpoint: server.URL, Timezone: "Europe/London", ResultsPollInterval: time.Hour}, store, league.NewResultService(store), time.Now)
	defer stop()
	if poll == nil {
		t.Fatal("configured relay disabled")
	}
	if err := poll(context.Background()); err == nil {
		t.Fatal("receipt-bearing memory ingestion accepted")
	}
	if acks.Load() != 0 {
		t.Fatal("memory acknowledged")
	}
	items, err := store.ListPendingResults(context.Background(), league.PendingResultStatusPending)
	if err != nil || len(items) != 0 {
		t.Fatalf("memory mutated: %v", err)
	}
}
