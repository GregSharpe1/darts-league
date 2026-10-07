package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/greg/darts-league/backend/internal/autodarts"
	"github.com/greg/darts-league/backend/internal/league"
	"github.com/jackc/pgx/v5"
)

func (s *Store) DurableImports() {}

func (s *Store) InsertImport(ctx context.Context, imported autodarts.Import, received time.Time) (league.ImportOutcome, error) {
	var outcome league.ImportOutcome
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return outcome, err
	}
	defer tx.Rollback(ctx)
	// Serialize only this source identity, including changed digests. The unique
	// index remains the final replay guard. Hash collisions only reduce concurrency.
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1,0))`, imported.Source+":"+imported.ExternalMatchID); err != nil {
		return outcome, err
	}
	var existing int64
	err = tx.QueryRow(ctx, `SELECT id,changed_import FROM pending_results WHERE source=$1 AND external_match_id=$2 AND digest=$3`, imported.Source, imported.ExternalMatchID, imported.Digest).Scan(&existing, &outcome.Changed)
	if err == nil {
		outcome.Pending, err = getPendingInTx(ctx, tx, existing)
		if err != nil {
			return outcome, err
		}
		outcome.Duplicate = true
		return outcome, tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return outcome, err
	}
	if err := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM pending_results WHERE source=$1 AND external_match_id=$2)`, imported.Source, imported.ExternalMatchID).Scan(&outcome.Changed); err != nil {
		return outcome, err
	}
	a, b := imported.Players[0], imported.Players[1]
	status := league.PendingResultStatusPending
	if imported.ReviewReason != "" {
		status = league.PendingResultStatusReviewBlocked
	}
	var id int64
	err = tx.QueryRow(ctx, `INSERT INTO pending_results
		(source,external_match_id,digest,source_payload,played_at_original,settings_evidence,review_reason,changed_import,
		 player_one_name,player_one_legs,player_one_average,player_two_name,player_two_legs,player_two_average,status,received_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) RETURNING id`,
		imported.Source, imported.ExternalMatchID, imported.Digest, []byte(imported.Payload), imported.PlayedAt, imported.SettingsEvidence, imported.ReviewReason, outcome.Changed,
		a.DisplayName, a.LegsWon, a.Average(), b.DisplayName, b.LegsWon, b.Average(), status, received).Scan(&id)
	if err != nil {
		return outcome, err
	}
	for i, player := range imported.Players {
		stats := autodarts.Stats{}
		if player.Stats != nil {
			stats = *player.Stats
		}
		_, err := tx.Exec(ctx, `INSERT INTO import_players(pending_result_id,source_player_id,source_order,account_id,display_name,legs_won,
			match_average,points_scored,darts_thrown,checkout_hits,checkout_attempts) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`,
			id, player.ID, i, player.AccountID, player.DisplayName, player.LegsWon, stats.MatchAverage, stats.PointsScored, stats.DartsThrown, stats.CheckoutHits, stats.CheckoutAttempts)
		if err != nil {
			return outcome, err
		}
	}
	outcome.Pending = league.PendingResult{ID: id, ExternalMatchID: imported.ExternalMatchID, PlayerOneName: a.DisplayName, PlayerOneLegs: a.LegsWon, PlayerOneAverage: a.Average(), PlayerTwoName: b.DisplayName, PlayerTwoLegs: b.LegsWon, PlayerTwoAverage: b.Average(), Status: status, ReceivedAt: received}
	return outcome, tx.Commit(ctx)
}

func getPendingInTx(ctx context.Context, tx pgx.Tx, id int64) (league.PendingResult, error) {
	return scanPendingResult(tx.QueryRow(ctx, `SELECT id,external_match_id,player_one_name,player_one_legs,player_one_average,
		player_two_name,player_two_legs,player_two_average,status,received_at,confirmed_at,confirmed_by FROM pending_results WHERE id=$1`, id))
}

func (s *Store) GetImport(ctx context.Context, id int64) (league.ImportRecord, error) {
	var record league.ImportRecord
	var payload []byte
	var approval []byte
	err := s.pool.QueryRow(ctx, `SELECT source,COALESCE(external_match_id,''),COALESCE(digest,''),source_payload,played_at_original,settings_evidence,
		review_reason,changed_import,season_id,fixture_id,result_id,source_active,approval_metadata FROM pending_results WHERE id=$1`, id).Scan(
		&record.Import.Source, &record.Import.ExternalMatchID, &record.Import.Digest, &payload, &record.Import.PlayedAt, &record.Import.SettingsEvidence,
		&record.Import.ReviewReason, &record.Changed, &record.SeasonID, &record.FixtureID, &record.ResultID, &record.Active, &approval)
	if errors.Is(err, pgx.ErrNoRows) {
		return record, league.ErrPendingResultNotFound
	}
	if err != nil {
		return record, err
	}
	record.Pending, err = s.GetPendingResult(ctx, id)
	if err != nil {
		return record, err
	}
	record.Import.Payload = payload
	if len(approval) > 0 {
		if err := json.Unmarshal(approval, &record.Approval); err != nil {
			return record, err
		}
		record.Mapping = record.Approval.Mapping
	}
	if record.Import.SettingsEvidence == "source_reported" {
		var p autodarts.Payload
		if err := json.Unmarshal(payload, &p); err != nil {
			return record, err
		}
		record.Import.Players, record.Import.Detail = p.Players, p.Detail
	} else {
		legacy, err := league.LegacyImportRecord(record.Pending)
		if err != nil {
			return record, err
		}
		record.Import.Players = legacy.Import.Players
		if record.Import.Digest == "" {
			record.Import = legacy.Import
		}
	}
	return record, nil
}
