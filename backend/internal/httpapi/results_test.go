package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

func TestRecordResultAndStandingsFlow(t *testing.T) {
	t.Parallel()

	store := league.NewMemoryStore()
	clock := func() time.Time { return time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC) }
	registration := NewRegistrationHandler(league.NewRegistrationServiceWithNow(store, clock))
	season := NewSeasonHandler(league.NewSeasonServiceWithNow(store, clock), league.NewFixtureServiceWithNow(store, clock), "Darts League")
	results := NewResultHandler(league.NewResultServiceWithNow(store, clock))

	registerTestPlayers(t, registration, []string{"Luke Humphries", "Michael Smith", "Peter Wright", "Gerwyn Price"})
	hitEndpoint(t, season.handleSeasonStart, httptest.NewRequest(http.MethodPost, "/api/admin/season/start", nil), http.StatusCreated)

	fixtureList := hitEndpoint(t, season.handlePublicFixtures, httptest.NewRequest(http.MethodGet, "/api/fixtures", nil), http.StatusOK)
	var fixtures struct {
		Weeks []fixtureWeekResponse `json:"weeks"`
	}
	if err := json.Unmarshal(fixtureList.Body.Bytes(), &fixtures); err != nil {
		t.Fatalf("expected valid fixtures response, got %v", err)
	}
	fixtureID := fixtures.Weeks[0].Fixtures[0].ID

	request := httptest.NewRequest(http.MethodPost, "/api/admin/fixtures/1/result", bytes.NewBufferString(`{"player_one_legs":3,"player_two_legs":1,"player_one_average":95.4,"player_two_average":88.2}`))
	request.SetPathValue("fixtureID", strconv.FormatInt(fixtureID, 10))
	hitEndpoint(t, results.handleRecordResult, request, http.StatusCreated)

	standingsResponse := hitEndpoint(t, results.handleStandings, httptest.NewRequest(http.MethodGet, "/api/standings", nil), http.StatusOK)
	var standings struct {
		Rows []standingRowResponse `json:"standings"`
	}
	if err := json.Unmarshal(standingsResponse.Body.Bytes(), &standings); err != nil {
		t.Fatalf("expected valid standings response, got %v", err)
	}
	if len(standings.Rows) != 4 {
		t.Fatalf("expected 4 standings rows, got %d", len(standings.Rows))
	}
	if standings.Rows[0].Points != 2 {
		t.Fatalf("expected top row to have 2 points, got %+v", standings.Rows[0])
	}
	if fixtureID <= 0 {
		t.Fatalf("expected positive fixture id, got %d", fixtureID)
	}
}

func TestRecordResultRejectsInvalidScorelines(t *testing.T) {
	t.Parallel()

	store := league.NewMemoryStore()
	clock := func() time.Time { return time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC) }
	registration := NewRegistrationHandler(league.NewRegistrationServiceWithNow(store, clock))
	season := NewSeasonHandler(league.NewSeasonServiceWithNow(store, clock), league.NewFixtureServiceWithNow(store, clock), "Darts League")
	results := NewResultHandler(league.NewResultServiceWithNow(store, clock))

	registerTestPlayers(t, registration, []string{"Luke Humphries", "Michael Smith"})
	hitEndpoint(t, season.handleSeasonStart, httptest.NewRequest(http.MethodPost, "/api/admin/season/start", nil), http.StatusCreated)

	request := httptest.NewRequest(http.MethodPost, "/api/admin/fixtures/1/result", bytes.NewBufferString(`{"player_one_legs":2,"player_two_legs":2}`))
	request.SetPathValue("fixtureID", "1")
	recorder := hitEndpoint(t, results.handleRecordResult, request, http.StatusBadRequest)
	assertErrorCode(t, recorder.Body.Bytes(), "invalid_result")
}

