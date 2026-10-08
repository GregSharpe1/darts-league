package postgres

import (
	"context"
	_ "embed"
)

//go:embed import_expansion.sql
var importExpansionSchema string

func (s *Store) migrateImportExpansion(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(390039)`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, initialSchema); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, importExpansionSchema); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
