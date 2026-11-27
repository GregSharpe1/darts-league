package postgres

import (
	"context"
	_ "embed"
	"errors"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/playeraccess"
	"github.com/jackc/pgx/v5"
)

//go:embed player_access.sql
var playerAccessSchema string

func (s *Store) ReserveSignIn(ctx context.Context, email, hash, path string, now time.Time) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(390041)`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM player_signin_links WHERE created_at < $1`, now.Add(-time.Hour)); err != nil {
		return err
	}
	var recipient, total int
	if err := tx.QueryRow(ctx, `SELECT count(*) FILTER (WHERE email = $1), count(*) FROM player_signin_links`, email).Scan(&recipient, &total); err != nil {
		return err
	}
	if recipient >= 4 || total >= 120 {
		return playeraccess.ErrRate
	}
	if _, err := tx.Exec(ctx, `INSERT INTO player_signin_links (token_hash, email, return_path, created_at) VALUES ($1, $2, $3, $4)`, hash, email, path, now); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) CancelSignIn(ctx context.Context, hash string) error {
	// Invalidate failed deliveries and retain the attempt for the rate limit.
	_, err := s.pool.Exec(ctx, `UPDATE player_signin_links SET used = TRUE WHERE token_hash = $1`, hash)
	return err
}

func (s *Store) ExchangeSignIn(ctx context.Context, linkHash, sessionHash string, now time.Time) (string, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return "", err
	}
	defer tx.Rollback(ctx)
	var email, path string
	err = tx.QueryRow(ctx, `UPDATE player_signin_links SET used = TRUE
		WHERE token_hash = $1 AND NOT used AND created_at > $2 RETURNING email, return_path`, linkHash, now.Add(-20*time.Minute)).Scan(&email, &path)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", playeraccess.ErrExpired
	}
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO player_accounts(email) VALUES ($1) ON CONFLICT DO NOTHING`, email); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM player_sessions WHERE expires_at <= $1`, now); err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO player_sessions (token_hash, email, expires_at) VALUES ($1, $2, $3)`, sessionHash, email, now.Add(playeraccess.SessionLifetime)); err != nil {
		return "", err
	}
	return path, tx.Commit(ctx)
}

func (s *Store) PlayerAccount(ctx context.Context, hash string, now time.Time) (playeraccess.Account, error) {
	var a playeraccess.Account
	err := s.pool.QueryRow(ctx, `
		SELECT a.email, COALESCE(a.identity_id, 0), COALESCE(p.display_name, ''), COALESCE(p.nickname, ''), COALESCE(c.status, '')
		FROM player_sessions s JOIN player_accounts a USING (email)
		LEFT JOIN LATERAL (
			SELECT display_name, nickname FROM player_profile_entries WHERE identity_id = a.identity_id ORDER BY source, entry_id DESC LIMIT 1
		) p ON TRUE
		LEFT JOIN LATERAL (
			SELECT status FROM player_profile_claims WHERE email = a.email ORDER BY id DESC LIMIT 1
		) c ON TRUE
		WHERE s.token_hash = $1 AND s.expires_at > $2`, hash, now).Scan(&a.Email, &a.IdentityID, &a.DisplayName, &a.Nickname, &a.ClaimStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, playeraccess.ErrSession
	}
	return a, err
}

func (s *Store) EndPlayerSession(ctx context.Context, hash string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM player_sessions WHERE token_hash = $1`, hash)
	return err
}

func (s *Store) SubmitPlayerClaim(ctx context.Context, email string, identity int64) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockPlayerIdentities(ctx, tx); err != nil {
		return err
	}
	tag, err := tx.Exec(ctx, `
		INSERT INTO player_profile_claims(email, identity_id)
		SELECT $1, $2
		WHERE EXISTS (SELECT 1 FROM player_accounts WHERE email = $1 AND identity_id IS NULL)
		  AND EXISTS (SELECT 1 FROM player_profile_entries WHERE identity_id = $2)
		  AND NOT EXISTS (SELECT 1 FROM player_accounts WHERE identity_id = $2)
		ON CONFLICT DO NOTHING`, email, identity)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return playeraccess.ErrClaim
	}
	return tx.Commit(ctx)
}

func (s *Store) ListPlayerClaims(ctx context.Context) ([]playeraccess.Claim, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT c.id, c.email, c.identity_id, COALESCE(p.display_name, ''), COALESCE(p.nickname, ''), COALESCE(p.competition, '')
		FROM player_profile_claims c
		LEFT JOIN LATERAL (
			SELECT display_name, nickname, competition FROM player_profile_entries WHERE identity_id = c.identity_id ORDER BY source, entry_id DESC LIMIT 1
		) p ON TRUE
		WHERE c.status = 'pending' ORDER BY c.id`)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(row pgx.CollectableRow) (playeraccess.Claim, error) {
		var c playeraccess.Claim
		err := row.Scan(&c.ID, &c.Email, &c.IdentityID, &c.DisplayName, &c.Nickname, &c.Competition)
		return c, err
	})
}

func (s *Store) DecidePlayerClaim(ctx context.Context, id int64, approve bool, actor string) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	if err := lockPlayerIdentities(ctx, tx); err != nil {
		return err
	}
	var identity int64
	var email string
	err = tx.QueryRow(ctx, `SELECT identity_id, email FROM player_profile_claims WHERE id = $1 AND status = 'pending'`, id).Scan(&identity, &email)
	if errors.Is(err, pgx.ErrNoRows) {
		return playeraccess.ErrClaim
	}
	if err != nil {
		return err
	}
	status := "rejected"
	if approve {
		tag, err := tx.Exec(ctx, `
			UPDATE player_accounts SET identity_id = $1
			WHERE email = $2 AND identity_id IS NULL
			  AND EXISTS (SELECT 1 FROM player_profile_entries WHERE identity_id = $1)
			  AND NOT EXISTS (SELECT 1 FROM player_accounts WHERE identity_id = $1)`, identity, email)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return playeraccess.ErrClaim
		}
		status = "approved"
	}
	if _, err := tx.Exec(ctx, `UPDATE player_profile_claims SET status = $2, decided_by = $3, decided_at = NOW() WHERE id = $1`, id, status, actor); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func registrationIdentity(ctx context.Context, tx pgx.Tx) (*int64, error) {
	email := league.VerifiedPlayerEmail(ctx)
	if email == "" {
		return nil, nil
	}
	if err := lockPlayerIdentities(ctx, tx); err != nil {
		return nil, err
	}
	var identity int64
	err := tx.QueryRow(ctx, `SELECT identity_id FROM player_accounts WHERE email = $1 AND identity_id IS NOT NULL`, email).Scan(&identity)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, playeraccess.ErrClaim
	}
	return &identity, err
}

func lockPlayerIdentities(ctx context.Context, tx pgx.Tx) error {
	_, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(390040)`)
	return err
}
