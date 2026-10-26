package notifications

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/resultsrelay"
)

func TestAutoSubmitNotifierMessage(t *testing.T) {
	ctx := context.Background()
	store := league.NewMemoryStore()
	divisions, err := store.ReplaceDivisions(ctx, 1, []league.Division{{SeasonID: 1, Name: "Premier <A>", Slug: "premier"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []league.Player{{SeasonID: 1, DisplayName: "Alice", Nickname: "Ace"}, {SeasonID: 1, DisplayName: "Bob"}} {
		p.DivisionID = &divisions[0].ID
		if _, err := store.CreatePlayer(ctx, p); err != nil {
			t.Fatal(err)
		}
	}
	fixtures, err := store.CreateFixtures(ctx, []league.Fixture{{SeasonID: 1, DivisionID: divisions[0].ID, PlayerOneID: 1, PlayerTwoID: 2, GameVariant: "501", LegsToWin: 3, ScheduledAt: time.Now()}})
	if err != nil {
		t.Fatal(err)
	}
	poster := &stubPoster{}
	NewAutoSubmitNotifier(store, poster, " CADMIN ", "https://darts.example.com/")(ctx, league.Result{FixtureID: fixtures[0].ID, PlayerOneLegs: 3, PlayerTwoLegs: 1})
	if len(poster.messages) != 1 || poster.messages[0].channelID != "CADMIN" {
		t.Fatalf("messages %+v", poster.messages)
	}
	for _, want := range []string{"recorded automatically", "Ace (Alice) 3-1 Bob", "Division: Premier &lt;A&gt;", "<https://darts.example.com/admin/matches/1|View match>"} {
		if !strings.Contains(poster.messages[0].text, want) {
			t.Errorf("%q missing %q", poster.messages[0].text, want)
		}
	}
}

type failingClient struct{ messages []resultsrelay.Message }

func (c failingClient) ReceiveMessages(context.Context) ([]resultsrelay.Message, error) {
	return c.messages, nil
}
func (failingClient) AcknowledgeMessages(context.Context, []resultsrelay.Message) error { return nil }

func TestRelayFailureAlertsOncePerMessageAndSurvivesSlackFailure(t *testing.T) {
	for _, slackErr := range []error{nil, errors.New("slack down")} {
		store := league.NewMemoryStore()
		service := league.NewPendingResultService(store, league.NewResultService(store))
		poster := &stubPoster{err: slackErr}
		message := resultsrelay.Message{MessageID: "m-1", ReceiptHandle: "r-1", Body: `{"external_match_id":"ad-42","secret":"x"`}
		poller := resultsrelay.NewPoller(failingClient{[]resultsrelay.Message{message}}, service, time.UTC, time.Minute, nil).
			WithDurableIngest(func(ctx context.Context, m resultsrelay.Message) error {
				_, err := service.IngestPayload(ctx, []byte(m.Body))
				return err
			}).
			WithFailureNotifier(NewRelayFailureNotifier(poster, "CADMIN", "https://darts.example.com"))
		for range 2 {
			if err := poller.PollNow(context.Background()); err == nil {
				t.Fatal("expected processing failure to be returned")
			}
		}
		if len(poster.messages) != 1 {
			t.Fatalf("expected one alert across redeliveries, got %d", len(poster.messages))
		}
		text := poster.messages[0].text
		for _, want := range []string{"Autodarts import failed", "payload rejected as invalid", "Relay message: m-1"} {
			if !strings.Contains(text, want) {
				t.Errorf("%q missing %q", text, want)
			}
		}
		if strings.Contains(text, "secret") {
			t.Fatal("alert leaked payload")
		}
	}
}

func TestRelayFailureNotifierIncludesMatchID(t *testing.T) {
	poster := &stubPoster{}
	NewRelayFailureNotifier(poster, "CADMIN", "")(context.Background(), resultsrelay.Message{Body: `{"external_match_id":"ad-42"}`}, "reason")
	if len(poster.messages) != 1 || !strings.Contains(poster.messages[0].text, "Autodarts match: ad-42") || strings.Contains(poster.messages[0].text, "<") {
		t.Fatalf("%+v", poster.messages)
	}
}
