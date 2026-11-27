package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/playeraccess"
	"github.com/greg/darts-league/backend/internal/playerstats"
	"github.com/greg/darts-league/backend/internal/tournament"
)

const playerSessionCookie = "darts_player_session"

type PlayerAccessHandler struct {
	Access       playeraccess.Service
	Directory    playerstats.Service
	Registration league.RegistrationService
	Tournament   tournament.Service
}

func (h PlayerAccessHandler) RegisterRoutes(mux *http.ServeMux, requireAdmin func(http.HandlerFunc) http.HandlerFunc) {
	mux.HandleFunc("GET /api/player-access", matchNoStore(h.session))
	if h.Access.Store == nil {
		return
	}
	mux.HandleFunc("POST /api/player-access/link", matchNoStore(h.sameOrigin(h.issue)))
	mux.HandleFunc("POST /api/player-access/verify", matchNoStore(h.sameOrigin(h.verify)))
	mux.HandleFunc("POST /api/player-access/sign-out", matchNoStore(h.sameOrigin(h.signOut)))
	mux.HandleFunc("POST /api/player-access/claims", matchNoStore(h.sameOrigin(h.claim)))
	mux.HandleFunc("POST /api/player-access/leagues/{seasonID}/entry", matchNoStore(h.sameOrigin(h.enterLeague)))
	mux.HandleFunc("POST /api/player-access/tournaments/{slug}/entry", matchNoStore(h.sameOrigin(h.enterTournament)))
	mux.HandleFunc("GET /api/admin/player-claims", matchNoStore(requireAdmin(h.claims)))
	mux.HandleFunc("POST /api/admin/player-claims/{id}/decision", matchNoStore(requireAdmin(h.sameOrigin(h.decide))))
}

func (h PlayerAccessHandler) sameOrigin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if origin := r.Header.Get("Origin"); origin != "" && origin != h.Access.Origin {
			writeError(w, http.StatusForbidden, "invalid_origin", "Open the competition sign-up page to continue.")
			return
		}
		next(w, r)
	}
}

func playerSessionHash(r *http.Request) string {
	cookie, err := r.Cookie(playerSessionCookie)
	if err != nil || len(cookie.Value) != 43 {
		return ""
	}
	return playeraccess.Hash(cookie.Value)
}

func (h PlayerAccessHandler) account(r *http.Request) (playeraccess.Account, error) {
	hash := playerSessionHash(r)
	if hash == "" {
		return playeraccess.Account{}, playeraccess.ErrSession
	}
	return h.Access.Store.PlayerAccount(r.Context(), hash, time.Now())
}

func (h PlayerAccessHandler) session(w http.ResponseWriter, r *http.Request) {
	response := struct {
		Enabled bool                  `json:"enabled"`
		Account *playeraccess.Account `json:"account,omitempty"`
	}{Enabled: h.Access.Store != nil}
	if response.Enabled {
		account, err := h.account(r)
		switch {
		case err == nil:
			response.Account = &account
		case !errors.Is(err, playeraccess.ErrSession):
			writeAccessError(w, err)
			return
		}
	}
	writeJSON(w, http.StatusOK, response)
}

func (h PlayerAccessHandler) issue(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Email string `json:"email"`
		Next  string `json:"next"`
	}
	if !readTournamentJSON(w, r, &req) {
		return
	}
	if err := h.Access.RequestLink(r.Context(), req.Email, req.Next); err != nil {
		writeAccessError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h PlayerAccessHandler) cookie(w http.ResponseWriter, token string, maxAge int) {
	http.SetCookie(w, &http.Cookie{
		Name: playerSessionCookie, Value: token, Path: "/api/player-access",
		HttpOnly: true, Secure: strings.HasPrefix(h.Access.Origin, "https://"), SameSite: http.SameSiteLaxMode, MaxAge: maxAge,
	})
}

func (h PlayerAccessHandler) verify(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if !readTournamentJSON(w, r, &req) {
		return
	}
	if len(req.Token) != 43 {
		writeAccessError(w, playeraccess.ErrExpired)
		return
	}
	session, err := playeraccess.NewToken()
	if err != nil {
		writeAccessError(w, err)
		return
	}
	next, err := h.Access.Store.ExchangeSignIn(r.Context(), playeraccess.Hash(req.Token), playeraccess.Hash(session), time.Now())
	if err != nil {
		writeAccessError(w, err)
		return
	}
	h.cookie(w, session, int(playeraccess.SessionLifetime.Seconds()))
	writeJSON(w, http.StatusOK, map[string]string{"next": playeraccess.ReturnPath(next)})
}

