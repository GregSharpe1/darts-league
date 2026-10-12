package postgres

import (
	"context"
	"errors"
	"github.com/greg/darts-league/backend/internal/league"
	"github.com/jackc/pgx/v5/pgconn"
)

func (s *Store) ListAutodartsMappings(ctx context.Context, seasonID int64) ([]league.AutodartsMapping, error) {
	rows, err := s.pool.Query(ctx, `SELECT id, autodarts_username FROM players WHERE season_id=$1 AND autodarts_username IS NOT NULL ORDER BY id`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	mappings := []league.AutodartsMapping{}
	for rows.Next() {
		var row league.AutodartsMapping
		if err := rows.Scan(&row.PlayerID, &row.Username); err != nil {
			return nil, err
		}
		mappings = append(mappings, row)
	}
	return mappings, rows.Err()
}

func (s *Store) SetAutodartsUsername(ctx context.Context, seasonID, playerID int64, username *string) error {
	tag, err := s.pool.Exec(ctx, `UPDATE players SET autodarts_username=$3 WHERE season_id=$1 AND id=$2`, seasonID, playerID, username)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return league.ErrAutodartsUsernameConflict
	}
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return league.ErrPlayerNotFound
	}
	return nil
}
