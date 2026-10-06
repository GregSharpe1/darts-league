package postgres

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/resultsrelay"
)

type importRelay struct {
	messages []resultsrelay.Message
	ack      func([]resultsrelay.Message) error
}

func (r *importRelay) ReceiveMessages(context.Context) ([]resultsrelay.Message, error) {
	return r.messages, nil
}
func (r *importRelay) AcknowledgeMessages(_ context.Context, m []resultsrelay.Message) error {
	return r.ack(m)
}

func TestRelayAcknowledgesCommittedVersionedImport(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	s, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer s.Close()
	id := "relay-" + strings.ReplaceAll(time.Now().Format("150405.000000000"), ".", "-")
	body := `{"schema_version":"autodarts.import.v1","source":"autodarts","external_match_id":"` + id + `","playedAt":null,"settings":{"base_score":501,"legs_to_win":3,"out":"double"},"completed":true,"players":[{"match_player_id":"a","account_id":null,"display_name":"A","legs_won":3,"stats":null},{"match_player_id":"b","account_id":null,"display_name":"B","legs_won":1,"stats":null}],"detail":null}`
	notifications := 0
	service := league.NewPendingResultService(s, league.NewResultService(s)).WithNotifier(func(context.Context, league.PendingResult) { notifications++ })
	acked := 0
	lostAck := errors.New("lost ack")
	relay := &importRelay{messages: []resultsrelay.Message{{MessageID: "versioned", ReceiptHandle: "receipt", Body: body}, {MessageID: "bad", ReceiptHandle: "bad-receipt", Body: `{}`}}}
	relay.ack = func(messages []resultsrelay.Message) error {
		if len(messages) != 1 || messages[0].MessageID != "versioned" {
			t.Fatal("invalid message acknowledged")
		}
		var count int
		if err := s.pool.QueryRow(ctx, `SELECT count(*) FROM pending_results WHERE external_match_id=$1`, id).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatal("ack preceded visible commit")
		}
		acked++
		return lostAck
	}
	poller := resultsrelay.NewPoller(relay, service, time.UTC, time.Minute, log.New(io.Discard, "", 0)).WithDurableIngest(func(ctx context.Context, m resultsrelay.Message) error {
		_, err := service.IngestDurablePayload(ctx, []byte(m.Body))
		return err
	})
	if err := poller.PollNow(ctx); !errors.Is(err, lostAck) {
		t.Fatalf("expected lost ack: %v", err)
	}
	relay.messages = relay.messages[:1]
	relay.messages[0].ReceiptHandle = "redelivered"
	lostAck = nil
	if err := poller.PollNow(ctx); err != nil {
		t.Fatal(err)
	}
	if acked != 2 || notifications != 1 {
		t.Fatalf("acks=%d notifications=%d", acked, notifications)
	}
	s.Close()
	if err := poller.PollNow(ctx); err == nil {
		t.Fatal("storage failure succeeded")
	}
	if acked != 2 {
		t.Fatal("storage failure acknowledged")
	}
}
