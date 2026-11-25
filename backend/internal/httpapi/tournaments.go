package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/tournament"
)

type TournamentHandler struct {
	Service tournament.Service
}

func (h TournamentHandler) RegisterRoutes(mux *http.ServeMux, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /api/tournaments/{slug}", h.getTournament)
	mux.HandleFunc("POST /api/tournaments/{slug}/entries", h.register)
	mux.HandleFunc("GET /api/admin/tournaments", requireAdmin(h.listTournaments))
	mux.HandleFunc("POST /api/admin/tournaments", requireAdmin(h.createTournament))
	mux.HandleFunc("PUT /api/admin/tournaments/{slug}/registration", requireAdmin(h.setRegistration))
	mux.HandleFunc("GET /api/admin/tournaments/{slug}/entries", requireAdmin(h.listEntries))
	mux.HandleFunc("DELETE /api/admin/tournaments/{slug}/entries/{id}", requireAdmin(h.deleteEntry))
}

func (h TournamentHandler) getTournament(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	event, err := h.Service.Store.GetTournament(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeTournamentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (h TournamentHandler) register(w http.ResponseWriter, r *http.Request) {
	var req registerPlayerRequest
	if !readTournamentJSON(w, r, &req) {
		return
	}
	entry, err := h.Service.Register(r.Context(), r.PathValue("slug"), req.DisplayName, req.Nickname)
	if err != nil {
		writeTournamentError(w, err)
		return
	}
	preferredName := entry.Nickname
	if preferredName == "" {
		preferredName = entry.DisplayName
	}
	writeJSON(w, http.StatusCreated, map[string]string{"preferred_name": preferredName})
}

func (h TournamentHandler) listTournaments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	events, err := h.Service.Store.ListTournaments(r.Context())
	if err != nil {
		writeTournamentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, events)
}

func (h TournamentHandler) createTournament(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Slug     string `json:"slug"`
		Name     string `json:"name"`
		StartsOn string `json:"starts_on"`
	}
	if !readTournamentJSON(w, r, &req) {
		return
	}
	event, err := h.Service.Create(r.Context(), req.Slug, req.Name, req.StartsOn)
	if err != nil {
		writeTournamentError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, event)
}

func (h TournamentHandler) setRegistration(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Open *bool `json:"registration_open"`
	}
	if !readTournamentJSON(w, r, &req) {
		return
	}
	if req.Open == nil {
		writeError(w, http.StatusBadRequest, "invalid_registration_state", "Choose whether registration should be open.")
		return
	}
	event, err := h.Service.Store.SetTournamentRegistration(r.Context(), r.PathValue("slug"), *req.Open)
	if err != nil {
		writeTournamentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (h TournamentHandler) listEntries(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	entries, err := h.Service.Store.ListTournamentEntries(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeTournamentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, entries)
}

func (h TournamentHandler) deleteEntry(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id < 1 {
		writeError(w, http.StatusBadRequest, "invalid_entry_id", "Entry id must be a positive integer.")
		return
	}
	if err := h.Service.Store.DeleteTournamentEntry(r.Context(), r.PathValue("slug"), id); err != nil {
		writeTournamentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func readTournamentJSON[T any](w http.ResponseWriter, r *http.Request, dst *T) bool {
	decoder := json.NewDecoder(http.MaxBytesReader(w, r.Body, 4096))
	decoder.DisallowUnknownFields()
	var request *T
	err := decoder.Decode(&request)
	if err == nil && request != nil {
		err = decoder.Decode(new(json.RawMessage))
		if errors.Is(err, io.EOF) {
			*dst = *request
			return true
		}
	}
	var sizeError *http.MaxBytesError
	if errors.As(err, &sizeError) {
		writeError(w, http.StatusRequestEntityTooLarge, "request_too_large", "Request body must be at most 4096 bytes.")
	} else {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must contain one valid JSON object with the expected fields.")
	}
	return false
}

func writeTournamentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, tournament.ErrInvalidName):
		writeError(w, http.StatusBadRequest, "invalid_tournament_name", "Tournament name must contain 2 to 60 characters without control characters.")
	case errors.Is(err, tournament.ErrInvalidSlug):
		writeError(w, http.StatusBadRequest, "invalid_tournament_slug", "Use up to 60 lowercase letters or digits, with single hyphens between words.")
	case errors.Is(err, tournament.ErrInvalidDate):
		writeError(w, http.StatusBadRequest, "invalid_tournament_date", "Enter a valid tournament start date.")
	case errors.Is(err, tournament.ErrNotFound):
		writeError(w, http.StatusNotFound, "tournament_not_found", "Tournament was not found.")
	case errors.Is(err, tournament.ErrEntryNotFound):
		writeError(w, http.StatusNotFound, "tournament_entry_not_found", "Entry was not found in this tournament.")
	case errors.Is(err, tournament.ErrDuplicateSlug):
		writeError(w, http.StatusConflict, "duplicate_tournament_slug", "This tournament link is already in use.")
	case errors.Is(err, league.ErrRegistrationClosed):
		writeError(w, http.StatusConflict, "registration_closed", "Registration is closed for this tournament.")
	case errors.Is(err, league.ErrDuplicatePlayerName):
		writeError(w, http.StatusConflict, "duplicate_display_name", "Display name is already registered for this tournament.")
	default:
		writeDomainError(w, err)
	}
}
