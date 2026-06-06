package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/greg/darts-league/backend/internal/league"
)

type SeasonHandler struct {
	seasons      league.SeasonService
	fixtures     league.FixtureService
	instanceName string
}

func NewSeasonHandler(seasons league.SeasonService, fixtures league.FixtureService, instanceName string) SeasonHandler {
	return SeasonHandler{seasons: seasons, fixtures: fixtures, instanceName: instanceName}
}

func (h SeasonHandler) RegisterRoutes(mux *http.ServeMux, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /api/season", h.handleSeasonSummary)
	mux.HandleFunc("GET /api/divisions", h.handleDivisions)
	mux.HandleFunc("GET /api/divisions/{divisionSlug}/fixtures", h.handlePublicFixtures)
	mux.HandleFunc("PUT /api/admin/season", requireAdmin(h.handleSeasonUpdate))
	mux.HandleFunc("PUT /api/admin/season/config", requireAdmin(h.handleSeasonUpdateConfig))
	mux.HandleFunc("POST /api/admin/season/start", requireAdmin(h.handleSeasonStart))
	mux.HandleFunc("GET /api/admin/season/preview", requireAdmin(h.handleSchedulePreview))
	mux.HandleFunc("GET /api/admin/season/presets", requireAdmin(h.handleGamesPerWeekPresets))
	mux.HandleFunc("GET /api/admin/divisions", requireAdmin(h.handleAdminDivisions))
	mux.HandleFunc("POST /api/admin/divisions/provision", requireAdmin(h.handleProvisionDivisions))
	mux.HandleFunc("PUT /api/admin/divisions/{divisionID}", requireAdmin(h.handleUpdateDivision))
	mux.HandleFunc("GET /api/admin/divisions/{divisionSlug}/fixtures", requireAdmin(h.handleAdminFixtures))
}

func (h SeasonHandler) handleSeasonSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := h.seasons.Summary(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, h.toSeasonSummaryResponse(summary))
}

func (h SeasonHandler) handleSeasonStart(w http.ResponseWriter, r *http.Request) {
	summary, err := h.seasons.StartSeason(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, h.toSeasonSummaryResponse(summary))
}

type updateSeasonRequest struct {
	Name string `json:"name"`
}

func (h SeasonHandler) handleSeasonUpdate(w http.ResponseWriter, r *http.Request) {
	var req updateSeasonRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.")
		return
	}

	summary, err := h.seasons.UpdateName(r.Context(), req.Name)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, h.toSeasonSummaryResponse(summary))
}

type updateSeasonConfigRequest struct {
	GameVariant  string `json:"game_variant"`
	LegsToWin    int    `json:"legs_to_win"`
	GamesPerWeek int    `json:"games_per_week"`
}

func (h SeasonHandler) handleSeasonUpdateConfig(w http.ResponseWriter, r *http.Request) {
	var req updateSeasonConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.")
		return
	}

	summary, err := h.seasons.UpdateConfig(r.Context(), req.GameVariant, req.LegsToWin, req.GamesPerWeek)
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, h.toSeasonSummaryResponse(summary))
}

func (h SeasonHandler) handleSchedulePreview(w http.ResponseWriter, r *http.Request) {
	preview, err := h.seasons.SchedulePreview(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"player_count":   preview.PlayerCount,
		"game_variant":   preview.GameVariant,
		"legs_to_win":    preview.LegsToWin,
		"games_per_week": preview.GamesPerWeek,
		"week_count":     preview.WeekCount,
		"total_fixtures": preview.TotalFixtures,
	})
}

func (h SeasonHandler) handleGamesPerWeekPresets(w http.ResponseWriter, r *http.Request) {
	summary, err := h.seasons.Summary(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	presets := league.GamesPerWeekPresets(summary.PlayerCount)
	response := make([]map[string]any, 0, len(presets))
	for _, preset := range presets {
		response = append(response, map[string]any{
			"games_per_week": preset.GamesPerWeek,
			"week_count":     preset.WeekCount,
		})
	}

	writeJSON(w, http.StatusOK, map[string]any{"presets": response})
}

func (h SeasonHandler) handlePublicFixtures(w http.ResponseWriter, r *http.Request) {
	weeks, currentWeek, err := h.fixtures.PublicSchedule(r.Context(), r.PathValue("divisionSlug"))
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"current_week": currentWeek,
		"weeks":        toFixtureWeekResponses(weeks),
	})
}

