package league

import (
	"context"
	"testing"
	"time"
)

func TestValidateResultScoreAcceptsOnlyFirstToThree(t *testing.T) {
	t.Parallel()

	valid := [][2]int{{3, 0}, {3, 1}, {3, 2}, {0, 3}, {1, 3}, {2, 3}}
	for _, score := range valid {
		if err := ValidateResultScore(score[0], score[1], 3); err != nil {
			t.Fatalf("expected score %v to be valid, got %v", score, err)
		}
	}

	invalid := [][2]int{{3, 3}, {2, 2}, {4, 0}, {0, 0}, {1, 2}}
	for _, score := range invalid {
		if err := ValidateResultScore(score[0], score[1], 3); err == nil {
			t.Fatalf("expected score %v to be invalid", score)
		}
	}
}

func TestValidateResultScoreWithVariableLegsToWin(t *testing.T) {
	t.Parallel()

	// First to 1: only 1-0 or 0-1 valid
	if err := ValidateResultScore(1, 0, 1); err != nil {
		t.Fatalf("expected 1-0 to be valid for first-to-1, got %v", err)
	}
	if err := ValidateResultScore(0, 1, 1); err != nil {
		t.Fatalf("expected 0-1 to be valid for first-to-1, got %v", err)
	}
	if err := ValidateResultScore(1, 1, 1); err == nil {
		t.Fatal("expected 1-1 to be invalid for first-to-1")
	}

	// First to 5: valid scores include 5-0 through 5-4
	for loser := 0; loser < 5; loser++ {
		if err := ValidateResultScore(5, loser, 5); err != nil {
			t.Fatalf("expected 5-%d to be valid for first-to-5, got %v", loser, err)
		}
		if err := ValidateResultScore(loser, 5, 5); err != nil {
			t.Fatalf("expected %d-5 to be valid for first-to-5, got %v", loser, err)
		}
	}
	if err := ValidateResultScore(5, 5, 5); err == nil {
		t.Fatal("expected 5-5 to be invalid for first-to-5")
	}
	if err := ValidateResultScore(4, 3, 5); err == nil {
		t.Fatal("expected 4-3 to be invalid for first-to-5")
	}
}

func TestBuildStandingsOrdersByPointsLegDifferenceLegsForThenAlphabetical(t *testing.T) {
	t.Parallel()

	players := []Player{
		{ID: 1, DisplayName: "Luke Humphries", Nickname: "The Freeze"},
		{ID: 2, DisplayName: "Michael Smith", Nickname: "Bully Boy"},
		{ID: 3, DisplayName: "Gerwyn Price", Nickname: "The Iceman"},
	}
	fixtures := []Fixture{
		{ID: 1, PlayerOneID: 1, PlayerTwoID: 2},
		{ID: 2, PlayerOneID: 2, PlayerTwoID: 3},
		{ID: 3, PlayerOneID: 3, PlayerTwoID: 1},
	}
	results := []Result{
		{FixtureID: 1, PlayerOneLegs: 3, PlayerTwoLegs: 2, WinnerID: 1},
		{FixtureID: 2, PlayerOneLegs: 3, PlayerTwoLegs: 1, WinnerID: 2},
		{FixtureID: 3, PlayerOneLegs: 1, PlayerTwoLegs: 3, WinnerID: 1},
	}

	standings := BuildStandings(players, fixtures, results)
	if len(standings) != 3 {
		t.Fatalf("expected 3 standings rows, got %d", len(standings))
	}
	if standings[0].PreferredName != "The Freeze" {
		t.Fatalf("expected The Freeze top, got %q", standings[0].PreferredName)
	}
	if standings[0].Points != 4 || standings[0].LegsFor != 6 || standings[0].LegsAgainst != 3 {
		t.Fatalf("unexpected top row stats: %+v", standings[0])
	}
	if standings[1].PreferredName != "Bully Boy" {
		t.Fatalf("expected Bully Boy second, got %q", standings[1].PreferredName)
	}
	if standings[2].Lost != 2 {
		t.Fatalf("expected last row to have 2 losses, got %+v", standings[2])
	}
}

func TestEditResultUpdatesStandingsAndWritesAuditLog(t *testing.T) {
	t.Parallel()

	store := NewMemoryStore()
	now := time.Date(2026, time.March, 18, 12, 0, 0, 0, time.UTC)
	registration := NewRegistrationServiceWithNow(store, func() time.Time { return now })
	seasonService := NewSeasonServiceWithNow(store, func() time.Time { return now })
	resultService := NewResultServiceWithNow(store, func() time.Time { return now })
	ctx := context.Background()

	for _, player := range []Player{{DisplayName: "Luke Humphries", Nickname: "The Freeze"}, {DisplayName: "Michael Smith", Nickname: "Bully Boy"}} {
		if _, err := registration.RegisterPlayer(ctx, player); err != nil {
			t.Fatalf("expected player registration to succeed, got %v", err)
		}
	}
	if _, err := seasonService.StartSeason(ctx); err != nil {
		t.Fatalf("expected season start to succeed, got %v", err)
	}
	playerOneAverage := 96.4
	playerTwoAverage := 84.1
	if _, err := resultService.RecordResult(ctx, 1, 3, 0, &playerOneAverage, &playerTwoAverage); err != nil {
		t.Fatalf("expected initial result to succeed, got %v", err)
	}

	now = now.Add(time.Hour)
	updatedPlayerOneAverage := 98.2
	updatedPlayerTwoAverage := 91.7
	if _, err := resultService.EditResult(ctx, 1, 3, 2, &updatedPlayerOneAverage, &updatedPlayerTwoAverage, "admin"); err != nil {
		t.Fatalf("expected result edit to succeed, got %v", err)
	}

	standings, err := resultService.Standings(ctx)
	if err != nil {
		t.Fatalf("expected standings, got %v", err)
	}
	if standings[0].LegsAgainst != 2 {
		t.Fatalf("expected edited result to affect standings, got %+v", standings[0])
	}
	audit, err := resultService.AuditLog(ctx)
	if err != nil {
		t.Fatalf("expected audit log, got %v", err)
	}
	if len(audit) != 1 || audit[0].OldResult.PlayerTwoLegs != 0 || audit[0].NewResult.PlayerTwoLegs != 2 {
		t.Fatalf("expected one audit entry with before/after result, got %+v", audit)
	}
	if audit[0].NewResult.PlayerOneAverage == nil || *audit[0].NewResult.PlayerOneAverage != updatedPlayerOneAverage {
		t.Fatalf("expected updated average in audit entry, got %+v", audit[0])
	}
}
