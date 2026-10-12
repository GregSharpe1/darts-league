package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/greg/darts-league/backend/internal/league"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPendingDetailSuggestionHTTPContract(t *testing.T) {
	ctx := context.Background()
	store := league.NewMemoryStore()
	division := int64(1)
	for _, name := range []string{"league one", "league two"} {
		if _, err := store.CreatePlayer(ctx, league.Player{SeasonID: 1, DivisionID: &division, Status: league.PlayerStatusAssigned, DisplayName: name}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := store.CreateFixtures(ctx, []league.Fixture{{SeasonID: 1, DivisionID: 1, PlayerOneID: 1, PlayerTwoID: 2, GameVariant: "501", LegsToWin: 3}}); err != nil {
		t.Fatal(err)
	}
	s := league.NewPendingResultService(store, league.NewResultService(store))
	out, err := s.IngestPayload(ctx, []byte(`{"matchId":"http-suggestion","player1":{"name":"source two","legsWon":3},"player2":{"name":"source one","legsWon":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	auth := NewAuthHandler("admin", "secret", "test-key")
	mux := http.NewServeMux()
	auth.RegisterRoutes(mux)
	NewPendingResultHandler(s).RegisterRoutes(mux, auth.RequireAdmin)
	login := httptest.NewRecorder()
	mux.ServeHTTP(login, httptest.NewRequest("POST", "/api/admin/login", strings.NewReader(`{"username":"admin","password":"secret"}`)))
	if login.Code != 200 {
		t.Fatal(login.Body.String())
	}
	cookie := login.Result().Cookies()[0]
	for i := range 3 {
		if i > 0 {
			r := httptest.NewRequest("PUT", fmt.Sprintf("/api/admin/autodarts-mappings/%d", i), strings.NewReader(fmt.Sprintf(`{"season_id":1,"username":"source %s"}`, []string{"", "one", "two"}[i])))
			r.AddCookie(cookie)
			w := httptest.NewRecorder()
			mux.ServeHTTP(w, r)
			if w.Code != 200 {
				t.Fatal(w.Body.String())
			}
		}
		r := httptest.NewRequest("GET", fmt.Sprintf("/api/admin/pending-results/%d", out.Pending.ID), nil)
		r.AddCookie(cookie)
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		var detail struct {
			Suggestion league.ImportSuggestion `json:"suggestion"`
			Mapping    map[string]int64        `json:"mapping"`
		}
		if err := json.Unmarshal(w.Body.Bytes(), &detail); err != nil {
			t.Fatal(err)
		}
		if w.Code != 200 || detail.Suggestion.SeasonID == nil || *detail.Suggestion.SeasonID != 1 || len(detail.Suggestion.Mapping) != i || len(detail.Mapping) != 0 {
			t.Fatalf("response %s", w.Body.String())
		}
		if (detail.Suggestion.FixtureID != nil) != (i == 2) {
			t.Fatalf("fixture suggestion %s", w.Body.String())
		}
		if i == 2 && (detail.Suggestion.Mapping["legacy-1"] != 2 || detail.Suggestion.Mapping["legacy-2"] != 1) {
			t.Fatal("source order changed")
		}
	}
	rows, err := store.ListResultsBySeason(ctx, 1)
	if err != nil || len(rows) != 0 {
		t.Fatalf("GET mutated results %+v %v", rows, err)
	}
}
