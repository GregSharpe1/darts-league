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

// DefaultLegsToWin is the legacy default used when no season config is set.
const DefaultLegsToWin = 3

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

	gameVariant := season.GameVariant
	if gameVariant == "" {
		gameVariant = GameVariant501
	}
	legsToWin := season.LegsToWin
	if legsToWin < 1 {
		legsToWin = DefaultLegsToWin
	}
	gamesPerWeek := season.GamesPerWeek
	if gamesPerWeek < 1 {
		gamesPerWeek = 1
	}

	// Generate all round-robin pairings in order.
	rounds := len(rotation) - 1
	half := len(rotation) / 2
	allPairings := make([]Fixture, 0, rounds*half)

	for round := 0; round < rounds; round++ {
		for i := 0; i < half; i++ {
			left := rotation[i]
			right := rotation[len(rotation)-1-i]
			if left.ID == 0 || right.ID == 0 {
				continue
			}

			fixture := Fixture{
				SeasonID:    season.ID,
				PlayerOneID: left.ID,
				PlayerTwoID: right.ID,
				GameVariant: gameVariant,
				LegsToWin:   legsToWin,
				Status:      "scheduled",
			}

			if round%2 == 1 && i == 0 {
				fixture.PlayerOneID = right.ID
				fixture.PlayerTwoID = left.ID
			}

			allPairings = append(allPairings, fixture)
		}

		rotation = rotatePlayers(rotation)
	}

	// Pack pairings into weeks based on gamesPerWeek.
	// Each week can hold at most gamesPerWeek games per player.
	firstReveal, err := FirstWeekRevealAt(*season.StartedAt, loc)
	if err != nil {
		return nil, err
	}

	fixtures := packPairingsIntoWeeks(allPairings, gamesPerWeek, season.ID, firstReveal)
	return fixtures, nil
}

// packPairingsIntoWeeks assigns fixtures to weeks such that no player
// appears more than gamesPerWeek times in a single week.
func packPairingsIntoWeeks(pairings []Fixture, gamesPerWeek int, seasonID int64, firstReveal time.Time) []Fixture {
	if len(pairings) == 0 {
		return nil
	}

	assigned := make([]bool, len(pairings))
	result := make([]Fixture, 0, len(pairings))
	weekNumber := 0
	totalAssigned := 0

	for totalAssigned < len(pairings) {
		weekNumber++
		revealAt := WeekRevealTime(firstReveal, weekNumber-1)
		playerGamesThisWeek := map[int64]int{}

		for i, pairing := range pairings {
			if assigned[i] {
				continue
			}
			p1, p2 := pairing.PlayerOneID, pairing.PlayerTwoID
			if playerGamesThisWeek[p1] >= gamesPerWeek || playerGamesThisWeek[p2] >= gamesPerWeek {
				continue
			}

			fixture := pairing
			fixture.WeekNumber = weekNumber
			fixture.ScheduledAt = revealAt
			result = append(result, fixture)
			assigned[i] = true
			totalAssigned++
			playerGamesThisWeek[p1]++
			playerGamesThisWeek[p2]++
		}
	}

	return result
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

// WeekScheduleTime returns the match time for a week (legacy: 19:30 local).
// Kept for backward compatibility with existing fixture display.
func WeekScheduleTime(firstReveal time.Time, weekOffset int) time.Time {
	reveal := firstReveal.In(firstReveal.Location()).AddDate(0, 0, 7*weekOffset)
	return time.Date(reveal.Year(), reveal.Month(), reveal.Day(), 19, 30, 0, 0, firstReveal.Location())
}

// WeekRevealTime returns the Monday 09:00 reveal time for a given week offset.
func WeekRevealTime(firstReveal time.Time, weekOffset int) time.Time {
	return firstReveal.In(firstReveal.Location()).AddDate(0, 0, 7*weekOffset)
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
