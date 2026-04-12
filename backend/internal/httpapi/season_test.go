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

func TestSeasonSummaryShowsRegistrationOpenState(t *testing.T) {
	t.Parallel()

	handler := newSeasonHandlerWithNow(time.Date(2026, time.March, 20, 10, 0, 0, 0, time.UTC))
	recorder := hitEndpoint(t, handler.season.handleSeasonSummary, httptest.NewRequest(http.MethodGet, "/api/season", nil), http.StatusOK)

	var response seasonSummaryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid season response, got %v", err)
	}

	if response.Status != string(league.SeasonStatusRegistrationOpen) {
		t.Fatalf("expected registration_open status, got %q", response.Status)
	}
	if response.InstanceName != "Cardiff Office - Darts League" {
		t.Fatalf("expected instance name in response, got %q", response.InstanceName)
	}
	if !response.RegistrationOpen {
		t.Fatal("expected registration to be open")
	}
}

func TestSeasonStartGeneratesFixturesAndClosesRegistration(t *testing.T) {
	t.Parallel()

	handler := newSeasonHandlerWithNow(time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC))
	hitEndpoint(t, handler.registration.handleRegisterPlayer, httptest.NewRequest(http.MethodPost, "/api/players/register", bytes.NewBufferString(`{"display_name":"Luke Humphries","nickname":"The Freeze"}`)), http.StatusCreated)
	hitEndpoint(t, handler.registration.handleRegisterPlayer, httptest.NewRequest(http.MethodPost, "/api/players/register", bytes.NewBufferString(`{"display_name":"Michael Smith","nickname":"Bully Boy"}`)), http.StatusCreated)
	registerTestPlayers(t, handler.registration, []string{"Peter Wright", "Gerwyn Price"})

	recorder := hitEndpoint(t, handler.season.handleSeasonStart, httptest.NewRequest(http.MethodPost, "/api/admin/season/start", nil), http.StatusCreated)

	var response seasonSummaryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid season response, got %v", err)
	}

	if response.Status != string(league.SeasonStatusStarted) {
		t.Fatalf("expected started status, got %q", response.Status)
	}
	if response.InstanceName != "Cardiff Office - Darts League" {
		t.Fatalf("expected instance name in start response, got %q", response.InstanceName)
	}
	if response.RegistrationOpen {
		t.Fatal("expected registration to be closed after season start")
	}
	if response.WeekCount != 3 {
		t.Fatalf("expected 3 weeks for 4 players, got %d", response.WeekCount)
	}
}

func TestSeasonUpdateRenamesActiveSeasonBeforeStart(t *testing.T) {
	t.Parallel()

	handler := newSeasonHandlerWithNow(time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC))
	request := httptest.NewRequest(http.MethodPut, "/api/admin/season", bytes.NewBufferString(`{"name":"  Cardiff   Premier   League  "}`))

	recorder := hitEndpoint(t, handler.season.handleSeasonUpdate, request, http.StatusOK)

	var response seasonSummaryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid season response, got %v", err)
	}

	if response.Name != "Cardiff Premier League" {
		t.Fatalf("expected normalized season name, got %q", response.Name)
	}
	if !response.RegistrationOpen {
		t.Fatal("expected registration to remain open after rename")
	}
}

func TestSeasonUpdateRejectsInvalidName(t *testing.T) {
	t.Parallel()

	handler := newSeasonHandlerWithNow(time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC))
	request := httptest.NewRequest(http.MethodPut, "/api/admin/season", bytes.NewBufferString(`{"name":" "}`))

	recorder := hitEndpoint(t, handler.season.handleSeasonUpdate, request, http.StatusBadRequest)
	assertErrorCode(t, recorder.Body.Bytes(), "season_name_required")
}

