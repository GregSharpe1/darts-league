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

func TestPlayerFixtureLabel(t *testing.T) {
	t.Parallel()

	nicknamed := Player{DisplayName: "Luke Humphries", Nickname: "The Freeze"}
	if nicknamed.FixtureLabel() != "The Freeze (Luke Humphries)" {
		t.Fatalf("unexpected fixture label %q", nicknamed.FixtureLabel())
	}

	unnamed := Player{DisplayName: "Michael Smith"}
	if unnamed.FixtureLabel() != "Michael Smith" {
		t.Fatalf("unexpected fixture label fallback %q", unnamed.FixtureLabel())
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

func TestValidateGameVariant(t *testing.T) {
	t.Parallel()

	if err := ValidateGameVariant("501"); err != nil {
		t.Fatalf("expected 501 to be valid, got %v", err)
	}
	if err := ValidateGameVariant("301"); err != nil {
		t.Fatalf("expected 301 to be valid, got %v", err)
	}
	if err := ValidateGameVariant("401"); err == nil {
		t.Fatal("expected 401 to be invalid")
	}
	if err := ValidateGameVariant(""); err == nil {
		t.Fatal("expected empty string to be invalid")
	}
}

func TestValidateLegsToWin(t *testing.T) {
	t.Parallel()

	if err := ValidateLegsToWin(1); err != nil {
		t.Fatalf("expected 1 to be valid, got %v", err)
	}
	if err := ValidateLegsToWin(10); err != nil {
		t.Fatalf("expected 10 to be valid, got %v", err)
	}
	if err := ValidateLegsToWin(0); err == nil {
		t.Fatal("expected 0 to be invalid")
	}
	if err := ValidateLegsToWin(-1); err == nil {
		t.Fatal("expected -1 to be invalid")
	}
}

func TestValidateGamesPerWeek(t *testing.T) {
	t.Parallel()

	if err := ValidateGamesPerWeek(1, 4); err != nil {
		t.Fatalf("expected 1 game/week with 4 players to be valid, got %v", err)
	}
	if err := ValidateGamesPerWeek(3, 4); err != nil {
		t.Fatalf("expected 3 games/week with 4 players to be valid, got %v", err)
	}
	if err := ValidateGamesPerWeek(4, 4); err == nil {
		t.Fatal("expected 4 games/week with 4 players to be invalid (max is 3)")
	}
	if err := ValidateGamesPerWeek(0, 4); err == nil {
		t.Fatal("expected 0 games/week to be invalid")
	}
}

func TestGamesPerWeekPresets(t *testing.T) {
	t.Parallel()

	presets := GamesPerWeekPresets(6)
	if len(presets) != 5 {
		t.Fatalf("expected 5 presets for 6 players, got %d", len(presets))
	}
	// 1 game/week = 5 weeks
	if presets[0].GamesPerWeek != 1 || presets[0].WeekCount != 5 {
		t.Fatalf("expected 1 game/week = 5 weeks, got %+v", presets[0])
	}
	// 5 games/week = 1 week
	if presets[4].GamesPerWeek != 5 || presets[4].WeekCount != 1 {
		t.Fatalf("expected 5 games/week = 1 week, got %+v", presets[4])
	}
	// 2 games/week = 3 weeks (ceil(5/2))
	if presets[1].GamesPerWeek != 2 || presets[1].WeekCount != 3 {
		t.Fatalf("expected 2 games/week = 3 weeks, got %+v", presets[1])
	}
}

func TestGamesPerWeekPresetsReturnNilForTooFewPlayers(t *testing.T) {
	t.Parallel()

	if presets := GamesPerWeekPresets(1); presets != nil {
		t.Fatalf("expected nil presets for 1 player, got %+v", presets)
	}
	if presets := GamesPerWeekPresets(0); presets != nil {
		t.Fatalf("expected nil presets for 0 players, got %+v", presets)
	}
}

func TestComputeSchedulePreview(t *testing.T) {
	t.Parallel()

	preview := ComputeSchedulePreview(8, "501", 3, 2)
	if preview.PlayerCount != 8 {
		t.Fatalf("expected 8 players, got %d", preview.PlayerCount)
	}
	if preview.TotalFixtures != 28 {
		t.Fatalf("expected 28 total fixtures for 8 players, got %d", preview.TotalFixtures)
	}
	if preview.WeekCount != 4 {
		t.Fatalf("expected 4 weeks (ceil(7/2)), got %d", preview.WeekCount)
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
