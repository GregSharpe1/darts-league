package tournament

import (
	"testing"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

func TestImportedBracketAdvancesEntrantIDsWithSharedPublicNames(t *testing.T) {
	entries := []Entry{
		{ID: 1, DisplayName: "Alice", Nickname: "Flight"},
		{ID: 2, DisplayName: "Bob", Nickname: "Flight"},
		{ID: 3, DisplayName: "Casey"},
	}
	event := Event{Slug: "cup", MatchFormat: DefaultMatchFormat(), Bracket: buildBracket(entries)}
	for i, names := range [][2]string{{"Casey", "Bob"}, {"Bob", "Alice"}} {
		players := []autodarts.Player{
			{ID: "a", DisplayName: names[0], LegsWon: 1},
			{ID: "b", DisplayName: names[1], LegsWon: 3},
		}
		imported := autodarts.Import{Players: players}
		candidates := event.ImportCandidates(entries, imported)
		if len(candidates) != 1 || len(candidates[0].Mappings) != 1 {
			t.Fatalf("match %d candidates: %+v", i, candidates)
		}
		candidate := candidates[0]
		if i == 1 {
			aliases := []autodarts.Player{{ID: "a", DisplayName: "Flight", LegsWon: 1}, {ID: "b", DisplayName: "Flight", LegsWon: 3}}
			choices := event.ImportCandidates(entries, autodarts.Import{Players: aliases})
			if len(choices) != 1 || len(choices[0].Mappings) != 2 {
				t.Fatalf("shared names should require an explicit entrant mapping: %+v", choices)
			}
		}
		result, err := event.RecordImportedMatch(candidate.MatchNumber, players, candidate.Mappings[0])
		if err != nil || result.WinnerID != int64(2-i) {
			t.Fatalf("match %d result: %+v, %v", i, result, err)
		}
		if candidates := event.ImportCandidates(entries, imported); len(candidates) != 0 {
			t.Fatalf("scored match remains eligible: %+v", candidates)
		}
	}
	final := event.Bracket[1][0]
	if final.WinnerID != entries[0].ID || final.Slots[0].Player != "Flight" || final.Slots[1].Player != "Flight" ||
		final.Slots[1].WinnerOf == 0 || final.Slots[0].EntryID == final.Slots[1].EntryID {
		t.Fatalf("final lost entrant identities or draw: %+v", final)
	}
}

func TestLegacyBracketIdentityResolutionPreservesDraw(t *testing.T) {
	entries := []Entry{{ID: 5, DisplayName: "Alice", Nickname: "Ace"}, {ID: 7, DisplayName: "Bob"}}
	bracket := Bracket{{{Number: 3, Slots: [2]BracketSlot{{Player: "Bob"}, {Player: "Ace"}}}}}
	bracket.ResolvePublishedEntries(entries)
	if match := bracket[0][0]; match.Number != 3 || match.Slots != [2]BracketSlot{{Player: "Bob", EntryID: 7}, {Player: "Ace", EntryID: 5}} {
		t.Fatal(match)
	}
	entries[1].Nickname = "Ace"
	bracket[0][0].Slots = [2]BracketSlot{{Player: "Ace"}, {Player: "Ace"}}
	bracket.ResolvePublishedEntries(entries)
	if match := bracket[0][0]; match.ready() || match.Slots[0].Player != "Ace" || match.Slots[1].Player != "Ace" {
		t.Fatal(match)
	}
}
