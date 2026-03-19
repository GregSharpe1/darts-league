package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/greg/darts-league/backend/internal/league"
)

type ResultHandler struct {
	results league.ResultService
}

func NewResultHandler(results league.ResultService) ResultHandler {
	return ResultHandler{results: results}
}

func (h ResultHandler) RegisterRoutes(mux *http.ServeMux, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /api/standings", h.handleStandings)
	mux.HandleFunc("POST /api/admin/fixtures/{fixtureID}/result", requireAdmin(h.handleRecordResult))
	mux.HandleFunc("PUT /api/admin/fixtures/{fixtureID}/result", requireAdmin(h.handleEditResult))
	mux.HandleFunc("DELETE /api/admin/fixtures/{fixtureID}/result", requireAdmin(h.handleDeleteResult))
	mux.HandleFunc("GET /api/admin/audit", requireAdmin(h.handleAuditLog))
}

type resultRequest struct {
	PlayerOneLegs    int      `json:"player_one_legs"`
	PlayerTwoLegs    int      `json:"player_two_legs"`
	PlayerOneAverage *float64 `json:"player_one_average"`
	PlayerTwoAverage *float64 `json:"player_two_average"`
}

type standingRowResponse struct {
	Player        string `json:"player"`
	DisplayName   string `json:"display_name"`
	Played        int    `json:"played"`
	Won           int    `json:"won"`
	Lost          int    `json:"lost"`
	LegsFor       int    `json:"legs_for"`
	LegsAgainst   int    `json:"legs_against"`
	LegDifference int    `json:"leg_difference"`
	Points        int    `json:"points"`
}

func (h ResultHandler) handleStandings(w http.ResponseWriter, r *http.Request) {
	rows, err := h.results.Standings(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	response := make([]standingRowResponse, 0, len(rows))
	for _, row := range rows {
		response = append(response, standingRowResponse{
			Player:        row.PreferredName,
			DisplayName:   row.DisplayName,
			Played:        row.Played,
			Won:           row.Won,
			Lost:          row.Lost,
			LegsFor:       row.LegsFor,
			LegsAgainst:   row.LegsAgainst,
			LegDifference: row.LegDifference,
			Points:        row.Points,
		})
	}
	writeJSON(w, http.StatusOK, map[string]any{"standings": response})
}

func (h ResultHandler) handleRecordResult(w http.ResponseWriter, r *http.Request) {
	fixtureID, err := strconv.ParseInt(r.PathValue("fixtureID"), 10, 64)
	if err != nil || fixtureID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_fixture_id", "Fixture id must be a positive integer.")
		return
	}
	var req resultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.")
		return
	}
	result, err := h.results.RecordResult(r.Context(), fixtureID, req.PlayerOneLegs, req.PlayerTwoLegs, req.PlayerOneAverage, req.PlayerTwoAverage)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"id":                 result.ID,
		"fixture_id":         result.FixtureID,
		"player_one_legs":    result.PlayerOneLegs,
		"player_two_legs":    result.PlayerTwoLegs,
		"player_one_average": result.PlayerOneAverage,
		"player_two_average": result.PlayerTwoAverage,
		"winner_id":          result.WinnerID,
	})
}

func (h ResultHandler) handleEditResult(w http.ResponseWriter, r *http.Request) {
	fixtureID, err := strconv.ParseInt(r.PathValue("fixtureID"), 10, 64)
	if err != nil || fixtureID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_fixture_id", "Fixture id must be a positive integer.")
		return
	}
	var req resultRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.")
		return
	}
	actor := adminActorFromContext(r.Context())
	if actor == "" {
		actor = r.Header.Get("X-Admin-Actor")
	}
	if actor == "" {
		actor = "admin"
	}
	result, err := h.results.EditResult(r.Context(), fixtureID, req.PlayerOneLegs, req.PlayerTwoLegs, req.PlayerOneAverage, req.PlayerTwoAverage, actor)
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":                 result.ID,
		"fixture_id":         result.FixtureID,
		"player_one_legs":    result.PlayerOneLegs,
		"player_two_legs":    result.PlayerTwoLegs,
		"player_one_average": result.PlayerOneAverage,
		"player_two_average": result.PlayerTwoAverage,
		"winner_id":          result.WinnerID,
	})
}

func (h ResultHandler) handleDeleteResult(w http.ResponseWriter, r *http.Request) {
	fixtureID, err := strconv.ParseInt(r.PathValue("fixtureID"), 10, 64)
	if err != nil || fixtureID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_fixture_id", "Fixture id must be a positive integer.")
		return
	}
	actor := adminActorFromContext(r.Context())
	if actor == "" {
		actor = "admin"
	}
	if err := h.results.DeleteResult(r.Context(), fixtureID, actor); err != nil {
		writeDomainError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h ResultHandler) handleAuditLog(w http.ResponseWriter, r *http.Request) {
	entries, err := h.results.AuditLog(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}
	response := make([]map[string]any, 0, len(entries))
	for _, entry := range entries {
		item := map[string]any{
			"id":            entry.ID,
			"fixture_id":    entry.FixtureID,
			"fixture_label": entry.FixtureLabel,
			"action":        entry.Action,
			"actor":         entry.Actor,
			"created_at":    entry.CreatedAt.UTC().Format(http.TimeFormat),
		}
		if entry.OldResult != nil {
			item["old_result"] = entry.OldResult
		}
		if entry.NewResult != nil {
			item["new_result"] = entry.NewResult
		}
		response = append(response, item)
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": response})
}