func (h SeasonHandler) handleAdminFixtures(w http.ResponseWriter, r *http.Request) {
	weeks, err := h.fixtures.AdminSchedule(r.Context(), r.PathValue("divisionSlug"))
	if err != nil {
		writeDomainError(w, err)
		return
	}
	response := make([]map[string]any, 0, len(weeks))
	for _, week := range weeks {
		fixtures := make([]map[string]any, 0, len(week.Fixtures))
		for _, fixture := range week.Fixtures {
			item := map[string]any{
				"id":           fixture.ID,
				"player_one":   fixture.PlayerOne,
				"player_two":   fixture.PlayerTwo,
				"scheduled_at": fixture.ScheduledAt.UTC().Format(http.TimeFormat),
				"game_variant": fixture.GameVariant,
				"legs_to_win":  fixture.LegsToWin,
				"status":       fixture.Status,
			}
			if fixture.Result != nil {
				item["result"] = fixture.Result
			}
			fixtures = append(fixtures, item)
		}
		response = append(response, map[string]any{
			"week_number": week.WeekNumber,
			"reveal_at":   week.RevealAt.UTC().Format(http.TimeFormat),
			"fixtures":    fixtures,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"weeks": response})
}

func (h SeasonHandler) handleDivisions(w http.ResponseWriter, r *http.Request) {
	divisions, err := h.seasons.ListDivisions(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"divisions": toDivisionResponses(divisions)})
}

func (h SeasonHandler) handleAdminDivisions(w http.ResponseWriter, r *http.Request) {
	h.handleDivisions(w, r)
}

type provisionDivisionsRequest struct {
	Count int `json:"count"`
}

func (h SeasonHandler) handleProvisionDivisions(w http.ResponseWriter, r *http.Request) {
	var req provisionDivisionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.")
		return
	}
	divisions, err := h.seasons.ProvisionDivisions(r.Context(), req.Count)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"divisions": toDivisionResponses(divisions)})
}

type updateDivisionRequest struct {
	Name                 string `json:"name"`
	Slug                 string `json:"slug"`
	SlackPublicChannelID string `json:"slack_public_channel_id"`
}

func (h SeasonHandler) handleUpdateDivision(w http.ResponseWriter, r *http.Request) {
	divisionID, err := strconv.ParseInt(r.PathValue("divisionID"), 10, 64)
	if err != nil || divisionID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_division_id", "Division id must be a positive integer.")
		return
	}
	var req updateDivisionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.")
		return
	}
	division, err := h.seasons.UpdateDivision(r.Context(), divisionID, req.Name, req.Slug, req.SlackPublicChannelID)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, toDivisionResponse(division))
}

