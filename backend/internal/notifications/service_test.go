package notifications

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

func TestRegistrationNotifierPostsAdminSignupMessage(t *testing.T) {
	t.Parallel()

	poster := &stubPoster{}
	loc, _ := time.LoadLocation("Europe/London")
	notifier := NewRegistrationNotifier(poster, "CADMIN", loc, nil)

	notifier.NotifyPlayerRegistered(context.Background(), league.Player{
		DisplayName:  "Luke Humphries",
		Nickname:     "The Freeze",
		RegisteredAt: time.Date(2026, time.March, 18, 12, 30, 0, 0, time.UTC),
	}, 12)

	if len(poster.messages) != 1 {
		t.Fatalf("expected 1 signup message, got %d", len(poster.messages))
	}
	if poster.messages[0].channelID != "CADMIN" {
		t.Fatalf("expected admin channel, got %q", poster.messages[0].channelID)
	}
	if !strings.Contains(poster.messages[0].text, "The Freeze (Luke Humphries)") {
		t.Fatalf("expected admin label in message, got %q", poster.messages[0].text)
	}
	if !strings.Contains(poster.messages[0].text, "Total registered: 12") {
		t.Fatalf("expected total count in message, got %q", poster.messages[0].text)
	}
}

func TestComposeWeeklyFixturesMessageUsesCurrentPublicWeek(t *testing.T) {
	t.Parallel()

	store := seededWeeklyStore(t)
	service := NewWeeklyService(store, func() time.Time {
		loc, _ := time.LoadLocation("Europe/London")
		return time.Date(2026, time.March, 30, 9, 0, 0, 0, loc)
	}, nil, "CPUBLIC")

	message, ok, err := service.ComposeWeeklyFixturesMessage(context.Background())
	if err != nil {
		t.Fatalf("expected fixtures message, got %v", err)
	}
	if !ok {
		t.Fatal("expected fixtures message to be available")
	}
	if !strings.Contains(message, "🎯 Week 2 Fixtures") {
		t.Fatalf("expected current week header, got %q", message)
	}
	if !strings.Contains(message, "📅") {
		t.Fatalf("expected date line, got %q", message)
	}
	if !strings.Contains(message, "Bully Boy vs The Iceman") {
		t.Fatalf("expected week 2 fixture names, got %q", message)
	}
	if !strings.Contains(message, "🏆 Bully Boy vs The Iceman") {
		t.Fatalf("expected emoji matchup format, got %q", message)
	}
	if strings.Contains(message, "The Freeze vs Bully Boy") {
		t.Fatalf("expected prior week fixture to be excluded, got %q", message)
	}
}

func TestComposeWeeklySummaryMessageIncludesResultsAndFullStandings(t *testing.T) {
	t.Parallel()

	store := seededWeeklyStore(t)
	service := NewWeeklyService(store, func() time.Time {
		loc, _ := time.LoadLocation("Europe/London")
		return time.Date(2026, time.March, 30, 9, 0, 0, 0, loc)
	}, nil, "CPUBLIC")

	resultService := league.NewResultServiceWithNow(store, func() time.Time {
		return time.Date(2026, time.April, 3, 8, 0, 0, 0, time.UTC)
	})
	if _, err := resultService.RecordResult(context.Background(), 3, 3, 1, nil, nil); err != nil {
		t.Fatalf("expected first result to succeed, got %v", err)
	}
	if _, err := resultService.RecordResult(context.Background(), 4, 3, 2, nil, nil); err != nil {
		t.Fatalf("expected second result to succeed, got %v", err)
	}

	message, ok, err := service.ComposeWeeklySummaryMessage(context.Background())
	if err != nil {
		t.Fatalf("expected summary message, got %v", err)
	}
	if !ok {
		t.Fatal("expected summary message to be available")
	}
	if !strings.Contains(message, "📣 Week 2 Results + Standings") {
		t.Fatalf("expected summary header, got %q", message)
	}
	if !strings.Contains(message, "Bully Boy 3-1 The Iceman") {
		t.Fatalf("expected recorded result, got %q", message)
	}
	if !strings.Contains(message, "Snakebite 3-2 The Freeze") {
		t.Fatalf("expected second recorded result, got %q", message)
	}
	if !strings.Contains(message, "👑 Leader: Bully Boy on 2 pts") {
		t.Fatalf("expected leader line, got %q", message)
	}
	if !strings.Contains(message, "📊 Standings") {
		t.Fatalf("expected standings section, got %q", message)
	}
	if !strings.Contains(message, "The Freeze") || !strings.Contains(message, "Snakebite") || !strings.Contains(message, "Bully Boy") || !strings.Contains(message, "The Iceman") {
		t.Fatalf("expected full table to include all players, got %q", message)
	}
	if !strings.Contains(message, "Pts") {
		t.Fatalf("expected standings header, got %q", message)
	}
}

func TestPostWeeklySummaryUsesPublicChannel(t *testing.T) {
	t.Parallel()

	store := seededWeeklyStore(t)
	poster := &stubPoster{}
	service := NewWeeklyService(store, func() time.Time {
		loc, _ := time.LoadLocation("Europe/London")
		return time.Date(2026, time.March, 30, 9, 0, 0, 0, loc)
	}, poster, "CPUBLIC")

	posted, err := service.PostWeeklyFixtures(context.Background())
	if err != nil {
		t.Fatalf("expected weekly fixtures post to succeed, got %v", err)
	}
	if !posted {
		t.Fatal("expected fixtures post to be sent")
	}
	if len(poster.messages) != 1 || poster.messages[0].channelID != "CPUBLIC" {
		t.Fatalf("expected one public message, got %+v", poster.messages)
	}
}

func seededWeeklyStore(t *testing.T) *league.MemoryStore {
	t.Helper()

	store := league.NewMemoryStore()
	ctx := context.Background()
	now := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	registration := league.NewRegistrationServiceWithNow(store, func() time.Time { return now })
	seasonService := league.NewSeasonServiceWithNow(store, func() time.Time { return now })

	players := []league.Player{
		{DisplayName: "Luke Humphries", Nickname: "The Freeze"},
		{DisplayName: "Michael Smith", Nickname: "Bully Boy"},
		{DisplayName: "Peter Wright", Nickname: "Snakebite"},
		{DisplayName: "Gerwyn Price", Nickname: "The Iceman"},
	}
	for _, player := range players {
		if _, err := registration.RegisterPlayer(ctx, player); err != nil {
			t.Fatalf("expected registration to succeed, got %v", err)
		}
	}
	if _, err := seasonService.StartSeason(ctx); err != nil {
		t.Fatalf("expected season start to succeed, got %v", err)
	}

	return store
}

type postedMessage struct {
	channelID string
	text      string
}

type stubPoster struct {
	messages []postedMessage
	err      error
}

func (p *stubPoster) PostMessage(_ context.Context, channelID, text string) error {
	p.messages = append(p.messages, postedMessage{channelID: channelID, text: text})
	return p.err
}
