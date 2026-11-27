package playerstats

import (
	"context"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/tournament"
)

func must[T any](t *testing.T, value T, err error) T {
	t.Helper()
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func TestHistoricalLeaguePlacementAndTournamentIdentity(t *testing.T) {
	ctx := context.Background()
	store := league.NewMemoryStore()
	events := tournament.NewMemoryStore()
	now := time.Date(2026, 4, 6, 8, 0, 0, 0, time.UTC) // Monday at 09:00 UK time.
	service := Service{League: store, Tournament: events, Now: func() time.Time { return now }}

	season, err := store.GetActiveSeason(ctx)
	season = must(t, season, err)
	divisions, err := store.ReplaceDivisions(ctx, season.ID, []league.Division{{Name: "Premier", Slug: "premier"}})
	divisionID := must(t, divisions, err)[0].ID
	player, err := store.CreatePlayer(ctx, league.Player{SeasonID: season.ID, DivisionID: &divisionID, Status: league.PlayerStatusAssigned, DisplayName: "Alex Morgan", Nickname: "Arrow"})
	player = must(t, player, err)
	opponent, err := store.CreatePlayer(ctx, league.Player{SeasonID: season.ID, DivisionID: &divisionID, Status: league.PlayerStatusAssigned, DisplayName: "Sam Patel"})
	opponent = must(t, opponent, err)
	_, err = store.UpsertSeason(ctx, season.Start(now))
	if err != nil {
		t.Fatal(err)
	}
	fixtures, err := store.CreateFixtures(ctx, []league.Fixture{{SeasonID: season.ID, DivisionID: divisionID, PlayerOneID: player.ID, PlayerTwoID: opponent.ID, WeekNumber: 1, ScheduledAt: now.Add(10 * time.Hour)}})
	fixture := must(t, fixtures, err)[0]
	result, err := store.CreateResult(ctx, league.Result{FixtureID: fixture.ID, WinnerID: opponent.ID, PlayerOneLegs: 1, PlayerTwoLegs: 3})
	must(t, result, err)

	_, err = events.CreateTournament(ctx, tournament.Event{Slug: "cup", Name: "City Cup", RegistrationOpen: true, MatchFormat: tournament.DefaultMatchFormat()})
	if err != nil {
		t.Fatal(err)
	}
	entry, err := events.CreateTournamentEntry(ctx, "cup", tournament.Entry{DisplayName: "alex  MORGAN", Nickname: "Different nickname"})
	entry = must(t, entry, err)
	other, err := events.CreateTournamentEntry(ctx, "cup", tournament.Entry{DisplayName: "Taylor Evans", Nickname: "Arrow"})
	other = must(t, other, err)
	_, err = events.SetTournamentRegistration(ctx, "cup", false)
	if err != nil {
		t.Fatal(err)
	}
	event, err := events.StartTournament(ctx, "cup")
	event = must(t, event, err)
	// Record a result in the drawn final. Read the entrants in draw order.
	final := &event.Bracket[0][0]
	legs := [2]int{0, 3}
	if final.Slots[0].EntryID == entry.ID {
		legs = [2]int{3, 0}
	}
	final.Legs, final.WinnerID = &legs, entry.ID

	got, err := service.ForLeaguePlayer(ctx, season.ID, player.ID)
	if err != nil || got.League != nil || got.Tournament == nil || got.Tournament.Position != 1 {
		t.Fatalf("active league and completed tournament: %+v, %v", got, err)
	}
	if err := store.CloseSeason(ctx, season.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateNextSeason(ctx, season.ID, league.NewSeason("Next Season")); err != nil {
		t.Fatal(err)
	}
	now = now.Add(-time.Second)
	got, err = service.ForLeaguePlayer(ctx, season.ID, player.ID)
	if err != nil || got.League != nil {
		t.Fatalf("locked result exposed: %+v, %v", got.League, err)
	}
	now = now.Add(time.Second)
	got, err = service.ForLeaguePlayer(ctx, season.ID, player.ID)
	if err != nil || got.League == nil || got.League.Position != 2 || got.League.Division != "Premier" {
		t.Fatalf("historical league placement: %+v, %v", got.League, err)
	}
	tournamentOnly, err := service.ForTournamentPlayer(ctx, "cup", other.ID)
	if err != nil || tournamentOnly.League != nil || tournamentOnly.Tournament == nil || tournamentOnly.Tournament.Position != 2 {
		t.Fatalf("shared nickname must not join distinct entrants: %+v, %v", tournamentOnly, err)
	}
	_, err = events.CreateTournament(ctx, tournament.Event{Slug: "later-cup", Name: "Later Cup", StartsOn: "2026-05-01", RegistrationOpen: true, MatchFormat: tournament.DefaultMatchFormat()})
	if err != nil {
		t.Fatal(err)
	}
	repeat, err := events.CreateTournamentEntry(ctx, "later-cup", tournament.Entry{DisplayName: "taylor EVANS", Nickname: "New nickname"})
	repeat = must(t, repeat, err)
	guest, err := events.CreateTournamentEntry(ctx, "later-cup", tournament.Entry{DisplayName: "New player"})
	must(t, guest, err)
	event, err = events.SetTournamentRegistration(ctx, "later-cup", false)
	must(t, event, err)
	event, err = events.StartTournament(ctx, "later-cup")
	must(t, event, err)

	directory, err := service.Directory(ctx)
	directory = must(t, directory, err)
	if len(directory) != 4 || directory[2].EntryID != repeat.ID || directory[2].TournamentSlug != "later-cup" {
		t.Fatalf("directory: %+v", directory)
	}
}

func TestTournamentFinishUsesRecordedEliminationsAndByes(t *testing.T) {
	scored := [2]int{3, 0}
	bracket := tournament.Bracket{
		{
			{Slots: [2]tournament.BracketSlot{{EntryID: 1}, {EntryID: 2}}, Legs: &scored, WinnerID: 1},
			{Slots: [2]tournament.BracketSlot{{EntryID: 3}, {Bye: true}}},
		},
		{{Slots: [2]tournament.BracketSlot{{EntryID: 1}, {EntryID: 3}}}},
	}
	for _, test := range []struct {
		entryID  int64
		position int
		tied     bool
	}{{1, 0, false}, {2, 3, false}, {3, 0, false}} {
		position, tied := tournamentFinish(bracket, test.entryID)
		if position != test.position || tied != test.tied {
			t.Errorf("entrant %d got position %d (tied %t)", test.entryID, position, tied)
		}
	}
	bracket[0][1].Slots[1] = tournament.BracketSlot{EntryID: 4}
	if position, tied := tournamentFinish(bracket, 2); position != 3 || !tied {
		t.Errorf("four-player semifinal elimination got position %d (tied %t)", position, tied)
	}
}