type seasonSummaryResponse struct {
	ID               int64  `json:"id"`
	InstanceName     string `json:"instance_name"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	Timezone         string `json:"timezone"`
	StartedAt        string `json:"started_at,omitempty"`
	RegistrationOpen bool   `json:"registration_open"`
	SeasonStarted    bool   `json:"season_started"`
	AdminLocked      bool   `json:"admin_locked"`
	CanStartSeason   bool   `json:"can_start_season"`
	CanEditSettings  bool   `json:"can_edit_settings"`
	CanEditDivisionChannel bool `json:"can_edit_division_channel"`
	CanEditDivisions bool   `json:"can_edit_divisions"`
	CanAssignPlayers bool   `json:"can_assign_players"`
	PlayerCount      int    `json:"player_count"`
	WeekCount        int    `json:"week_count"`
	GameVariant      string `json:"game_variant"`
	LegsToWin        int    `json:"legs_to_win"`
	GamesPerWeek     int    `json:"games_per_week"`
	TotalFixtures    int    `json:"total_fixtures"`
	DivisionCount    int    `json:"division_count"`
	AssignedCount    int    `json:"assigned_count"`
	WaitlistCount    int    `json:"waitlist_count"`
}

type divisionResponse struct {
	ID                   int64  `json:"id"`
	Name                 string `json:"name"`
	Slug                 string `json:"slug"`
	Position             int    `json:"position"`
	SlackPublicChannelID string `json:"slack_public_channel_id,omitempty"`
}

type fixtureWeekResponse struct {
	WeekNumber int                     `json:"week_number"`
	Status     string                  `json:"status"`
	RevealAt   string                  `json:"reveal_at"`
	Fixtures   []publicFixtureResponse `json:"fixtures"`
}

type publicFixtureResponse struct {
	ID          int64                  `json:"id"`
	PlayerOne   string                 `json:"player_one"`
	PlayerTwo   string                 `json:"player_two"`
	ScheduledAt string                 `json:"scheduled_at,omitempty"`
	GameVariant string                 `json:"game_variant,omitempty"`
	LegsToWin   int                    `json:"legs_to_win,omitempty"`
	Result      *league.ResultSnapshot `json:"result,omitempty"`
}

func (h SeasonHandler) toSeasonSummaryResponse(summary league.SeasonSummary) seasonSummaryResponse {
	response := seasonSummaryResponse{
		ID:               summary.ID,
		InstanceName:     h.instanceName,
		Name:             summary.Name,
		Status:           string(summary.Status),
		Timezone:         summary.Timezone,
		RegistrationOpen: summary.RegistrationOpen,
		SeasonStarted:    summary.SeasonStarted,
		AdminLocked:      summary.AdminLocked,
		CanStartSeason:   summary.CanStartSeason,
		CanEditSettings:  summary.CanEditSettings,
		CanEditDivisionChannel: summary.CanEditDivisionChannel,
		CanEditDivisions: summary.CanEditDivisions,
		CanAssignPlayers: summary.CanAssignPlayers,
		PlayerCount:      summary.PlayerCount,
		WeekCount:        summary.WeekCount,
		GameVariant:      summary.GameVariant,
		LegsToWin:        summary.LegsToWin,
		GamesPerWeek:     summary.GamesPerWeek,
		TotalFixtures:    summary.TotalFixtures,
		DivisionCount:    summary.DivisionCount,
		AssignedCount:    summary.AssignedCount,
		WaitlistCount:    summary.WaitlistCount,
	}
	if summary.StartedAt != nil {
		response.StartedAt = summary.StartedAt.UTC().Format(http.TimeFormat)
	}
	return response
}

func toDivisionResponses(divisions []league.Division) []divisionResponse {
	response := make([]divisionResponse, 0, len(divisions))
	for _, division := range divisions {
		response = append(response, toDivisionResponse(division))
	}
	return response
}

func toDivisionResponse(division league.Division) divisionResponse {
	return divisionResponse{
		ID:                   division.ID,
		Name:                 division.Name,
		Slug:                 division.Slug,
		Position:             division.Position,
		SlackPublicChannelID: division.SlackPublicChannelID,
	}
}

func toFixtureWeekResponses(weeks []league.PublicFixtureWeek) []fixtureWeekResponse {
	response := make([]fixtureWeekResponse, 0, len(weeks))
	for _, week := range weeks {
		response = append(response, toFixtureWeekResponse(week))
	}
	return response
}

func toFixtureWeekResponse(week league.PublicFixtureWeek) fixtureWeekResponse {
	fixtures := make([]publicFixtureResponse, 0, len(week.Fixtures))
	for _, fixture := range week.Fixtures {
		item := publicFixtureResponse{
			ID:        fixture.ID,
			PlayerOne: fixture.PlayerOne,
			PlayerTwo: fixture.PlayerTwo,
		}
		if fixture.ScheduledAt != nil {
			item.ScheduledAt = fixture.ScheduledAt.UTC().Format(http.TimeFormat)
			item.GameVariant = fixture.GameVariant
			item.LegsToWin = fixture.LegsToWin
			item.Result = fixture.Result
		}
		fixtures = append(fixtures, item)
	}

	return fixtureWeekResponse{
		WeekNumber: week.WeekNumber,
		Status:     week.Status,
		RevealAt:   week.RevealAt.UTC().Format(http.TimeFormat),
		Fixtures:   fixtures,
	}
}
