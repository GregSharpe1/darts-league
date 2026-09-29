package resultsrelay

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

// resultMessage mirrors the JSON payload produced by the score-scrape relay:
//
//	{"matchId":"...","player1":{"name":"TERRY","legsWon":2,"matchAverage":143.1},...}
type resultMessage struct {
	MatchID string        `json:"matchId"`
	Player1 playerMessage `json:"player1"`
	Player2 playerMessage `json:"player2"`
}

type playerMessage struct {
	Name         string  `json:"name"`
	LegsWon      int     `json:"legsWon"`
	MatchAverage float64 `json:"matchAverage"`
}

// Poller periodically checks the queue for new results and stores each one
// as a pending result awaiting admin confirmation.
type Poller struct {
	client   Client
	ingest   league.PendingResultService
	loc      *time.Location
	interval time.Duration
	now      func() time.Time
	logger   *log.Logger
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
				p.logger.Printf("results relay: poll failed: %v", err)
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
	messages, err := p.client.ReceiveMessages(ctx)
	if err != nil {
		return err
	}

	for _, message := range messages {
		if err := p.processMessage(ctx, message); err != nil {
			p.logger.Printf("results relay: leaving message for retry after processing error: %v", err)
			continue
		}
	}

	return nil
}

// processMessage parses and ingests a message. Malformed payloads are
// treated as processed (and deleted) since retrying them cannot succeed;
// transient storage errors are returned so the message stays in the queue.
func (p *Poller) processMessage(ctx context.Context, message Message) error {
	var parsed resultMessage
	if err := json.Unmarshal([]byte(message.Body), &parsed); err != nil {
		p.logger.Printf("results relay: discarding malformed message: %v", err)
		return nil
	}
	if parsed.Player1.Name == "" || parsed.Player2.Name == "" {
		p.logger.Printf("results relay: discarding message missing player names")
		return nil
	}

	playerOneAverage := parsed.Player1.MatchAverage
	playerTwoAverage := parsed.Player2.MatchAverage
	_, err := p.ingest.Ingest(ctx, parsed.MatchID, parsed.Player1.Name, parsed.Player1.LegsWon, &playerOneAverage, parsed.Player2.Name, parsed.Player2.LegsWon, &playerTwoAverage)
	if err != nil && !errors.Is(err, league.ErrDuplicateExternalMatch) {
		return err
	}
	return nil
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
