package resultsrelay

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

// Poller periodically checks the queue for new results and stores each one
// as a pending result awaiting admin confirmation.
type Poller struct {
	client        Client
	ingest        league.PendingResultService
	loc           *time.Location
	interval      time.Duration
	now           func() time.Time
	logger        *log.Logger
	durableIngest func(context.Context, Message) error
}

var ErrDurableIngestRequired = errors.New("relay acknowledgement requires a durable ingestion callback")

// WithDurableIngest returns a configured copy. The callback must validate and
// commit the import before returning nil, or verify a persisted duplicate before
// returning league.ErrDuplicateExternalMatch. It must honor context cancellation.
// Never wire this to the development in-memory store.
func (p *Poller) WithDurableIngest(ingest func(context.Context, Message) error) *Poller {
	configured := *p
	configured.durableIngest = ingest
	return &configured
}

// PollWindow describes the weekday/hour window during which polling occurs.
type PollWindow struct {
	StartHour int // inclusive, 24h local time
	EndHour   int // exclusive, 24h local time
}

// DefaultPollWindow polls on weekdays between 9am and 5pm.
var DefaultPollWindow = PollWindow{StartHour: 9, EndHour: 17}

func NewPoller(client Client, ingest league.PendingResultService, loc *time.Location, interval time.Duration, logger *log.Logger) *Poller {
	if loc == nil {
		loc = time.UTC
	}
	if interval <= 0 {
		interval = 15 * time.Minute
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Poller{client: client, ingest: ingest, loc: loc, interval: interval, now: time.Now, logger: logger}
}

// Run blocks, polling the queue on the configured interval whenever the
// current time falls within the poll window, until ctx is cancelled.
func (p *Poller) Run(ctx context.Context) {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		if ShouldPoll(p.now(), p.loc, DefaultPollWindow) {
			if err := p.pollOnce(ctx); err != nil {
				p.logger.Printf("results relay: poll failed; unacknowledged messages retained for retry")
			}
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

// PollNow immediately reads and processes the relay, regardless of the
// scheduled polling window.
func (p *Poller) PollNow(ctx context.Context) error {
	return p.pollOnce(ctx)
}

func (p *Poller) pollOnce(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	messages, err := p.client.ReceiveMessages(ctx)
	if err != nil {
		return err
	}

	var failures []error
	var committed []Message
	for _, message := range messages {
		if err := p.processMessage(ctx, message); err != nil {
			failures = append(failures, err)
			continue
		}
		if message.validReceipt() {
			committed = append(committed, message)
		}
	}
	acknowledged := 0
	if len(committed) > 0 {
		if err := p.client.AcknowledgeMessages(ctx, committed); err != nil {
			failures = append(failures, err)
		} else {
			acknowledged = len(committed)
		}
	}
	p.logger.Printf("results relay: received=%d committed=%d acknowledged=%d failures=%d", len(messages), len(committed), acknowledged, len(failures))
	return errors.Join(failures...)
}

// Invalid messages remain in SQS for bounded retry and native DLQ redrive.
func (p *Poller) processMessage(ctx context.Context, message Message) error {
	if message.Rejection != "" || len(message.Body) > 256*1024 {
		return ErrInvalidDelivery
	}
	if message.MessageID != "" || message.ReceiptHandle != "" {
		if !message.validReceipt() {
			return ErrInvalidDelivery
		}
		if p.durableIngest == nil {
			return ErrDurableIngestRequired
		}
		err := p.durableIngest(ctx, message)
		if errors.Is(err, league.ErrDuplicateExternalMatch) {
			return nil
		}
		return err
	}

	// Legacy responses have no receipt and cannot be acknowledged by this poller.
	if p.durableIngest != nil {
		return p.durableIngest(ctx, message)
	}
	_, err := p.ingest.IngestPayload(ctx, []byte(message.Body))
	return err
}

// ShouldPoll reports whether now (evaluated in loc) falls on a weekday within
// the given poll window.
func ShouldPoll(now time.Time, loc *time.Location, window PollWindow) bool {
	if loc == nil {
		loc = time.UTC
	}
	local := now.In(loc)
	if local.Weekday() == time.Saturday || local.Weekday() == time.Sunday {
		return false
	}
	return local.Hour() >= window.StartHour && local.Hour() < window.EndHour
}
