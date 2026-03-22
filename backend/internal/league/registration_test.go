package league

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestNormalizeDisplayName(t *testing.T) {
	t.Parallel()

	got := NormalizeDisplayName("  The   Power  ")
	if got != "the power" {
		t.Fatalf("expected normalized name to be %q, got %q", "the power", got)
	}
}

func TestPlayerPreferredName(t *testing.T) {
	t.Parallel()

	player := Player{DisplayName: "Phil Taylor", Nickname: "The Power"}
	if player.PreferredName() != "The Power" {
		t.Fatalf("expected nickname to be preferred, got %q", player.PreferredName())
	}

	unnamed := Player{DisplayName: "Michael Smith"}
	if unnamed.PreferredName() != "Michael Smith" {
		t.Fatalf("expected display name fallback, got %q", unnamed.PreferredName())
	}
}

func TestPlayerAdminLabel(t *testing.T) {
	t.Parallel()

	player := Player{DisplayName: "Gerwyn Price", Nickname: "The Iceman"}
	if player.AdminLabel() != "The Iceman (Gerwyn Price)" {
		t.Fatalf("unexpected admin label %q", player.AdminLabel())
	}
}

func TestRegistrationBookRejectsDuplicateDisplayNamesCaseInsensitive(t *testing.T) {
	t.Parallel()

	season := NewSeason("Spring 2026")
	book := NewRegistrationBook([]Player{{DisplayName: "Luke Humphries"}})

	err := book.ValidateNewPlayer(season, Player{DisplayName: "  LUKE   humphries "})
	if !errors.Is(err, ErrDuplicatePlayerName) {
		t.Fatalf("expected duplicate player error, got %v", err)
	}
}

func TestRegistrationBookRejectsBlankDisplayNames(t *testing.T) {
	t.Parallel()

	season := NewSeason("Spring 2026")
	book := NewRegistrationBook(nil)

	err := book.ValidateNewPlayer(season, Player{DisplayName: "   "})
	if !errors.Is(err, ErrDisplayNameRequired) {
		t.Fatalf("expected display name required error, got %v", err)
	}
}

func TestRegistrationBookRejectsNewPlayersAfterSeasonStart(t *testing.T) {
	t.Parallel()

	season := NewSeason("Spring 2026").Start(time.Date(2026, time.March, 23, 9, 0, 0, 0, time.UTC))
	book := NewRegistrationBook(nil)

	err := book.ValidateNewPlayer(season, Player{DisplayName: "Nathan Aspinall"})
	if !errors.Is(err, ErrRegistrationClosed) {
		t.Fatalf("expected registration closed error, got %v", err)
	}
}

func TestRegistrationBookAllowsPlayerDeleteOnlyBeforeSeasonStart(t *testing.T) {
	t.Parallel()

	book := NewRegistrationBook(nil)

	if err := book.ValidatePlayerDelete(NewSeason("Spring 2026")); err != nil {
		t.Fatalf("expected pre-season delete to be allowed, got %v", err)
	}

	started := NewSeason("Spring 2026").Start(time.Date(2026, time.March, 23, 9, 0, 0, 0, time.UTC))
	err := book.ValidatePlayerDelete(started)
	if !errors.Is(err, ErrPlayerDeleteLocked) {
		t.Fatalf("expected delete locked error, got %v", err)
	}
}

func TestRegistrationServiceNotifiesAfterSuccessfulSignup(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	notifier := &stubRegistrationNotifier{}
	service := NewRegistrationServiceWithNowAndNotifier(store, func() time.Time {
		return time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	}, notifier)

	player, err := service.RegisterPlayer(context.Background(), Player{DisplayName: "Luke Humphries", Nickname: "The Freeze"})
	if err != nil {
		t.Fatalf("expected registration to succeed, got %v", err)
	}

	if len(notifier.players) != 1 {
		t.Fatalf("expected notifier to receive 1 player, got %d", len(notifier.players))
	}

	if notifier.players[0].ID != player.ID {
		t.Fatalf("expected notified player id %d, got %d", player.ID, notifier.players[0].ID)
	}

	if notifier.totals[0] != 1 {
		t.Fatalf("expected total registered count 1, got %d", notifier.totals[0])
	}
}

func TestRegistrationServiceSkipsNotifierOnValidationFailure(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	notifier := &stubRegistrationNotifier{}
	service := NewRegistrationServiceWithNowAndNotifier(store, time.Now, notifier)

	if _, err := service.RegisterPlayer(context.Background(), Player{DisplayName: "   "}); !errors.Is(err, ErrDisplayNameRequired) {
		t.Fatalf("expected display name required error, got %v", err)
	}

	if len(notifier.players) != 0 {
		t.Fatalf("expected notifier not to be called, got %d calls", len(notifier.players))
	}
}

type stubRegistrationNotifier struct {
	players []Player
	totals  []int
}

func (n *stubRegistrationNotifier) NotifyPlayerRegistered(_ context.Context, player Player, totalRegistered int) {
	n.players = append(n.players, player)
	n.totals = append(n.totals, totalRegistered)
}
