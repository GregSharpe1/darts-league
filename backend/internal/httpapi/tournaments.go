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
	mux.HandleFunc("GET /api/tournaments", h.listPublicTournaments)
	mux.HandleFunc("GET /api/tournaments/{slug}", h.getTournament)
	mux.HandleFunc("POST /api/tournaments/{slug}/entries", h.register)
	mux.HandleFunc("GET /api/admin/tournaments", requireAdmin(h.listTournaments))
	mux.HandleFunc("POST /api/admin/tournaments", requireAdmin(h.createTournament))
	mux.HandleFunc("DELETE /api/admin/tournaments/{slug}", requireAdmin(h.deleteTournament))
	mux.HandleFunc("POST /api/admin/tournaments/{slug}/start", requireAdmin(h.startTournament))
	mux.HandleFunc("PUT /api/admin/tournaments/{slug}/registration", requireAdmin(h.setRegistration))
	mux.HandleFunc("PUT /api/admin/tournaments/{slug}/format", requireAdmin(h.updateFormat))
	mux.HandleFunc("PUT /api/admin/tournaments/{slug}/end-date", requireAdmin(h.updateEndDate))
	mux.HandleFunc("PUT /api/admin/tournaments/{slug}/matches/{number}/score", requireAdmin(h.correctScore))
	mux.HandleFunc("GET /api/admin/tournaments/{slug}/entries", requireAdmin(h.listEntries))
	mux.HandleFunc("DELETE /api/admin/tournaments/{slug}/entries/{id}", requireAdmin(h.deleteEntry))
}

type tournamentSummary struct {
	Slug             string `json:"slug"`
	Name             string `json:"name"`
	StartsOn         string `json:"starts_on"`
	EndsOn           string `json:"ends_on,omitempty"`
	RegistrationOpen bool   `json:"registration_open"`
	Started          bool   `json:"started"`
}

func (h TournamentHandler) listPublicTournaments(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Cache-Control", "no-store")
	events, err := h.Service.Store.ListTournaments(r.Context())
	if err != nil {
		writeTournamentError(w, err)
		return
	}
	summaries := make([]tournamentSummary, 0, len(events))
	for _, event := range events {
		summaries = append(summaries, tournamentSummary{
			Slug:             event.Slug,
			Name:             event.Name,
			StartsOn:         event.StartsOn,
			EndsOn:           event.EndsOn,
			RegistrationOpen: event.RegistrationOpen,
			Started:          len(event.Bracket) > 0,
		})
	}
	writeJSON(w, http.StatusOK, summaries)
}

