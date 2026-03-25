package postgres

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/greg/darts-league/backend/internal/league"
)

//go:embed schema.sql
var initialSchema string

type Store struct {
	pool *pgxpool.Pool
}

func Open(ctx context.Context, databaseURL string) (*Store, error) {
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return nil, err
	}
	store := &Store{pool: pool}
	if err := store.pingAndMigrate(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return store, nil
}

func (s *Store) Close() {
	if s.pool != nil {
		s.pool.Close()
	}
}

func (s *Store) pingAndMigrate(ctx context.Context) error {
	if err := s.pool.Ping(ctx); err != nil {
		return err
	}
	_, err := s.pool.Exec(ctx, initialSchema)
	return err
}

func (s *Store) EnsureActiveSeason(ctx context.Context, season league.Season) (league.Season, error) {
	existing, err := s.GetActiveSeason(ctx)
	if err == nil {
		return existing, nil
	}
	if !errors.Is(err, league.ErrSeasonNotFound) {
		return league.Season{}, err
	}
	return s.UpsertSeason(ctx, season)
}

func (s *Store) GetActiveSeason(ctx context.Context) (league.Season, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, name, status, timezone, started_at, game_variant, legs_to_win, games_per_week
		FROM seasons
		ORDER BY id DESC
		LIMIT 1
	`)
	var season league.Season
	if err := row.Scan(&season.ID, &season.Name, &season.Status, &season.Timezone, &season.StartedAt, &season.GameVariant, &season.LegsToWin, &season.GamesPerWeek); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return league.Season{}, league.ErrSeasonNotFound
		}
		return league.Season{}, err
	}
	return season, nil
}

func (s *Store) ListPlayersBySeason(ctx context.Context, seasonID int64) ([]league.Player, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, season_id, display_name, nickname, registered_at
		FROM players
		WHERE season_id = $1
		ORDER BY registered_at ASC, id ASC
	`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	players := []league.Player{}
	for rows.Next() {
		var player league.Player
		var nickname *string
		if err := rows.Scan(&player.ID, &player.SeasonID, &player.DisplayName, &nickname, &player.RegisteredAt); err != nil {
			return nil, err
		}
		player.Nickname = valueOrBlank(nickname)
		players = append(players, player)
	}
	return players, rows.Err()
}

func (s *Store) ListFixturesBySeason(ctx context.Context, seasonID int64) ([]league.Fixture, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, season_id, week_number, scheduled_at, player_one_id, player_two_id, game_variant, legs_to_win, status
		FROM fixtures
		WHERE season_id = $1
		ORDER BY week_number ASC, scheduled_at ASC, id ASC
	`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	fixtures := []league.Fixture{}
	for rows.Next() {
		var fixture league.Fixture
		if err := rows.Scan(&fixture.ID, &fixture.SeasonID, &fixture.WeekNumber, &fixture.ScheduledAt, &fixture.PlayerOneID, &fixture.PlayerTwoID, &fixture.GameVariant, &fixture.LegsToWin, &fixture.Status); err != nil {
			return nil, err
		}
		fixtures = append(fixtures, fixture)
	}
	return fixtures, rows.Err()
}

func (s *Store) GetFixture(ctx context.Context, fixtureID int64) (league.Fixture, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, season_id, week_number, scheduled_at, player_one_id, player_two_id, game_variant, legs_to_win, status
		FROM fixtures
		WHERE id = $1
	`, fixtureID)
	var fixture league.Fixture
	if err := row.Scan(&fixture.ID, &fixture.SeasonID, &fixture.WeekNumber, &fixture.ScheduledAt, &fixture.PlayerOneID, &fixture.PlayerTwoID, &fixture.GameVariant, &fixture.LegsToWin, &fixture.Status); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return league.Fixture{}, league.ErrFixtureNotFound
		}
		return league.Fixture{}, err
	}
	return fixture, nil
}

