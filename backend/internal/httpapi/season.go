package httpapi

import (
	"net/http"

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
	mux.HandleFunc("POST /api/admin/season/start", requireAdmin(h.handleSeasonStart))
	mux.HandleFunc("GET /api/admin/fixtures", requireAdmin(h.handleAdminFixtures))
	mux.HandleFunc("GET /api/fixtures", h.handlePublicFixtures)
	mux.HandleFunc("GET /api/fixtures/current-week", h.handleCurrentWeek)
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

func (h SeasonHandler) handlePublicFixtures(w http.ResponseWriter, r *http.Request) {
	weeks, currentWeek, err := h.fixtures.PublicSchedule(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"current_week": currentWeek,
		"weeks":        toFixtureWeekResponses(weeks),
	})
}

func (h SeasonHandler) handleCurrentWeek(w http.ResponseWriter, r *http.Request) {
	weeks, currentWeek, err := h.fixtures.PublicSchedule(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	for _, week := range weeks {
		if week.WeekNumber == currentWeek {
			writeJSON(w, http.StatusOK, map[string]any{
				"current_week": currentWeek,
				"week":         toFixtureWeekResponse(week),
			})
			return
		}
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"current_week": currentWeek,
		"week":         nil,
	})
}

func (h SeasonHandler) handleAdminFixtures(w http.ResponseWriter, r *http.Request) {
	weeks, err := h.fixtures.AdminSchedule(r.Context())
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

type seasonSummaryResponse struct {
	ID               int64  `json:"id"`
	InstanceName     string `json:"instance_name"`
	Name             string `json:"name"`
	Status           string `json:"status"`
	Timezone         string `json:"timezone"`
	StartedAt        string `json:"started_at,omitempty"`
	RegistrationOpen bool   `json:"registration_open"`
	PlayerCount      int    `json:"player_count"`
	WeekCount        int    `json:"week_count"`
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
		PlayerCount:      summary.PlayerCount,
		WeekCount:        summary.WeekCount,
	}
	if summary.StartedAt != nil {
		response.StartedAt = summary.StartedAt.UTC().Format(http.TimeFormat)
	}
	return response
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
