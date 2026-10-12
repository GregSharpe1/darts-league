package httpapi

import (
	"context"
	"encoding/json"
	"github.com/greg/darts-league/backend/internal/league"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestMappingHTTPContract(t *testing.T) {
	ctx := context.Background()
	store := league.NewMemoryStore()
	for _, name := range []string{"One", "Two"} {
		if _, err := store.CreatePlayer(ctx, league.Player{SeasonID: 1, DisplayName: name}); err != nil {
			t.Fatal(err)
		}
	}
	s := league.NewPendingResultService(store, league.NewResultService(store))
	auth := NewAuthHandler("admin", "secret", "test-key")
	mux := http.NewServeMux()
	auth.RegisterRoutes(mux)
	NewPendingResultHandler(s).RegisterRoutes(mux, auth.RequireAdmin)
	NewRegistrationHandler(league.NewRegistrationService(store)).RegisterRoutes(mux, auth.RequireAdmin)
	var cookie *http.Cookie
	send := func(method, path, body string, authenticated bool) *httptest.ResponseRecorder {
		r := httptest.NewRequest(method, path, strings.NewReader(body))
		if authenticated {
			r.AddCookie(cookie)
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, r)
		return w
	}
	base := "/api/admin/autodarts-mappings"
	for _, method := range []string{"GET", "PUT", "DELETE"} {
		path := base
		if method != "GET" {
			path += "/1"
		}
		if w := send(method, path, `{"season_id":1,"username":"private-source"}`, false); w.Code != 401 {
			t.Fatalf("%s auth %d", method, w.Code)
		}
	}
	login := send("POST", "/api/admin/login", `{"username":"admin","password":"secret"}`, false)
	if login.Code != 200 {
		t.Fatal(login.Body.String())
	}
	cookie = login.Result().Cookies()[0]
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"PUT", base + "/1", `{"season_id":1,"username":" private-source "}`, 200},
		{"PUT", base + "/2", `{"season_id":1,"username":"private-source"}`, 409},
		{"PUT", base + "/1", `{"season_id":2,"username":"stale"}`, 409},
		{"PUT", base + "/999", `{"season_id":1,"username":"absent"}`, 404},
		{"PUT", base + "/bad", `{"season_id":1,"username":"bad"}`, 400},
		{"PUT", base + "/1", `{"username":"missing-season"}`, 400},
		{"PUT", base + "/1", `{"season_id":1,"username":""}`, 400},
		{"PUT", base + "/1", `{"season_id":1,"username":"bad\u0000name"}`, 400},
		{"PUT", base + "/1", `{"season_id":1,"username":"x","extra":true}`, 400},
		{"PUT", base + "/1", `{"season_id":1,"username":"x"} {}`, 400},
		{"PUT", base + "/1", `null`, 400},
		{"PUT", base + "/1", strings.Repeat(" ", 17000) + `{}`, 400},
		{"DELETE", base + "/1", `{"season_id":1,"username":"x"}`, 400},
	} {
		if w := send(tc.method, tc.path, tc.body, true); w.Code != tc.status {
			t.Fatalf("%s %s: %d %s", tc.method, tc.body, w.Code, w.Body.String())
		}
	}
	w := send("GET", base, "", true)
	var view league.AutodartsMappingView
	if err := json.Unmarshal(w.Body.Bytes(), &view); err != nil {
		t.Fatal(err)
	}
	if w.Code != 200 || w.Header().Get("Cache-Control") != "no-store" || view.SeasonID != 1 || view.AutoSubmitEnabled || len(view.Mappings) != 1 || view.Mappings[0].Username != "private-source" {
		t.Fatal(w.Body.String())
	}
	if w := send("GET", "/api/players", "", false); strings.Contains(w.Body.String(), "private-source") || strings.Contains(w.Body.String(), "autodarts") {
		t.Fatal("private mapping leaked")
	}
	for range 2 {
		if w := send("DELETE", base+"/1", `{"season_id":1}`, true); w.Code != 204 {
			t.Fatal(w.Body.String())
		}
	}
}
