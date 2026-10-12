package league

import (
	"context"
	"errors"
	"strings"
	"unicode/utf8"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

var ErrAutodartsUsernameInvalid = errors.New("invalid Autodarts username")
var ErrAutodartsUsernameConflict = errors.New("Autodarts username already mapped")

type AutodartsMapping struct {
	PlayerID int64  `json:"player_id"`
	Username string `json:"username"`
}

type AutodartsMappingView struct {
	SeasonID          int64              `json:"season_id"`
	AutoSubmitEnabled bool               `json:"auto_submit_enabled"`
	Mappings          []AutodartsMapping `json:"mappings"`
}

type AutodartsMappingStore interface {
	ListAutodartsMappings(context.Context, int64) ([]AutodartsMapping, error)
	SetAutodartsUsername(context.Context, int64, int64, *string) error
}

func (s PendingResultService) AutodartsMappings(ctx context.Context) (AutodartsMappingView, error) {
	var view AutodartsMappingView
	err := transact(ctx, s.store, func(tx Store) error {
		season, err := tx.GetActiveSeason(ctx)
		if err != nil {
			return err
		}
		store, ok := tx.(AutodartsMappingStore)
		if !ok {
			return ErrImportStoreRequired
		}
		view.SeasonID, view.AutoSubmitEnabled = season.ID, s.autoSubmit
		view.Mappings, err = store.ListAutodartsMappings(ctx, season.ID)
		return err
	})
	return view, err
}

func (s PendingResultService) SaveAutodartsMapping(ctx context.Context, seasonID, playerID int64, username string) (AutodartsMapping, error) {
	username = strings.TrimSpace(username)
	if !utf8.ValidString(username) || !autodarts.ValidLabel(username) {
		return AutodartsMapping{}, ErrAutodartsUsernameInvalid
	}
	err := s.writeAutodartsMapping(ctx, seasonID, playerID, &username)
	return AutodartsMapping{PlayerID: playerID, Username: username}, err
}

func (s PendingResultService) ClearAutodartsMapping(ctx context.Context, seasonID, playerID int64) error {
	return s.writeAutodartsMapping(ctx, seasonID, playerID, nil)
}

func (s PendingResultService) writeAutodartsMapping(ctx context.Context, seasonID, playerID int64, username *string) error {
	return transact(ctx, s.store, func(tx Store) error {
		season, err := tx.GetActiveSeason(ctx)
		if err != nil {
			return err
		}
		if season.ID != seasonID {
			return ErrSeasonTransition
		}
		store, ok := tx.(AutodartsMappingStore)
		if !ok {
			return ErrImportStoreRequired
		}
		return store.SetAutodartsUsername(ctx, seasonID, playerID, username)
	})
}
