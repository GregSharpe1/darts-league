package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

func TestProtectedAdminRouteRequiresLogin(t *testing.T) {
	t.Parallel()

	mux, _ := newAuthTestMux()
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/admin/players", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized status, got %d", recorder.Code)
	}
	assertErrorCode(t, recorder.Body.Bytes(), "unauthorized")
}

func TestAdminLoginUnlocksProtectedRoutes(t *testing.T) {
	t.Parallel()

	mux, _ := newAuthTestMux()
	loginRecorder := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewBufferString(`{"username":"admin","password":"secret"}`))
	mux.ServeHTTP(loginRecorder, loginReq)

	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login success, got %d", loginRecorder.Code)
	}
	cookies := loginRecorder.Result().Cookies()
	if len(cookies) == 0 {
		t.Fatal("expected session cookie after login")
	}

	protectedRecorder := httptest.NewRecorder()
	protectedReq := httptest.NewRequest(http.MethodGet, "/api/admin/players", nil)
	protectedReq.AddCookie(cookies[0])
	mux.ServeHTTP(protectedRecorder, protectedReq)

	if protectedRecorder.Code != http.StatusOK {
		t.Fatalf("expected protected route to succeed, got %d with body %s", protectedRecorder.Code, protectedRecorder.Body.String())
	}
}

func TestAdminLogoutExpiresSession(t *testing.T) {
	t.Parallel()

	mux, _ := newAuthTestMux()
	loginRecorder := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewBufferString(`{"username":"admin","password":"secret"}`))
	mux.ServeHTTP(loginRecorder, loginReq)
	sessionCookie := loginRecorder.Result().Cookies()[0]

	logoutRecorder := httptest.NewRecorder()
	logoutReq := httptest.NewRequest(http.MethodPost, "/api/admin/logout", nil)
	logoutReq.AddCookie(sessionCookie)
	mux.ServeHTTP(logoutRecorder, logoutReq)

	if logoutRecorder.Code != http.StatusOK {
		t.Fatalf("expected logout success, got %d", logoutRecorder.Code)
	}
	logoutCookie := logoutRecorder.Result().Cookies()[0]
	if logoutCookie.MaxAge != -1 {
		t.Fatalf("expected logout cookie to expire session, got max age %d", logoutCookie.MaxAge)
	}

	protectedRecorder := httptest.NewRecorder()
	protectedReq := httptest.NewRequest(http.MethodGet, "/api/admin/players", nil)
	protectedReq.AddCookie(logoutCookie)
	mux.ServeHTTP(protectedRecorder, protectedReq)

	if protectedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected expired session to be rejected, got %d", protectedRecorder.Code)
	}
}

func TestLoginRejectsInvalidCredentials(t *testing.T) {
	t.Parallel()

	mux, _ := newAuthTestMux()
	recorder := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewBufferString(`{"username":"admin","password":"wrong"}`))
	mux.ServeHTTP(recorder, req)

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected invalid credentials to fail, got %d", recorder.Code)
	}
	assertErrorCode(t, recorder.Body.Bytes(), "invalid_credentials")
}

func newAuthTestMux() (*http.ServeMux, *league.MemoryStore) {
	store := league.NewMemoryStore()
	clock := func() time.Time { return time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC) }
	auth := NewAuthHandlerWithNow("admin", "secret", "test-secret", clock)
	registration := NewRegistrationHandler(league.NewRegistrationServiceWithNow(store, clock))
	mux := http.NewServeMux()
	auth.RegisterRoutes(mux)
	registration.RegisterRoutes(mux, auth.RequireAdmin)
	return mux, store
}

func TestProtectedEditUsesSessionActorForAudit(t *testing.T) {
	t.Parallel()

	clockNow := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return clockNow }
	store := league.NewMemoryStore()
	auth := NewAuthHandlerWithNow("admin", "secret", "test-secret", clock)
	registration := NewRegistrationHandler(league.NewRegistrationServiceWithNow(store, clock))
	season := NewSeasonHandler(league.NewSeasonServiceWithNow(store, clock), league.NewFixtureServiceWithNow(store, clock), "Darts League")
	results := NewResultHandler(league.NewResultServiceWithNow(store, clock))
	mux := http.NewServeMux()
	auth.RegisterRoutes(mux)
	registration.RegisterRoutes(mux, auth.RequireAdmin)
	season.RegisterRoutes(mux, auth.RequireAdmin)
	results.RegisterRoutes(mux, auth.RequireAdmin)

	for _, name := range []string{"Luke Humphries", "Michael Smith"} {
		body, _ := json.Marshal(map[string]string{"display_name": name})
		req := httptest.NewRequest(http.MethodPost, "/api/players/register", bytes.NewBuffer(body))
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
	}

	loginRecorder := httptest.NewRecorder()
	loginReq := httptest.NewRequest(http.MethodPost, "/api/admin/login", bytes.NewBufferString(`{"username":"admin","password":"secret"}`))
	mux.ServeHTTP(loginRecorder, loginReq)
	sessionCookie := loginRecorder.Result().Cookies()[0]

	startReq := httptest.NewRequest(http.MethodPost, "/api/admin/season/start", nil)
	startReq.AddCookie(sessionCookie)
	startRec := httptest.NewRecorder()
	mux.ServeHTTP(startRec, startReq)

	recordReq := httptest.NewRequest(http.MethodPost, "/api/admin/fixtures/1/result", bytes.NewBufferString(`{"player_one_legs":3,"player_two_legs":0}`))
	recordReq.AddCookie(sessionCookie)
	recordRec := httptest.NewRecorder()
	mux.ServeHTTP(recordRec, recordReq)

	clockNow = clockNow.Add(time.Hour)
	editReq := httptest.NewRequest(http.MethodPut, "/api/admin/fixtures/1/result", bytes.NewBufferString(`{"player_one_legs":3,"player_two_legs":2}`))
	editReq.AddCookie(sessionCookie)
	editRec := httptest.NewRecorder()
	mux.ServeHTTP(editRec, editReq)

	auditReq := httptest.NewRequest(http.MethodGet, "/api/admin/audit", nil)
	auditReq.AddCookie(sessionCookie)
	auditRec := httptest.NewRecorder()
	mux.ServeHTTP(auditRec, auditReq)

	var response struct {
		Entries []struct {
			Actor string `json:"actor"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(auditRec.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected audit payload, got %v", err)
	}
	if len(response.Entries) != 1 || response.Entries[0].Actor != "admin" {
		t.Fatalf("expected audit actor from session, got %+v", response.Entries)
	}
}
