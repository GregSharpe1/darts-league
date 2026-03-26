package league

import (
	"fmt"
	"testing"
	"time"
)

func TestGenerateRoundRobinFixturesCreatesSingleRoundRobin(t *testing.T) {
	t.Parallel()

	season := NewSeason("Spring 2026")
	season.ID = 1
	season = season.Start(time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC))
	players := []Player{
		{ID: 1, DisplayName: "Luke Humphries"},
		{ID: 2, DisplayName: "Michael Smith"},
		{ID: 3, DisplayName: "Peter Wright"},
		{ID: 4, DisplayName: "Gerwyn Price"},
	}

	fixtures, err := GenerateRoundRobinFixtures(season, players)
	if err != nil {
		t.Fatalf("expected fixtures to be generated, got %v", err)
	}

	if len(fixtures) != 6 {
		t.Fatalf("expected 6 fixtures, got %d", len(fixtures))
	}

	pairings := map[[2]int64]bool{}
	for _, fixture := range fixtures {
		pair := [2]int64{fixture.PlayerOneID, fixture.PlayerTwoID}
		reverse := [2]int64{fixture.PlayerTwoID, fixture.PlayerOneID}
		if pairings[pair] || pairings[reverse] {
			t.Fatalf("duplicate pairing found: %+v", pair)
		}
		pairings[pair] = true
	}

	if fixtures[0].WeekNumber != 1 || fixtures[len(fixtures)-1].WeekNumber != 3 {
		t.Fatalf("expected weeks to span 1..3, got first=%d last=%d", fixtures[0].WeekNumber, fixtures[len(fixtures)-1].WeekNumber)
	}
	if fixtures[0].GameVariant != GameVariant501 || fixtures[0].LegsToWin != DefaultLegsToWin {
		t.Fatalf("expected default match format, got %s first-to-%d", fixtures[0].GameVariant, fixtures[0].LegsToWin)
	}
}

func TestFirstWeekRevealAtUsesNextMondayAtNineInLondon(t *testing.T) {
	t.Parallel()

	loc, _ := time.LoadLocation("Europe/London")
	startedAt := time.Date(2026, time.March, 18, 20, 0, 0, 0, time.UTC)

	revealAt, err := FirstWeekRevealAt(startedAt, loc)
	if err != nil {
		t.Fatalf("expected reveal calculation to work, got %v", err)
	}

	expected := time.Date(2026, time.March, 23, 9, 0, 0, 0, loc)
	if !revealAt.Equal(expected) {
		t.Fatalf("expected reveal at %s, got %s", expected, revealAt)
	}
}

func TestWeekScheduleTimePreservesLondonLocalTimeAcrossDST(t *testing.T) {
	t.Parallel()

	loc, _ := time.LoadLocation("Europe/London")
	firstReveal := time.Date(2026, time.March, 23, 9, 0, 0, 0, loc)

	weekTwo := WeekScheduleTime(firstReveal, 1)
	if weekTwo.Hour() != 19 || weekTwo.Minute() != 30 {
		t.Fatalf("expected week two to stay at 19:30 local, got %s", weekTwo)
	}
	if weekTwo.Location().String() != "Europe/London" {
		t.Fatalf("expected London location, got %s", weekTwo.Location())
	}
}

