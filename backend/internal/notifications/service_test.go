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
	}, nil, "CPUBLIC", "")

	division, err := store.GetDivisionBySlug(context.Background(), 1, "division-1")
	if err != nil {
		t.Fatalf("expected division, got %v", err)
	}
	message, ok, err := service.ComposeWeeklyFixturesMessage(context.Background(), division)
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
	if !strings.Contains(message, "Bully Boy (Michael Smith) vs The Iceman (Gerwyn Price)") {
		t.Fatalf("expected week 2 fixture names, got %q", message)
	}
	if !strings.Contains(message, "🏆 Bully Boy (Michael Smith) vs The Iceman (Gerwyn Price)") {
		t.Fatalf("expected emoji matchup format, got %q", message)
	}
	if strings.Contains(message, "The Freeze (Luke Humphries) vs Bully Boy (Michael Smith)") {
		t.Fatalf("expected prior week fixture to be excluded, got %q", message)
	}
}

func TestComposeWeeklySummaryMessageIncludesResultsAndFullStandings(t *testing.T) {
	t.Parallel()

	store := seededWeeklyStore(t)
	service := NewWeeklyService(store, func() time.Time {
		loc, _ := time.LoadLocation("Europe/London")
		return time.Date(2026, time.March, 30, 9, 0, 0, 0, loc)
	}, nil, "CPUBLIC", "")
	division, err := store.GetDivisionBySlug(context.Background(), 1, "division-1")
	if err != nil {
		t.Fatalf("expected division, got %v", err)
	}

	resultService := league.NewResultServiceWithNow(store, func() time.Time {
		return time.Date(2026, time.April, 3, 8, 0, 0, 0, time.UTC)
	})
	if _, err := resultService.RecordResult(context.Background(), 3, 3, 1, nil, nil); err != nil {
		t.Fatalf("expected first result to succeed, got %v", err)
	}
	if _, err := resultService.RecordResult(context.Background(), 4, 3, 2, nil, nil); err != nil {
		t.Fatalf("expected second result to succeed, got %v", err)
	}

	message, ok, err := service.ComposeWeeklySummaryMessage(context.Background(), division)
	if err != nil {
		t.Fatalf("expected summary message, got %v", err)
	}
	if !ok {
		t.Fatal("expected summary message to be available")
	}
	if !strings.Contains(message, "📣 Week 2 Results + Standings") {
		t.Fatalf("expected summary header, got %q", message)
	}
	if !strings.Contains(message, "Bully Boy (Michael Smith) 3-1 The Iceman (Gerwyn Price)") {
		t.Fatalf("expected recorded result, got %q", message)
	}
	if !strings.Contains(message, "Snakebite (Peter Wright) 3-2 The Freeze (Luke Humphries)") {
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
	}, poster, "CPUBLIC", "")

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

