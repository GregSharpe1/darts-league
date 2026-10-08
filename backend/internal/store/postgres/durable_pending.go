package postgres

import (
	"context"
	"errors"
	"strings"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/jackc/pgx/v5"
)

// CreatePendingResultDurably commits before reporting success. The existing
// unique external_match_id arbitrates concurrent consumers, including replays
// of results that admins have already confirmed or rejected.
func (s *Store) CreatePendingResultDurably(ctx context.Context, pending league.PendingResult) (league.PendingResult, error) {
	if strings.TrimSpace(pending.ExternalMatchID) == "" {
		return league.PendingResult{}, league.ErrInvalidResult
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.ReadCommitted})
	if err != nil {
		return league.PendingResult{}, err
	}
	defer tx.Rollback(ctx)
	row := tx.QueryRow(ctx, `
		INSERT INTO pending_results (external_match_id, player_one_name, player_one_legs, player_one_average, player_two_name, player_two_legs, player_two_average, status, received_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (external_match_id) DO NOTHING
		RETURNING id, external_match_id, player_one_name, player_one_legs, player_one_average, player_two_name, player_two_legs, player_two_average, status, received_at, confirmed_at, confirmed_by
	`, pending.ExternalMatchID, pending.PlayerOneName, pending.PlayerOneLegs, nullableFloat(pending.PlayerOneAverage), pending.PlayerTwoName, pending.PlayerTwoLegs, nullableFloat(pending.PlayerTwoAverage), pending.Status, pending.ReceivedAt)
	created, err := scanPendingResult(row)
	duplicate := errors.Is(err, pgx.ErrNoRows)
	if duplicate {
		// A new READ COMMITTED statement sees the row committed by a competing insert.
		var id int64
		err = tx.QueryRow(ctx, `SELECT id FROM pending_results WHERE external_match_id = $1`, pending.ExternalMatchID).Scan(&id)
	}
	if err != nil {
		return league.PendingResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return league.PendingResult{}, err
	}
	if duplicate {
		return league.PendingResult{}, league.ErrDuplicateExternalMatch
	}
	return created, nil
}
