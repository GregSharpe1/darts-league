package postgres

import (
	"context"
	_ "embed"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/playerstats"
	"github.com/jackc/pgx/v5"
)

//go:embed identity.sql
var identitySchema string

func (s *Store) IdentitySnapshot(ctx context.Context) (playerstats.IdentitySnapshot, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT p.identity_id, 'league', p.id, p.season_id::text,
			s.name, p.display_name, COALESCE(p.nickname, ''), COALESCE(p.requested_identity_id, 0)
		FROM players p JOIN seasons s ON s.id = p.season_id
		UNION ALL
		SELECT e.identity_id, 'tournament', e.id, e.tournament_slug,
			t.name, e.display_name, e.nickname, COALESCE(e.requested_identity_id, 0)
		FROM tournament_entries e JOIN tournaments t ON t.slug = e.tournament_slug
		ORDER BY 1, 2, 3
	`)
	if err != nil {
		return playerstats.IdentitySnapshot{}, err
	}
	entries, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (playerstats.IdentityEntry, error) {
		var e playerstats.IdentityEntry
		err := row.Scan(&e.IdentityID, &e.Kind, &e.EntryID, &e.Scope, &e.Competition, &e.DisplayName, &e.Nickname, &e.RequestedIdentityID)
		return e, err
	})
	if err != nil {
		return playerstats.IdentitySnapshot{}, err
	}
	rows, err = s.pool.Query(ctx, `SELECT left_id, right_id FROM player_identity_separations`)
	if err != nil {
		return playerstats.IdentitySnapshot{}, err
	}
	defer rows.Close()
	separated := make(map[playerstats.IdentityPair]bool)
	for rows.Next() {
		var key playerstats.IdentityPair
		if err := rows.Scan(&key.Left, &key.Right); err != nil {
			return playerstats.IdentitySnapshot{}, err
		}
		separated[key] = true
	}
	return playerstats.IdentitySnapshot{Entries: entries, Separated: separated}, rows.Err()
}

func (s *Store) DecideIdentity(ctx context.Context, decision playerstats.IdentityDecision) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	// Serialize review decisions, and lock entry rows against concurrent deletion.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(390040)`); err != nil {
		return err
	}
	left, right := min(decision.Left, decision.Right), max(decision.Left, decision.Right)
	for _, table := range []string{"players", "tournament_entries"} {
		rows, err := tx.Query(ctx, `SELECT id FROM `+table+` WHERE identity_id IN ($1, $2) ORDER BY id FOR UPDATE`, left, right)
		if err != nil {
			return err
		}
		for rows.Next() {
		}
		rows.Close()
		if err := rows.Err(); err != nil {
			return err
		}
	}
	snapshot, err := (&Store{pool: tx}).IdentitySnapshot(ctx)
	if err != nil {
		return err
	}
	if err := snapshot.ValidateDecision(decision); err != nil {
		return err
	}
	if decision.Action == "link" {
		if err := mergeIdentities(ctx, tx, left, right); err != nil {
			return err
		}
	} else if _, err := tx.Exec(ctx, `INSERT INTO player_identity_separations VALUES ($1, $2) ON CONFLICT DO NOTHING`, left, right); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO player_identity_decisions (left_id, right_id, action, actor) VALUES ($1, $2, $3, $4)`, left, right, decision.Action, decision.Actor); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func mergeIdentities(ctx context.Context, tx pgx.Tx, keep, retire int64) error {
	if _, err := tx.Exec(ctx, `UPDATE players SET identity_id = $1 WHERE identity_id = $2`, keep, retire); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE tournament_entries SET identity_id = $1 WHERE identity_id = $2`, keep, retire); err != nil {
		return err
	}
	for _, table := range []string{"players", "tournament_entries"} {
		if _, err := tx.Exec(ctx, `UPDATE `+table+` SET requested_identity_id = $1 WHERE requested_identity_id = $2`, keep, retire); err != nil {
			return err
		}
	}
	// Keep prior decisions to separate either profile from a third person.
	_, err := tx.Exec(ctx, `
		WITH moved AS (
			DELETE FROM player_identity_separations WHERE left_id = $2 OR right_id = $2
			RETURNING CASE WHEN left_id = $2 THEN right_id ELSE left_id END AS other
		)
		INSERT INTO player_identity_separations (left_id, right_id)
		SELECT LEAST($1, other), GREATEST($1, other) FROM moved WHERE other <> $1
		ON CONFLICT DO NOTHING`, keep, retire)
	return err
}

// Keep the selected identity stable while inserting the returning player's entry.
func verifyReturningIdentity(ctx context.Context, tx pgx.Tx, identityID, seasonID int64, tournamentSlug string) error {
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(390040)`); err != nil {
		return err
	}
	var exists, entered bool
	if err := tx.QueryRow(ctx, `SELECT
		EXISTS (SELECT 1 FROM players WHERE identity_id = $1) OR EXISTS (SELECT 1 FROM tournament_entries WHERE identity_id = $1),
		EXISTS (SELECT 1 FROM players WHERE season_id = $2 AND (identity_id = $1 OR requested_identity_id = $1))
		  OR EXISTS (SELECT 1 FROM tournament_entries WHERE tournament_slug = $3 AND (identity_id = $1 OR requested_identity_id = $1))
	`, identityID, seasonID, tournamentSlug).Scan(&exists, &entered); err != nil {
		return err
	}
	if !exists {
		return playerstats.ErrIdentityNotFound
	}
	if entered {
		return league.ErrDuplicatePlayerName
	}
	return nil
}
