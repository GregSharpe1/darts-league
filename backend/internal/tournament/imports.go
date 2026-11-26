package tournament

import (
	"fmt"
	"strings"

	"github.com/greg/darts-league/backend/internal/autodarts"
	"github.com/greg/darts-league/backend/internal/league"
)

// ResolvePublishedEntries restores identities when an older draw's stored labels
// uniquely identify entrants. The original draw and its labels stay in place.
func (b Bracket) ResolvePublishedEntries(entries []Entry) {
	ids := make(map[string]int64, len(entries))
	for _, entry := range entries {
		label := entry.preferredName()
		if _, exists := ids[label]; exists {
			ids[label] = 0
		} else {
			ids[label] = entry.ID
		}
	}
	for _, round := range b {
		for i := range round {
			for j, slot := range round[i].Slots {
				if slot.EntryID == 0 && slot.WinnerOf == 0 {
					round[i].Slots[j].EntryID = ids[slot.Player]
				}
			}
		}
	}
}

func (m BracketMatch) ready() bool {
	return m.Legs == nil && m.Slots[0].EntryID > 0 && m.Slots[1].EntryID > 0 &&
		m.Slots[0].EntryID != m.Slots[1].EntryID && !m.Slots[0].Bye && !m.Slots[1].Bye
}

func (e Event) ImportCandidates(entries []Entry, imported autodarts.Import) []league.TournamentImportCandidate {
	if e.GameVariant != league.GameVariant501 || (e.LegsToWin != 2 && e.LegsToWin != 3) || len(imported.Players) != 2 {
		return nil
	}
	a, b := imported.Players[0], imported.Players[1]
	if league.ValidateResultScore(a.LegsWon, b.LegsWon, int(e.LegsToWin)) != nil {
		return nil
	}
	byID := make(map[int64]Entry, len(entries))
	for _, entry := range entries {
		byID[entry.ID] = entry
	}
	matches := func(entry Entry, player autodarts.Player) bool {
		name := strings.TrimSpace(player.DisplayName)
		return name != "" && (entry.DisplayName == name || entry.Nickname == name)
	}
	adminLabel := func(entry Entry) string {
		if entry.Nickname != "" && entry.Nickname != entry.DisplayName {
			return fmt.Sprintf("%s (%s)", entry.DisplayName, entry.Nickname)
		}
		return entry.DisplayName
	}
	var candidates []league.TournamentImportCandidate
	for _, round := range e.Bracket {
		for _, match := range round {
			if !match.ready() {
				continue
			}
			left, right := byID[match.Slots[0].EntryID], byID[match.Slots[1].EntryID]
			candidate := league.TournamentImportCandidate{
				TournamentSlug: e.Slug, TournamentName: e.Name, MatchNumber: match.Number,
				Players: [2]string{adminLabel(left), adminLabel(right)},
			}
			if matches(left, a) && matches(right, b) {
				candidate.Mappings = append(candidate.Mappings, map[string]int64{a.ID: left.ID, b.ID: right.ID})
			}
			if matches(right, a) && matches(left, b) {
				candidate.Mappings = append(candidate.Mappings, map[string]int64{a.ID: right.ID, b.ID: left.ID})
			}
			if len(candidate.Mappings) > 0 {
				candidates = append(candidates, candidate)
			}
		}
	}
	return candidates
}

func (e Event) RecordImportedMatch(number int, players []autodarts.Player, mapping map[string]int64) (league.TournamentImportResult, error) {
	if len(players) != 2 {
		return league.TournamentImportResult{}, league.ErrInvalidMapping
	}
	for _, round := range e.Bracket {
		for i := range round {
			match := &round[i]
			if match.Number != number || !match.ready() {
				continue
			}
			a, b := players[0], players[1]
			if mapping[a.ID] != match.Slots[0].EntryID {
				a, b = b, a
			}
			if mapping[a.ID] != match.Slots[0].EntryID || mapping[b.ID] != match.Slots[1].EntryID {
				return league.TournamentImportResult{}, league.ErrInvalidMapping
			}
			if err := league.ValidateResultScore(a.LegsWon, b.LegsWon, int(e.LegsToWin)); err != nil {
				return league.TournamentImportResult{}, err
			}
			winner := match.Slots[0]
			if b.LegsWon > a.LegsWon {
				winner = match.Slots[1]
			}
			legs := [2]int{a.LegsWon, b.LegsWon}
			match.Legs, match.WinnerID = &legs, winner.EntryID
			for _, laterRound := range e.Bracket {
				for j := range laterRound {
					for k, slot := range laterRound[j].Slots {
						if slot.WinnerOf == number {
							slot.EntryID, slot.Player = winner.EntryID, winner.Player
							laterRound[j].Slots[k] = slot
						}
					}
				}
			}
			return league.TournamentImportResult{TournamentSlug: e.Slug, MatchNumber: number, Legs: legs, WinnerID: winner.EntryID}, nil
		}
	}
	return league.TournamentImportResult{}, league.ErrApprovalConflict
}
