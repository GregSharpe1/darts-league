package postgres

import (
	"context"
	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/resultsrelay"
	"io"
	"log"
	"testing"
	"time"
)

func TestAutoSubmitRelayAcknowledgesOnlyCommittedApproval(t *testing.T) {
	ctx := context.Background()
	store, s, body, req := autoPG(t)
	if _, err := store.pool.Exec(ctx, `CREATE FUNCTION fail_auto_ack() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'audit unavailable'; END $$; CREATE TRIGGER fail_auto_ack BEFORE INSERT ON admin_audit_log FOR EACH ROW EXECUTE FUNCTION fail_auto_ack()`); err != nil {
		t.Fatal(err)
	}
	acks, notifications := 0, 0
	s = s.WithNotifier(func(context.Context, league.PendingResult) { notifications++ })
	relay := &importRelay{messages: []resultsrelay.Message{{MessageID: "new", ReceiptHandle: "receipt", Body: string(body)}}}
	relay.ack = func(messages []resultsrelay.Message) error {
		acks++
		rows, err := store.ImportsBySource(ctx, "autodarts", "new-auto")
		if err != nil || len(rows) != 1 || rows[0].Pending.Status != league.PendingResultStatusConfirmed {
			t.Fatalf("ack before approval commit: %+v %v", rows, err)
		}
		audits, err := store.ListAuditLogsBySeason(ctx, req.SeasonID)
		if err != nil || len(audits) != 1 || audits[0].Actor != "autodarts:auto-submit" {
			t.Fatalf("ack before audit commit: %+v %v", audits, err)
		}
		return nil
	}
	poller := resultsrelay.NewPoller(relay, s, time.UTC, time.Minute, log.New(io.Discard, "", 0)).WithDurableIngest(func(ctx context.Context, message resultsrelay.Message) error {
		_, err := s.IngestDurablePayload(ctx, []byte(message.Body))
		return err
	})
	if err := poller.PollNow(ctx); err == nil || acks != 0 || notifications != 0 {
		t.Fatalf("failed audit acknowledged: %d %d %v", acks, notifications, err)
	}
	if _, err := store.pool.Exec(ctx, `DROP TRIGGER fail_auto_ack ON admin_audit_log`); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		if err := poller.PollNow(ctx); err != nil {
			t.Fatal(err)
		}
	}
	if acks != 2 || notifications != 0 {
		t.Fatalf("acks=%d notifications=%d", acks, notifications)
	}
}
