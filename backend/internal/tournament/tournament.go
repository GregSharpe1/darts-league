package tournament

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/greg/darts-league/backend/internal/league"
)

var (
	ErrNotFound      = errors.New("tournament not found")
	ErrEntryNotFound = errors.New("tournament entry not found")
	ErrDuplicateSlug = errors.New("tournament link is already in use")
	ErrDeleteLocked  = errors.New("only tournaments with open registration can be deleted")
	ErrInvalidName   = errors.New("tournament name must contain 2 to 60 characters without control characters")
	ErrInvalidSlug   = errors.New("tournament link must contain up to 60 lowercase letters, digits, or single hyphens between words")
	ErrInvalidDate   = errors.New("enter a valid tournament start date")
)

var validSlug = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

type Event struct {
	Slug             string  `json:"slug"`
	Name             string  `json:"name"`
	StartsOn         string  `json:"starts_on"`
	RegistrationOpen bool    `json:"registration_open"`
	Bracket          Bracket `json:"bracket,omitempty"`
}

type Entry struct {
	ID           int64     `json:"id"`
	DisplayName  string    `json:"display_name"`
	Nickname     string    `json:"nickname,omitempty"`
	RegisteredAt time.Time `json:"registered_at"`
}

func (e Event) CanDelete() bool {
	return e.RegistrationOpen && len(e.Bracket) == 0
}

type Store interface {
	CreateTournament(context.Context, Event) (Event, error)
	GetTournament(context.Context, string) (Event, error)
	ListTournaments(context.Context) ([]Event, error)
	DeleteTournament(context.Context, string) error
	SetTournamentRegistration(context.Context, string, bool) (Event, error)
	StartTournament(context.Context, string) (Event, error)
	CreateTournamentEntry(context.Context, string, Entry) (Entry, error)
	ListTournamentEntries(context.Context, string) ([]Entry, error)
	DeleteTournamentEntry(context.Context, string, int64) error
}

type Service struct {
	Store Store
	Now   func() time.Time
}

func (s Service) Create(ctx context.Context, slug, name, startsOn string) (Event, error) {
	normalizedName := league.NormalizeWhitespace(name)
	if strings.ContainsFunc(name, unicode.IsControl) || utf8.RuneCountInString(normalizedName) < 2 || utf8.RuneCountInString(normalizedName) > 60 {
		return Event{}, ErrInvalidName
	}
	if len(slug) > 60 || !validSlug.MatchString(slug) {
		return Event{}, ErrInvalidSlug
	}
	date, err := time.Parse(time.DateOnly, startsOn)
	if err != nil || date.Year() < 1 {
		return Event{}, ErrInvalidDate
	}
	return s.Store.CreateTournament(ctx, Event{Slug: slug, Name: normalizedName, StartsOn: startsOn})
}

func (s Service) Register(ctx context.Context, slug, displayName, nickname string) (Entry, error) {
	if err := league.ValidatePlayerName(displayName, nickname); err != nil {
		return Entry{}, err
	}
	return s.Store.CreateTournamentEntry(ctx, slug, Entry{
		DisplayName:  league.NormalizeWhitespace(displayName),
		Nickname:     league.NormalizeWhitespace(nickname),
		RegisteredAt: s.Now().UTC(),
	})
}
