package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/greg/darts-league/backend/internal/league"
)

type PlayerStatisticsHandler struct{ results league.ResultService }

func NewPlayerStatisticsHandler(results league.ResultService) PlayerStatisticsHandler {
	return PlayerStatisticsHandler{results: results}
}

func (h PlayerStatisticsHandler) RegisterRoutes(mux *http.ServeMux, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /api/seasons/{seasonID}/players/{playerID}/statistics", matchNoStore(h.handle(false)))
	mux.HandleFunc("GET /api/admin/seasons/{seasonID}/players/{playerID}/statistics", matchNoStore(requireAdmin(h.handle(true))))
}

func (h PlayerStatisticsHandler) handle(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		seasonID, seasonErr := strconv.ParseInt(r.PathValue("seasonID"), 10, 64)
		playerID, playerErr := strconv.ParseInt(r.PathValue("playerID"), 10, 64)
		if seasonErr != nil || playerErr != nil || seasonID <= 0 || playerID <= 0 {
			writeError(w, http.StatusNotFound, "not_found", "Player not found in this season.")
			return
		}
		lookup := h.results.PublicPlayerStatistics
		if admin {
			lookup = h.results.AdminPlayerStatistics
		}
		response, err := lookup(r.Context(), seasonID, playerID)
		if errors.Is(err, league.ErrPlayerNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Player not found in this season.")
			return
		}
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "Player statistics are temporarily unavailable.")
			return
		}
		writeJSON(w, http.StatusOK, response)
	}
}
