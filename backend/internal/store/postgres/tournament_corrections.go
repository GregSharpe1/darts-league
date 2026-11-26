package postgres

import (
	"context"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/tournament"
)

func (s *Store) CorrectTournamentScore(ctx context.Context, slug string, number int, correction tournament.ScoreCorrection) (tournament.Event, error) {
	var event tournament.Event
	err := s.Transaction(ctx, func(scoped league.Store) error {
		store := scoped.(*Store)
		if err := store.LockTournamentImports(ctx); err != nil {
			return err
		}
		current, err := store.GetTournament(ctx, slug)
		if err != nil {
			return err
		}
		bracket, changes, err := current.ApplyScoreCorrection(number, correction)
		if err != nil {
			return err
		}
		event = current
		if len(changes) == 0 {
			return nil
		}

		for _, change := range changes {
			// Keep the original receipt and its confirmed status so it cannot score again.
			if _, err := store.pool.Exec(ctx, `WITH former AS (
				DELETE FROM tournament_import_results WHERE tournament_slug = $1 AND match_number = $2
				RETURNING pending_result_id
			)
			UPDATE pending_results SET source_active = FALSE WHERE id IN (SELECT pending_result_id FROM former)`,
				slug, change.Before.Number); err != nil {
				return err
			}
			if _, err := store.pool.Exec(ctx, `INSERT INTO tournament_score_audit
				(tournament_slug, match_number, actor, reason, previous_match, updated_match)
				VALUES ($1, $2, $3, $4, $5, $6)`,
				slug, change.Before.Number, correction.Actor, correction.Reason, change.Before, change.After); err != nil {
				return err
			}
		}
		event, err = scanTournament(store.pool.QueryRow(ctx, `
			UPDATE tournaments SET bracket = $2 WHERE slug = $1 RETURNING `+tournamentColumns, slug, bracket))
		return err
	})
	return event, err
}
