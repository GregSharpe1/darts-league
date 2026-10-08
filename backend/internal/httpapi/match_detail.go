package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/greg/darts-league/backend/internal/league"
)

type MatchDetailHandler struct{ fixtures league.FixtureService }

func NewMatchDetailHandler(fixtures league.FixtureService) MatchDetailHandler {
	return MatchDetailHandler{fixtures: fixtures}
}

func (h MatchDetailHandler) RegisterRoutes(mux *http.ServeMux, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /api/fixtures/{fixtureID}/autodarts", matchNoStore(h.handle(false)))
	mux.HandleFunc("GET /api/admin/fixtures/{fixtureID}/autodarts", matchNoStore(requireAdmin(h.handle(true))))
}

func matchNoStore(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		next(w, r)
	}
}

func (h MatchDetailHandler) handle(admin bool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.ParseInt(r.PathValue("fixtureID"), 10, 64)
		if err != nil || id <= 0 {
			writeError(w, http.StatusNotFound, "not_found", "Match not found.")
			return
		}
		response, err := h.fixtures.MatchDetail(r.Context(), id, admin)
		if errors.Is(err, league.ErrFixtureNotFound) {
			writeError(w, http.StatusNotFound, "not_found", "Match not found.")
			return
		}
		if err != nil {
			writeError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "Match details are temporarily unavailable.")
			return
		}
		writeJSON(w, http.StatusOK, response)
	}
}
