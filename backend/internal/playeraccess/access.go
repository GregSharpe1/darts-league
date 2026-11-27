package playeraccess

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"net/mail"
	"net/url"
	"regexp"
	"strings"
	"time"
)

var (
	ErrInvalid = errors.New("enter a valid email address or use a new sign-in link")
	ErrExpired = errors.New("the sign-in link is invalid or has expired")
	ErrRate    = errors.New("too many sign-in links requested; try again later")
	ErrClaim   = errors.New("this claim cannot be submitted or has already been reviewed")
	ErrSession = errors.New("sign in with your player email to continue")
)

const SessionLifetime = 30 * 24 * time.Hour

type Account struct {
	Email       string `json:"email"`
	IdentityID  int64  `json:"identity_id,omitempty"`
	DisplayName string `json:"display_name,omitempty"`
	Nickname    string `json:"nickname,omitempty"`
	ClaimStatus string `json:"claim_status,omitempty"`
}

type Claim struct {
	ID int64 `json:"id"`
	Account
	Competition string `json:"competition"`
}

type Store interface {
	ReserveSignIn(context.Context, string, string, string, time.Time) error
	CancelSignIn(context.Context, string) error
	ExchangeSignIn(context.Context, string, string, time.Time) (string, error)
	PlayerAccount(context.Context, string, time.Time) (Account, error)
	EndPlayerSession(context.Context, string) error
	SubmitPlayerClaim(context.Context, string, int64) error
	ListPlayerClaims(context.Context) ([]Claim, error)
	DecidePlayerClaim(context.Context, int64, bool, string) error
}

type Service struct {
	Store  Store
	Origin string
	Send   func(context.Context, string, string) error
}

func EmailAddress(email string) (string, error) {
	email = strings.TrimSpace(email)
	address, err := mail.ParseAddress(email)
	if err != nil || address.Address != email || len(email) > 254 || strings.ContainsAny(email, "\r\n") {
		return "", ErrInvalid
	}
	at := strings.LastIndexByte(email, '@')
	return email[:at+1] + strings.ToLower(email[at+1:]), nil
}

var tournamentPath = regexp.MustCompile(`^/tournaments/[\p{L}\p{N}\p{M}-]+$`)

func ReturnPath(path string) string {
	decoded, err := url.PathUnescape(path)
	if err == nil && (decoded == "/register" || tournamentPath.MatchString(decoded)) {
		return decoded
	}
	return "/register"
}

func NewToken() (string, error) {
	var random [32]byte
	if _, err := rand.Read(random[:]); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(random[:]), nil
}

func Hash(token string) string {
	digest := sha256.Sum256([]byte(token))
	return hex.EncodeToString(digest[:])
}

func (s Service) RequestLink(ctx context.Context, email, returnPath string) error {
	email, err := EmailAddress(email)
	if err != nil {
		return err
	}
	token, err := NewToken()
	if err != nil {
		return err
	}
	hash := Hash(token)
	if err := s.Store.ReserveSignIn(ctx, email, hash, ReturnPath(returnPath), time.Now()); err != nil {
		return err
	}
	if err := s.Send(ctx, email, s.Origin+"/player-access#token="+token); err != nil {
		_ = s.Store.CancelSignIn(ctx, hash)
		return err
	}
	return nil
}