func TestSeasonUpdateLocksAfterSeasonStart(t *testing.T) {
	t.Parallel()

	handler := newSeasonHandlerWithNow(time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC))
	registerTestPlayers(t, handler.registration, []string{"Luke Humphries", "Michael Smith"})
	hitEndpoint(t, handler.season.handleSeasonStart, httptest.NewRequest(http.MethodPost, "/api/admin/season/start", nil), http.StatusCreated)

	request := httptest.NewRequest(http.MethodPut, "/api/admin/season", bytes.NewBufferString(`{"name":"Locked League"}`))
	recorder := hitEndpoint(t, handler.season.handleSeasonUpdate, request, http.StatusConflict)
	assertErrorCode(t, recorder.Body.Bytes(), "season_started")
}

func TestPublicFixturesHideFutureWeekDetailsUntilUnlock(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	handler := newSeasonHandlerWithNow(startTime)
	resultService := league.NewResultServiceWithNow(handler.store, handler.clock.Now)
	hitEndpoint(t, handler.registration.handleRegisterPlayer, httptest.NewRequest(http.MethodPost, "/api/players/register", bytes.NewBufferString(`{"display_name":"Luke Humphries","nickname":"The Freeze"}`)), http.StatusCreated)
	hitEndpoint(t, handler.registration.handleRegisterPlayer, httptest.NewRequest(http.MethodPost, "/api/players/register", bytes.NewBufferString(`{"display_name":"Michael Smith","nickname":"Bully Boy"}`)), http.StatusCreated)
	registerTestPlayers(t, handler.registration, []string{"Peter Wright", "Gerwyn Price"})
	hitEndpoint(t, handler.season.handleSeasonStart, httptest.NewRequest(http.MethodPost, "/api/admin/season/start", nil), http.StatusCreated)
	fixtures, err := handler.store.ListFixturesBySeason(httptest.NewRequest(http.MethodGet, "/", nil).Context(), 1)
	if err != nil {
		t.Fatalf("expected fixtures to be listed, got %v", err)
	}
	fixtureID := int64(0)
	for _, fixture := range fixtures {
		if fixture.WeekNumber == 1 {
			fixtureID = fixture.ID
			break
		}
	}
	if fixtureID == 0 {
		t.Fatal("expected a week 1 fixture to exist")
	}
	if _, err := resultService.RecordResult(httptest.NewRequest(http.MethodGet, "/", nil).Context(), fixtureID, 3, 1, nil, nil); err != nil {
		t.Fatalf("expected result to be recorded, got %v", err)
	}

	handler.clock.Set(time.Date(2026, time.March, 23, 10, 0, 0, 0, mustLoadLondon(t)))
	recorder := hitEndpoint(t, handler.season.handlePublicFixtures, httptest.NewRequest(http.MethodGet, "/api/fixtures", nil), http.StatusOK)

	var response struct {
		CurrentWeek int                   `json:"current_week"`
		Weeks       []fixtureWeekResponse `json:"weeks"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid fixtures response, got %v", err)
	}

	if response.CurrentWeek != 1 {
		t.Fatalf("expected current week 1, got %d", response.CurrentWeek)
	}
	if len(response.Weeks) != 3 {
		t.Fatalf("expected 3 public weeks, got %d", len(response.Weeks))
	}
	if response.Weeks[0].Fixtures[0].ScheduledAt == "" {
		t.Fatal("expected unlocked week to include schedule details")
	}
	foundFixtureLabel := false
	for _, fixture := range response.Weeks[0].Fixtures {
		if fixture.PlayerOne == "The Freeze (Luke Humphries)" || fixture.PlayerTwo == "The Freeze (Luke Humphries)" {
			foundFixtureLabel = true
			break
		}
	}
	if !foundFixtureLabel {
		t.Fatalf("expected unlocked week to show full fixture label, got %+v", response.Weeks[0].Fixtures)
	}
	foundPlayedResult := false
	for _, fixture := range response.Weeks[0].Fixtures {
		if fixture.Result != nil && fixture.Result.PlayerOneLegs == 3 && fixture.Result.PlayerTwoLegs == 1 {
			foundPlayedResult = true
			break
		}
	}
	if !foundPlayedResult {
		t.Fatalf("expected unlocked week to include played result, got %+v", response.Weeks[0].Fixtures)
	}
	if response.Weeks[1].Fixtures[0].ScheduledAt != "" {
		t.Fatal("expected locked future week to hide scheduled details")
	}
	if response.Weeks[1].Fixtures[0].Result != nil {
		t.Fatal("expected locked future week to hide played result details")
	}
	if response.Weeks[1].Fixtures[0].PlayerOne != "I knew you'd look" || response.Weeks[1].Fixtures[0].PlayerTwo != "Nothing to see here" {
		t.Fatalf("expected locked future week to use funny placeholders, got %+v", response.Weeks[1].Fixtures[0])
	}
}

func TestCurrentWeekEndpointReturnsUnlockedWeekOnly(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	handler := newSeasonHandlerWithNow(startTime)
	registerTestPlayers(t, handler.registration, []string{"Luke Humphries", "Michael Smith", "Peter Wright", "Gerwyn Price"})
	hitEndpoint(t, handler.season.handleSeasonStart, httptest.NewRequest(http.MethodPost, "/api/admin/season/start", nil), http.StatusCreated)

	handler.clock.Set(time.Date(2026, time.March, 30, 10, 0, 0, 0, mustLoadLondon(t)))
	recorder := hitEndpoint(t, handler.season.handleCurrentWeek, httptest.NewRequest(http.MethodGet, "/api/fixtures/current-week", nil), http.StatusOK)

	var response struct {
		CurrentWeek int                 `json:"current_week"`
		Week        fixtureWeekResponse `json:"week"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid current week response, got %v", err)
	}

	if response.CurrentWeek != 2 || response.Week.WeekNumber != 2 {
		t.Fatalf("expected current week response for week 2, got current=%d week=%d", response.CurrentWeek, response.Week.WeekNumber)
	}
	if response.Week.Status != "unlocked" {
		t.Fatalf("expected current week to be unlocked, got %q", response.Week.Status)
	}
}