func (h TournamentHandler) startTournament(w http.ResponseWriter, r *http.Request) {
	event, err := h.Service.Store.StartTournament(r.Context(), r.PathValue("slug"))
	if err != nil {
		writeTournamentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (h TournamentHandler) deleteTournament(w http.ResponseWriter, r *http.Request) {
	if err := h.Service.Store.DeleteTournament(r.Context(), r.PathValue("slug")); err != nil {
		writeTournamentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
		LegacySlug  string  `json:"slug"` // Accepted for older clients. The name determines the slug.
		Name        string  `json:"name"`
		StartsOn    string  `json:"starts_on"`
		EndsOn      string  `json:"ends_on"`
		GameVariant *string `json:"game_variant"`
		LegsToWin   *int32  `json:"legs_to_win"`
	}
	if !readTournamentJSON(w, r, &req) {
		return
	}
	format := tournament.DefaultMatchFormat()
	if req.GameVariant != nil {
		format.GameVariant = *req.GameVariant
	}
	if req.LegsToWin != nil {
		format.LegsToWin = *req.LegsToWin
	}
	event, err := h.Service.Create(r.Context(), req.Name, req.StartsOn, req.EndsOn, format)
	if err != nil {
		writeTournamentError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, event)
}

func (h TournamentHandler) correctScore(w http.ResponseWriter, r *http.Request) {
	number, err := strconv.Atoi(r.PathValue("number"))
	if err != nil || number <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_match_number", "Match number must be a positive integer.")
		return
	}
	var req struct {
		Legs         []int   `json:"legs"`
		PreviousLegs []int   `json:"previous_legs"`
		Entrants     []int64 `json:"entrants"`
		ClearMatches []int   `json:"clear_matches"`
		Reason       string  `json:"reason"`
	}
	if !readTournamentJSON(w, r, &req) {
		return
	}
	if len(req.Legs) != 2 || len(req.PreviousLegs) != 2 || len(req.Entrants) != 2 {
		writeError(w, http.StatusBadRequest, "invalid_score_correction", "Provide the corrected score, current score, and both entrants.")
		return
	}
	event, err := h.Service.CorrectScore(r.Context(), r.PathValue("slug"), number, tournament.ScoreCorrection{
		Legs: [2]int(req.Legs), PreviousLegs: [2]int(req.PreviousLegs), Entrants: [2]int64(req.Entrants),
		ClearMatches: req.ClearMatches, Reason: req.Reason, Actor: adminActorFromContext(r.Context()),
	})
	if err != nil {
		writeTournamentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (h TournamentHandler) updateEndDate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		EndsOn *string `json:"ends_on"`
	}
	if !readTournamentJSON(w, r, &req) {
		return
	}
	if req.EndsOn == nil {
		writeError(w, http.StatusBadRequest, "end_date_required", "Supply an end date or an empty string to clear it.")
		return
	}
	event, err := h.Service.UpdateEndDate(r.Context(), r.PathValue("slug"), *req.EndsOn)
	if err != nil {
		writeTournamentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, event)
}

func (h TournamentHandler) updateFormat(w http.ResponseWriter, r *http.Request) {
	var format tournament.MatchFormat
	if !readTournamentJSON(w, r, &format) {
		return
	}
	event, err := h.Service.UpdateFormat(r.Context(), r.PathValue("slug"), format)
	if err != nil {
		writeTournamentError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, event)
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
	case errors.Is(err, tournament.ErrInvalidEndDate):
		writeError(w, http.StatusBadRequest, "invalid_tournament_end_date", err.Error())
	case errors.Is(err, tournament.ErrInvalidDate):
		writeError(w, http.StatusBadRequest, "invalid_tournament_date", "Enter a valid tournament start date.")
	case errors.Is(err, tournament.ErrScoreNotRecorded):
		writeError(w, http.StatusConflict, "score_not_recorded", err.Error())
	case errors.Is(err, tournament.ErrScoreChanged):
		writeError(w, http.StatusConflict, "tournament_score_changed", err.Error())
	case errors.Is(err, tournament.ErrCorrectionReason):
		writeError(w, http.StatusBadRequest, "score_correction_reason", err.Error())
	case errors.Is(err, tournament.ErrScoreUnavailable):
		writeError(w, http.StatusServiceUnavailable, "tournament_score_unavailable", err.Error())
	case errors.Is(err, tournament.ErrNotFound):
		writeError(w, http.StatusNotFound, "tournament_not_found", "Tournament was not found.")
	case errors.Is(err, tournament.ErrEntryNotFound):
		writeError(w, http.StatusNotFound, "tournament_entry_not_found", "Entry was not found in this tournament.")
	case errors.Is(err, tournament.ErrDeleteLocked):
		writeError(w, http.StatusConflict, "tournament_delete_locked", "Only tournaments with open registration can be deleted.")
	case errors.Is(err, tournament.ErrFormatLocked):
		writeError(w, http.StatusConflict, "tournament_format_locked", "Match format is locked once the bracket is published.")
	case errors.Is(err, tournament.ErrTournamentStarted):
		writeError(w, http.StatusConflict, "tournament_started", "The roster is locked because this tournament has started.")
	case errors.Is(err, tournament.ErrCloseBeforeStart):
		writeError(w, http.StatusConflict, "tournament_registration_open", "Close registration before starting the tournament.")
	case errors.Is(err, tournament.ErrTooFewEntrants):
		writeError(w, http.StatusConflict, "not_enough_entrants", "At least two entrants are required to start the tournament.")
	case errors.Is(err, league.ErrRegistrationClosed):
		writeError(w, http.StatusConflict, "registration_closed", "Registration is closed for this tournament.")
	case errors.Is(err, league.ErrDuplicatePlayerName):
		writeError(w, http.StatusConflict, "duplicate_display_name", "Display name is already registered for this tournament.")
	default:
		writeDomainError(w, err)
	}
}
