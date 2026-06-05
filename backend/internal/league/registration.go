package league

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrRegistrationClosed    = errors.New("registration is closed")
	ErrDisplayNameRequired   = errors.New("display name is required")
	ErrDuplicatePlayerName   = errors.New("display name already exists")
	ErrPlayerDeleteLocked    = errors.New("players can only be deleted before the season starts")
	ErrPlayerAssignLocked    = errors.New("players can only be assigned before the season starts")
	ErrSeasonNameRequired    = errors.New("season name is required")
	ErrSeasonNameLength      = errors.New("season name must be between 2 and 60 characters")
	ErrSeasonRenameLocked    = errors.New("season name can only be changed before the season starts")
	ErrSeasonConfigLocked    = errors.New("match configuration can only be changed before the season starts")
	ErrDivisionNotFound      = errors.New("division not found")
	ErrDivisionSlugLocked    = errors.New("division names and slugs can only be changed before the season starts")
	ErrInvalidDivisionCount  = errors.New("division count must be at least 1")
	ErrDivisionNameRequired  = errors.New("division name is required")
	ErrDivisionSlugRequired  = errors.New("division slug is required")
	ErrDuplicateDivisionSlug = errors.New("division slug already exists in this season")
	ErrInvalidGameVariant    = errors.New("game variant must be 301 or 501")
	ErrInvalidLegsToWin      = errors.New("legs to win must be at least 1")
	ErrInvalidGamesPerWeek   = errors.New("games per week must be at least 1")
	ErrGamesPerWeekTooHigh   = errors.New("games per week exceeds available fixtures per player")
)

type PlayerStatus string

const (
	PlayerStatusWaitlist PlayerStatus = "waitlist"
	PlayerStatusAssigned PlayerStatus = "assigned"
)

type SeasonStatus string

const (
	SeasonStatusRegistrationOpen SeasonStatus = "registration_open"
	SeasonStatusStarted          SeasonStatus = "started"
)

type Season struct {
	ID           int64
	Name         string
	Status       SeasonStatus
	Timezone     string
	StartedAt    *time.Time
	GameVariant  string
	LegsToWin    int
	GamesPerWeek int
}

const (
	GameVariant301 = "301"
	GameVariant501 = "501"
)

func NewSeason(name string) Season {
	return Season{
		Name:         name,
		Status:       SeasonStatusRegistrationOpen,
		Timezone:     "Europe/London",
		GameVariant:  GameVariant501,
		LegsToWin:    3,
		GamesPerWeek: 1,
	}
}

func (s Season) RegistrationOpen() bool {
	return s.Status == SeasonStatusRegistrationOpen
}

func (s Season) CanDeletePlayers() bool {
	return s.RegistrationOpen()
}

func (s Season) Start(startedAt time.Time) Season {
	s.Status = SeasonStatusStarted
	s.StartedAt = &startedAt
	return s
}

type Player struct {
	ID           int64
	SeasonID     int64
	DivisionID   *int64
	DisplayName  string
	Nickname     string
	Status       PlayerStatus
	RegisteredAt time.Time
}

type Division struct {
	ID                   int64
	SeasonID             int64
	Name                 string
	Slug                 string
	Position             int
	SlackPublicChannelID string
}

func (p Player) PreferredName() string {
	if strings.TrimSpace(p.Nickname) != "" {
		return strings.TrimSpace(p.Nickname)
	}

	return strings.TrimSpace(p.DisplayName)
}

func (p Player) AdminLabel() string {
	return p.FixtureLabel()
}

func (p Player) FixtureLabel() string {
	preferred := p.PreferredName()
	displayName := strings.TrimSpace(p.DisplayName)
	if strings.TrimSpace(p.Nickname) == "" || preferred == displayName {
		return displayName
	}

	return fmt.Sprintf("%s (%s)", preferred, displayName)
}

type RegistrationBook struct {
	existing []Player
}

func NewRegistrationBook(existing []Player) RegistrationBook {
	copyOfPlayers := make([]Player, len(existing))
	copy(copyOfPlayers, existing)

	return RegistrationBook{existing: copyOfPlayers}
}

