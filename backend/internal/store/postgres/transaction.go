package postgres

import (
	"context"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type database interface {
	Begin(context.Context) (pgx.Tx, error)
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
	Query(context.Context, string, ...any) (pgx.Rows, error)
	QueryRow(context.Context, string, ...any) pgx.Row
}

func (s *Store) Transaction(ctx context.Context, fn func(league.Store) error) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// ponytail: league-wide writes are serialized; narrow to season/source locks if throughput matters.
	// EXCLUSIVE also conflicts with legacy FOR UPDATE row-share locks, avoiding lock-upgrade deadlocks.
	if _, err := tx.Exec(ctx, `LOCK TABLE seasons IN EXCLUSIVE MODE; LOCK TABLE pending_results IN SHARE ROW EXCLUSIVE MODE`); err != nil {
		return err
	}
	if err := fn(&Store{pool: tx}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
