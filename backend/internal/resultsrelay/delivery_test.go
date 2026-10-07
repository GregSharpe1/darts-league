package resultsrelay

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"os"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/autodarts"
	"github.com/greg/darts-league/backend/internal/league"
)

type deliveryClient struct {
	messages []Message
	acked    []Message
	ackErr   error
}

func TestReceiptlessDetailRemainsBlockedInMemory(t *testing.T) {
	b, err := os.ReadFile("../../../docs/autodarts/examples-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var examples map[string]json.RawMessage
	if err := json.Unmarshal(b, &examples); err != nil {
		t.Fatal(err)
	}
	var payload autodarts.Payload
	if err := json.Unmarshal(examples["producer"], &payload); err != nil {
		t.Fatal(err)
	}
	payload.Detail.Legs[0].Visits[0].EndRemaining = 2
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	client := &deliveryClient{messages: []Message{{Body: string(body)}}}
	store := league.NewMemoryStore()
	service := league.NewPendingResultService(store, league.NewResultService(store))
	poller := NewPoller(client, service, time.UTC, time.Minute, log.New(io.Discard, "", 0))
	if err := poller.PollNow(context.Background()); err != nil {
		t.Fatal(err)
	}
	blocked, err := store.ListPendingResults(context.Background(), league.PendingResultStatusReviewBlocked)
	if err != nil || len(blocked) != 1 || len(client.acked) != 0 {
		t.Fatalf("lost blocked detail: %v %+v", err, blocked)
	}
	if _, err := service.Confirm(context.Background(), blocked[0].ID, 1, 2, "admin"); !errors.Is(err, league.ErrPendingResultNotPending) {
		t.Fatal("confirmed contradictory detail", err)
	}
}

func TestMemoryDurableCallbackNeverAcknowledges(t *testing.T) {
	client := &deliveryClient{messages: []Message{delivery("memory")}}
	store := league.NewMemoryStore()
	service := league.NewPendingResultService(store, league.NewResultService(store))
	poller := NewPoller(client, service, time.UTC, time.Minute, log.New(io.Discard, "", 0)).WithDurableIngest(func(ctx context.Context, m Message) error {
		_, err := service.IngestDurablePayload(ctx, []byte(m.Body))
		return err
	})
	if err := poller.PollNow(context.Background()); !errors.Is(err, league.ErrDurableStoreRequired) || len(client.acked) != 0 {
		t.Fatal("memory acknowledged", err)
	}
}

func (c *deliveryClient) ReceiveMessages(context.Context) ([]Message, error) { return c.messages, nil }
func (c *deliveryClient) AcknowledgeMessages(_ context.Context, messages []Message) error {
	c.acked = append(c.acked, messages...)
	return c.ackErr
}

func delivery(id string) Message {
	return Message{MessageID: id, ReceiptHandle: "receipt-" + id,
		Body: `{"matchId":"` + id + `","player1":{"name":"Alice","legsWon":3},"player2":{"name":"Bob","legsWon":2}}`}
}

func TestPollRequiresExplicitDurableBoundaryBeforeAcknowledging(t *testing.T) {
	client := &deliveryClient{messages: []Message{delivery("id")}}
	store := league.NewMemoryStore()
	service := league.NewPendingResultService(store, league.NewResultService(store))
	poller := NewPoller(client, service, time.UTC, time.Minute, log.New(io.Discard, "", 0))
	err := poller.PollNow(context.Background())
	if !errors.Is(err, ErrDurableIngestRequired) || len(client.acked) != 0 {
		t.Fatalf("unsafe poll: %v, %+v", err, client.acked)
	}
	pending, err := service.List(context.Background())
	if err != nil || len(pending) != 0 {
		t.Fatalf("unconfigured poll must not import into fallback memory: %v", err)
	}
}

func TestPollAcknowledgesOnlyCommittedMessagesInPartialBatch(t *testing.T) {
	client := &deliveryClient{messages: []Message{delivery("good"), delivery("db-failure"), delivery("duplicate")}}
	storageErr := errors.New("storage unavailable")
	committed := map[string]bool{"duplicate": true}
	poller := NewPoller(client, league.PendingResultService{}, time.UTC, time.Minute, log.New(io.Discard, "", 0)).WithDurableIngest(
		func(_ context.Context, message Message) error {
			if len(client.acked) != 0 {
				t.Fatal("ack preceded completion of ingestion")
			}
			if message.MessageID == "db-failure" {
				return storageErr
			}
			if committed[message.MessageID] {
				return league.ErrDuplicateExternalMatch
			}
			committed[message.MessageID] = true
			return nil
		})
	err := poller.PollNow(context.Background())
	if !errors.Is(err, storageErr) {
		t.Fatalf("missing partial failure: %v", err)
	}
	if len(client.acked) != 2 || client.acked[0].MessageID != "good" || client.acked[1].MessageID != "duplicate" {
		t.Fatalf("unsafe ack: %+v", client.acked)
	}
}

func TestPollRedeliveryAfterLostAckCreatesOneImport(t *testing.T) {
	client := &deliveryClient{messages: []Message{delivery("id")}, ackErr: errors.New("lost ack")}
	imports := 0
	poller := NewPoller(client, league.PendingResultService{}, time.UTC, time.Minute, log.New(io.Discard, "", 0)).WithDurableIngest(
		func(context.Context, Message) error {
			if imports > 0 {
				return league.ErrDuplicateExternalMatch
			}
			imports++
			return nil
		})
	if err := poller.PollNow(context.Background()); !errors.Is(err, client.ackErr) {
		t.Fatalf("lost ack not signalled: %v", err)
	}
	client.ackErr = nil
	client.messages[0].ReceiptHandle = "new-receipt"
	if err := poller.PollNow(context.Background()); err != nil {
		t.Fatal(err)
	}
	if imports != 1 || len(client.acked) != 2 || client.acked[1].ReceiptHandle != "new-receipt" {
		t.Fatalf("unsafe replay: imports=%d acks=%+v", imports, client.acked)
	}
}

func TestPollLeavesMalformedMessagesForRedrive(t *testing.T) {
	for _, body := range []string{"{", `{}`, `{"matchId":"","player1":{"name":"A"},"player2":{"name":"B"}}`} {
		t.Run("malformed", func(t *testing.T) {
			message := delivery("bad")
			message.Body = body
			client := &deliveryClient{messages: []Message{message}}
			poller := NewPoller(client, league.PendingResultService{}, time.UTC, time.Minute, log.New(io.Discard, "", 0)).WithDurableIngest(
				func(_ context.Context, message Message) error {
					_, err := autodarts.Parse([]byte(message.Body))
					return err
				})
			if err := poller.PollNow(context.Background()); err == nil || len(client.acked) != 0 {
				t.Fatalf("malformed message discarded: %v", err)
			}
		})
	}
}