func TestAdminFixturesIncludeRecordedResults(t *testing.T) {
	t.Parallel()

	startTime := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	handler := newSeasonHandlerWithNow(startTime)
	resultService := league.NewResultServiceWithNow(handler.store, handler.clock.Now)
	registerTestPlayers(t, handler.registration, []string{"Luke Humphries", "Michael Smith"})
	hitEndpoint(t, handler.season.handleSeasonStart, httptest.NewRequest(http.MethodPost, "/api/admin/season/start", nil), http.StatusCreated)
	if _, err := resultService.RecordResult(httptest.NewRequest(http.MethodGet, "/", nil).Context(), 1, 3, 1, nil, nil); err != nil {
		t.Fatalf("expected result to be recorded, got %v", err)
	}

	recorder := hitEndpoint(t, handler.season.handleAdminFixtures, httptest.NewRequest(http.MethodGet, "/api/admin/fixtures", nil), http.StatusOK)

	var response struct {
		Weeks []struct {
			Fixtures []struct {
				ID     int64 `json:"id"`
				Result struct {
					PlayerOneLegs int `json:"player_one_legs"`
					PlayerTwoLegs int `json:"player_two_legs"`
				} `json:"result"`
			} `json:"fixtures"`
		} `json:"weeks"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid admin fixtures response, got %v", err)
	}
	if len(response.Weeks) != 1 || len(response.Weeks[0].Fixtures) != 1 {
		t.Fatalf("expected one admin fixture, got %+v", response.Weeks)
	}
	if response.Weeks[0].Fixtures[0].Result.PlayerOneLegs != 3 || response.Weeks[0].Fixtures[0].Result.PlayerTwoLegs != 1 {
		t.Fatalf("expected admin fixtures to include recorded result, got %+v", response.Weeks[0].Fixtures[0].Result)
	}
}

func TestSeasonSummaryIncludesMatchConfig(t *testing.T) {
	t.Parallel()

	handler := newSeasonHandlerWithNow(time.Date(2026, time.March, 20, 10, 0, 0, 0, time.UTC))
	recorder := hitEndpoint(t, handler.season.handleSeasonSummary, httptest.NewRequest(http.MethodGet, "/api/season", nil), http.StatusOK)

	var response seasonSummaryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid season response, got %v", err)
	}

	if response.GameVariant != "501" {
		t.Fatalf("expected default game variant 501, got %q", response.GameVariant)
	}
	if response.LegsToWin != 3 {
		t.Fatalf("expected default legs to win 3, got %d", response.LegsToWin)
	}
	if response.GamesPerWeek != 1 {
		t.Fatalf("expected default games per week 1, got %d", response.GamesPerWeek)
	}
}

func TestSeasonUpdateConfigBeforeStart(t *testing.T) {
	t.Parallel()

	handler := newSeasonHandlerWithNow(time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC))
	registerTestPlayers(t, handler.registration, []string{"Luke Humphries", "Michael Smith", "Peter Wright", "Gerwyn Price"})

	request := httptest.NewRequest(http.MethodPut, "/api/admin/season/config", bytes.NewBufferString(`{"game_variant":"301","legs_to_win":5,"games_per_week":2}`))
	recorder := hitEndpoint(t, handler.season.handleSeasonUpdateConfig, request, http.StatusOK)

	var response seasonSummaryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid season response, got %v", err)
	}

	if response.GameVariant != "301" {
		t.Fatalf("expected 301, got %q", response.GameVariant)
	}
	if response.LegsToWin != 5 {
		t.Fatalf("expected 5, got %d", response.LegsToWin)
	}
	if response.GamesPerWeek != 2 {
		t.Fatalf("expected 2, got %d", response.GamesPerWeek)
	}
}

func TestSeasonUpdateConfigRejectsInvalidVariant(t *testing.T) {
	t.Parallel()

	handler := newSeasonHandlerWithNow(time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC))
	request := httptest.NewRequest(http.MethodPut, "/api/admin/season/config", bytes.NewBufferString(`{"game_variant":"401","legs_to_win":3,"games_per_week":1}`))
	recorder := hitEndpoint(t, handler.season.handleSeasonUpdateConfig, request, http.StatusBadRequest)
	assertErrorCode(t, recorder.Body.Bytes(), "invalid_game_variant")
}

func TestSeasonUpdateConfigLockedAfterStart(t *testing.T) {
	t.Parallel()

	handler := newSeasonHandlerWithNow(time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC))
	registerTestPlayers(t, handler.registration, []string{"Luke Humphries", "Michael Smith"})
	hitEndpoint(t, handler.season.handleSeasonStart, httptest.NewRequest(http.MethodPost, "/api/admin/season/start", nil), http.StatusCreated)

	request := httptest.NewRequest(http.MethodPut, "/api/admin/season/config", bytes.NewBufferString(`{"game_variant":"301","legs_to_win":5,"games_per_week":1}`))
	recorder := hitEndpoint(t, handler.season.handleSeasonUpdateConfig, request, http.StatusConflict)
	assertErrorCode(t, recorder.Body.Bytes(), "season_started")
}

func TestGamesPerWeekPresetsEndpoint(t *testing.T) {
	t.Parallel()

	handler := newSeasonHandlerWithNow(time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC))
	registerTestPlayers(t, handler.registration, []string{"Luke Humphries", "Michael Smith", "Peter Wright", "Gerwyn Price"})

	recorder := hitEndpoint(t, handler.season.handleGamesPerWeekPresets, httptest.NewRequest(http.MethodGet, "/api/admin/season/presets", nil), http.StatusOK)

	var response struct {
		Presets []struct {
			GamesPerWeek int `json:"games_per_week"`
			WeekCount    int `json:"week_count"`
		} `json:"presets"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid presets response, got %v", err)
	}

	if len(response.Presets) != 3 {
		t.Fatalf("expected 3 presets for 4 players, got %d", len(response.Presets))
	}
	if response.Presets[0].GamesPerWeek != 1 || response.Presets[0].WeekCount != 3 {
		t.Fatalf("expected first preset 1 game/week = 3 weeks, got %+v", response.Presets[0])
	}
}

