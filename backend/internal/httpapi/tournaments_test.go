package httpapi

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/tournament"
)

func TestTournamentRegistrationLifecycle(t *testing.T) {
	mux := http.NewServeMux()
	auth := NewAuthHandler("admin", "test-password", "test-secret")
	auth.RegisterRoutes(mux)
	TournamentHandler{Service: tournament.Service{Store: tournament.NewMemoryStore(), Now: time.Now}}.RegisterRoutes(mux, auth.RequireAdmin)

	request := func(method, path, body string, cookie *http.Cookie, status int) *httptest.ResponseRecorder {
		t.Helper()
		req := httptest.NewRequest(method, path, strings.NewReader(body))
		if cookie != nil {
			req.AddCookie(cookie)
		}
		res := httptest.NewRecorder()
		mux.ServeHTTP(res, req)
		if res.Code != status {
			t.Fatalf("%s %s returned %d, want %d: %s", method, path, res.Code, status, res.Body)
		}
		return res
	}

	const adminPath = "/api/admin/tournaments"
	const adminEvent = adminPath + "/city-cup"
	const publicEvent = "/api/tournaments/city-cup"
	const eventData = `{"slug":"city-cup","name":"City Cup","starts_on":"2027-03-20"}`
	const entrant = `{"display_name":"  Maya  Evans  ","nickname":"  Flight  "}`
	for _, endpoint := range []struct{ method, path string }{
		{http.MethodPost, adminPath},
		{http.MethodGet, adminPath},
		{http.MethodPut, adminEvent + "/registration"},
		{http.MethodGet, adminEvent + "/entries"},
		{http.MethodDelete, adminEvent + "/entries/1"},
	} {
		request(endpoint.method, endpoint.path, "", nil, http.StatusUnauthorized)
	}
	request("GET", publicEvent, "", nil, http.StatusNotFound)

	login := request("POST", "/api/admin/login", `{"username":"admin","password":"test-password"}`, nil, http.StatusOK)
	cookie := login.Result().Cookies()[0]
	request("POST", adminPath, strings.Replace(eventData, "2027-03-20", "2027-02-29", 1), cookie, http.StatusBadRequest)
	request("POST", adminPath, eventData, cookie, http.StatusCreated)
	request("POST", adminPath, eventData, cookie, http.StatusConflict)
	request("POST", publicEvent+"/entries", entrant, nil, http.StatusConflict)
	request("PUT", adminEvent+"/registration", `{}`, cookie, http.StatusBadRequest)
	request("PUT", adminEvent+"/registration", `{"registration_open":true}`, cookie, http.StatusOK)
	request("POST", publicEvent+"/entries", `{"display_name":"   "}`, nil, http.StatusBadRequest)

	signup := request("POST", publicEvent+"/entries", entrant, nil, http.StatusCreated)
	if strings.TrimSpace(signup.Body.String()) != `{"preferred_name":"Flight"}` {
		t.Fatalf("unexpected public signup response: %s", signup.Body)
	}
	request("POST", publicEvent+"/entries", `{"display_name":"maya EVANS"}`, nil, http.StatusConflict)
	event := request("GET", publicEvent, "", nil, http.StatusOK)
	if strings.Contains(event.Body.String(), "Maya") || strings.Contains(event.Body.String(), "Flight") {
		t.Fatalf("public event leaked entrant details: %s", event.Body)
	}
	request("GET", publicEvent+"/entries", "", nil, http.StatusMethodNotAllowed)

	roster := request("GET", adminEvent+"/entries", "", cookie, http.StatusOK)
	var entries []tournament.Entry
	if err := json.Unmarshal(roster.Body.Bytes(), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].DisplayName != "Maya Evans" || entries[0].Nickname != "Flight" {
		t.Fatalf("unexpected roster: %+v", entries)
	}

	const secondEvent = "/api/tournaments/another-cup"
	request("POST", adminPath, strings.Replace(eventData, "city-cup", "another-cup", 1), cookie, http.StatusCreated)
	request("PUT", adminPath+"/another-cup/registration", `{"registration_open":true}`, cookie, http.StatusOK)
	request("POST", secondEvent+"/entries", entrant, nil, http.StatusCreated)
	request("DELETE", adminPath+"/another-cup/entries/1", "", cookie, http.StatusNotFound)

	request("PUT", adminEvent+"/registration", `{"registration_open":false}`, cookie, http.StatusOK)
	request("POST", publicEvent+"/entries", `{"display_name":"Late Entrant"}`, nil, http.StatusConflict)
	request("DELETE", adminEvent+"/entries/1", "", cookie, http.StatusNoContent)
	request("PUT", adminEvent+"/registration", `{"registration_open":true}`, cookie, http.StatusOK)
	request("POST", publicEvent+"/entries", entrant, nil, http.StatusCreated)
}
