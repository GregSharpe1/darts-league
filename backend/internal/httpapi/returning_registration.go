package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/playerstats"
	"github.com/greg/darts-league/backend/internal/tournament"
)

type ReturningRegistrationHandler struct {
	Profiles     playerstats.Service
	Registration league.RegistrationService
	Tournament   tournament.Service
}

func (h ReturningRegistrationHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/seasons/{seasonID}/returning-entry", matchNoStore(h.leagueEntry))
	mux.HandleFunc("POST /api/tournaments/{slug}/returning-entry", matchNoStore(h.tournamentEntry))
}

type returningEntry struct {
	IdentityID int64 `json:"identity_id"`
}

func (h ReturningRegistrationHandler) selection(w http.ResponseWriter, r *http.Request) (int64, string, string, bool) {
	var req returningEntry
	if !readTournamentJSON(w, r, &req) {
		return 0, "", "", false
	}
	if req.IdentityID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_profile", "Choose an existing player profile.")
		return 0, "", "", false
	}
	displayName, nickname, err := h.Profiles.ReturningName(r.Context(), req.IdentityID)
	if err != nil {
		if !writeReturningError(w, err) {
			writeError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "The player directory is temporarily unavailable.")
		}
		return 0, "", "", false
	}
	return req.IdentityID, displayName, nickname, true
}

func (h ReturningRegistrationHandler) leagueEntry(w http.ResponseWriter, r *http.Request) {
	seasonID, err := strconv.ParseInt(r.PathValue("seasonID"), 10, 64)
	if err != nil || seasonID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_season", "Open the current league sign-up page to enter.")
		return
	}
	id, displayName, nickname, ok := h.selection(w, r)
	if !ok {
		return
	}
	player, err := h.Registration.RegisterPlayer(r.Context(), league.Player{
		SeasonID: seasonID, DisplayName: displayName, Nickname: nickname, RequestedIdentityID: id,
	})
	if err != nil {
		if !writeReturningError(w, err) {
			writeDomainError(w, err)
		}
		return
	}
	writeJSON(w, http.StatusCreated, toPlayerResponse(player))
}

func (h ReturningRegistrationHandler) tournamentEntry(w http.ResponseWriter, r *http.Request) {
	id, displayName, nickname, ok := h.selection(w, r)
	if !ok {
		return
	}
	entry, err := h.Tournament.RegisterEntry(r.Context(), r.PathValue("slug"), tournament.Entry{
		DisplayName: displayName, Nickname: nickname, RequestedIdentityID: id,
	})
	if err != nil {
		if !writeReturningError(w, err) {
			writeTournamentError(w, err)
		}
		return
	}
	label := entry.Nickname
	if label == "" {
		label = entry.DisplayName
	}
	writeJSON(w, http.StatusCreated, map[string]string{"preferred_name": label})
}

func writeReturningError(w http.ResponseWriter, err error) bool {
	if !errors.Is(err, playerstats.ErrIdentityNotFound) {
		return false
	}
	writeError(w, http.StatusConflict, "profile_changed", "This player profile has changed. Refresh the list and choose it again.")
	return true
}
