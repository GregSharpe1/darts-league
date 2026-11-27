package httpapi

import (
	"errors"
	"net/http"

	"github.com/greg/darts-league/backend/internal/playerstats"
)

type PlayerIdentityHandler struct {
	Store playerstats.IdentityStore
}

func (h PlayerIdentityHandler) RegisterRoutes(mux *http.ServeMux, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /api/admin/player-identities", matchNoStore(requireAdmin(h.review)))
	mux.HandleFunc("POST /api/admin/player-identities/decisions", matchNoStore(requireAdmin(h.decide)))
}

func (h PlayerIdentityHandler) review(w http.ResponseWriter, r *http.Request) {
	snapshot, err := h.Store.IdentitySnapshot(r.Context())
	if err != nil {
		writeError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "Player identity review is temporarily unavailable.")
		return
	}
	writeJSON(w, http.StatusOK, snapshot.Review())
}

func (h PlayerIdentityHandler) decide(w http.ResponseWriter, r *http.Request) {
	var decision playerstats.IdentityDecision
	if !readTournamentJSON(w, r, &decision) {
		return
	}
	decision.Actor = adminActorFromContext(r.Context())
	err := h.Store.DecideIdentity(r.Context(), decision)
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, playerstats.ErrIdentityDecision):
		writeError(w, http.StatusBadRequest, "invalid_decision", err.Error())
	case errors.Is(err, playerstats.ErrIdentityNotFound), errors.Is(err, playerstats.ErrIdentityOverlap):
		writeError(w, http.StatusConflict, "identity_conflict", err.Error())
	default:
		writeError(w, http.StatusServiceUnavailable, "temporarily_unavailable", "The identity decision could not be saved.")
	}
}