func TestSchedulePreviewEndpoint(t *testing.T) {
	t.Parallel()

	handler := newSeasonHandlerWithNow(time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC))
	registerTestPlayers(t, handler.registration, []string{"Luke Humphries", "Michael Smith", "Peter Wright", "Gerwyn Price"})

	recorder := hitEndpoint(t, handler.season.handleSchedulePreview, httptest.NewRequest(http.MethodGet, "/api/admin/season/preview", nil), http.StatusOK)

	var response struct {
		PlayerCount   int    `json:"player_count"`
		GameVariant   string `json:"game_variant"`
		LegsToWin     int    `json:"legs_to_win"`
		GamesPerWeek  int    `json:"games_per_week"`
		WeekCount     int    `json:"week_count"`
		TotalFixtures int    `json:"total_fixtures"`
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid preview response, got %v", err)
	}

	if response.PlayerCount != 4 || response.TotalFixtures != 6 || response.WeekCount != 3 {
		t.Fatalf("unexpected preview: %+v", response)
	}
}

func TestSeasonStartWithCustomConfig(t *testing.T) {
	t.Parallel()

	handler := newSeasonHandlerWithNow(time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC))
	registerTestPlayers(t, handler.registration, []string{"Luke Humphries", "Michael Smith", "Peter Wright", "Gerwyn Price"})

	// Set custom config
	configReq := httptest.NewRequest(http.MethodPut, "/api/admin/season/config", bytes.NewBufferString(`{"game_variant":"301","legs_to_win":2,"games_per_week":2}`))
	hitEndpoint(t, handler.season.handleSeasonUpdateConfig, configReq, http.StatusOK)

	// Start the season
	recorder := hitEndpoint(t, handler.season.handleSeasonStart, httptest.NewRequest(http.MethodPost, "/api/admin/season/start", nil), http.StatusCreated)

	var response seasonSummaryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("expected valid season response, got %v", err)
	}

	if response.GameVariant != "301" {
		t.Fatalf("expected 301, got %q", response.GameVariant)
	}
	if response.LegsToWin != 2 {
		t.Fatalf("expected first-to-2, got %d", response.LegsToWin)
	}
	// 4 players, 3 games per player, 2 games/week = 2 weeks
	if response.WeekCount != 2 {
		t.Fatalf("expected 2 weeks with custom config, got %d", response.WeekCount)
	}
	if response.TotalFixtures != 6 {
		t.Fatalf("expected 6 total fixtures, got %d", response.TotalFixtures)
	}
}