func TestCompletedSeasonDoesNotPostWeeklyMessages(t *testing.T) {
	store := seededWeeklyStore(t)
	ctx := context.Background()
	fixtures, err := store.ListFixturesBySeason(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	results := league.NewResultService(store)
	for _, fixture := range fixtures {
		if _, err := results.RecordResult(ctx, fixture.ID, 3, 0, nil, nil); err != nil {
			t.Fatal(err)
		}
	}
	if err := store.CloseSeason(ctx, 1); err != nil {
		t.Fatal(err)
	}
	poster := &stubPoster{}
	service := NewWeeklyService(store, func() time.Time { return time.Date(2026, 4, 10, 10, 0, 0, 0, time.UTC) }, poster, "CPUBLIC", "https://darts.example.com")
	for _, post := range []func(context.Context) (bool, error){service.PostWeeklyFixtures, service.PostWeeklySummary} {
		if posted, err := post(ctx); err != nil || posted {
			t.Fatalf("completed notification: %v %v", posted, err)
		}
	}
	if len(poster.messages) != 0 {
		t.Fatal("posted completed season")
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
	divisions, err := seasonService.ProvisionDivisions(ctx, 1)
	if err != nil {
		t.Fatalf("expected division provisioning to succeed, got %v", err)
	}
	registeredPlayers, err := registration.ListPlayers(ctx)
	if err != nil {
		t.Fatalf("expected players to be listed, got %v", err)
	}
	for _, player := range registeredPlayers {
		if _, err := registration.AssignPlayer(ctx, player.ID, &divisions[0].ID); err != nil {
			t.Fatalf("expected assignment to succeed, got %v", err)
		}
	}
	if _, err := seasonService.UpdateDivision(ctx, divisions[0].ID, divisions[0].Name, divisions[0].Slug, "CPUBLIC"); err != nil {
		t.Fatalf("expected division slack channel update to succeed, got %v", err)
	}
	if _, err := seasonService.StartSeason(ctx); err != nil {
		t.Fatalf("expected season start to succeed, got %v", err)
	}

	return store
}

func TestPlayerLabelsAreEscapedForSlack(t *testing.T) {
	ctx := context.Background()
	store := league.NewMemoryStore()
	now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	poster := &stubPoster{}
	registration := league.NewRegistrationServiceWithNowAndNotifier(store, func() time.Time { return now }, NewRegistrationNotifier(poster, "TEST", time.UTC, nil))
	season := league.NewSeasonServiceWithNow(store, func() time.Time { return now })
	divisions, err := season.ProvisionDivisions(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"<!channel>", "<@U123> & Co"} {
		player, err := registration.RegisterPlayer(ctx, league.Player{DisplayName: name})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := registration.AssignPlayer(ctx, player.ID, &divisions[0].ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := season.StartSeason(ctx); err != nil {
		t.Fatal(err)
	}
	weekly := NewWeeklyService(store, func() time.Time { return time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC) }, nil, "TEST", "https://example.com")
	messages := []string{poster.messages[0].text, poster.messages[1].text}
	for _, compose := range []func(context.Context, league.Division) (string, bool, error){weekly.ComposeWeeklyFixturesMessage, weekly.ComposeWeeklySummaryMessage} {
		message, ok, err := compose(ctx, divisions[0])
		if err != nil || !ok {
			t.Fatalf("compose: %v %v", ok, err)
		}
		if !strings.Contains(message, "<https://example.com/divisions/division-1/standings|here>") {
			t.Fatal("intentional link lost")
		}
		messages = append(messages, message)
	}
	for _, message := range messages {
		if strings.Contains(message, "<!channel>") || strings.Contains(message, "<@U123>") || strings.Contains(message, " & Co") {
			t.Fatalf("unescaped label: %s", message)
		}
		if !strings.Contains(message, "&lt;") {
			t.Fatalf("missing escaped label: %s", message)
		}
	}
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

func TestWeeklyMessagesAreScopedToEachDivision(t *testing.T) {
	for name, channels := range map[string][]string{
		"shared channel":  {"CSHARED", "CSHARED", "CSHARED"},
		"public fallback": {"", "", ""},
		"mixed":           {"CDIV1", "", "CDIV1"},
	} {
		t.Run(name, func(t *testing.T) {
			ctx := context.Background()
			store := league.NewMemoryStore()
			now := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
			registration := league.NewRegistrationServiceWithNow(store, func() time.Time { return now })
			season := league.NewSeasonServiceWithNow(store, func() time.Time { return now })
			divisions, err := season.ProvisionDivisions(ctx, 3)
			if err != nil {
				t.Fatal(err)
			}
			for i, division := range divisions {
				if _, err := season.UpdateDivision(ctx, division.ID, division.Name, division.Slug, channels[i]); err != nil {
					t.Fatal(err)
				}
			}
			roster := [][]league.Player{
				{{DisplayName: "Aaron Able"}, {DisplayName: "Bob Brown", Nickname: "Bravo"}},
				{{DisplayName: "Carl Cole", Nickname: "Charlie"}, {DisplayName: "Dan Dee"}},
			}
			for i, players := range roster {
				for _, player := range players {
					registered, err := registration.RegisterPlayer(ctx, player)
					if err != nil {
						t.Fatal(err)
					}
					if _, err := registration.AssignPlayer(ctx, registered.ID, &divisions[i].ID); err != nil {
						t.Fatal(err)
					}
				}
			}
			if _, err := season.StartSeason(ctx); err != nil {
				t.Fatal(err)
			}
			fixtures, err := store.ListFixturesByDivision(ctx, divisions[0].ID)
			if err != nil || len(fixtures) != 1 {
				t.Fatalf("division 1 fixtures: %v %v", fixtures, err)
			}
			legs := map[bool][2]int{true: {0, 3}, false: {3, 0}}[fixtures[0].PlayerOneID == 1]
			if _, err := league.NewResultService(store).RecordResult(ctx, fixtures[0].ID, legs[0], legs[1], nil, nil); err != nil {
				t.Fatal(err)
			}

			poster := &stubPoster{}
			weekly := NewWeeklyService(store, func() time.Time { return time.Date(2026, 3, 23, 10, 0, 0, 0, time.UTC) }, poster, "CPUBLIC", "https://darts.example.com")
			if posted, err := weekly.PostWeeklySummary(ctx); err != nil || !posted {
				t.Fatalf("summary: %v %v", posted, err)
			}
			if len(poster.messages) != 2 {
				t.Fatalf("expected one message per non-empty division, got %+v", poster.messages)
			}
			want := []struct{ channel, slug, leader, other string }{
				{channels[0], "division-1", "Leader: Bravo on 2 pts", "Charlie"},
				{channels[1], "division-2", "Leader: Charlie on 0 pts", "Aaron"},
			}
			for i, w := range want {
				message := poster.messages[i]
				if w.channel == "" {
					w.channel = "CPUBLIC"
				}
				if message.channelID != w.channel {
					t.Fatalf("message %d channel = %q, want %q", i, message.channelID, w.channel)
				}
				for _, s := range []string{divisions[i].Name, "/divisions/" + w.slug + "/standings", w.leader} {
					if !strings.Contains(message.text, s) {
						t.Fatalf("message %d missing %q: %s", i, s, message.text)
					}
				}
				if strings.Contains(message.text, w.other) || strings.Contains(message.text, "Division 3") {
					t.Fatalf("message %d leaks another division: %s", i, message.text)
				}
			}
		})
	}
}