func (b RegistrationBook) ValidateNewPlayer(season Season, player Player) error {
	if !season.RegistrationOpen() {
		return ErrRegistrationClosed
	}

	if NormalizeDisplayName(player.DisplayName) == "" {
		return ErrDisplayNameRequired
	}

	newName := NormalizeDisplayName(player.DisplayName)
	for _, existingPlayer := range b.existing {
		if NormalizeDisplayName(existingPlayer.DisplayName) == newName {
			return ErrDuplicatePlayerName
		}
	}

	return nil
}

func ValidateDivisionCount(count int) error {
	if count < 1 {
		return ErrInvalidDivisionCount
	}
	return nil
}

func ValidateDivisionName(name string) error {
	if strings.TrimSpace(name) == "" {
		return ErrDivisionNameRequired
	}
	return nil
}

func NormalizeDivisionName(name string) string {
	return strings.Join(strings.Fields(name), " ")
}

func NormalizeDivisionSlug(slug string) string {
	slug = strings.ToLower(strings.TrimSpace(slug))
	fields := strings.FieldsFunc(slug, func(r rune) bool {
		return !(r >= 'a' && r <= 'z' || r >= '0' && r <= '9')
	})
	return strings.Join(fields, "-")
}

func ValidateDivisionSlug(slug string) error {
	if NormalizeDivisionSlug(slug) == "" {
		return ErrDivisionSlugRequired
	}
	return nil
}

func (b RegistrationBook) ValidatePlayerDelete(season Season) error {
	if !season.CanDeletePlayers() {
		return ErrPlayerDeleteLocked
	}

	return nil
}

func NormalizeDisplayName(name string) string {
	return strings.ToLower(strings.Join(strings.Fields(name), " "))
}

func NormalizeSeasonName(name string) string {
	return strings.Join(strings.Fields(name), " ")
}

func ValidateGameVariant(variant string) error {
	if variant != GameVariant301 && variant != GameVariant501 {
		return ErrInvalidGameVariant
	}
	return nil
}

func ValidateLegsToWin(legs int) error {
	if legs < 1 {
		return ErrInvalidLegsToWin
	}
	return nil
}

func ValidateGamesPerWeek(gamesPerWeek, playerCount int) error {
	if gamesPerWeek < 1 {
		return ErrInvalidGamesPerWeek
	}
	if playerCount >= 2 && gamesPerWeek > playerCount-1 {
		return ErrGamesPerWeekTooHigh
	}
	return nil
}

// GamesPerWeekPresets returns all valid games-per-week options for a
// given player count, each labelled with the resulting number of weeks.
type GamesPerWeekPreset struct {
	GamesPerWeek int
	WeekCount    int
}

func GamesPerWeekPresets(playerCount int) []GamesPerWeekPreset {
	if playerCount < 2 {
		return nil
	}
	totalGames := playerCount - 1 // games per player in a round-robin
	presets := make([]GamesPerWeekPreset, 0, totalGames)
	for gpw := 1; gpw <= totalGames; gpw++ {
		weeks := (totalGames + gpw - 1) / gpw // ceiling division
		presets = append(presets, GamesPerWeekPreset{GamesPerWeek: gpw, WeekCount: weeks})
	}
	return presets
}

// SchedulePreview computes a summary of the schedule that would be
// generated for a given player count and season config.
type SchedulePreview struct {
	PlayerCount   int
	GameVariant   string
	LegsToWin     int
	GamesPerWeek  int
	WeekCount     int
	TotalFixtures int
}

func ComputeSchedulePreview(playerCount int, gameVariant string, legsToWin, gamesPerWeek int) SchedulePreview {
	totalFixtures := playerCount * (playerCount - 1) / 2
	gamesPerPlayer := playerCount - 1
	weekCount := 0
	if gamesPerWeek > 0 && playerCount >= 2 {
		weekCount = (gamesPerPlayer + gamesPerWeek - 1) / gamesPerWeek
	}
	return SchedulePreview{
		PlayerCount:   playerCount,
		GameVariant:   gameVariant,
		LegsToWin:     legsToWin,
		GamesPerWeek:  gamesPerWeek,
		WeekCount:     weekCount,
		TotalFixtures: totalFixtures,
	}
}

func ValidateSeasonName(name string) error {
	normalized := NormalizeSeasonName(name)
	if normalized == "" {
		return ErrSeasonNameRequired
	}

	length := utf8.RuneCountInString(normalized)
	if length < 2 || length > 60 {
		return ErrSeasonNameLength
	}

	return nil
}
