package league

import (
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
