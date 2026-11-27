package postgres

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/playeraccess"
	"github.com/greg/darts-league/backend/internal/tournament"
)

func TestVerifiedPlayerCanReuseClaimAcrossCompetitions(t *testing.T) {
	store, seasonID, _ := completedFixtures(t)
	ctx := context.Background()
	now := time.Now().UTC()
	email := "player@example.test"
	if err := store.ReserveSignIn(ctx, email, "stale", "/register", now.Add(-21*time.Minute)); err != nil {
		t.Fatal(err)
	}
	if _, err := store.ExchangeSignIn(ctx, "stale", "stale-session", now); !errors.Is(err, playeraccess.ErrExpired) {
		t.Fatalf("expired link: %v", err)
	}
	if err := store.ReserveSignIn(ctx, email, "fresh", "/register", now); err != nil {
		t.Fatal(err)
	}
	if path, err := store.ExchangeSignIn(ctx, "fresh", "session", now); err != nil || path != "/register" {
		t.Fatalf("exchange: %q %v", path, err)
	}
	if _, err := store.ExchangeSignIn(ctx, "fresh", "replay", now); !errors.Is(err, playeraccess.ErrExpired) {
		t.Fatalf("replayed link: %v", err)
	}
	account, err := store.PlayerAccount(ctx, "session", now)
	if err != nil || account.IdentityID != 0 {
		t.Fatalf("verified email should initially be unclaimed: %+v %v", account, err)
	}

	var identity int64
	if err := store.pool.QueryRow(ctx, `SELECT identity_id FROM players WHERE season_id = $1 AND display_name = 'Alice'`, seasonID).Scan(&identity); err != nil {
		t.Fatal(err)
	}
	if err := store.SubmitPlayerClaim(ctx, email, identity); err != nil {
		t.Fatal(err)
	}
	if err := store.SubmitPlayerClaim(ctx, email, identity); !errors.Is(err, playeraccess.ErrClaim) {
		t.Fatalf("duplicate pending claim: %v", err)
	}
	claims, err := store.ListPlayerClaims(ctx)
	if err != nil || len(claims) != 1 || claims[0].IdentityID != identity {
		t.Fatalf("claim queue: %+v %v", claims, err)
	}
	if err := store.DecidePlayerClaim(ctx, claims[0].ID, true, "admin"); err != nil {
		t.Fatal(err)
	}
	account, err = store.PlayerAccount(ctx, "session", now)
	if err != nil || account.IdentityID != identity || account.DisplayName != "Alice" {
		t.Fatalf("approved account: %+v %v", account, err)
	}

	if err := store.CloseSeason(ctx, seasonID); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateNextSeason(ctx, seasonID, league.NewSeason("Returning Players")); err != nil {
		t.Fatal(err)
	}
	next, err := store.GetActiveSeason(ctx)
	if err != nil {
		t.Fatal(err)
	}
	verified := league.WithVerifiedPlayerEmail(ctx, email)
	registration := league.NewRegistrationService(store)
	if _, err := registration.RegisterPlayer(verified, league.Player{SeasonID: seasonID, DisplayName: account.DisplayName}); !errors.Is(err, league.ErrSeasonTransition) {
		t.Fatalf("a stale sign-up page should not enter a new season: %v", err)
	}
	player, err := registration.RegisterPlayer(verified, league.Player{SeasonID: next.ID, DisplayName: account.DisplayName, Nickname: account.Nickname})
	if err != nil {
		t.Fatal(err)
	}

	service := tournament.Service{Store: store, Now: time.Now}
	cup, err := service.Create(ctx, "Verified Player Cup", "2027-04-01", "", tournament.DefaultMatchFormat())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetTournamentRegistration(ctx, cup.Slug, true); err != nil {
		t.Fatal(err)
	}
	entry, err := service.Register(verified, cup.Slug, account.DisplayName, account.Nickname)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Register(verified, cup.Slug, "Another Name", ""); !errors.Is(err, league.ErrDuplicatePlayerName) {
		t.Fatalf("the same identity cannot enter twice with a different name: %v", err)
	}
	var leagueIdentity, cupIdentity int64
	if err := store.pool.QueryRow(ctx, `SELECT
		(SELECT identity_id FROM players WHERE id = $1),
		(SELECT identity_id FROM tournament_entries WHERE id = $2)`, player.ID, entry.ID).Scan(&leagueIdentity, &cupIdentity); err != nil {
		t.Fatal(err)
	}
	if leagueIdentity != identity || cupIdentity != identity {
		t.Fatalf("entries must reuse the approved identity: league %d, cup %d, expected %d", leagueIdentity, cupIdentity, identity)
	}
	if err := store.EndPlayerSession(ctx, "session"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.PlayerAccount(ctx, "session", now); !errors.Is(err, playeraccess.ErrSession) {
		t.Fatalf("signed-out session: %v", err)
	}
	t.Cleanup(func() {
		_, _ = store.pool.Exec(ctx, `DELETE FROM player_signin_links WHERE email = $1`, email)
		_, _ = store.pool.Exec(ctx, `DELETE FROM player_accounts WHERE email = $1`, email)
		_, _ = store.pool.Exec(ctx, `DELETE FROM tournament_entries WHERE tournament_slug = $1`, cup.Slug)
		_, _ = store.pool.Exec(ctx, `DELETE FROM tournaments WHERE slug = $1`, cup.Slug)
	})
}
