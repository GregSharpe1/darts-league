package playerstats

import (
	"context"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/autodarts"
	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/tournament"
)

type tournamentWithRecord struct {
	*tournament.MemoryStore
	source league.ImportRecord
}

func (s tournamentWithRecord) TournamentRecordImport(_ context.Context, _ string, _ int) (league.ImportRecord, error) {
	return s.source, nil
}

func TestRecordsCombineLinkedEntriesAndRespectReveals(t *testing.T) {
	ctx := context.Background()
	store := league.NewMemoryStore()
	events := tournamentWithRecord{MemoryStore: tournament.NewMemoryStore()}
	now := time.Date(2026, 4, 6, 8, 0, 0, 0, time.UTC)
	service := Service{League: store, Tournament: events, Now: func() time.Time { return now }}
	service.Identities = &MemoryIdentities{Source: service.IdentityEntries}

	season, err := store.GetActiveSeason(ctx)
	season = must(t, season, err)
	divisions, err := store.ReplaceDivisions(ctx, season.ID, []league.Division{{Name: "Premier", Slug: "premier"}})
	division := must(t, divisions, err)[0]
	createPlayer := func(name string, identityID int64) league.Player {
		player, err := store.CreatePlayer(ctx, league.Player{
			SeasonID: season.ID, DisplayName: name, DivisionID: &division.ID, Status: league.PlayerStatusAssigned, ReturningIdentityID: identityID,
		})
		return must(t, player, err)
	}
	player, opponent, third := createPlayer("Alex Morgan", 0), createPlayer("Sam Patel", 0), createPlayer("Casey Brooks", 0)
	started, err := store.UpsertSeason(ctx, season.Start(now))
	season = must(t, started, err)
	fixtures, err := store.CreateFixtures(ctx, []league.Fixture{
		{SeasonID: season.ID, DivisionID: division.ID, PlayerOneID: player.ID, PlayerTwoID: opponent.ID, WeekNumber: 1, ScheduledAt: now.Add(10 * time.Hour)},
		{SeasonID: season.ID, DivisionID: division.ID, PlayerOneID: player.ID, PlayerTwoID: third.ID, WeekNumber: 2, ScheduledAt: now.Add(7*24*time.Hour + 10*time.Hour)},
	})
	fixtures = must(t, fixtures, err)
	average, hidden := 87.25, 125.0
	result, err := store.CreateResult(ctx, league.Result{FixtureID: fixtures[0].ID, WinnerID: player.ID, PlayerOneLegs: 3, PlayerTwoLegs: 1, PlayerOneAverage: &average})
	must(t, result, err)
	result, err = store.CreateResult(ctx, league.Result{FixtureID: fixtures[1].ID, WinnerID: player.ID, PlayerOneLegs: 3, PlayerTwoLegs: 1, PlayerOneAverage: &hidden})
	must(t, result, err)
	snapshot, err := service.Identities.IdentitySnapshot(ctx)
	identity := must(t, snapshot, err).index()[entryKey{"league", player.ID}]

	event, err := events.CreateTournament(ctx, tournament.Event{Slug: "cup", Name: "City Cup", RegistrationOpen: true, MatchFormat: tournament.DefaultMatchFormat()})
	event = must(t, event, err)
	entrant, err := events.CreateTournamentEntry(ctx, event.Slug, tournament.Entry{DisplayName: "Arrow", ReturningIdentityID: identity})
	entrant = must(t, entrant, err)
	guest, err := events.CreateTournamentEntry(ctx, event.Slug, tournament.Entry{DisplayName: "Guest player"})
	guest = must(t, guest, err)
	event, err = events.SetTournamentRegistration(ctx, event.Slug, false)
	event = must(t, event, err)
	event, err = events.StartTournament(ctx, event.Slug)
	event = must(t, event, err)
	tournamentAverage, maxima := 91.5, 2
	sourceID := "a"
	score := [2]int{3, 0}
	if event.Bracket[0][0].Slots[0].EntryID != entrant.ID {
		score = [2]int{0, 3}
	}
	match := &event.Bracket[0][0]
	match.Legs, match.WinnerID = &score, entrant.ID
	events.source = league.ImportRecord{
		Active: true, Pending: league.PendingResult{Status: league.PendingResultStatusConfirmed},
		Approval: &league.ImportAudit{TournamentSlug: event.Slug, MatchNumber: match.Number},
		Mapping:  map[string]int64{"a": entrant.ID, "b": guest.ID},
		Import: autodarts.Import{
			Players: []autodarts.Player{
				{ID: "a", LegsWon: 3, Stats: &autodarts.Stats{MatchAverage: &tournamentAverage, Total180: &maxima}},
				{ID: "b", LegsWon: 0},
			},
			Detail: &autodarts.Detail{Coverage: "partial", Legs: []autodarts.Leg{{Completed: true, WinnerID: &sourceID,
				Visits: []autodarts.Visit{{PlayerID: "a", StartRemaining: 50, EndRemaining: 0,
					Throws: []autodarts.Throw{{Segment: autodarts.Segment{Bed: "inner_bull", Number: 25}}}}},
			}}},
		},
	}
	service.Tournament = events

	records, err := service.Records(ctx)
	records = must(t, records, err)
	for _, metric := range []struct {
		key   string
		value float64
		known int
	}{{"wins", 2, 2}, {"total_180", 2, 1}, {"highest_finish", 50, 1}, {"bullseyes", 1, 1}, {"match_average", 91.5, 2}} {
		rows := records[metric.key]
		if len(rows) != 1 || rows[0].Profile.IdentityID != identity || rows[0].Value != metric.value || rows[0].KnownMatches != metric.known || rows[0].Played != 2 {
			t.Errorf("%s: %+v", metric.key, rows)
		}
	}
	if len(records["best_leg"]) != 0 {
		t.Errorf("partial legs must not establish a fastest leg: %+v", records["best_leg"])
	}

	events.source.Active = false
	service.Tournament = events
	now = now.Add(7 * 24 * time.Hour)
	records, err = service.Records(ctx)
	records = must(t, records, err)
	if wins, average := records["wins"], records["match_average"]; len(wins) != 1 || wins[0].Value != 3 ||
		len(average) != 1 || average[0].Value != 125 || len(records["total_180"]) != 0 || len(records["bullseyes"]) != 0 {
		t.Fatalf("revealed league records and detached tournament evidence: %+v", records)
	}
}

