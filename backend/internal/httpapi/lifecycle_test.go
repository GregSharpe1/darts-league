package httpapi

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

func TestLifecycleRoutes(t *testing.T) {
	ctx := context.Background()
	store := league.NewMemoryStore()
	clock := func() time.Time { return time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC) }
	service := league.NewSeasonServiceWithNow(store, clock)
	season := NewSeasonHandler(service, league.NewFixtureServiceWithNow(store, clock), "Test")
	auth := NewAuthHandlerWithNow("admin", "secret", "test-secret", clock)
	mux := http.NewServeMux()
	auth.RegisterRoutes(mux)
	season.RegisterRoutes(mux, auth.RequireAdmin)
	NewResultHandler(league.NewResultService(store)).RegisterRoutes(mux, auth.RequireAdmin)
	for _, path := range []string{"/api/admin/season/close", "/api/admin/season/next"} {
		hitEndpoint(t, mux.ServeHTTP, httptest.NewRequest("POST", path, bytes.NewBufferString(`{"season_id":1}`)), 401)
	}
	login := hitEndpoint(t, mux.ServeHTTP, httptest.NewRequest("POST", "/api/admin/login", bytes.NewBufferString(`{"username":"admin","password":"secret"}`)), 200)
	cookie := login.Result().Cookies()[0]
	request := func(method, path, body string, status int) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		req.AddCookie(cookie)
		return hitEndpoint(t, mux.ServeHTTP, req, status)
	}
	for _, path := range []string{"/api/admin/season/close", "/api/admin/season/next"} {
		for _, body := range []string{`{`, `{}`, `{"season_id":-1}`, `{"season_id":"1"}`, `{"season_id":1} {}`} {
			request("POST", path, body, 400)
		}
	}
	request("POST", "/api/admin/season/close", `{"season_id":1}`, 409)
	registration := NewRegistrationHandler(league.NewRegistrationServiceWithNow(store, clock))
	registerTestPlayers(t, registration, []string{"Alice", "Bob"})
	assignPlayersToDivisionInStore(t, store, clock)
	if _, err := service.StartSeason(ctx); err != nil {
		t.Fatal(err)
	}
	request("POST", "/api/admin/season/close", `{"season_id":1}`, 409)
	request("POST", "/api/admin/fixtures/1/result", `{"player_one_legs":3,"player_two_legs":1}`, 201)
	before := request("GET", "/api/divisions/division-1/standings", "", 200).Body.String()
	request("POST", "/api/admin/season/close", `{"season_id":1}`, 200)
	if after := request("GET", "/api/divisions/division-1/standings", "", 200).Body.String(); after != before {
		t.Fatal("closing changed standings")
	}
	for _, method := range []string{"POST", "PUT", "DELETE"} {
		recorder := request(method, "/api/admin/fixtures/1/result", `{"player_one_legs":3,"player_two_legs":0}`, 409)
		assertErrorCode(t, recorder.Body.Bytes(), "season_completed")
	}
	request("POST", "/api/admin/season/next", `{"season_id":1,"name":" "}`, 400)
	request("POST", "/api/admin/season/next", `{"season_id":99,"name":"Next"}`, 409)
	request("POST", "/api/admin/season/next", `{"season_id":1,"name":"Next"}`, 201)
	request("POST", "/api/admin/season/next", `{"season_id":1,"name":"Again"}`, 409)
	request("DELETE", "/api/admin/fixtures/1/result", "", 409)
	request("GET", "/api/divisions/division-1/standings", "", 404)
}
