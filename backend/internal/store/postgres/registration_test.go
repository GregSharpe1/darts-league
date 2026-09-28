package postgres

import (
	"context"
	"os"
	"testing"

	"github.com/greg/darts-league/backend/internal/league"
)

func TestRegistrationTreatsSQLAsData(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	store, err := Open(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	resetTables(t, ctx, store)
	if _, err := store.EnsureActiveSeason(ctx, league.NewSeason("Registration")); err != nil {
		t.Fatal(err)
	}
	service := league.NewRegistrationService(store)
	want := []league.Player{
		{DisplayName: "Existing"},
		{DisplayName: "Robert'); DROP TABLE players;--", Nickname: "DROP DATABASE darts"},
		{DisplayName: "DROP DATABASE darts", Nickname: "R'); DROP TABLE players;--"},
		{DisplayName: "O'Brien"},
	}
	for _, player := range want {
		if _, err := service.RegisterPlayer(ctx, player); err != nil {
			t.Fatal(err)
		}
	}
	players, err := service.ListPlayers(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(players) != len(want) {
		t.Fatalf("got %d players", len(players))
	}
	for _, expected := range want {
		found := false
		for _, actual := range players {
			if actual.DisplayName == expected.DisplayName && actual.Nickname == expected.Nickname {
				found = true
			}
		}
		if !found {
			t.Fatalf("missing unchanged player: %+v", expected)
		}
	}
}