func (s *Store) ListResultsBySeason(ctx context.Context, seasonID int64) ([]league.Result, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT r.id, r.fixture_id, r.player_one_legs, r.player_two_legs, r.player_one_average, r.player_two_average, r.winner_id, r.entered_at, r.updated_at
		FROM results r
		JOIN fixtures f ON f.id = r.fixture_id
		WHERE f.season_id = $1
		ORDER BY r.entered_at ASC, r.id ASC
	`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	results := []league.Result{}
	for rows.Next() {
		var result league.Result
		var playerOneAverage *float64
		var playerTwoAverage *float64
		if err := rows.Scan(&result.ID, &result.FixtureID, &result.PlayerOneLegs, &result.PlayerTwoLegs, &playerOneAverage, &playerTwoAverage, &result.WinnerID, &result.EnteredAt, &result.UpdatedAt); err != nil {
			return nil, err
		}
		result.PlayerOneAverage = playerOneAverage
		result.PlayerTwoAverage = playerTwoAverage
		results = append(results, result)
	}
	return results, rows.Err()
}

func (s *Store) GetResultByFixture(ctx context.Context, fixtureID int64) (league.Result, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT id, fixture_id, player_one_legs, player_two_legs, player_one_average, player_two_average, winner_id, entered_at, updated_at
		FROM results
		WHERE fixture_id = $1
	`, fixtureID)
	var result league.Result
	var playerOneAverage *float64
	var playerTwoAverage *float64
	if err := row.Scan(&result.ID, &result.FixtureID, &result.PlayerOneLegs, &result.PlayerTwoLegs, &playerOneAverage, &playerTwoAverage, &result.WinnerID, &result.EnteredAt, &result.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return league.Result{}, league.ErrResultNotFound
		}
		return league.Result{}, err
	}
	result.PlayerOneAverage = playerOneAverage
	result.PlayerTwoAverage = playerTwoAverage
	return result, nil
}

func (s *Store) ListAuditLogsBySeason(ctx context.Context, seasonID int64) ([]league.AuditLogEntry, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT a.id, a.fixture_id, a.action, a.actor, a.old_payload, a.new_payload, a.created_at
		FROM admin_audit_log a
		JOIN fixtures f ON f.id = a.fixture_id
		WHERE f.season_id = $1
		ORDER BY a.created_at DESC, a.id DESC
	`, seasonID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	entries := []league.AuditLogEntry{}
	for rows.Next() {
		var entry league.AuditLogEntry
		var oldPayload []byte
		var newPayload []byte
		if err := rows.Scan(&entry.ID, &entry.FixtureID, &entry.Action, &entry.Actor, &oldPayload, &newPayload, &entry.CreatedAt); err != nil {
			return nil, err
		}
		if len(oldPayload) > 0 {
			entry.OldResult = &league.ResultSnapshot{}
			if err := json.Unmarshal(oldPayload, entry.OldResult); err != nil {
				return nil, err
			}
		}
		if len(newPayload) > 0 {
			entry.NewResult = &league.ResultSnapshot{}
			if err := json.Unmarshal(newPayload, entry.NewResult); err != nil {
				return nil, err
			}
		}
		entries = append(entries, entry)
	}
	return entries, rows.Err()
}

func (s *Store) CreatePlayer(ctx context.Context, player league.Player) (league.Player, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO players (season_id, display_name, display_name_normalized, nickname, registered_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id, season_id, display_name, nickname, registered_at
	`, player.SeasonID, player.DisplayName, league.NormalizeDisplayName(player.DisplayName), nullIfBlank(player.Nickname), player.RegisteredAt)
	var created league.Player
	var nickname *string
	if err := row.Scan(&created.ID, &created.SeasonID, &created.DisplayName, &nickname, &created.RegisteredAt); err != nil {
		if isUniqueViolation(err) {
			return league.Player{}, league.ErrDuplicatePlayerName
		}
		return league.Player{}, err
	}
	created.Nickname = valueOrBlank(nickname)
	return created, nil
}

