package notifications

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestWeeklyMessagesIncludeDivisionStandingsLink(t *testing.T) {
	t.Parallel()

	for _, baseURL := range []string{"https://darts.example.com", " https://darts.example.com/ ", ""} {
		for _, kind := range []string{"fixtures", "summary"} {
			t.Run(kind+"/"+baseURL, func(t *testing.T) {
				store := seededWeeklyStore(t)
				division, err := store.GetDivisionBySlug(context.Background(), 1, "division-1")
				if err != nil {
					t.Fatal(err)
				}
				division.Name = "Premier"
				division.Slug = "premier"
				if _, err := store.UpdateDivision(context.Background(), division); err != nil {
					t.Fatal(err)
				}
				poster := &stubPoster{}
				service := NewWeeklyService(store, func() time.Time {
					return time.Date(2026, time.March, 30, 8, 0, 0, 0, time.UTC)
				}, poster, "CPUBLIC", baseURL)
				post := service.PostWeeklyFixtures
				header := "Week 2 Fixtures \u2014 Premier"
				if kind == "summary" {
					post = service.PostWeeklySummary
					header = "Week 2 Results + Standings \u2014 Premier"
				}

				posted, err := post(context.Background())

				if err != nil || !posted || len(poster.messages) != 1 {
					t.Fatalf("expected one posted message, got posted=%v, err=%v, messages=%v", posted, err, poster.messages)
				}
				message := poster.messages[0].text
				if !strings.Contains(strings.Split(message, "\n")[0], header) {
					t.Fatalf("expected division header %q, got %q", header, message)
				}
				if baseURL == "" {
					if strings.Contains(message, "View the standings") {
						t.Fatalf("expected no link without a public URL, got %q", message)
					}
				} else if got := strings.Split(message, "\n")[2]; got != "View the standings <https://darts.example.com/divisions/premier/standings|here>." {
					t.Fatalf("unexpected standings link line: %q", got)
				}
			})
		}
	}
}
