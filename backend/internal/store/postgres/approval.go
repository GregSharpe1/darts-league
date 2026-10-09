package postgres

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/jackc/pgx/v5"
)

func (s *Store) ImportsBySource(ctx context.Context, source, external string) ([]league.ImportRecord, error) {
	rows, err := s.pool.Query(ctx, `SELECT id FROM pending_results WHERE source=$1 AND COALESCE(external_match_id,'')=$2 ORDER BY id`, source, external)
	if err != nil {
		return nil, err
	}
	ids := []int64{}
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, err
	}
	records := []league.ImportRecord{}
	for _, id := range ids {
		r, err := s.GetImport(ctx, id)
		if err != nil {
			return nil, err
		}
		records = append(records, r)
	}
	return records, nil
}

func (s *Store) ImportByFixture(ctx context.Context, id int64) (league.ImportRecord, error) {
	var pendingID int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM pending_results WHERE fixture_id=$1 AND source_active`, id).Scan(&pendingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return league.ImportRecord{}, league.ErrPendingResultNotFound
	}
	if err != nil {
		return league.ImportRecord{}, err
	}
	return s.GetImport(ctx, pendingID)
}

func (s *Store) ImportByResult(ctx context.Context, id int64) (league.ImportRecord, error) {
	var pendingID int64
	err := s.pool.QueryRow(ctx, `SELECT id FROM pending_results WHERE result_id=$1 AND source_active`, id).Scan(&pendingID)
	if errors.Is(err, pgx.ErrNoRows) {
		return league.ImportRecord{}, league.ErrPendingResultNotFound
	}
	if err != nil {
		return league.ImportRecord{}, err
	}
	return s.GetImport(ctx, pendingID)
}

func (s *Store) SaveImportApproval(ctx context.Context, r league.ImportRecord) error {
	if _, err := s.pool.Exec(ctx, `UPDATE pending_results SET digest=$2,source_payload=$3,settings_evidence=$4 WHERE id=$1 AND digest IS NULL`, r.Pending.ID, r.Import.Digest, []byte(r.Import.Payload), r.Import.SettingsEvidence); err != nil {
		return err
	}
	if _, err := s.pool.Exec(ctx, `INSERT INTO import_players(pending_result_id,source_player_id,source_order,display_name,legs_won,match_average)
	 SELECT id,'legacy-1',0,player_one_name,player_one_legs,player_one_average FROM pending_results WHERE id=$1 AND settings_evidence='legacy_stored_summary'
	 UNION ALL SELECT id,'legacy-2',1,player_two_name,player_two_legs,player_two_average FROM pending_results WHERE id=$1 AND settings_evidence='legacy_stored_summary'
	 ON CONFLICT DO NOTHING`, r.Pending.ID); err != nil {
		return err
	}
	metadata, err := json.Marshal(r.Approval)
	if err != nil {
		return err
	}
	tag, err := s.pool.Exec(ctx, `UPDATE pending_results SET status=$2,confirmed_at=$3,confirmed_by=$4,
  season_id=$5,fixture_id=$6,result_id=$7,source_active=$8,approval_metadata=$9 WHERE id=$1`,
		r.Pending.ID, r.Pending.Status, r.Pending.ConfirmedAt, r.Pending.ConfirmedBy, r.SeasonID, r.FixtureID, r.ResultID, r.Active, metadata)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return league.ErrPendingResultNotFound
	}
	for source, playerID := range r.Mapping {
		var slot int
		if err := s.pool.QueryRow(ctx, `SELECT CASE WHEN player_one_id=$2 THEN 1 WHEN player_two_id=$2 THEN 2 ELSE 0 END FROM fixtures WHERE id=$1`, r.FixtureID, playerID).Scan(&slot); err != nil {
			return err
		}
		if slot == 0 {
			return league.ErrInvalidMapping
		}
		if _, err := s.pool.Exec(ctx, `UPDATE import_players SET league_player_id=$3,fixture_slot=$4 WHERE pending_result_id=$1 AND source_player_id=$2`, r.Pending.ID, source, playerID, slot); err != nil {
			return err
		}
	}
	return nil
}
