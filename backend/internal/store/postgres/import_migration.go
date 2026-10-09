package postgres

import (
	"context"
	"crypto/sha256"
	_ "embed"
	"encoding/hex"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

//go:embed imports.sql
var importSchema string

//go:embed approval.sql
var approvalSchema string

func (s *Store) migrateImports(ctx context.Context) error {
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
	if _, err := tx.Exec(ctx, importSchema); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, approvalSchema); err != nil {
		return err
	}
	// Old rows never contained raw payloads or source time. Preserve the stored
	// summary as reconstructed evidence, not as a claim to recovered source bytes.
	rows, err := tx.Query(ctx, `SELECT id,jsonb_build_object('matchId',COALESCE(external_match_id,''),
		'player1',jsonb_build_object('name',player_one_name,'legsWon',player_one_legs,'matchAverage',player_one_average),
		'player2',jsonb_build_object('name',player_two_name,'legsWon',player_two_legs,'matchAverage',player_two_average))
		FROM pending_results WHERE digest IS NULL FOR UPDATE`)
	if err != nil {
		return err
	}
	type original struct {
		id      int64
		payload []byte
	}
	var originals []original
	for rows.Next() {
		var o original
		if err := rows.Scan(&o.id, &o.payload); err != nil {
			rows.Close()
			return err
		}
		originals = append(originals, o)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return err
	}
	for _, o := range originals {
		canonical, err := jsoncanonicalizer.Transform(o.payload)
		if err != nil {
			return err
		}
		digest := sha256.Sum256(canonical)
		if _, err := tx.Exec(ctx, `UPDATE pending_results SET source_payload=$2,digest=$3,settings_evidence='legacy_stored_summary' WHERE id=$1`, o.id, canonical, hex.EncodeToString(digest[:])); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `INSERT INTO import_players(pending_result_id,source_player_id,source_order,display_name,legs_won,match_average)
			SELECT id,'legacy-1',0,player_one_name,player_one_legs,player_one_average FROM pending_results WHERE id=$1
			UNION ALL SELECT id,'legacy-2',1,player_two_name,player_two_legs,player_two_average FROM pending_results WHERE id=$1
			ON CONFLICT DO NOTHING`, o.id); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