type seasonHandlerBundle struct {
	store        *league.MemoryStore
	clock        *testClock
	registration RegistrationHandler
	season       SeasonHandler
}

func newSeasonHandlerWithNow(now time.Time) seasonHandlerBundle {
	store := league.NewMemoryStore()
	clock := &testClock{now: now}
	registration := league.NewRegistrationServiceWithNow(store, clock.Now)
	seasons := league.NewSeasonServiceWithNow(store, clock.Now)
	fixtures := league.NewFixtureServiceWithNow(store, clock.Now)
	registrationHandler := NewRegistrationHandler(registration)
	seasonHandler := NewSeasonHandler(seasons, fixtures, "Cardiff Office - Darts League")
	return seasonHandlerBundle{store: store, clock: clock, registration: registrationHandler, season: seasonHandler}
}

type testClock struct {
	now time.Time
}

func (c *testClock) Now() time.Time {
	return c.now
}

func (c *testClock) Set(now time.Time) {
	c.now = now
}

func registerTestPlayers(t *testing.T, handler RegistrationHandler, names []string) {
	t.Helper()
	for _, name := range names {
		body := []byte(`{"display_name":"` + name + `"}`)
		hitEndpoint(t, handler.handleRegisterPlayer, httptest.NewRequest(http.MethodPost, "/api/players/register", bytes.NewBuffer(body)), http.StatusCreated)
	}
}

func mustLoadLondon(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		t.Fatalf("expected london timezone, got %v", err)
	}
	return loc
}
