package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

func TestPlayerStatisticsScopeAndVisibility(t *testing.T) {
	ctx := context.Background()
	store := league.NewMemoryStore()
	if _, err := store.EnsureActiveSeason(ctx, league.NewSeason("Statistics")); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	seasons := league.NewSeasonServiceWithNow(store, clock)
	reg := league.NewRegistrationServiceWithNow(store, clock)
	divisions, err := seasons.ProvisionDivisions(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, p := range []league.Player{{DisplayName: "Alice", Nickname: "Arrow"}, {DisplayName: "Bob"}} {
		p, err := reg.RegisterPlayer(ctx, p)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := reg.AssignPlayer(ctx, p.ID, &divisions[0].ID); err != nil {
			t.Fatal(err)
		}
	}
	season, err := seasons.StartSeason(ctx)
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := store.ListFixturesBySeason(ctx, season.ID)
	if err != nil {
		t.Fatal(err)
	}
	f := fixtures[0]
	results := league.NewResultServiceWithNow(store, clock)
	manual, err := results.RecordResult(ctx, f.ID, 3, 1, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	auth := NewAuthHandlerWithNow("admin", "secret", "test-only", clock)
	mux := http.NewServeMux()
	NewPlayerStatisticsHandler(results).RegisterRoutes(mux, auth.RequireAdmin)
	get := func(path string, authenticated bool, want int) league.PlayerStatistics {
		t.Helper()
		req := httptest.NewRequest("GET", path, nil)
		if authenticated {
			req.AddCookie(&http.Cookie{Name: adminSessionCookieName, Value: auth.signSession("admin", now.Add(time.Hour))})
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != want || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
		}
		var got league.PlayerStatistics
		if want == 200 {
			if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
				t.Fatal(err)
			}
			for _, secret := range []string{"Alice", "account_id", "source_payload", "display_name", "seat-a", "seat-b", "fictional-match", "digest"} {
				if strings.Contains(w.Body.String(), secret) {
					t.Fatalf("leaked %s", secret)
				}
			}
		}
		return got
	}
	path := fmt.Sprintf("/api/seasons/%d/players/%d/statistics", season.ID, f.PlayerOneID)
	adminPath := strings.Replace(path, "/api/", "/api/admin/", 1)
	get(adminPath, false, 401)
	get("/api/admin/seasons/bad/players/0/statistics", false, 401)
	if got := get(path, false, 200); got.Played != 0 || len(got.History) != 0 || got.PreferredName != "Arrow" {
		t.Fatalf("hidden match: %+v", got)
	}
	if got := get(adminPath, true, 200); got.Played != 1 || got.Points != 2 {
		t.Fatalf("admin projection: %+v", got)
	}
	for _, id := range []string{"0", "-1", "bad", "9223372036854775808", "999"} {
		get("/api/seasons/"+id+"/players/1/statistics", false, 404)
		get("/api/seasons/1/players/"+id+"/statistics", false, 404)
	}
	now = time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
	if got := get(path, false, 200); got.Played != 1 || got.Coverage.EligibleMatches != 1 || len(got.History) != 1 || got.MatchAverageMean != nil {
		t.Fatalf("revealed match: %+v", got)
	}
	pending := league.NewPendingResultServiceWithNow(store, results, clock)
	raw, err := os.ReadFile("../../../docs/autodarts/examples-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var examples map[string]json.RawMessage
	if err := json.Unmarshal(raw, &examples); err != nil {
		t.Fatal(err)
	}
	outcome, err := pending.IngestPayload(ctx, examples["producer"])
	if err != nil {
		t.Fatal(err)
	}
	if got := get(path, false, 200); got.Coverage.MatchesWithDetail != 0 {
		t.Fatal("pending source leaked")
	}
	if _, err := pending.Approve(ctx, league.ApprovalRequest{
		PendingID: outcome.Pending.ID, SeasonID: season.ID, FixtureID: f.ID,
		Mapping: map[string]int64{"seat-a": f.PlayerOneID, "seat-b": f.PlayerTwoID},
		Actor:   "admin", Reason: "verified replacement", MissingDateReason: "not provided",
		Replace: true, ExpectedResult: league.ExpectedFromResult(manual),
	}); err != nil {
		t.Fatal(err)
	}
	now = time.Date(2026, 3, 29, 8, 0, 0, 0, time.UTC)
	if got := get(path, false, 200); got.Played != 0 || got.Coverage.RecordedThrows != 0 || len(got.Throws) != 0 {
		t.Fatal("unrevealed source leaked")
	}
	if got := get(adminPath, true, 200); got.Coverage.MatchesWithDetail != 1 || len(got.Throws) == 0 {
		t.Fatal("admin lost source detail")
	}
	now = time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
	if got := get(path, false, 200); got.Coverage.MatchesWithDetail != 1 || len(got.Throws) == 0 {
		t.Fatal("revealed source missing")
	}
	standingsReq := httptest.NewRequest("GET", "/api/divisions/standings", nil)
	standingsReq.SetPathValue("divisionSlug", divisions[0].Slug)
	w := httptest.NewRecorder()
	NewResultHandler(results).handleStandings(w, standingsReq)
	var standings struct {
		Rows []standingRowResponse `json:"standings"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &standings); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || len(standings.Rows) != 2 || standings.Rows[0].PlayerID != f.PlayerOneID {
		t.Fatalf("standings lost stable ID: %s", w.Body.String())
	}
	if err := store.CloseSeason(ctx, season.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateNextSeason(ctx, season.ID, league.NewSeason("Next")); err != nil {
		t.Fatal(err)
	}
	if got := get(path, false, 200); got.SeasonID != season.ID || got.Played != 1 {
		t.Fatalf("historical scope: %+v", got)
	}
	get(fmt.Sprintf("/api/seasons/%d/players/%d/statistics", season.ID+1, f.PlayerOneID), false, 404)
	returning, err := reg.RegisterPlayer(ctx, league.Player{DisplayName: "Alice", Nickname: "NewArrow"})
	if err != nil {
		t.Fatal(err)
	}
	if got := get(fmt.Sprintf("/api/seasons/%d/players/%d/statistics", returning.SeasonID, returning.ID), false, 200); got.Played != 0 || got.PreferredName != "NewArrow" {
		t.Fatalf("returning identity merged: %+v", got)
	}
	if got := get(path, false, 200); got.PreferredName != "Arrow" || got.Played != 1 {
		t.Fatal("historical identity changed")
	}
}
