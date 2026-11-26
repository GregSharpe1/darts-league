package tournament

import (
	"context"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/greg/darts-league/backend/internal/league"
)

var (
	ErrScoreNotRecorded = errors.New("no score is recorded for this match")
	ErrScoreChanged     = errors.New("the match or a later round changed. Reload the tournament and review the correction.")
	ErrCorrectionReason = errors.New("enter a correction reason of up to 500 characters")
	ErrScoreUnavailable = errors.New("tournament score corrections require persistent storage")
)

type ScoreCorrection struct {
	Legs         [2]int
	PreviousLegs [2]int
	Entrants     [2]int64
	ClearMatches []int
	Reason       string
	Actor        string
}

type ScoreChange struct {
	Before BracketMatch
	After  BracketMatch
}

type scoreStore interface {
	CorrectTournamentScore(context.Context, string, int, ScoreCorrection) (Event, error)
}

func (s Service) CorrectScore(ctx context.Context, slug string, number int, correction ScoreCorrection) (Event, error) {
	correction.Reason = strings.TrimSpace(correction.Reason)
	if correction.Reason == "" || utf8.RuneCountInString(correction.Reason) > 500 {
		return Event{}, ErrCorrectionReason
	}
	store, ok := s.Store.(scoreStore)
	if !ok {
		return Event{}, ErrScoreUnavailable
	}
	return store.CorrectTournamentScore(ctx, slug, number, correction)
}

func (b Bracket) dependentMatch(number int) (*BracketMatch, *BracketSlot) {
	for _, round := range b {
		for i := range round {
			for j := range round[i].Slots {
				if round[i].Slots[j].WinnerOf == number {
					return &round[i], &round[i].Slots[j]
				}
			}
		}
	}
	return nil, nil
}

// ApplyScoreCorrection works on a copy so stale or unconfirmed changes leave the draw intact.
func (e Event) ApplyScoreCorrection(number int, correction ScoreCorrection) (Bracket, []ScoreChange, error) {
	if err := league.ValidateResultScore(correction.Legs[0], correction.Legs[1], int(e.LegsToWin)); err != nil {
		return nil, nil, err
	}

	bracket := make(Bracket, len(e.Bracket))
	var match *BracketMatch
	for i, round := range e.Bracket {
		bracket[i] = slices.Clone(round)
		for j := range bracket[i] {
			if bracket[i][j].Number == number {
				match = &bracket[i][j]
			}
		}
	}
	if match == nil || match.Legs == nil {
		return nil, nil, ErrScoreNotRecorded
	}
	if *match.Legs != correction.PreviousLegs || [2]int64{match.Slots[0].EntryID, match.Slots[1].EntryID} != correction.Entrants {
		return nil, nil, ErrScoreChanged
	}
	if *match.Legs == correction.Legs && len(correction.ClearMatches) == 0 {
		return bracket, nil, nil
	}

	before := *match
	winner := match.Slots[0]
	if correction.Legs[1] > correction.Legs[0] {
		winner = match.Slots[1]
	}
	match.Legs, match.WinnerID = &correction.Legs, winner.EntryID
	changes := []ScoreChange{{Before: before, After: *match}}
	var cleared []int
	if before.WinnerID != winner.EntryID {
		for next, slot := bracket.dependentMatch(number); next != nil; next, slot = bracket.dependentMatch(next.Number) {
			previous := *next
			slot.EntryID, slot.Player = winner.EntryID, winner.Player
			if next.Legs == nil {
				break
			}
			next.Legs, next.WinnerID = nil, 0
			cleared = append(cleared, next.Number)
			changes = append(changes, ScoreChange{Before: previous, After: *next})
			winner = BracketSlot{}
		}
	}
	if !slices.Equal(cleared, correction.ClearMatches) {
		return nil, nil, ErrScoreChanged
	}
	return bracket, changes, nil
}
