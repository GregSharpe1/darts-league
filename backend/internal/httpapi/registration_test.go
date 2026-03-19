package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
)

func TestRegisterPlayerReturnsCreatedPlayer(t *testing.T) {
	t.Parallel()

	_, handler := newTestHandler()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/players/register", bytes.NewBufferString(`{"display_name":"Luke Humphries","nickname":"The Freeze"}`))

	handler.handleRegisterPlayer(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}

	var response map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid json response, got %v", err)
	}

	if got := response["preferred_name"]; got != "The Freeze" {
		t.Fatalf("expected preferred name %q, got %#v", "The Freeze", got)
	}

	if got := response["display_name"]; got != "Luke Humphries" {
		t.Fatalf("expected display name %q, got %#v", "Luke Humphries", got)
	}
}

func TestRegisterPlayerRejectsDuplicateDisplayNames(t *testing.T) {
	t.Parallel()

	_, handler := newTestHandler()

	first := httptest.NewRequest(http.MethodPost, "/api/players/register", bytes.NewBufferString(`{"display_name":"Luke Humphries"}`))
	hitEndpoint(t, handler.handleRegisterPlayer, first, http.StatusCreated)

	second := httptest.NewRequest(http.MethodPost, "/api/players/register", bytes.NewBufferString(`{"display_name":"  luke   HUMPHRIES "}`))
	recorder := hitEndpoint(t, handler.handleRegisterPlayer, second, http.StatusConflict)

	assertErrorCode(t, recorder.Body.Bytes(), "duplicate_display_name")
}

func TestListPlayersReturnsAlphabeticalAdminView(t *testing.T) {
	t.Parallel()

	_, handler := newTestHandler()
	hitEndpoint(t, handler.handleRegisterPlayer, httptest.NewRequest(http.MethodPost, "/api/players/register", bytes.NewBufferString(`{"display_name":"Peter Wright","nickname":"Snakebite"}`)), http.StatusCreated)
	hitEndpoint(t, handler.handleRegisterPlayer, httptest.NewRequest(http.MethodPost, "/api/players/register", bytes.NewBufferString(`{"display_name":"Luke Humphries","nickname":"The Freeze"}`)), http.StatusCreated)

	recorder := hitEndpoint(t, handler.handleListPlayers, httptest.NewRequest(http.MethodGet, "/api/admin/players", nil), http.StatusOK)

	var response struct {
		Players []struct {
			DisplayName string `json:"display_name"`
			AdminLabel  string `json:"admin_label"`
		} `json:"players"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid list response, got %v", err)
	}

	if len(response.Players) != 2 {
		t.Fatalf("expected 2 players, got %d", len(response.Players))
	}

	if response.Players[0].DisplayName != "Luke Humphries" {
		t.Fatalf("expected Luke Humphries first, got %q", response.Players[0].DisplayName)
	}

	if response.Players[1].AdminLabel != "Snakebite (Peter Wright)" {
		t.Fatalf("unexpected admin label %q", response.Players[1].AdminLabel)
	}
}

func TestDeletePlayerRemovesPlayerBeforeSeasonStart(t *testing.T) {
	t.Parallel()

	_, handler := newTestHandler()
	createResponse := hitEndpoint(t, handler.handleRegisterPlayer, httptest.NewRequest(http.MethodPost, "/api/players/register", bytes.NewBufferString(`{"display_name":"Nathan Aspinall"}`)), http.StatusCreated)

	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(createResponse.Body.Bytes(), &created); err != nil {
		t.Fatalf("expected valid create response, got %v", err)
	}

	request := httptest.NewRequest(http.MethodDelete, "/api/admin/players/1", nil)
	request.SetPathValue("playerID", strconv.FormatInt(created.ID, 10))
	recorder := hitEndpoint(t, handler.handleDeletePlayer, request, http.StatusNoContent)

	if recorder.Body.Len() != 0 {
		t.Fatalf("expected empty delete response body, got %q", recorder.Body.String())
	}

	listResponse := hitEndpoint(t, handler.handleListPlayers, httptest.NewRequest(http.MethodGet, "/api/admin/players", nil), http.StatusOK)
	var list struct {
		Players []any `json:"players"`
	}
	if err := json.Unmarshal(listResponse.Body.Bytes(), &list); err != nil {
		t.Fatalf("expected valid list response, got %v", err)
	}

	if len(list.Players) != 0 {
		t.Fatalf("expected player list to be empty after delete, got %d players", len(list.Players))
	}
}

func TestDeletePlayerFailsAfterSeasonStart(t *testing.T) {
	t.Parallel()

	store, handler := newTestHandler()
	hitEndpoint(t, handler.handleRegisterPlayer, httptest.NewRequest(http.MethodPost, "/api/players/register", bytes.NewBufferString(`{"display_name":"Michael Smith"}`)), http.StatusCreated)

	season, err := store.GetActiveSeason(context.Background())
	if err != nil {
		t.Fatalf("expected active season, got %v", err)
	}

	_, err = store.UpsertSeason(context.Background(), season.Start(time.Date(2026, time.March, 23, 9, 0, 0, 0, time.UTC)))
	if err != nil {
		t.Fatalf("expected season start to succeed, got %v", err)
	}

	request := httptest.NewRequest(http.MethodDelete, "/api/admin/players/1", nil)
	request.SetPathValue("playerID", "1")
	recorder := hitEndpoint(t, handler.handleDeletePlayer, request, http.StatusConflict)

	assertErrorCode(t, recorder.Body.Bytes(), "season_started")
}

func newTestHandler() (*testStore, RegistrationHandler) {
	store := newTestStore()
	service := league.NewRegistrationService(store)
	return store, NewRegistrationHandler(service)
}

type testStore struct {
	*league.MemoryStore
}

func newTestStore() *testStore {
	return &testStore{MemoryStore: league.NewMemoryStore()}
}

func hitEndpoint(t *testing.T, handler func(http.ResponseWriter, *http.Request), request *http.Request, expectedStatus int) *httptest.ResponseRecorder {
	t.Helper()
	recorder := httptest.NewRecorder()
	handler(recorder, request)
	if recorder.Code != expectedStatus {
		t.Fatalf("expected status %d, got %d with body %s", expectedStatus, recorder.Code, recorder.Body.String())
	}
	return recorder
}

func assertErrorCode(t *testing.T, body []byte, expected string) {
	t.Helper()
	var response struct {
		Error struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(body, &response); err != nil {
		t.Fatalf("expected valid error payload, got %v", err)
	}
	if response.Error.Code != expected {
		t.Fatalf("expected error code %q, got %q", expected, response.Error.Code)
	}
}