func (h PlayerAccessHandler) signOut(w http.ResponseWriter, r *http.Request) {
	if hash := playerSessionHash(r); hash != "" {
		if err := h.Access.Store.EndPlayerSession(r.Context(), hash); err != nil {
			writeAccessError(w, err)
			return
		}
	}
	h.cookie(w, "", -1)
	w.WriteHeader(http.StatusNoContent)
}

func (h PlayerAccessHandler) claim(w http.ResponseWriter, r *http.Request) {
	account, err := h.account(r)
	if err != nil {
		writeAccessError(w, err)
		return
	}
	var req struct {
		IdentityID int64 `json:"identity_id"`
	}
	if !readTournamentJSON(w, r, &req) {
		return
	}
	profiles, err := h.Directory.Directory(r.Context())
	if err != nil {
		writeAccessError(w, err)
		return
	}
	for _, profile := range profiles {
		if profile.IdentityID == req.IdentityID {
			if err := h.Access.Store.SubmitPlayerClaim(r.Context(), account.Email, req.IdentityID); err != nil {
				writeAccessError(w, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
			return
		}
	}
	writeAccessError(w, playeraccess.ErrClaim)
}

func (h PlayerAccessHandler) registrationAccount(r *http.Request) (playeraccess.Account, error) {
	account, err := h.account(r)
	if err == nil && (account.IdentityID == 0 || account.DisplayName == "") {
		err = playeraccess.ErrClaim
	}
	return account, err
}

func (h PlayerAccessHandler) enterLeague(w http.ResponseWriter, r *http.Request) {
	account, err := h.registrationAccount(r)
	if err != nil {
		writeAccessError(w, err)
		return
	}
	seasonID, err := strconv.ParseInt(r.PathValue("seasonID"), 10, 64)
	if err != nil || seasonID <= 0 {
		writeError(w, http.StatusBadRequest, "invalid_season", "Open the current league sign-up page to enter.")
		return
	}
	ctx := league.WithVerifiedPlayerEmail(r.Context(), account.Email)
	p, err := h.Registration.RegisterPlayer(ctx, league.Player{SeasonID: seasonID, DisplayName: account.DisplayName, Nickname: account.Nickname})
	if err != nil {
		writeDomainError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, toPlayerResponse(p))
}

func (h PlayerAccessHandler) enterTournament(w http.ResponseWriter, r *http.Request) {
	account, err := h.registrationAccount(r)
	if err != nil {
		writeAccessError(w, err)
		return
	}
	ctx := league.WithVerifiedPlayerEmail(r.Context(), account.Email)
	entry, err := h.Tournament.Register(ctx, r.PathValue("slug"), account.DisplayName, account.Nickname)
	if err != nil {
		writeTournamentError(w, err)
		return
	}
	label := entry.Nickname
	if label == "" {
		label = entry.DisplayName
	}
	writeJSON(w, http.StatusCreated, map[string]string{"preferred_name": label})
}

func (h PlayerAccessHandler) claims(w http.ResponseWriter, r *http.Request) {
	claims, err := h.Access.Store.ListPlayerClaims(r.Context())
	if err != nil {
		writeAccessError(w, err)
		return
	}
	if claims == nil {
		claims = []playeraccess.Claim{}
	}
	writeJSON(w, http.StatusOK, claims)
}

func (h PlayerAccessHandler) decide(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		writeError(w, http.StatusNotFound, "not_found", "Claim not found.")
		return
	}
	var req struct {
		Approve *bool `json:"approve"`
	}
	if !readTournamentJSON(w, r, &req) {
		return
	}
	if req.Approve == nil {
		writeError(w, http.StatusBadRequest, "invalid_decision", "Choose whether to approve or reject this claim.")
		return
	}
	if err := h.Access.Store.DecidePlayerClaim(r.Context(), id, *req.Approve, adminActorFromContext(r.Context())); err != nil {
		writeAccessError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func writeAccessError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, playeraccess.ErrInvalid), errors.Is(err, playeraccess.ErrExpired):
		writeError(w, http.StatusBadRequest, "invalid_player_access", err.Error())
	case errors.Is(err, playeraccess.ErrSession):
		writeError(w, http.StatusUnauthorized, "player_sign_in_required", err.Error())
	case errors.Is(err, playeraccess.ErrClaim):
		writeError(w, http.StatusConflict, "player_claim_unavailable", err.Error())
	case errors.Is(err, playeraccess.ErrRate):
		writeError(w, http.StatusTooManyRequests, "too_many_attempts", err.Error())
	default:
		writeError(w, http.StatusServiceUnavailable, "player_access_unavailable", "Player sign-in is temporarily unavailable.")
	}
}
