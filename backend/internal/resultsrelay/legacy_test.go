package resultsrelay

import (
	"context"
	"errors"
	"io"
	"log"
	"strings"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

func TestLegacyBoundaryRejectsBeforeDurableCallback(t *testing.T) {
	valid := delivery("id").Body
	for _, body := range []string{
		strings.Replace(valid, `"matchId"`, `"schemaVersion":1,"matchId"`, 1),
		strings.Replace(valid, `"matchId"`, `"legs":[],"matchId"`, 1),
		strings.Replace(valid, `"legsWon":2`, `"legsWon":null`, 1),
		strings.Replace(valid, `"legsWon":2`, `"legsWon":-1`, 1),
		strings.Replace(valid, `"legsWon":2`, `"legsWon":3`, 1),
		strings.Replace(valid, `"legsWon":2`, `"legsWon":2147483648`, 1),
		strings.Replace(valid, `"legsWon":2`, `"legsWon":2.5`, 1),
		strings.Replace(valid, `"legsWon":2`, `"matchAverage":181,"legsWon":2`, 1),
		strings.Replace(valid, `"legsWon":2`, `"matchAverage":1e999,"legsWon":2`, 1),
		strings.Replace(valid, `"Alice"`, `"A\nB"`, 1),
		strings.Replace(valid, `"Alice"`, `"`+strings.Repeat("a", 61)+`"`, 1),
		strings.Replace(valid, `"matchId":"id"`, `"matchId":" "`, 1),
		valid + `{}`,
		strings.Repeat(" ", 256*1024+1),
	} {
		t.Run(body[:min(len(body), 70)], func(t *testing.T) {
			message := delivery("id")
			message.Body = body
			client := &deliveryClient{messages: []Message{message}}
			poller := NewPoller(client, league.PendingResultService{}, time.UTC, time.Minute, log.New(io.Discard, "", 0)).WithDurableIngest(
				func(context.Context, league.PendingResult) error {
					t.Fatal("invalid delivery reached storage")
					return nil
				})
			if err := poller.PollNow(context.Background()); !errors.Is(err, ErrInvalidDelivery) || len(client.acked) != 0 {
				t.Fatalf("invalid delivery acknowledged: %v", err)
			}
		})
	}
}

func TestLegacyReceiptlessPreservesNullableAverage(t *testing.T) {
	store := league.NewMemoryStore()
	service := league.NewPendingResultService(store, league.NewResultService(store))
	client := &deliveryClient{messages: []Message{{Body: delivery("dev").Body}}}
	poller := NewPoller(client, service, time.UTC, time.Minute, log.New(io.Discard, "", 0))
	if err := poller.PollNow(context.Background()); err != nil {
		t.Fatal(err)
	}
	pending, err := service.List(context.Background())
	if err != nil || len(pending) != 1 || pending[0].PlayerOneAverage != nil || len(client.acked) != 0 {
		t.Fatalf("receiptless nullable average lost: %+v, %v", pending, err)
	}
}
