CREATE TABLE IF NOT EXISTS player_accounts (
    email TEXT PRIMARY KEY,
    identity_id BIGINT UNIQUE
);

CREATE TABLE IF NOT EXISTS player_signin_links (
    token_hash TEXT PRIMARY KEY,
    email TEXT NOT NULL,
    return_path TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT FALSE
);

CREATE INDEX IF NOT EXISTS player_signin_links_created_idx ON player_signin_links (created_at);

CREATE TABLE IF NOT EXISTS player_sessions (
    token_hash TEXT PRIMARY KEY,
    email TEXT NOT NULL REFERENCES player_accounts (email) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS player_sessions_expires_idx ON player_sessions (expires_at);

CREATE TABLE IF NOT EXISTS player_profile_claims (
    id BIGSERIAL PRIMARY KEY,
    email TEXT NOT NULL REFERENCES player_accounts (email) ON DELETE CASCADE,
    identity_id BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    decided_by TEXT,
    decided_at TIMESTAMPTZ
);

CREATE UNIQUE INDEX IF NOT EXISTS player_profile_claims_pending_email_idx
    ON player_profile_claims (email) WHERE status = 'pending';

CREATE OR REPLACE VIEW player_profile_entries AS
SELECT identity_id, display_name, COALESCE(nickname, '') AS nickname, s.name AS competition, 0 AS source, p.id AS entry_id
FROM players p JOIN seasons s ON s.id = p.season_id
UNION ALL
SELECT identity_id, display_name, nickname, t.name, 1, e.id
FROM tournament_entries e JOIN tournaments t ON t.slug = e.tournament_slug;
