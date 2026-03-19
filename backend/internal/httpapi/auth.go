package httpapi

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const adminSessionCookieName = "darts_league_admin"

var ErrInvalidCredentials = errors.New("invalid admin credentials")

type contextKey string

const adminActorContextKey contextKey = "admin_actor"

type AuthHandler struct {
	username string
	password string
	secret   []byte
	now      func() time.Time
}

func NewAuthHandler(username, password, secret string) AuthHandler {
	return NewAuthHandlerWithNow(username, password, secret, time.Now)
}

func NewAuthHandlerWithNow(username, password, secret string, now func() time.Time) AuthHandler {
	return AuthHandler{
		username: username,
		password: password,
		secret:   []byte(secret),
		now:      now,
	}
}

func (h AuthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("POST /api/admin/login", h.handleLogin)
	mux.HandleFunc("POST /api/admin/logout", h.RequireAdmin(h.handleLogout))
}

func (h AuthHandler) RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		actor, err := h.adminFromRequest(r)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Admin login is required.")
			return
		}
		next(w, r.WithContext(context.WithValue(r.Context(), adminActorContextKey, actor)))
	}
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h AuthHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "Request body must be valid JSON.")
		return
	}
	if req.Username != h.username || req.Password != h.password {
		writeError(w, http.StatusUnauthorized, "invalid_credentials", "Username or password is incorrect.")
		return
	}

	expiresAt := h.now().UTC().Add(12 * time.Hour)
	token := h.signSession(req.Username, expiresAt)
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  expiresAt,
		MaxAge:   int((12 * time.Hour).Seconds()),
	})
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": true, "actor": req.Username})
}

func (h AuthHandler) handleLogout(w http.ResponseWriter, _ *http.Request) {
	http.SetCookie(w, &http.Cookie{
		Name:     adminSessionCookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		Expires:  time.Unix(0, 0).UTC(),
		MaxAge:   -1,
	})
	writeJSON(w, http.StatusOK, map[string]any{"authenticated": false})
}

func (h AuthHandler) adminFromRequest(r *http.Request) (string, error) {
	cookie, err := r.Cookie(adminSessionCookieName)
	if err != nil {
		return "", err
	}
	parts := strings.Split(cookie.Value, ".")
	if len(parts) != 2 {
		return "", ErrInvalidCredentials
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", err
	}
	expectedSig := h.computeSignature(parts[0])
	if subtle.ConstantTimeCompare([]byte(parts[1]), []byte(expectedSig)) != 1 {
		return "", ErrInvalidCredentials
	}
	payload := strings.Split(string(payloadBytes), "|")
	if len(payload) != 2 {
		return "", ErrInvalidCredentials
	}
	expiresUnix, err := strconv.ParseInt(payload[1], 10, 64)
	if err != nil {
		return "", err
	}
	if h.now().UTC().After(time.Unix(expiresUnix, 0).UTC()) {
		return "", ErrInvalidCredentials
	}
	return payload[0], nil
}

func (h AuthHandler) signSession(username string, expiresAt time.Time) string {
	payload := fmt.Sprintf("%s|%d", username, expiresAt.Unix())
	encodedPayload := base64.RawURLEncoding.EncodeToString([]byte(payload))
	return encodedPayload + "." + h.computeSignature(encodedPayload)
}

func (h AuthHandler) computeSignature(payload string) string {
	mac := hmac.New(sha256.New, h.secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func adminActorFromContext(ctx context.Context) string {
	actor, _ := ctx.Value(adminActorContextKey).(string)
	return actor
}
