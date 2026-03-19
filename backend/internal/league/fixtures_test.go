package league

import (
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
	if fixtures[0].GameVariant != GameVariant501 || fixtures[0].LegsToWin != LegsToWin {
		t.Fatalf("expected fixed match format, got %s first-to-%d", fixtures[0].GameVariant, fixtures[0].LegsToWin)
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
