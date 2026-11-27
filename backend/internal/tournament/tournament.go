package tournament

import (
	"context"
	"errors"
	"strconv"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/greg/darts-league/backend/internal/league"
	"golang.org/x/text/unicode/norm"
)

var (
	ErrNotFound       = errors.New("tournament not found")
	ErrEntryNotFound  = errors.New("tournament entry not found")
	ErrDuplicateSlug  = errors.New("tournament link is already in use")
	ErrDeleteLocked   = errors.New("only tournaments with open registration can be deleted")
	ErrFormatLocked   = errors.New("tournament match format is locked once the bracket is published")
	ErrInvalidName    = errors.New("tournament name must contain 2 to 60 characters without control characters")
	ErrInvalidDate    = errors.New("enter a valid tournament start date")
	ErrInvalidEndDate = errors.New("enter a tournament end date on or after the start date")
)

type Event struct {
	Slug             string  `json:"slug"`
	Name             string  `json:"name"`
	StartsOn         string  `json:"starts_on"`
	EndsOn           string  `json:"ends_on,omitempty"`
	RegistrationOpen bool    `json:"registration_open"`
	Bracket          Bracket `json:"bracket,omitempty"`
	MatchFormat
}

type MatchFormat struct {
	GameVariant string `json:"game_variant"`
	LegsToWin   int32  `json:"legs_to_win"`
}

func DefaultMatchFormat() MatchFormat {
	return MatchFormat{GameVariant: league.GameVariant501, LegsToWin: league.DefaultLegsToWin}
}

func (f MatchFormat) validate() error {
	if err := league.ValidateGameVariant(f.GameVariant); err != nil {
		return err
	}
	return league.ValidateLegsToWin(int(f.LegsToWin))
}

type Entry struct {
	ReturningIdentityID int64     `json:"-"`
	ID                  int64     `json:"id"`
	DisplayName         string    `json:"display_name"`
	Nickname            string    `json:"nickname,omitempty"`
	RegisteredAt        time.Time `json:"registered_at"`
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
	UpdateTournamentFormat(context.Context, string, MatchFormat) (Event, error)
	UpdateTournamentEndDate(context.Context, string, string) (Event, error)
	StartTournament(context.Context, string) (Event, error)
	CreateTournamentEntry(context.Context, string, Entry) (Entry, error)
	ListTournamentEntries(context.Context, string) ([]Entry, error)
	DeleteTournamentEntry(context.Context, string, int64) error
}

type Service struct {
	Store Store
	Now   func() time.Time
}

func (s Service) Create(ctx context.Context, name, startsOn, endsOn string, format MatchFormat) (Event, error) {
	normalizedName := league.NormalizeWhitespace(name)
	if strings.ContainsFunc(name, unicode.IsControl) || utf8.RuneCountInString(normalizedName) < 2 || utf8.RuneCountInString(normalizedName) > 60 {
		return Event{}, ErrInvalidName
	}
	date, err := time.Parse(time.DateOnly, startsOn)
	if err != nil || date.Year() < 1 {
		return Event{}, ErrInvalidDate
	}
	if err := validateEndDate(startsOn, endsOn); err != nil {
		return Event{}, err
	}
	if err := format.validate(); err != nil {
		return Event{}, err
	}
	base := tournamentSlug(normalizedName)
	event := Event{Slug: base, Name: normalizedName, StartsOn: startsOn, EndsOn: endsOn, MatchFormat: format}
	for suffix := 2; ; suffix++ {
		created, err := s.Store.CreateTournament(ctx, event)
		if !errors.Is(err, ErrDuplicateSlug) {
			return created, err
		}
		event.Slug = base + "-" + strconv.Itoa(suffix)
	}
}

func tournamentSlug(name string) string {
	keepMarks := false
	slug := strings.Map(func(r rune) rune {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			keepMarks = unicode.IsLetter(r) && !unicode.Is(unicode.Latin, r)
			return unicode.ToLower(r)
		case unicode.IsMark(r):
			if keepMarks {
				return r
			}
			return -1
		default:
			keepMarks = false
			return ' '
		}
	}, norm.NFD.String(name))
	slug = norm.NFC.String(strings.Join(strings.Fields(slug), "-"))
	if slug == "" {
		return "tournament"
	}
	return slug
}

func validateEndDate(startsOn, endsOn string) error {
	if endsOn == "" {
		return nil
	}
	end, err := time.Parse(time.DateOnly, endsOn)
	if err != nil || end.Year() < 1 || endsOn < startsOn {
		return ErrInvalidEndDate
	}
	return nil
}

func (s Service) UpdateEndDate(ctx context.Context, slug, endsOn string) (Event, error) {
	event, err := s.Store.GetTournament(ctx, slug)
	if err != nil {
		return Event{}, err
	}
	if err := validateEndDate(event.StartsOn, endsOn); err != nil {
		return Event{}, err
	}
	return s.Store.UpdateTournamentEndDate(ctx, slug, endsOn)
}

func (s Service) UpdateFormat(ctx context.Context, slug string, format MatchFormat) (Event, error) {
	if err := format.validate(); err != nil {
		return Event{}, err
	}
	return s.Store.UpdateTournamentFormat(ctx, slug, format)
}

func (s Service) Register(ctx context.Context, slug, displayName, nickname string) (Entry, error) {
	return s.RegisterEntry(ctx, slug, Entry{DisplayName: displayName, Nickname: nickname})
}

func (s Service) RegisterEntry(ctx context.Context, slug string, entry Entry) (Entry, error) {
	if err := league.ValidatePlayerName(entry.DisplayName, entry.Nickname); err != nil {
		return Entry{}, err
	}
	entry.ID = 0
	entry.DisplayName = league.NormalizeWhitespace(entry.DisplayName)
	entry.Nickname = league.NormalizeWhitespace(entry.Nickname)
	entry.RegisteredAt = s.Now().UTC()
	return s.Store.CreateTournamentEntry(ctx, slug, entry)
}