func (s *Store) CreateFixtures(ctx context.Context, fixtures []league.Fixture) ([]league.Fixture, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	created := make([]league.Fixture, 0, len(fixtures))
	for _, fixture := range fixtures {
		row := tx.QueryRow(ctx, `
			INSERT INTO fixtures (season_id, week_number, scheduled_at, player_one_id, player_two_id, game_variant, legs_to_win, status)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			RETURNING id, season_id, week_number, scheduled_at, player_one_id, player_two_id, game_variant, legs_to_win, status
		`, fixture.SeasonID, fixture.WeekNumber, fixture.ScheduledAt, fixture.PlayerOneID, fixture.PlayerTwoID, fixture.GameVariant, fixture.LegsToWin, fixture.Status)
		var createdFixture league.Fixture
		if err := row.Scan(&createdFixture.ID, &createdFixture.SeasonID, &createdFixture.WeekNumber, &createdFixture.ScheduledAt, &createdFixture.PlayerOneID, &createdFixture.PlayerTwoID, &createdFixture.GameVariant, &createdFixture.LegsToWin, &createdFixture.Status); err != nil {
			return nil, err
		}
		created = append(created, createdFixture)
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return created, nil
}

func (s *Store) CreateResult(ctx context.Context, result league.Result) (league.Result, error) {
	row := s.pool.QueryRow(ctx, `
		INSERT INTO results (fixture_id, player_one_legs, player_two_legs, player_one_average, player_two_average, winner_id, entered_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, fixture_id, player_one_legs, player_two_legs, player_one_average, player_two_average, winner_id, entered_at, updated_at
	`, result.FixtureID, result.PlayerOneLegs, result.PlayerTwoLegs, nullableFloat(result.PlayerOneAverage), nullableFloat(result.PlayerTwoAverage), result.WinnerID, result.EnteredAt, result.UpdatedAt)
	var created league.Result
	var playerOneAverage *float64
	var playerTwoAverage *float64
	if err := row.Scan(&created.ID, &created.FixtureID, &created.PlayerOneLegs, &created.PlayerTwoLegs, &playerOneAverage, &playerTwoAverage, &created.WinnerID, &created.EnteredAt, &created.UpdatedAt); err != nil {
		if isUniqueViolation(err) {
			return league.Result{}, league.ErrResultAlreadyExists
		}
		return league.Result{}, err
	}
	created.PlayerOneAverage = playerOneAverage
	created.PlayerTwoAverage = playerTwoAverage
	return created, nil
}

func (s *Store) UpdateResult(ctx context.Context, result league.Result) (league.Result, error) {
	row := s.pool.QueryRow(ctx, `
		UPDATE results
		SET player_one_legs = $2, player_two_legs = $3, player_one_average = $4, player_two_average = $5, winner_id = $6, updated_at = $7
		WHERE id = $1
		RETURNING id, fixture_id, player_one_legs, player_two_legs, player_one_average, player_two_average, winner_id, entered_at, updated_at
	`, result.ID, result.PlayerOneLegs, result.PlayerTwoLegs, nullableFloat(result.PlayerOneAverage), nullableFloat(result.PlayerTwoAverage), result.WinnerID, result.UpdatedAt)
	var updated league.Result
	var playerOneAverage *float64
	var playerTwoAverage *float64
	if err := row.Scan(&updated.ID, &updated.FixtureID, &updated.PlayerOneLegs, &updated.PlayerTwoLegs, &playerOneAverage, &playerTwoAverage, &updated.WinnerID, &updated.EnteredAt, &updated.UpdatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return league.Result{}, league.ErrResultNotFound
		}
		return league.Result{}, err
	}
	updated.PlayerOneAverage = playerOneAverage
	updated.PlayerTwoAverage = playerTwoAverage
	return updated, nil
}

