package notifications

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/resultsrelay"
	"github.com/greg/darts-league/backend/internal/slack"
)

func TestPendingResultNotificationFromRepeatedRelayPolls(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, err := w.Write([]byte(`{"messages":[{"body":"{\"matchId\":\"match-1\",\"player1\":{\"name\":\"Alice & <@U123>\",\"legsWon\":3},\"player2\":{\"name\":\"Bob\",\"legsWon\":2}}"}]}`))
		if err != nil {
			t.Error(err)
		}
	}))
	defer server.Close()
	client, err := resultsrelay.NewClient(server.URL)
	if err != nil {
		t.Fatal(err)
	}
	store := league.NewMemoryStore()
	poster := &stubPoster{}
	service := league.NewPendingResultService(store, league.NewResultService(store)).WithNotifier(
		NewPendingResultNotifier(poster, " CADMIN ", " https://darts.example.com/ "),
	)
	poller := resultsrelay.NewPoller(client, service, time.UTC, time.Minute, nil)

	for range 2 {
		if err := poller.PollNow(context.Background()); err != nil {
			t.Fatal(err)
		}
	}

	if len(poster.messages) != 1 {
		t.Fatalf("expected one notification across repeated polls, got %d", len(poster.messages))
	}
	message := poster.messages[0]
	if message.channelID != "CADMIN" {
		t.Fatalf("unexpected channel: %q", message.channelID)
	}
	for _, want := range []string{"Alice &amp; &lt;@U123&gt; 3-2 Bob", "<https://darts.example.com/admin/pending-results|Review pending results>"} {
		if !strings.Contains(message.text, want) {
			t.Errorf("message %q missing %q", message.text, want)
		}
	}
}

func TestPendingResultNotifierConfigurationAndDeliveryFailure(t *testing.T) {
	t.Parallel()
	for _, tc := range []struct {
		name    string
		channel string
		baseURL string
		err     error
		posts   int
	}{
		{name: "missing channel", channel: " ", posts: 0},
		{name: "missing base URL", channel: "CADMIN", posts: 1},
		{name: "Slack failure", channel: "CADMIN", err: errors.New("Slack unavailable"), posts: 1},
		{name: "disabled Slack", channel: "CADMIN", err: slack.ErrDisabled, posts: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			poster := &stubPoster{err: tc.err}
			store := league.NewMemoryStore()
			service := league.NewPendingResultService(store, league.NewResultService(store)).WithNotifier(
				NewPendingResultNotifier(poster, tc.channel, tc.baseURL),
			)

			pending, err := service.Ingest(context.Background(), "match-1", "Alice", 3, nil, "Bob", 0, nil)

			if err != nil || pending.ID == 0 {
				t.Fatalf("notification failure must not fail ingestion: pending=%+v err=%v", pending, err)
			}
			if len(poster.messages) != tc.posts {
				t.Fatalf("expected %d post attempts, got %d", tc.posts, len(poster.messages))
			}
			if tc.posts > 0 && strings.Contains(poster.messages[0].text, "<") {
				t.Fatal("missing base URL must not produce a broken Slack link")
			}
		})
	}
}

type pendingWriteFailureStore struct {
	league.Store
}

func (pendingWriteFailureStore) CreatePendingResult(context.Context, league.PendingResult) (league.PendingResult, error) {
	return league.PendingResult{}, errors.New("storage unavailable")
}

func TestPendingResultNotificationSkippedWhenStorageFails(t *testing.T) {
	t.Parallel()
	store := pendingWriteFailureStore{Store: league.NewMemoryStore()}
	poster := &stubPoster{}
	service := league.NewPendingResultService(store, league.NewResultService(store)).WithNotifier(
		NewPendingResultNotifier(poster, "CADMIN", "https://darts.example.com"),
	)

	_, err := service.Ingest(context.Background(), "match-1", "Alice", 3, nil, "Bob", 1, nil)

	if err == nil || len(poster.messages) != 0 {
		t.Fatalf("expected storage error without notification, err=%v messages=%v", err, poster.messages)
	}
}
