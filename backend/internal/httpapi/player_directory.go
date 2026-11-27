package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/playerstats"
)

type PlayerDirectoryHandler struct {
	Service playerstats.Service
}

func (h PlayerDirectoryHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/players", matchNoStore(h.directory))
	mux.HandleFunc("GET /api/players/records", matchNoStore(h.records))
	mux.HandleFunc("GET /api/seasons/{seasonID}/players/{playerID}/placements", matchNoStore(h.leaguePlacements))
	mux.HandleFunc("GET /api/tournaments/{slug}/players/{entryID}/placements", matchNoStore(h.tournamentPlacements))
}

func (h PlayerDirectoryHandler) directory(w http.ResponseWriter, r *http.Request) {
	profiles, err := h.Service.Directory(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "The player directory is temporarily unavailable.")
		return
	}
	writeJSON(w, http.StatusOK, profiles)
}

func (h PlayerDirectoryHandler) records(w http.ResponseWriter, r *http.Request) {
	records, err := h.Service.Records(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "Player records are temporarily unavailable.")
		return
	}
	writeJSON(w, http.StatusOK, records)
}

func (h PlayerDirectoryHandler) leaguePlacements(w http.ResponseWriter, r *http.Request) {
	seasonID, seasonErr := strconv.ParseInt(r.PathValue("seasonID"), 10, 64)
	playerID, playerErr := strconv.ParseInt(r.PathValue("playerID"), 10, 64)
	if seasonErr != nil || playerErr != nil || seasonID <= 0 || playerID <= 0 {
		writePlacements(w, playerstats.Placements{}, league.ErrPlayerNotFound)
		return
	}
	placements, err := h.Service.ForLeaguePlayer(r.Context(), seasonID, playerID)
	writePlacements(w, placements, err)
}

func (h PlayerDirectoryHandler) tournamentPlacements(w http.ResponseWriter, r *http.Request) {
	entryID, err := strconv.ParseInt(r.PathValue("entryID"), 10, 64)
	if err != nil || entryID <= 0 {
		writePlacements(w, playerstats.Placements{}, league.ErrPlayerNotFound)
		return
	}
	placements, err := h.Service.ForTournamentPlayer(r.Context(), r.PathValue("slug"), entryID)
	writePlacements(w, placements, err)
}

func writePlacements(w http.ResponseWriter, placements playerstats.Placements, err error) {
	switch {
	case errors.Is(err, league.ErrPlayerNotFound):
		writeError(w, http.StatusNotFound, "not_found", "Player not found.")
	case err != nil:
		writeError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "Player placements are temporarily unavailable.")
	default:
		writeJSON(w, http.StatusOK, placements)
	}
}