func TestCompleteWinningLegProvidesFastestLegAndMaximumFallback(t *testing.T) {
	var player recordAccumulator
	dart := func(bed string, number int) autodarts.Throw {
		return autodarts.Throw{Segment: autodarts.Segment{Bed: bed, Number: number}}
	}
	id := "a"
	player.addRecordedDarts(autodarts.Player{ID: id}, &autodarts.Detail{Coverage: "complete", Legs: []autodarts.Leg{{
		Completed: true, WinnerID: &id,
		Visits: []autodarts.Visit{
			{PlayerID: id, StartRemaining: 501, EndRemaining: 321, Throws: []autodarts.Throw{dart("triple", 20), dart("triple", 20), dart("triple", 20)}},
			{PlayerID: id, StartRemaining: 321, EndRemaining: 170, Throws: []autodarts.Throw{dart("triple", 20), dart("triple", 17), dart("double", 20)}},
			{PlayerID: id, StartRemaining: 170, EndRemaining: 0, Throws: []autodarts.Throw{dart("triple", 20), dart("triple", 20), dart("inner_bull", 25)}},
		},
	}}})
	if player.leg.value != 9 || player.maximums.value != 1 || player.checkouts.value != 170 || player.bullseyes.value != 1 {
		t.Fatalf("nine-darter records: %+v", player)
	}
}
