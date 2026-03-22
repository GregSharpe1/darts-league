package league

import (
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"
)

var (
	ErrRegistrationClosed  = errors.New("registration is closed")
	ErrDisplayNameRequired = errors.New("display name is required")
	ErrDuplicatePlayerName = errors.New("display name already exists in this season")
	ErrPlayerDeleteLocked  = errors.New("players can only be deleted before the season starts")
	ErrSeasonNameRequired  = errors.New("season name is required")
	ErrSeasonNameLength    = errors.New("season name must be between 2 and 60 characters")
	ErrSeasonRenameLocked  = errors.New("season name can only be changed before the season starts")
)

type SeasonStatus string

const (
	SeasonStatusRegistrationOpen SeasonStatus = "registration_open"
	SeasonStatusStarted          SeasonStatus = "started"
)

type Season struct {
	ID        int64
	Name      string
	Status    SeasonStatus
	Timezone  string
	StartedAt *time.Time
}

func NewSeason(name string) Season {
	return Season{
		Name:     name,
		Status:   SeasonStatusRegistrationOpen,
		Timezone: "Europe/London",
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
	DisplayName  string
	Nickname     string
	RegisteredAt time.Time
}

func (p Player) PreferredName() string {
	if strings.TrimSpace(p.Nickname) != "" {
		return strings.TrimSpace(p.Nickname)
	}

	return strings.TrimSpace(p.DisplayName)
}

func (p Player) AdminLabel() string {
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
