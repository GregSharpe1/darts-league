package league

import (
	"errors"
	"sort"
	"time"
)

var (
	ErrSeasonAlreadyStarted = errors.New("season already started")
	ErrNotEnoughPlayers     = errors.New("at least two players are required to start the season")
	ErrTimezoneRequired     = errors.New("timezone is required")
)

const (
	GameVariant501 = "501"
	LegsToWin      = 3
)

type Fixture struct {
	ID          int64
	SeasonID    int64
	WeekNumber  int
	ScheduledAt time.Time
	PlayerOneID int64
	PlayerTwoID int64
	GameVariant string
	LegsToWin   int
	Status      string
}

type WeeklyFixtures struct {
	WeekNumber int
	RevealAt   time.Time
	Fixtures   []Fixture
}

func GenerateRoundRobinFixtures(season Season, players []Player) ([]Fixture, error) {
	if season.RegistrationOpen() {
		return nil, ErrSeasonAlreadyStarted
	}

	if len(players) < 2 {
		return nil, ErrNotEnoughPlayers
	}

	ordered := make([]Player, len(players))
	copy(ordered, players)
	sort.Slice(ordered, func(i, j int) bool {
		return NormalizeDisplayName(ordered[i].DisplayName) < NormalizeDisplayName(ordered[j].DisplayName)
	})

	rotation := make([]Player, len(ordered))
	copy(rotation, ordered)
	if len(rotation)%2 == 1 {
		rotation = append(rotation, Player{})
	}

	loc, err := time.LoadLocation(season.Timezone)
	if err != nil {
		return nil, err
	}

	weeks := len(rotation) - 1
	half := len(rotation) / 2
	fixtures := make([]Fixture, 0, weeks*half)
	firstReveal, err := FirstWeekRevealAt(*season.StartedAt, loc)
	if err != nil {
		return nil, err
	}

	for round := 0; round < weeks; round++ {
		weekNumber := round + 1
		scheduledAt := WeekScheduleTime(firstReveal, round)

		for i := 0; i < half; i++ {
			left := rotation[i]
			right := rotation[len(rotation)-1-i]
			if left.ID == 0 || right.ID == 0 {
				continue
			}

			fixture := Fixture{
				SeasonID:    season.ID,
				WeekNumber:  weekNumber,
				ScheduledAt: scheduledAt,
				PlayerOneID: left.ID,
				PlayerTwoID: right.ID,
				GameVariant: GameVariant501,
				LegsToWin:   LegsToWin,
				Status:      "scheduled",
			}

			if round%2 == 1 && i == 0 {
				fixture.PlayerOneID = right.ID
				fixture.PlayerTwoID = left.ID
			}

			fixtures = append(fixtures, fixture)
		}

		rotation = rotatePlayers(rotation)
	}

	return fixtures, nil
}

func FirstWeekRevealAt(startedAt time.Time, loc *time.Location) (time.Time, error) {
	if loc == nil {
		return time.Time{}, ErrTimezoneRequired
	}

	local := startedAt.In(loc)
	reveal := time.Date(local.Year(), local.Month(), local.Day(), 9, 0, 0, 0, loc)
	for reveal.Weekday() != time.Monday || reveal.Before(local) {
		reveal = reveal.AddDate(0, 0, 1)
		reveal = time.Date(reveal.Year(), reveal.Month(), reveal.Day(), 9, 0, 0, 0, loc)
	}

	return reveal, nil
}

func WeekScheduleTime(firstReveal time.Time, weekOffset int) time.Time {
	reveal := firstReveal.In(firstReveal.Location()).AddDate(0, 0, 7*weekOffset)
	return time.Date(reveal.Year(), reveal.Month(), reveal.Day(), 19, 30, 0, 0, firstReveal.Location())
}

func CurrentPublicWeek(fixtures []Fixture, now time.Time, loc *time.Location) int {
	if len(fixtures) == 0 || loc == nil {
		return 0
	}

	currentWeek := 0
	localNow := now.In(loc)
	for _, fixture := range fixtures {
		revealAt := time.Date(fixture.ScheduledAt.In(loc).Year(), fixture.ScheduledAt.In(loc).Month(), fixture.ScheduledAt.In(loc).Day(), 9, 0, 0, 0, loc)
		if !localNow.Before(revealAt) && fixture.WeekNumber > currentWeek {
			currentWeek = fixture.WeekNumber
		}
	}

	return currentWeek
}

func GroupFixturesByWeek(fixtures []Fixture, loc *time.Location) []WeeklyFixtures {
	if len(fixtures) == 0 {
		return nil
	}

	weeks := map[int]*WeeklyFixtures{}
	order := make([]int, 0)
	for _, fixture := range fixtures {
		entry, ok := weeks[fixture.WeekNumber]
		if !ok {
			reveal := time.Date(fixture.ScheduledAt.In(loc).Year(), fixture.ScheduledAt.In(loc).Month(), fixture.ScheduledAt.In(loc).Day(), 9, 0, 0, 0, loc)
			entry = &WeeklyFixtures{WeekNumber: fixture.WeekNumber, RevealAt: reveal}
			weeks[fixture.WeekNumber] = entry
			order = append(order, fixture.WeekNumber)
		}
		entry.Fixtures = append(entry.Fixtures, fixture)
	}

	sort.Ints(order)
	result := make([]WeeklyFixtures, 0, len(order))
	for _, week := range order {
		entry := weeks[week]
		sort.Slice(entry.Fixtures, func(i, j int) bool {
			return entry.Fixtures[i].ScheduledAt.Before(entry.Fixtures[j].ScheduledAt)
		})
		result = append(result, *entry)
	}

	return result
}

func rotatePlayers(players []Player) []Player {
	if len(players) <= 2 {
		return players
	}

	rotated := make([]Player, len(players))
	rotated[0] = players[0]
	rotated[1] = players[len(players)-1]
	copy(rotated[2:], players[1:len(players)-1])
	return rotated
}
