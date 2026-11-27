package postgres

import (
	"context"
	"errors"

	"github.com/greg/darts-league/backend/internal/autodarts"
	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/tournament"
	"github.com/jackc/pgx/v5"
)

func (s *Store) LockTournamentImports(ctx context.Context) error {
	// Publication takes a row lock before writing the bracket. EXCLUSIVE keeps
	// that operation ordered with the uniqueness check across published events.
	_, err := s.pool.Exec(ctx, `LOCK TABLE tournaments IN EXCLUSIVE MODE`)
	return err
}

func (s *Store) TournamentImportCandidates(ctx context.Context, imported autodarts.Import) ([]league.TournamentImportCandidate, error) {
	if len(imported.Players) != 2 {
		return nil, nil
	}
	rows, err := s.pool.Query(ctx, `SELECT `+tournamentColumns+` FROM tournaments
		WHERE bracket IS NOT NULL AND NOT ((bracket -> -1 -> 0) ? 'winner_id')
		AND game_variant = '501' AND legs_to_win = $1
		ORDER BY starts_on DESC, slug COLLATE "C"`, max(imported.Players[0].LegsWon, imported.Players[1].LegsWon))
	if err != nil {
		return nil, err
	}
	events, err := pgx.CollectRows(rows, func(row pgx.CollectableRow) (tournament.Event, error) { return scanTournament(row) })
	if err != nil {
		return nil, err
	}
	var candidates []league.TournamentImportCandidate
	for _, event := range events {
		entries, err := listTournamentEntries(ctx, s.pool, event.Slug)
		if err != nil {
			return nil, err
		}
		event.Bracket.ResolvePublishedEntries(entries)
		candidates = append(candidates, event.ImportCandidates(entries, imported)...)
	}
	return candidates, nil
}

func (s *Store) SaveTournamentImport(ctx context.Context, record league.ImportRecord, req league.ApprovalRequest) (league.TournamentImportResult, error) {
	event, err := s.GetTournament(ctx, req.TournamentSlug)
	if err != nil {
		return league.TournamentImportResult{}, err
	}
	entries, err := listTournamentEntries(ctx, s.pool, event.Slug)
	if err != nil {
		return league.TournamentImportResult{}, err
	}
	event.Bracket.ResolvePublishedEntries(entries)
	result, err := event.RecordImportedMatch(req.MatchNumber, record.Import.Players, req.Mapping)
	if err != nil {
		return result, err
	}
	if _, err := s.pool.Exec(ctx, `INSERT INTO tournament_import_results
		(tournament_slug, match_number, pending_result_id, source, external_match_id) VALUES ($1,$2,$3,$4,$5)`,
		event.Slug, req.MatchNumber, record.Pending.ID, record.Import.Source, record.Import.ExternalMatchID); err != nil {
		return result, err
	}
	if _, err := s.pool.Exec(ctx, `UPDATE tournaments SET bracket = $2 WHERE slug = $1`, event.Slug, event.Bracket); err != nil {
		return result, err
	}
	record.Active = true
	return result, s.SaveImportApproval(ctx, record)
}

// TournamentRecordImport reads the receipt still attached to a scored match.
// Score corrections remove this link before recording a replacement.
func (s *Store) TournamentRecordImport(ctx context.Context, slug string, number int) (league.ImportRecord, error) {
	var id int64
	err := s.pool.QueryRow(ctx, `SELECT r.pending_result_id
		FROM tournament_import_results r JOIN pending_results p ON p.id = r.pending_result_id
		WHERE r.tournament_slug = $1 AND r.match_number = $2 AND p.source_active`, slug, number).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return league.ImportRecord{}, league.ErrPendingResultNotFound
	}
	if err != nil {
		return league.ImportRecord{}, err
	}
	return s.GetImport(ctx, id)
}
