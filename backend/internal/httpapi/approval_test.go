package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

func TestAdminApprovalHTTPContract(t *testing.T) {
	ctx := context.Background()
	store := league.NewMemoryStore()
	clock := func() time.Time { return time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC) }
	seasons := league.NewSeasonServiceWithNow(store, clock)
	reg := league.NewRegistrationServiceWithNow(store, clock)
	divisions, err := seasons.ProvisionDivisions(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Alice", "Bob"} {
		p, err := reg.RegisterPlayer(ctx, league.Player{DisplayName: name})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := reg.AssignPlayer(ctx, p.ID, &divisions[0].ID); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := seasons.StartSeason(ctx); err != nil {
		t.Fatal(err)
	}
	fixtures, err := store.ListFixturesByDivision(ctx, divisions[0].ID)
	if err != nil {
		t.Fatal(err)
	}
	f := fixtures[0]
	pending := league.NewPendingResultServiceWithNow(store, league.NewResultServiceWithNow(store, clock), clock)
	outcome, err := pending.IngestPayload(ctx, []byte(`{"matchId":"api-old","player1":{"name":"A","legsWon":3},"player2":{"name":"B","legsWon":1}}`))
	if err != nil {
		t.Fatal(err)
	}
	auth := NewAuthHandlerWithNow("admin", "secret", "test-secret", clock)
	mux := http.NewServeMux()
	auth.RegisterRoutes(mux)
	NewPendingResultHandler(pending).RegisterRoutes(mux, auth.RequireAdmin)
	NewSeasonHandler(seasons, league.NewFixtureServiceWithNow(store, clock), "Test").RegisterRoutes(mux, auth.RequireAdmin)
	path := fmt.Sprintf("/api/admin/pending-results/%d", outcome.Pending.ID)
	send := func(method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	if rec := send("GET", path, "", nil); rec.Code != http.StatusUnauthorized {
		t.Fatalf("detail leaked: %d", rec.Code)
	}
	login := send("POST", "/api/admin/login", `{"username":"admin","password":"secret"}`, nil)
	if login.Code != 200 {
		t.Fatal(login.Body.String())
	}
	cookie := login.Result().Cookies()[0]
	detail := send("GET", path, "", cookie)
	if detail.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("admin source detail must not be cached")
	}
	var info struct {
		Evidence string  `json:"settings_evidence"`
		PlayedAt *string `json:"played_at"`
		Players  []struct {
			ID string `json:"match_player_id"`
		} `json:"players"`
	}
	if err := json.Unmarshal(detail.Body.Bytes(), &info); err != nil {
		t.Fatal(err)
	}
	if detail.Code != 200 || info.Evidence != "legacy_unverified" || info.PlayedAt != nil || len(info.Players) != 2 || info.Players[0].ID != "legacy-1" {
		t.Fatalf("detail: %s", detail.Body.String())
	}
	req := fmt.Sprintf(`{"season_id":%d,"fixture_id":%d,"mapping":{"legacy-1":%d,"legacy-2":%d},"attest_format":true,"missing_date_reason":"not provided","reason":"verified"}`, f.SeasonID, f.ID, f.PlayerTwoID, f.PlayerOneID)
	if rec := send("POST", path+"/confirm", req, cookie); rec.Code != 400 {
		t.Fatalf("missing expected accepted: %d %s", rec.Code, rec.Body.String())
	}
	req = req[:len(req)-1] + `,"expected_result":null}`
	if rec := send("POST", path+"/confirm", req, cookie); rec.Code != 200 {
		t.Fatalf("approval: %d %s", rec.Code, rec.Body.String())
	}
	if rec := send("POST", path+"/confirm", req, cookie); rec.Code != 409 {
		t.Fatalf("replay: %d", rec.Code)
	}
	schedule := send("GET", "/api/admin/divisions/"+divisions[0].Slug+"/fixtures", "", cookie)
	var response struct {
		Weeks []struct {
			Fixtures []struct {
				SeasonID int64                  `json:"season_id"`
				OneID    int64                  `json:"player_one_id"`
				TwoID    int64                  `json:"player_two_id"`
				Expected *league.ExpectedResult `json:"expected_result"`
			} `json:"fixtures"`
		} `json:"weeks"`
	}
	if err := json.Unmarshal(schedule.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if schedule.Code != 200 || len(response.Weeks) != 1 || len(response.Weeks[0].Fixtures) != 1 {
		t.Fatalf("schedule: %s", schedule.Body.String())
	}
	got := response.Weeks[0].Fixtures[0]
	if got.SeasonID != f.SeasonID || got.OneID != f.PlayerOneID || got.TwoID != f.PlayerTwoID || got.Expected == nil || got.Expected.PlayerTwoLegs != 3 || got.Expected.ID == 0 || got.Expected.UpdatedAt.IsZero() {
		t.Fatalf("fixture comparison: %+v", got)
	}
	logs, err := store.ListAuditLogsBySeason(ctx, f.SeasonID)
	if err != nil || len(logs) != 1 || logs[0].Actor != "admin" || logs[0].Import.Reason != "verified" {
		t.Fatalf("session actor/audit: %+v %v", logs, err)
	}
	public := send("GET", "/api/divisions/"+divisions[0].Slug+"/fixtures", "", nil)
	for _, private := range []string{"legacy-1", "account_id", "source_payload", "digest"} {
		if bytes.Contains(public.Body.Bytes(), []byte(private)) {
			t.Fatalf("public source leak: %s", private)
		}
	}
}
