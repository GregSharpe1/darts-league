package postgres

import (
	"context"
	"errors"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/tournament"
	"github.com/jackc/pgx/v5"
)

const tournamentColumns = `slug, name, to_char(starts_on, 'YYYY-MM-DD'), registration_open, bracket`
const tournamentEntryColumns = `id, display_name, nickname, registered_at`

func scanTournament(row pgx.Row) (tournament.Event, error) {
	var event tournament.Event
	err := row.Scan(&event.Slug, &event.Name, &event.StartsOn, &event.RegistrationOpen, &event.Bracket)
	if errors.Is(err, pgx.ErrNoRows) {
		err = tournament.ErrNotFound
	}
	return event, err
}

func (s *Store) CreateTournament(ctx context.Context, event tournament.Event) (tournament.Event, error) {
	created, err := scanTournament(s.pool.QueryRow(ctx, `
		INSERT INTO tournaments (slug, name, starts_on)
		VALUES ($1, $2, $3::text::date) ON CONFLICT (slug) DO NOTHING
		RETURNING `+tournamentColumns, event.Slug, event.Name, event.StartsOn))
	if errors.Is(err, tournament.ErrNotFound) {
		err = tournament.ErrDuplicateSlug
	}
	return created, err
}

func (s *Store) GetTournament(ctx context.Context, slug string) (tournament.Event, error) {
	return scanTournament(s.pool.QueryRow(ctx, `
		SELECT `+tournamentColumns+` FROM tournaments WHERE slug = $1`, slug))
}

func (s *Store) ListTournaments(ctx context.Context) ([]tournament.Event, error) {
	rows, err := s.pool.Query(ctx, `SELECT `+tournamentColumns+` FROM tournaments ORDER BY starts_on DESC, slug COLLATE "C"`)
	if err != nil {
		return nil, err
	}
	events, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (tournament.Event, error) { return scanTournament(row) })
	if events == nil && err == nil {
		events = []tournament.Event{}
	}
	return events, err
}

func (s *Store) DeleteTournament(ctx context.Context, slug string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	event, err := lockTournament(ctx, tx, slug)
	if err != nil {
		return err
	}
	if !event.CanDelete() {
		return tournament.ErrDeleteLocked
	}
	if _, err := tx.Exec(ctx, `DELETE FROM tournament_entries WHERE tournament_slug = $1`, slug); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM tournaments WHERE slug = $1`, slug); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) SetTournamentRegistration(ctx context.Context, slug string, open bool) (tournament.Event, error) {
	event, err := scanTournament(s.pool.QueryRow(ctx, `
		UPDATE tournaments SET registration_open = $2 WHERE slug = $1 AND bracket IS NULL
		RETURNING `+tournamentColumns, slug, open))
	if errors.Is(err, tournament.ErrNotFound) {
		if _, lookupErr := s.GetTournament(ctx, slug); lookupErr == nil {
			return tournament.Event{}, tournament.ErrTournamentStarted
		} else {
			return tournament.Event{}, lookupErr
		}
	}
	return event, err
}

func (s *Store) StartTournament(ctx context.Context, slug string) (tournament.Event, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return tournament.Event{}, err
	}
	defer tx.Rollback(ctx)
	event, err := lockTournament(ctx, tx, slug)
	if err != nil || len(event.Bracket) != 0 {
		return event, err
	}
	entries, err := listTournamentEntries(ctx, tx, slug)
	if err != nil {
		return tournament.Event{}, err
	}
	bracket, err := tournament.DrawBracket(event, entries)
	if err != nil {
		return tournament.Event{}, err
	}
	event, err = scanTournament(tx.QueryRow(ctx, `
		UPDATE tournaments SET bracket = $2 WHERE slug = $1
		RETURNING `+tournamentColumns, slug, bracket))
	if err != nil {
		return tournament.Event{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return tournament.Event{}, err
	}
	return event, nil
}

func lockTournament(ctx context.Context, tx pgx.Tx, slug string) (tournament.Event, error) {
	return scanTournament(tx.QueryRow(ctx, `
		SELECT `+tournamentColumns+` FROM tournaments WHERE slug = $1 FOR UPDATE`, slug))
}

func scanTournamentEntry(row pgx.Row) (tournament.Entry, error) {
	var entry tournament.Entry
	err := row.Scan(&entry.ID, &entry.DisplayName, &entry.Nickname, &entry.RegisteredAt)
	entry.RegisteredAt = entry.RegisteredAt.UTC()
	return entry, err
}

func (s *Store) CreateTournamentEntry(ctx context.Context, slug string, entry tournament.Entry) (tournament.Entry, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return tournament.Entry{}, err
	}
	defer tx.Rollback(ctx)

	// Keep the registration check and entry insert ordered with admin open/close updates.
	var open bool
	err = tx.QueryRow(ctx, `SELECT registration_open FROM tournaments WHERE slug = $1 FOR UPDATE`, slug).Scan(&open)
	if errors.Is(err, pgx.ErrNoRows) {
		return tournament.Entry{}, tournament.ErrNotFound
	}
	if err != nil {
		return tournament.Entry{}, err
	}
	if !open {
		return tournament.Entry{}, league.ErrRegistrationClosed
	}

	created, err := scanTournamentEntry(tx.QueryRow(ctx, `
		INSERT INTO tournament_entries (tournament_slug, display_name, display_name_normalized, nickname, registered_at)
		VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (tournament_slug, display_name_normalized) DO NOTHING
		RETURNING `+tournamentEntryColumns, slug, entry.DisplayName, league.NormalizeDisplayName(entry.DisplayName), entry.Nickname, entry.RegisteredAt))
	if errors.Is(err, pgx.ErrNoRows) {
		return tournament.Entry{}, league.ErrDuplicatePlayerName
	}
	if err != nil {
		return tournament.Entry{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return tournament.Entry{}, err
	}
	return created, nil
}

func (s *Store) ListTournamentEntries(ctx context.Context, slug string) ([]tournament.Entry, error) {
	if _, err := s.GetTournament(ctx, slug); err != nil {
		return nil, err
	}
	return listTournamentEntries(ctx, s.pool, slug)
}

func listTournamentEntries(ctx context.Context, db database, slug string) ([]tournament.Entry, error) {
	rows, err := db.Query(ctx, `
		SELECT `+tournamentEntryColumns+` FROM tournament_entries WHERE tournament_slug = $1 ORDER BY id`, slug)
	if err != nil {
		return nil, err
	}
	entries, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (tournament.Entry, error) { return scanTournamentEntry(row) })
	if entries == nil && err == nil {
		entries = []tournament.Entry{}
	}
	return entries, err
}

func (s *Store) DeleteTournamentEntry(ctx context.Context, slug string, id int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	event, err := lockTournament(ctx, tx, slug)
	if err != nil {
		return err
	}
	if len(event.Bracket) != 0 {
		return tournament.ErrTournamentStarted
	}
	tag, err := tx.Exec(ctx, `DELETE FROM tournament_entries WHERE tournament_slug = $1 AND id = $2`, slug, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return tournament.ErrEntryNotFound
	}
	return tx.Commit(ctx)
}
