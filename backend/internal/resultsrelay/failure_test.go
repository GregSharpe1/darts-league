package resultsrelay

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

func TestPollMemoryCallbackRefusesReceiptsAndLogsNoPayload(t *testing.T) {
	store := league.NewMemoryStore()
	service := league.NewPendingResultService(store, league.NewResultService(store))
	client := &deliveryClient{messages: []Message{delivery("secret-match")}}
	var logs bytes.Buffer
	poller := NewPoller(client, service, time.UTC, time.Minute, log.New(&logs, "", 0)).WithDurableIngest(service.IngestDurable)
	if err := poller.PollNow(context.Background()); !errors.Is(err, league.ErrDurablePendingStoreRequired) {
		t.Fatalf("memory callback accepted: %v", err)
	}
	if len(client.acked) != 0 {
		t.Fatal("memory acknowledged")
	}
	for _, secret := range []string{"secret-match", "receipt-", "Alice", "Bob", "matchId"} {
		if strings.Contains(logs.String(), secret) {
			t.Fatal("payload/receipt logged")
		}
	}
}

func TestPollRejectsIncompleteReceipts(t *testing.T) {
	for _, receipt := range []Message{
		{MessageID: "id"}, {ReceiptHandle: "receipt"},
		{MessageID: "id", ReceiptHandle: " "},
		{MessageID: "id", ReceiptHandle: strings.Repeat("r", 2049)},
	} {
		receipt.Body = delivery("id").Body
		client := &deliveryClient{messages: []Message{receipt}}
		poller := NewPoller(client, league.PendingResultService{}, time.UTC, time.Minute, log.Default()).WithDurableIngest(func(context.Context, league.PendingResult) error {
			t.Fatal("invalid receipt reached storage")
			return nil
		})
		if err := poller.PollNow(context.Background()); !errors.Is(err, ErrInvalidDelivery) || len(client.acked) != 0 {
			t.Fatalf("invalid receipt acknowledged: %v", err)
		}
	}
}

func TestClientTimeoutRetainsReceiveAndAck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if _, err := io.Copy(io.Discard, r.Body); err != nil {
			return
		}
		<-r.Context().Done()
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	client.(*httpClient).client.Timeout = 20 * time.Millisecond
	if _, err := client.ReceiveMessages(context.Background()); err == nil {
		t.Fatal("receive timeout hidden")
	}
	if err := client.AcknowledgeMessages(context.Background(), []Message{delivery("id")}); err == nil {
		t.Fatal("ack timeout hidden")
	}
}

func TestClientBoundsAckResponseAndSurfacesReceiveFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		if _, err := w.Write([]byte(strings.Repeat(" ", 32*1024+1))); err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client, err := NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := client.ReceiveMessages(context.Background()); err == nil {
		t.Fatal("receive failure hidden")
	}
	if err := client.AcknowledgeMessages(context.Background(), []Message{delivery("id")}); !errors.Is(err, ErrInvalidDelivery) {
		t.Fatalf("unbounded ack: %v", err)
	}
}

func TestPollPassesBoundedContextToStorage(t *testing.T) {
	client := &deliveryClient{messages: []Message{delivery("id")}}
	poller := NewPoller(client, league.PendingResultService{}, time.UTC, time.Minute, log.Default()).WithDurableIngest(func(ctx context.Context, _ league.PendingResult) error {
		deadline, ok := ctx.Deadline()
		if !ok || time.Until(deadline) > 45*time.Second {
			t.Fatal("unbounded ingestion")
		}
		return context.DeadlineExceeded
	})
	if err := poller.PollNow(context.Background()); !errors.Is(err, context.DeadlineExceeded) || len(client.acked) != 0 {
		t.Fatalf("storage timeout acknowledged: %v", err)
	}
}
