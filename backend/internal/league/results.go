package league

import (
	"errors"
	"sort"
	"strings"
	"time"
)

var (
	ErrFixtureNotFound     = errors.New("fixture not found")
	ErrInvalidResult       = errors.New("invalid result")
	ErrResultAlreadyExists = errors.New("result already exists")
	ErrAuditLogNotFound    = errors.New("audit log not found")
)

type Result struct {
	ID               int64
	FixtureID        int64
	PlayerOneLegs    int
	PlayerTwoLegs    int
	PlayerOneAverage *float64
	PlayerTwoAverage *float64
	WinnerID         int64
	EnteredAt        time.Time
	UpdatedAt        time.Time
}

type StandingRow struct {
	PlayerID      int64
	DisplayName   string
	PreferredName string
	Played        int
	Won           int
	Lost          int
	LegsFor       int
	LegsAgainst   int
	LegDifference int
	Points        int
}

type AuditLogEntry struct {
	ID           int64
	FixtureID    int64
	FixtureLabel string
	Action       string
	Actor        string
	OldResult    *ResultSnapshot
	NewResult    *ResultSnapshot
	CreatedAt    time.Time
}

type ResultSnapshot struct {
	PlayerOneLegs    int      `json:"player_one_legs"`
	PlayerTwoLegs    int      `json:"player_two_legs"`
	PlayerOneAverage *float64 `json:"player_one_average,omitempty"`
	PlayerTwoAverage *float64 `json:"player_two_average,omitempty"`
	WinnerID         int64    `json:"winner_id"`
}

func SnapshotFromResult(result Result) *ResultSnapshot {
	return &ResultSnapshot{
		PlayerOneLegs:    result.PlayerOneLegs,
		PlayerTwoLegs:    result.PlayerTwoLegs,
		PlayerOneAverage: result.PlayerOneAverage,
		PlayerTwoAverage: result.PlayerTwoAverage,
		WinnerID:         result.WinnerID,
	}
}

func ValidateResultScore(playerOneLegs, playerTwoLegs, legsToWin int) error {
	if legsToWin < 1 {
		legsToWin = DefaultLegsToWin
	}
	valid := (playerOneLegs == legsToWin && playerTwoLegs >= 0 && playerTwoLegs < legsToWin) ||
		(playerTwoLegs == legsToWin && playerOneLegs >= 0 && playerOneLegs < legsToWin)
	if !valid {
		return ErrInvalidResult
	}
	return nil
}

func WinnerIDForFixture(fixture Fixture, playerOneLegs, playerTwoLegs int) (int64, error) {
	if err := ValidateResultScore(playerOneLegs, playerTwoLegs, fixture.LegsToWin); err != nil {
		return 0, err
	}
	if playerOneLegs > playerTwoLegs {
		return fixture.PlayerOneID, nil
	}
	return fixture.PlayerTwoID, nil
}

func BuildStandings(players []Player, fixtures []Fixture, results []Result) []StandingRow {
	rowsByID := make(map[int64]*StandingRow, len(players))
	fixtureByID := make(map[int64]Fixture, len(fixtures))
	for _, player := range players {
		rowsByID[player.ID] = &StandingRow{
			PlayerID:      player.ID,
			DisplayName:   player.DisplayName,
			PreferredName: player.PreferredName(),
		}
	}
	for _, fixture := range fixtures {
		fixtureByID[fixture.ID] = fixture
	}

	for _, result := range results {
		fixture, ok := fixtureByID[result.FixtureID]
		if !ok {
			continue
		}
		playerOne := rowsByID[fixture.PlayerOneID]
		playerTwo := rowsByID[fixture.PlayerTwoID]
		if playerOne == nil || playerTwo == nil {
			continue
		}

		playerOne.Played++
		playerTwo.Played++
		playerOne.LegsFor += result.PlayerOneLegs
		playerOne.LegsAgainst += result.PlayerTwoLegs
		playerTwo.LegsFor += result.PlayerTwoLegs
		playerTwo.LegsAgainst += result.PlayerOneLegs

		if result.WinnerID == fixture.PlayerOneID {
			playerOne.Won++
			playerOne.Points += 2
			playerTwo.Lost++
		} else {
			playerTwo.Won++
			playerTwo.Points += 2
			playerOne.Lost++
		}
	}

	rows := make([]StandingRow, 0, len(rowsByID))
	for _, row := range rowsByID {
		row.LegDifference = row.LegsFor - row.LegsAgainst
		rows = append(rows, *row)
	}

	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Points != rows[j].Points {
			return rows[i].Points > rows[j].Points
		}
		if rows[i].LegDifference != rows[j].LegDifference {
			return rows[i].LegDifference > rows[j].LegDifference
		}
		if rows[i].LegsFor != rows[j].LegsFor {
			return rows[i].LegsFor > rows[j].LegsFor
		}
		return strings.ToLower(rows[i].DisplayName) < strings.ToLower(rows[j].DisplayName)
	})

	return rows
}