func (s *Store) DeleteResultByFixture(ctx context.Context, fixtureID int64) error {
	commandTag, err := s.pool.Exec(ctx, `DELETE FROM results WHERE fixture_id = $1`, fixtureID)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return league.ErrResultNotFound
	}
	return nil
}

func (s *Store) CreateAuditLog(ctx context.Context, entry league.AuditLogEntry) (league.AuditLogEntry, error) {
	var oldPayload []byte
	var newPayload []byte
	var err error
	if entry.OldResult != nil {
		oldPayload, err = json.Marshal(entry.OldResult)
		if err != nil {
			return league.AuditLogEntry{}, err
		}
	}
	if entry.NewResult != nil {
		newPayload, err = json.Marshal(entry.NewResult)
		if err != nil {
			return league.AuditLogEntry{}, err
		}
	}
	row := s.pool.QueryRow(ctx, `
		INSERT INTO admin_audit_log (fixture_id, action, actor, old_payload, new_payload, created_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, fixture_id, action, actor, created_at
	`, entry.FixtureID, entry.Action, entry.Actor, nullableJSON(oldPayload), nullableJSON(newPayload), entry.CreatedAt)
	var created league.AuditLogEntry
	if err := row.Scan(&created.ID, &created.FixtureID, &created.Action, &created.Actor, &created.CreatedAt); err != nil {
		return league.AuditLogEntry{}, err
	}
	created.OldResult = entry.OldResult
	created.NewResult = entry.NewResult
	return created, nil
}

func (s *Store) DeletePlayer(ctx context.Context, seasonID, playerID int64) error {
	commandTag, err := s.pool.Exec(ctx, `DELETE FROM players WHERE season_id = $1 AND id = $2`, seasonID, playerID)
	if err != nil {
		return err
	}
	if commandTag.RowsAffected() == 0 {
		return league.ErrPlayerNotFound
	}
	return nil
}

func (s *Store) UpsertSeason(ctx context.Context, season league.Season) (league.Season, error) {
	if season.ID == 0 {
		row := s.pool.QueryRow(ctx, `
			INSERT INTO seasons (name, status, timezone, started_at, game_variant, legs_to_win, games_per_week)
			VALUES ($1, $2, $3, $4, $5, $6, $7)
			RETURNING id, name, status, timezone, started_at, game_variant, legs_to_win, games_per_week
		`, season.Name, season.Status, season.Timezone, season.StartedAt, season.GameVariant, season.LegsToWin, season.GamesPerWeek)
		if err := row.Scan(&season.ID, &season.Name, &season.Status, &season.Timezone, &season.StartedAt, &season.GameVariant, &season.LegsToWin, &season.GamesPerWeek); err != nil {
			return league.Season{}, err
		}
		return season, nil
	}

	row := s.pool.QueryRow(ctx, `
		UPDATE seasons
		SET name = $2, status = $3, timezone = $4, started_at = $5, game_variant = $6, legs_to_win = $7, games_per_week = $8
		WHERE id = $1
		RETURNING id, name, status, timezone, started_at, game_variant, legs_to_win, games_per_week
	`, season.ID, season.Name, season.Status, season.Timezone, season.StartedAt, season.GameVariant, season.LegsToWin, season.GamesPerWeek)
	if err := row.Scan(&season.ID, &season.Name, &season.Status, &season.Timezone, &season.StartedAt, &season.GameVariant, &season.LegsToWin, &season.GamesPerWeek); err != nil {
		return league.Season{}, err
	}
	return season, nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func nullIfBlank(value string) any {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	return value
}

func valueOrBlank(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func nullableJSON(value []byte) any {
	if len(value) == 0 {
		return nil
	}
	return value
}

func nullableFloat(value *float64) any {
	if value == nil {
		return nil
	}
	return *value
}

func (s *Store) String() string {
	return fmt.Sprintf("postgres store")
}
