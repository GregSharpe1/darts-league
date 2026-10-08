package httpapi

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

func TestMatchDetailHTTPVisibilityAndAuth(t *testing.T) {
	for _, monday := range []string{"2026-03-23T09:00:00Z", "2026-03-30T08:00:00Z", "2026-10-19T08:00:00Z", "2026-10-26T09:00:00Z"} {
		t.Run(monday, func(t *testing.T) {
			reveal, err := time.Parse(time.RFC3339, monday)
			if err != nil {
				t.Fatal(err)
			}
			checkMatchDetailHTTPBoundary(t, reveal)
		})
	}
}

func checkMatchDetailHTTPBoundary(t *testing.T, reveal time.Time) {
	t.Helper()
	ctx := context.Background()
	store := league.NewMemoryStore()
	now := reveal.Add(-time.Second)
	clock := func() time.Time { return now }
	season, err := store.GetActiveSeason(ctx)
	if err != nil {
		t.Fatal(err)
	}
	p1, err := store.CreatePlayer(ctx, league.Player{SeasonID: season.ID, DisplayName: "Alice", Nickname: "Arrow"})
	if err != nil {
		t.Fatal(err)
	}
	p2, err := store.CreatePlayer(ctx, league.Player{SeasonID: season.ID, DisplayName: "Bob"})
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := store.CreateFixtures(ctx, []league.Fixture{{SeasonID: season.ID, PlayerOneID: p1.ID, PlayerTwoID: p2.ID, ScheduledAt: now.Add(time.Hour), WeekNumber: 1}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.CreateResult(ctx, league.Result{FixtureID: fixtures[0].ID, PlayerOneLegs: 3, PlayerTwoLegs: 1}); err != nil {
		t.Fatal(err)
	}
	auth := NewAuthHandlerWithNow("admin", "secret", "test-only", clock)
	mux := http.NewServeMux()
	auth.RegisterRoutes(mux)
	NewMatchDetailHandler(league.NewFixtureServiceWithNow(store, clock)).RegisterRoutes(mux, auth.RequireAdmin)
	server := httptest.NewServer(mux)
	defer server.Close()
	get := func(path string, cookie *http.Cookie) (int, string) {
		t.Helper()
		req, err := http.NewRequest("GET", server.URL+path, nil)
		if err != nil {
			t.Fatal(err)
		}
		if cookie != nil {
			req.AddCookie(cookie)
		}
		resp, err := server.Client().Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, err := io.ReadAll(resp.Body)
		if err != nil {
			t.Fatal(err)
		}
		if resp.Header.Get("Cache-Control") != "no-store" {
			t.Fatalf("cacheable response: %s", path)
		}
		return resp.StatusCode, string(body)
	}
	code, hidden := get("/api/fixtures/1/autodarts", nil)
	if code != 404 {
		t.Fatalf("guessed future ID: %d %s", code, hidden)
	}
	for _, id := range []string{"999", "-1", "invalid", "0"} {
		code, body := get("/api/fixtures/"+id+"/autodarts", nil)
		if code != 404 || body != hidden {
			t.Fatalf("distinguishable missing: %d %s", code, body)
		}
	}
	if code, _ := get("/api/admin/fixtures/1/autodarts", nil); code != 401 {
		t.Fatalf("unauthenticated: %d", code)
	}
	login, err := server.Client().Post(server.URL+"/api/admin/login", "application/json", strings.NewReader(`{"username":"admin","password":"secret"}`))
	if err != nil {
		t.Fatal(err)
	}
	cookie := login.Cookies()[0]
	if err := login.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if code, body := get("/api/admin/fixtures/1/autodarts", cookie); code != 200 || !strings.Contains(body, `"detail":null`) {
		t.Fatalf("admin summary: %d %s", code, body)
	}
	now = now.Add(time.Second)
	if code, body := get("/api/fixtures/1/autodarts", nil); code != 200 || !strings.Contains(body, `"preferred_name":"Arrow"`) {
		t.Fatalf("revealed: %d %s", code, body)
	}
	if err := store.DeleteResultByFixture(ctx, 1); err != nil {
		t.Fatal(err)
	}
	if code, body := get("/api/fixtures/1/autodarts", nil); code != 404 || body != hidden {
		t.Fatalf("unrecorded: %d %s", code, body)
	}
}
