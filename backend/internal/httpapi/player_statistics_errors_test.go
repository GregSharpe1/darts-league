package httpapi

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

type statisticsFailureStore struct {
	*league.MemoryStore
	lookups int
}

func (s *statisticsFailureStore) Transaction(context.Context, func(league.Store) error) error {
	s.lookups++
	return errors.New("private-database-credentials")
}

func TestPlayerStatisticsAuthenticatesBeforeLookupAndSanitizesErrors(t *testing.T) {
	store := &statisticsFailureStore{MemoryStore: league.NewMemoryStore()}
	now := time.Date(2026, 3, 30, 9, 0, 0, 0, time.UTC)
	auth := NewAuthHandlerWithNow("admin", "secret", "test-only", func() time.Time { return now })
	mux := http.NewServeMux()
	NewPlayerStatisticsHandler(league.NewResultService(store)).RegisterRoutes(mux, auth.RequireAdmin)
	for _, tc := range []struct {
		prefix, cookie  string
		status, lookups int
	}{
		{"admin/", "", 401, 0},
		{"admin/", "forged-session", 401, 0},
		{"admin/", auth.signSession("admin", now.Add(-time.Hour)), 401, 0},
		{"admin/", auth.signSession("admin", now.Add(time.Hour)), 503, 1},
		{"", "", 503, 2},
	} {
		req := httptest.NewRequest("GET", "/api/"+tc.prefix+"seasons/1/players/1/statistics", nil)
		if tc.cookie != "" {
			req.AddCookie(&http.Cookie{Name: adminSessionCookieName, Value: tc.cookie})
		}
		w := httptest.NewRecorder()
		mux.ServeHTTP(w, req)
		if w.Code != tc.status || store.lookups != tc.lookups || w.Header().Get("Cache-Control") != "no-store" || strings.Contains(w.Body.String(), "private-database") {
			t.Fatalf("status=%d lookups=%d body=%s", w.Code, store.lookups, w.Body.String())
		}
	}
}