func TestNoPairingAppearsInMultipleWeeks(t *testing.T) {
	t.Parallel()

	const playerCount = 32

	season := NewSeason("Big League 2026")
	season.ID = 1
	season = season.Start(time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC))

	players := make([]Player, playerCount)
	for i := range players {
		players[i] = Player{
			ID:          int64(i + 1),
			DisplayName: fmt.Sprintf("Player %02d", i+1),
		}
	}

	fixtures, err := GenerateRoundRobinFixtures(season, players)
	if err != nil {
		t.Fatalf("expected fixtures, got error: %v", err)
	}

	expectedFixtures := playerCount * (playerCount - 1) / 2
	if len(fixtures) != expectedFixtures {
		t.Fatalf("expected %d fixtures (C(%d,2)), got %d", expectedFixtures, playerCount, len(fixtures))
	}

	// Check no pairing appears in more than one week.
	type pair struct{ a, b int64 }
	pairWeek := map[pair]int{}
	for _, f := range fixtures {
		a, b := f.PlayerOneID, f.PlayerTwoID
		if a > b {
			a, b = b, a
		}
		p := pair{a, b}
		if prev, exists := pairWeek[p]; exists {
			t.Fatalf("players %d and %d appear in both week %d and week %d", a, b, prev, f.WeekNumber)
		}
		pairWeek[p] = f.WeekNumber
	}

	if len(pairWeek) != expectedFixtures {
		t.Fatalf("expected %d unique pairings, got %d", expectedFixtures, len(pairWeek))
	}

	// Check no player appears more than once in the same week.
	weekPlayers := map[int]map[int64]bool{}
	for _, f := range fixtures {
		if weekPlayers[f.WeekNumber] == nil {
			weekPlayers[f.WeekNumber] = map[int64]bool{}
		}
		if weekPlayers[f.WeekNumber][f.PlayerOneID] {
			t.Fatalf("player %d appears twice in week %d", f.PlayerOneID, f.WeekNumber)
		}
		weekPlayers[f.WeekNumber][f.PlayerOneID] = true
		if weekPlayers[f.WeekNumber][f.PlayerTwoID] {
			t.Fatalf("player %d appears twice in week %d", f.PlayerTwoID, f.WeekNumber)
		}
		weekPlayers[f.WeekNumber][f.PlayerTwoID] = true
	}

	expectedWeeks := playerCount - 1
	if len(weekPlayers) != expectedWeeks {
		t.Fatalf("expected %d weeks, got %d", expectedWeeks, len(weekPlayers))
	}
}

func TestGenerateFixturesUsesSeasonConfig(t *testing.T) {
	t.Parallel()

	season := NewSeason("Spring 2026")
	season.ID = 1
	season.GameVariant = GameVariant301
	season.LegsToWin = 5
	season = season.Start(time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC))
	players := []Player{
		{ID: 1, DisplayName: "Luke Humphries"},
		{ID: 2, DisplayName: "Michael Smith"},
	}

	fixtures, err := GenerateRoundRobinFixtures(season, players)
	if err != nil {
		t.Fatalf("expected fixtures, got %v", err)
	}

	if len(fixtures) != 1 {
		t.Fatalf("expected 1 fixture, got %d", len(fixtures))
	}
	if fixtures[0].GameVariant != GameVariant301 {
		t.Fatalf("expected 301 variant, got %s", fixtures[0].GameVariant)
	}
	if fixtures[0].LegsToWin != 5 {
		t.Fatalf("expected first-to-5, got %d", fixtures[0].LegsToWin)
	}
}

func TestGenerateFixturesWithMultipleGamesPerWeek(t *testing.T) {
	t.Parallel()

	season := NewSeason("Compressed League")
	season.ID = 1
	season.GamesPerWeek = 2
	season = season.Start(time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC))

	// 6 players = 5 games per player, 15 total fixtures
	// 2 games/week = ceil(5/2) = 3 weeks
	players := make([]Player, 6)
	for i := range players {
		players[i] = Player{ID: int64(i + 1), DisplayName: fmt.Sprintf("Player %d", i+1)}
	}

	fixtures, err := GenerateRoundRobinFixtures(season, players)
	if err != nil {
		t.Fatalf("expected fixtures, got %v", err)
	}

	if len(fixtures) != 15 {
		t.Fatalf("expected 15 fixtures, got %d", len(fixtures))
	}

	// Check no player exceeds 2 games in any week.
	weekPlayerCount := map[int]map[int64]int{}
	maxWeek := 0
	for _, f := range fixtures {
		if weekPlayerCount[f.WeekNumber] == nil {
			weekPlayerCount[f.WeekNumber] = map[int64]int{}
		}
		weekPlayerCount[f.WeekNumber][f.PlayerOneID]++
		weekPlayerCount[f.WeekNumber][f.PlayerTwoID]++
		if f.WeekNumber > maxWeek {
			maxWeek = f.WeekNumber
		}
	}

	for week, players := range weekPlayerCount {
		for playerID, count := range players {
			if count > 2 {
				t.Fatalf("player %d has %d games in week %d, expected max 2", playerID, count, week)
			}
		}
	}

	if maxWeek != 3 {
		t.Fatalf("expected 3 weeks with 2 games/week for 6 players, got %d", maxWeek)
	}
}