func TestEditResultCreatesAuditLogEntry(t *testing.T) {
	t.Parallel()

	store := league.NewMemoryStore()
	clockNow := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return clockNow }
	registration := NewRegistrationHandler(league.NewRegistrationServiceWithNow(store, clock))
	season := NewSeasonHandler(league.NewSeasonServiceWithNow(store, clock), league.NewFixtureServiceWithNow(store, clock), "Darts League")
	results := NewResultHandler(league.NewResultServiceWithNow(store, clock))

	registerTestPlayers(t, registration, []string{"Luke Humphries", "Michael Smith"})
	hitEndpoint(t, season.handleSeasonStart, httptest.NewRequest(http.MethodPost, "/api/admin/season/start", nil), http.StatusCreated)

	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/fixtures/1/result", bytes.NewBufferString(`{"player_one_legs":3,"player_two_legs":0,"player_one_average":92.6,"player_two_average":81.3}`))
	createReq.SetPathValue("fixtureID", "1")
	hitEndpoint(t, results.handleRecordResult, createReq, http.StatusCreated)

	clockNow = clockNow.Add(2 * time.Hour)
	editReq := httptest.NewRequest(http.MethodPut, "/api/admin/fixtures/1/result", bytes.NewBufferString(`{"player_one_legs":3,"player_two_legs":2,"player_one_average":97.1,"player_two_average":90.4}`))
	editReq.Header.Set("X-Admin-Actor", "league-admin")
	editReq.SetPathValue("fixtureID", "1")
	hitEndpoint(t, results.handleEditResult, editReq, http.StatusOK)

	auditResp := hitEndpoint(t, results.handleAuditLog, httptest.NewRequest(http.MethodGet, "/api/admin/audit", nil), http.StatusOK)
	var audit struct {
		Entries []struct {
			Actor     string                `json:"actor"`
			Action    string                `json:"action"`
			OldResult league.ResultSnapshot `json:"old_result"`
			NewResult league.ResultSnapshot `json:"new_result"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(auditResp.Body.Bytes(), &audit); err != nil {
		t.Fatalf("expected valid audit response, got %v", err)
	}
	if len(audit.Entries) != 1 {
		t.Fatalf("expected one audit entry, got %d", len(audit.Entries))
	}
	if audit.Entries[0].Actor != "league-admin" || audit.Entries[0].Action != "result_edited" {
		t.Fatalf("unexpected audit entry %+v", audit.Entries[0])
	}
	if audit.Entries[0].OldResult.PlayerTwoLegs != 0 || audit.Entries[0].NewResult.PlayerTwoLegs != 2 {
		t.Fatalf("expected audit snapshots to capture before/after scores, got %+v", audit.Entries[0])
	}
	if audit.Entries[0].OldResult.PlayerOneAverage == nil || *audit.Entries[0].NewResult.PlayerOneAverage != 97.1 {
		t.Fatalf("expected audit snapshots to include averages, got %+v", audit.Entries[0])
	}
}

func TestDeleteResultCreatesAuditLogEntry(t *testing.T) {
	t.Parallel()

	store := league.NewMemoryStore()
	clockNow := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return clockNow }
	registration := NewRegistrationHandler(league.NewRegistrationServiceWithNow(store, clock))
	season := NewSeasonHandler(league.NewSeasonServiceWithNow(store, clock), league.NewFixtureServiceWithNow(store, clock), "Darts League")
	results := NewResultHandler(league.NewResultServiceWithNow(store, clock))

	registerTestPlayers(t, registration, []string{"Luke Humphries", "Michael Smith"})
	hitEndpoint(t, season.handleSeasonStart, httptest.NewRequest(http.MethodPost, "/api/admin/season/start", nil), http.StatusCreated)

	createReq := httptest.NewRequest(http.MethodPost, "/api/admin/fixtures/1/result", bytes.NewBufferString(`{"player_one_legs":3,"player_two_legs":0}`))
	createReq.SetPathValue("fixtureID", "1")
	hitEndpoint(t, results.handleRecordResult, createReq, http.StatusCreated)

	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/admin/fixtures/1/result", nil)
	deleteReq.SetPathValue("fixtureID", "1")
	hitEndpoint(t, results.handleDeleteResult, deleteReq, http.StatusNoContent)

	auditResp := hitEndpoint(t, results.handleAuditLog, httptest.NewRequest(http.MethodGet, "/api/admin/audit", nil), http.StatusOK)
	var audit struct {
		Entries []struct {
			Action    string                 `json:"action"`
			OldResult league.ResultSnapshot  `json:"old_result"`
			NewResult *league.ResultSnapshot `json:"new_result"`
		} `json:"entries"`
	}
	if err := json.Unmarshal(auditResp.Body.Bytes(), &audit); err != nil {
		t.Fatalf("expected valid audit response, got %v", err)
	}
	if len(audit.Entries) != 1 || audit.Entries[0].Action != "result_deleted" {
		t.Fatalf("expected delete audit entry, got %+v", audit.Entries)
	}
	if audit.Entries[0].OldResult.PlayerOneLegs != 3 || audit.Entries[0].NewResult != nil {
		t.Fatalf("expected old result snapshot only, got %+v", audit.Entries[0])
	}
}
