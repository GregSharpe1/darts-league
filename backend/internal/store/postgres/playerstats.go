package postgres

import (
	"context"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ListSeasons(ctx context.Context) ([]league.Season, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, name, status, timezone, started_at, game_variant, legs_to_win, games_per_week
		FROM seasons
		ORDER BY id DESC
	`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (league.Season, error) {
		var season league.Season
		err := row.Scan(&season.ID, &season.Name, &season.Status, &season.Timezone, &season.StartedAt, &season.GameVariant, &season.LegsToWin, &season.GamesPerWeek)
		return season, err
	})
}