func TestGenerateFixturesPartialFinalWeek(t *testing.T) {
	t.Parallel()

	season := NewSeason("Odd Division")
	season.ID = 1
	season.GamesPerWeek = 2
	season = season.Start(time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC))

	// 4 players = 3 games per player, 6 total fixtures
	// 2 games/week = ceil(3/2) = 2 weeks
	// Week 1: 2 games per player, Week 2: 1 game per player
	players := make([]Player, 4)
	for i := range players {
		players[i] = Player{ID: int64(i + 1), DisplayName: fmt.Sprintf("Player %d", i+1)}
	}

	fixtures, err := GenerateRoundRobinFixtures(season, players)
	if err != nil {
		t.Fatalf("expected fixtures, got %v", err)
	}

	if len(fixtures) != 6 {
		t.Fatalf("expected 6 fixtures, got %d", len(fixtures))
	}

	weekCounts := map[int]int{}
	for _, f := range fixtures {
		weekCounts[f.WeekNumber]++
	}

	// With 4 players and 2 games/week: week 1 has 4 fixtures (2 per player), week 2 has 2 fixtures (1 per player)
	if weekCounts[1] != 4 {
		t.Fatalf("expected 4 fixtures in week 1, got %d", weekCounts[1])
	}
	if weekCounts[2] != 2 {
		t.Fatalf("expected 2 fixtures in week 2, got %d", weekCounts[2])
	}
}

func TestGenerateFixturesBlitzMode(t *testing.T) {
	t.Parallel()

	season := NewSeason("Blitz")
	season.ID = 1
	season.GamesPerWeek = 3 // all games in 1 week for 4 players
	season = season.Start(time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC))

	players := make([]Player, 4)
	for i := range players {
		players[i] = Player{ID: int64(i + 1), DisplayName: fmt.Sprintf("Player %d", i+1)}
	}

	fixtures, err := GenerateRoundRobinFixtures(season, players)
	if err != nil {
		t.Fatalf("expected fixtures, got %v", err)
	}

	maxWeek := 0
	for _, f := range fixtures {
		if f.WeekNumber > maxWeek {
			maxWeek = f.WeekNumber
		}
	}

	if maxWeek != 1 {
		t.Fatalf("expected all fixtures in 1 week for blitz mode, got %d weeks", maxWeek)
	}
}

func TestScheduledAtIsRevealTime(t *testing.T) {
	t.Parallel()

	season := NewSeason("Spring 2026")
	season.ID = 1
	season = season.Start(time.Date(2026, time.March, 10, 12, 0, 0, 0, time.UTC))
	players := []Player{
		{ID: 1, DisplayName: "Alice"},
		{ID: 2, DisplayName: "Bob"},
		{ID: 3, DisplayName: "Charlie"},
	}

	fixtures, err := GenerateRoundRobinFixtures(season, players)
	if err != nil {
		t.Fatalf("expected fixtures, got %v", err)
	}

	loc, _ := time.LoadLocation("Europe/London")
	for _, f := range fixtures {
		local := f.ScheduledAt.In(loc)
		if local.Hour() != 9 || local.Minute() != 0 {
			t.Fatalf("expected scheduled_at to be 09:00 (reveal time), got %s", local)
		}
	}
}

func TestCurrentPublicWeekUsesMondayNineUnlock(t *testing.T) {
	t.Parallel()

	loc, _ := time.LoadLocation("Europe/London")
	fixtures := []Fixture{
		{WeekNumber: 1, ScheduledAt: time.Date(2026, time.March, 23, 19, 30, 0, 0, loc)},
		{WeekNumber: 2, ScheduledAt: time.Date(2026, time.March, 30, 19, 30, 0, 0, loc)},
	}

	beforeUnlock := time.Date(2026, time.March, 30, 8, 59, 0, 0, loc)
	if got := CurrentPublicWeek(fixtures, beforeUnlock, loc); got != 1 {
		t.Fatalf("expected week 1 before unlock, got %d", got)
	}

	afterUnlock := time.Date(2026, time.March, 30, 9, 0, 0, 0, loc)
	if got := CurrentPublicWeek(fixtures, afterUnlock, loc); got != 2 {
		t.Fatalf("expected week 2 after unlock, got %d", got)
	}
}
