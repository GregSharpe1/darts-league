package httpapi

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/greg/darts-league/backend/internal/league"
)

type RegistrationHandler struct {
	service league.RegistrationService
}

func NewRegistrationHandler(service league.RegistrationService) RegistrationHandler {
	return RegistrationHandler{service: service}
}

func (h RegistrationHandler) RegisterRoutes(mux *http.ServeMux, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("POST /api/players/register", h.handleRegisterPlayer)
	mux.HandleFunc("GET /api/admin/players", requireAdmin(h.handleListPlayers))
	mux.HandleFunc("DELETE /api/admin/players/{playerID}", requireAdmin(h.handleDeletePlayer))
}

type registerPlayerRequest struct {
	DisplayName string `json:"display_name"`
	Nickname    string `json:"nickname"`
}

type playerResponse struct {
	ID            int64  `json:"id"`
	DisplayName   string `json:"display_name"`
	Nickname      string `json:"nickname,omitempty"`
	PreferredName string `json:"preferred_name"`
	AdminLabel    string `json:"admin_label"`
	RegisteredAt  string `json:"registered_at"`
}

func (h RegistrationHandler) handleRegisterPlayer(w http.ResponseWriter, r *http.Request) {
	var req registerPlayerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.")
		return
	}

	player, err := h.service.RegisterPlayer(r.Context(), league.Player{
		DisplayName: req.DisplayName,
		Nickname:    req.Nickname,
	})
	if err != nil {
		writeDomainError(w, err)
		return
	}

	writeJSON(w, http.StatusCreated, toPlayerResponse(player))
}

func (h RegistrationHandler) handleListPlayers(w http.ResponseWriter, r *http.Request) {
	players, err := h.service.ListPlayers(r.Context())
	if err != nil {
		writeDomainError(w, err)
		return
	}

	response := make([]playerResponse, 0, len(players))
	for _, player := range players {
		response = append(response, toPlayerResponse(player))
	}

	writeJSON(w, http.StatusOK, map[string]any{"players": response})
}

func (h RegistrationHandler) handleDeletePlayer(w http.ResponseWriter, r *http.Request) {
	playerID, err := strconv.ParseInt(r.PathValue("playerID"), 10, 64)
	if err != nil || playerID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_player_id", "Player id must be a positive integer.")
		return
	}

	if err := h.service.DeletePlayer(r.Context(), playerID); err != nil {
		writeDomainError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func toPlayerResponse(player league.Player) playerResponse {
	response := playerResponse{
		ID:            player.ID,
		DisplayName:   player.DisplayName,
		PreferredName: player.PreferredName(),
		AdminLabel:    player.AdminLabel(),
	}

	if strings.TrimSpace(player.Nickname) != "" {
		response.Nickname = player.Nickname
	}

	if !player.RegisteredAt.IsZero() {
		response.RegisteredAt = player.RegisteredAt.UTC().Format(http.TimeFormat)
	}

	return response
}

func writeDomainError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, league.ErrDisplayNameRequired):
		writeError(w, http.StatusBadRequest, "display_name_required", "Display name is required.")
	case errors.Is(err, league.ErrDuplicatePlayerName):
		writeError(w, http.StatusConflict, "duplicate_display_name", "Display name already exists for this season.")
	case errors.Is(err, league.ErrRegistrationClosed):
		writeError(w, http.StatusConflict, "registration_closed", "Registration is closed for the active season.")
	case errors.Is(err, league.ErrPlayerDeleteLocked):
		writeError(w, http.StatusConflict, "season_started", "Players can only be deleted before the season starts.")
	case errors.Is(err, league.ErrPlayerNotFound):
		writeError(w, http.StatusNotFound, "player_not_found", "Player was not found in the active season.")
	case errors.Is(err, league.ErrSeasonNotFound):
		writeError(w, http.StatusNotFound, "season_not_found", "No active season is available.")
	case errors.Is(err, league.ErrSeasonAlreadyStarted):
		writeError(w, http.StatusConflict, "season_started", "The active season has already started.")
	case errors.Is(err, league.ErrNotEnoughPlayers):
		writeError(w, http.StatusConflict, "not_enough_players", "At least two players are required to start the season.")
	case errors.Is(err, league.ErrFixtureNotFound):
		writeError(w, http.StatusNotFound, "fixture_not_found", "Fixture was not found.")
	case errors.Is(err, league.ErrInvalidResult):
		writeError(w, http.StatusBadRequest, "invalid_result", "Result must be a valid first-to-3 scoreline.")
	case errors.Is(err, league.ErrResultAlreadyExists):
		writeError(w, http.StatusConflict, "result_exists", "This fixture already has a recorded result.")
	case errors.Is(err, league.ErrResultNotFound):
		writeError(w, http.StatusNotFound, "result_not_found", "This fixture does not have a recorded result yet.")
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "Something went wrong.")
	}
}

func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{
			"code":    code,
			"message": message,
		},
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
