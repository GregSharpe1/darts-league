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
		{http.MethodDelete, adminEvent},
		{http.MethodPut, adminEvent + "/registration"},
		{http.MethodPut, adminEvent + "/format"},
		{http.MethodPost, adminEvent + "/start"},
		{http.MethodGet, adminEvent + "/entries"},
		{http.MethodDelete, adminEvent + "/entries/1"},
	} {
		request(endpoint.method, endpoint.path, "", nil, http.StatusUnauthorized)
	}
	request("GET", publicEvent, "", nil, http.StatusNotFound)
	empty := request("GET", "/api/tournaments", "", nil, http.StatusOK)
	if strings.TrimSpace(empty.Body.String()) != "[]" {
		t.Fatalf("expected an empty public tournament list, got %s", empty.Body)
	}

	login := request("POST", "/api/admin/login", `{"username":"admin","password":"test-password"}`, nil, http.StatusOK)
	cookie := login.Result().Cookies()[0]
	request("POST", adminPath, strings.Replace(eventData, "2027-03-20", "2027-02-29", 1), cookie, http.StatusBadRequest)
	created := request("POST", adminPath, eventData, cookie, http.StatusCreated)
	if !strings.Contains(created.Body.String(), `"game_variant":"501"`) || !strings.Contains(created.Body.String(), `"legs_to_win":3`) {
		t.Fatalf("missing default tournament format: %s", created.Body)
	}
	request("DELETE", adminEvent, "", cookie, http.StatusConflict)
	request("POST", adminPath, eventData, cookie, http.StatusConflict)
	request("POST", publicEvent+"/entries", entrant, nil, http.StatusConflict)
	request("PUT", adminEvent+"/registration", `{}`, cookie, http.StatusBadRequest)
	request("PUT", adminEvent+"/registration", `{"registration_open":true}`, cookie, http.StatusOK)
	request("PUT", adminEvent+"/format", `{"game_variant":"701","legs_to_win":3}`, cookie, http.StatusBadRequest)
	request("PUT", adminEvent+"/format", `{"game_variant":"301","legs_to_win":0}`, cookie, http.StatusBadRequest)
	request("PUT", adminEvent+"/format", `{"game_variant":"301","legs_to_win":5}`, cookie, http.StatusOK)
	request("POST", publicEvent+"/entries", `{"display_name":"   "}`, nil, http.StatusBadRequest)

	signup := request("POST", publicEvent+"/entries", entrant, nil, http.StatusCreated)
	if strings.TrimSpace(signup.Body.String()) != `{"preferred_name":"Flight"}` {
		t.Fatalf("unexpected public signup response: %s", signup.Body)
	}
	request("POST", publicEvent+"/entries", `{"display_name":"maya EVANS"}`, nil, http.StatusConflict)
	event := request("GET", publicEvent, "", nil, http.StatusOK)
	if !strings.Contains(event.Body.String(), `"game_variant":"301"`) || !strings.Contains(event.Body.String(), `"legs_to_win":5`) {
		t.Fatalf("public event did not show the saved format: %s", event.Body)
	}
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
	request("POST", adminPath, `{"slug":"another-cup","name":"Another Cup","starts_on":"2027-03-20","game_variant":"501","legs_to_win":2}`, cookie, http.StatusCreated)
	request("PUT", adminPath+"/another-cup/registration", `{"registration_open":true}`, cookie, http.StatusOK)
	request("POST", secondEvent+"/entries", entrant, nil, http.StatusCreated)
	request("DELETE", adminPath+"/another-cup/entries/1", "", cookie, http.StatusNotFound)

	request("PUT", adminEvent+"/registration", `{"registration_open":false}`, cookie, http.StatusOK)
	request("POST", publicEvent+"/entries", `{"display_name":"Late Entrant"}`, nil, http.StatusConflict)
	request("DELETE", adminEvent+"/entries/1", "", cookie, http.StatusNoContent)
	request("PUT", adminEvent+"/registration", `{"registration_open":true}`, cookie, http.StatusOK)
	request("POST", publicEvent+"/entries", entrant, nil, http.StatusCreated)
	request("POST", adminEvent+"/start", "", cookie, http.StatusConflict)
	request("PUT", adminEvent+"/registration", `{"registration_open":false}`, cookie, http.StatusOK)
	request("POST", adminEvent+"/start", "", cookie, http.StatusConflict)
	request("PUT", adminEvent+"/registration", `{"registration_open":true}`, cookie, http.StatusOK)
	request("POST", publicEvent+"/entries", `{"display_name":"Second Player"}`, nil, http.StatusCreated)
	request("PUT", adminEvent+"/registration", `{"registration_open":false}`, cookie, http.StatusOK)
	started := request("POST", adminEvent+"/start", "", cookie, http.StatusOK)
	public := request("GET", publicEvent, "", nil, http.StatusOK)
	if public.Body.String() != started.Body.String() || !strings.Contains(public.Body.String(), `"bracket"`) {
		t.Fatalf("public bracket does not match the published draw: %s", public.Body)
	}
	directory := request("GET", "/api/tournaments", "", nil, http.StatusOK)
	var summaries []tournamentSummary
	if err := json.Unmarshal(directory.Body.Bytes(), &summaries); err != nil {
		t.Fatal(err)
	}
	if len(summaries) != 2 || summaries[0].Slug != "another-cup" || !summaries[0].RegistrationOpen ||
		summaries[1].Slug != "city-cup" || summaries[1].RegistrationOpen || !summaries[1].Started {
		t.Fatalf("unexpected public tournament summaries: %+v", summaries)
	}
	if strings.Contains(directory.Body.String(), `"bracket"`) || strings.Contains(directory.Body.String(), "Flight") {
		t.Fatalf("public tournament list contains bracket or entrant details: %s", directory.Body)
	}
	request("PUT", adminEvent+"/registration", `{"registration_open":true}`, cookie, http.StatusConflict)
	request("PUT", adminEvent+"/format", `{"game_variant":"501","legs_to_win":2}`, cookie, http.StatusConflict)
	request("DELETE", adminEvent+"/entries/3", "", cookie, http.StatusConflict)
	request("DELETE", adminEvent, "", cookie, http.StatusConflict)

	secondAdminEvent := adminPath + "/another-cup"
	request("DELETE", secondAdminEvent, "", cookie, http.StatusNoContent)
	request("DELETE", secondAdminEvent, "", cookie, http.StatusNotFound)
	request("GET", secondEvent, "", nil, http.StatusNotFound)
	request("POST", secondEvent+"/entries", entrant, nil, http.StatusNotFound)
	request("GET", publicEvent, "", nil, http.StatusOK)
	request("POST", adminPath, strings.Replace(eventData, "city-cup", "another-cup", 1), cookie, http.StatusCreated)
	cleanRoster := request("GET", secondAdminEvent+"/entries", "", cookie, http.StatusOK)
	if strings.TrimSpace(cleanRoster.Body.String()) != "[]" {
		t.Fatalf("deleted entries persisted after recreating the tournament: %s", cleanRoster.Body)
	}
}
