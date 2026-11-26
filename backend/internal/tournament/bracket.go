package tournament

import (
	"errors"
	"math/rand/v2"
	"slices"
)

var (
	ErrTournamentStarted = errors.New("this tournament has already started")
	ErrCloseBeforeStart  = errors.New("close registration before starting the tournament")
	ErrTooFewEntrants    = errors.New("at least two entrants are required to start the tournament")
)

type BracketSlot struct {
	EntryID  int64  `json:"entry_id,omitempty"`
	Player   string `json:"player,omitempty"`
	WinnerOf int    `json:"winner_of,omitempty"`
	Bye      bool   `json:"bye,omitempty"`
}

type BracketMatch struct {
	WinnerID int64          `json:"winner_id,omitempty"`
	Legs     *[2]int        `json:"legs,omitempty"`
	Number   int            `json:"number"`
	Slots    [2]BracketSlot `json:"slots"`
}

type Bracket [][]BracketMatch

func DrawBracket(event Event, entries []Entry) (Bracket, error) {
	if len(event.Bracket) != 0 {
		return event.Bracket, nil
	}
	if event.RegistrationOpen {
		return nil, ErrCloseBeforeStart
	}
	if len(entries) < 2 {
		return nil, ErrTooFewEntrants
	}
	draw := slices.Clone(entries)
	rand.Shuffle(len(draw), func(i, j int) { draw[i], draw[j] = draw[j], draw[i] })
	return buildBracket(draw), nil
}

func buildBracket(entries []Entry) Bracket {
	seedOrder := []int{0, 1}
	for len(seedOrder) < len(entries) {
		size := len(seedOrder) * 2
		next := make([]int, 0, size)
		for _, seed := range seedOrder {
			next = append(next, seed, size-1-seed)
		}
		seedOrder = next
	}

	slots := make([]BracketSlot, len(seedOrder))
	for i, seed := range seedOrder {
		if seed >= len(entries) {
			slots[i] = BracketSlot{Bye: true}
			continue
		}
		entry := entries[seed]
		slots[i] = BracketSlot{Player: entry.preferredName(), EntryID: entry.ID}
	}

	var rounds Bracket
	matchNumber := 1
	for len(slots) > 1 {
		matches := make([]BracketMatch, 0, len(slots)/2)
		next := make([]BracketSlot, 0, len(slots)/2)
		for i := 0; i < len(slots); i += 2 {
			left, right := slots[i], slots[i+1]
			matches = append(matches, BracketMatch{Number: matchNumber, Slots: [2]BracketSlot{left, right}})
			switch {
			case left.Bye:
				next = append(next, right)
			case right.Bye:
				next = append(next, left)
			default:
				next = append(next, BracketSlot{WinnerOf: matchNumber})
			}
			matchNumber++
		}
		rounds = append(rounds, matches)
		slots = next
	}
	return rounds
}

func (e Entry) preferredName() string {
	if e.Nickname != "" {
		return e.Nickname
	}
	return e.DisplayName
}
