package tournament

import "testing"

func TestBuildBracket(t *testing.T) {
	for count := 2; count <= 32; count++ {
		entries := make([]Entry, count)
		for i := range entries {
			entries[i].DisplayName = string(rune('A' + i))
		}
		entries[0].Nickname = "Flight"

		bracket := buildBracket(entries)
		players := map[string]int{}
		byes := 0
		for _, match := range bracket[0] {
			for _, slot := range match.Slots {
				if slot.Bye {
					byes++
				} else {
					players[slot.Player]++
				}
			}
			if match.Slots[0].Bye && match.Slots[1].Bye {
				t.Fatalf("%d entrants: drew two byes together", count)
			}
		}
		if len(players) != count || players["Flight"] != 1 || byes != len(bracket[0])*2-count {
			t.Fatalf("%d entrants: got %v players and %d byes", count, players, byes)
		}
		for i, round := range bracket[1:] {
			for j, match := range round {
				for k, slot := range match.Slots {
					source := bracket[i][2*j+k]
					if source.Slots[0].Bye || source.Slots[1].Bye {
						if slot.Player == "" || slot.Player != source.Slots[0].Player+source.Slots[1].Player {
							t.Fatalf("%d entrants: expected bye player to advance, got %+v", count, slot)
						}
					} else if slot.WinnerOf != source.Number {
						t.Fatalf("%d entrants: expected match %d to feed match %d", count, source.Number, match.Number)
					}
				}
			}
		}
		if len(bracket[len(bracket)-1]) != 1 {
			t.Fatalf("%d entrants: expected a single final", count)
		}
	}
}
