package postgres

import (
	"context"
	"errors"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/jackc/pgx/v5"
)

func lockSeason(ctx context.Context, tx pgx.Tx, seasonID int64) (league.SeasonStatus, error) {
	var status league.SeasonStatus
	err := tx.QueryRow(ctx, `SELECT status FROM seasons WHERE id = $1 FOR UPDATE`, seasonID).Scan(&status)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", league.ErrSeasonTransition
	}
	return status, err
}

func (s *Store) CloseSeason(ctx context.Context, seasonID int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	status, err := lockSeason(ctx, tx, seasonID)
	if err != nil {
		return err
	}
	var currentID int64
	if err := tx.QueryRow(ctx, `SELECT MAX(id) FROM seasons`).Scan(&currentID); err != nil {
		return err
	}
	if currentID != seasonID || status != league.SeasonStatusStarted {
		return league.ErrSeasonTransition
	}
	var total, remaining int
	if err := tx.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE r.id IS NULL)
		FROM fixtures f LEFT JOIN results r ON r.fixture_id = f.id WHERE f.season_id = $1`, seasonID).Scan(&total, &remaining); err != nil {
		return err
	}
	if total == 0 || remaining != 0 {
		return league.ErrSeasonIncomplete
	}
	if _, err := tx.Exec(ctx, `UPDATE seasons SET status = 'completed' WHERE id = $1`, seasonID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) CreateNextSeason(ctx context.Context, seasonID int64, season league.Season) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	status, err := lockSeason(ctx, tx, seasonID)
	if err != nil {
		return err
	}
	var currentID int64
	if err := tx.QueryRow(ctx, `SELECT MAX(id) FROM seasons`).Scan(&currentID); err != nil {
		return err
	}
	if currentID != seasonID || status != league.SeasonStatusCompleted {
		return league.ErrSeasonTransition
	}
	if _, err := tx.Exec(ctx, `INSERT INTO seasons (name, status, timezone, game_variant, legs_to_win, games_per_week)
		VALUES ($1, $2, $3, $4, $5, $6)`, season.Name, season.Status, season.Timezone, season.GameVariant, season.LegsToWin, season.GamesPerWeek); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) beginSeasonWrite(ctx context.Context, seasonID int64) (pgx.Tx, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	status, err := lockSeason(ctx, tx, seasonID)
	if err == nil && status == league.SeasonStatusCompleted {
		err = league.ErrSeasonCompleted
	}
	if err != nil {
		if rollbackErr := tx.Rollback(ctx); rollbackErr != nil {
			return nil, errors.Join(err, rollbackErr)
		}
		return nil, err
	}
	return tx, nil
}

func (s *Store) beginResultWrite(ctx context.Context, fixtureID int64) (pgx.Tx, error) {
	var seasonID int64
	err := s.pool.QueryRow(ctx, `SELECT season_id FROM fixtures WHERE id = $1`, fixtureID).Scan(&seasonID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, league.ErrFixtureNotFound
	}
	if err != nil {
		return nil, err
	}
	return s.beginSeasonWrite(ctx, seasonID)
}
