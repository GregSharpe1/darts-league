package postgres

import (
	"context"
	"errors"
	"strconv"
	"testing"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/tournament"
)

func TestTournamentScoreCorrectionPreservesReceiptsAndClearsOnlyDependentScores(t *testing.T) {
	ctx := context.Background()
	store, importer, _, _ := autoPG(t)
	event, names := publishedCup(t, store, "correction-cup", 3, "Alice", "Bob", "Cara", "Dev")
	for i, match := range event.Bracket[0] {
		payload := cupPayload("correction-semi-"+strconv.Itoa(i), names[match.Slots[0].EntryID], names[match.Slots[1].EntryID], 3)
		if result := deliverCup(t, importer, payload); result.Pending.Status != league.PendingResultStatusConfirmed {
			t.Fatalf("semi-final was not scored: %+v", result)
		}
	}
	current, err := store.GetTournament(ctx, event.Slug)
	if err != nil {
		t.Fatal(err)
	}
	final := current.Bracket[1][0]
	finalPayload := cupPayload("correction-original-final", names[final.Slots[0].EntryID], names[final.Slots[1].EntryID], 3)
	finalImport := deliverCup(t, importer, finalPayload)
	if finalImport.Pending.Status != league.PendingResultStatusConfirmed {
		t.Fatalf("final was not scored: %+v", finalImport)
	}
	semi := current.Bracket[0][0]
	editor := tournament.Service{Store: store}
	correction := tournament.ScoreCorrection{
		Legs: [2]int{3, 2}, PreviousLegs: *semi.Legs, Entrants: [2]int64{semi.Slots[0].EntryID, semi.Slots[1].EntryID},
		Actor: "test-admin", Reason: "Verified the original score",
	}
	fixed, err := editor.CorrectScore(ctx, event.Slug, semi.Number, correction)
	if err != nil || fixed.Bracket[1][0].Legs == nil || fixed.Bracket[1][0].WinnerID != final.Slots[0].EntryID {
		t.Fatalf("same-winner correction changed the final: %+v, %v", fixed, err)
	}

	correction.Legs = [2]int{1, 3}
	if _, err := editor.CorrectScore(ctx, event.Slug, semi.Number, correction); !errors.Is(err, tournament.ErrScoreChanged) {
		t.Fatalf("stale score returned %v", err)
	}
	correction.PreviousLegs = [2]int{3, 2}
	if _, err := editor.CorrectScore(ctx, event.Slug, semi.Number, correction); !errors.Is(err, tournament.ErrScoreChanged) {
		t.Fatalf("unconfirmed final reset returned %v", err)
	}
	correction.ClearMatches = []int{final.Number}
	fixed, err = editor.CorrectScore(ctx, event.Slug, semi.Number, correction)
	if err != nil {
		t.Fatal(err)
	}
	updatedFinal := fixed.Bracket[1][0]
	if updatedFinal.Legs != nil || updatedFinal.WinnerID != 0 || updatedFinal.Slots[0].EntryID != semi.Slots[1].EntryID ||
		updatedFinal.Slots[1] != final.Slots[1] || fixed.Bracket[0][1].Legs == nil {
		t.Fatalf("invalid advancement after winner correction: %+v", fixed.Bracket)
	}
	receipt, err := store.GetImport(ctx, finalImport.Pending.ID)
	if err != nil || receipt.Active || receipt.Pending.Status != league.PendingResultStatusConfirmed || receipt.Approval == nil {
		t.Fatalf("original final receipt was lost or reusable: %+v, %v", receipt, err)
	}
	if replay := deliverCup(t, importer, finalPayload); !replay.Duplicate {
		t.Fatalf("original final replay was accepted: %+v", replay)
	}
	refinal := cupPayload("correction-replayed-final", names[updatedFinal.Slots[0].EntryID], names[updatedFinal.Slots[1].EntryID], 3)
	if result := deliverCup(t, importer, refinal); result.Pending.Status != league.PendingResultStatusConfirmed {
		t.Fatalf("replayed final was not scored: %+v", result)
	}

	var edits int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM tournament_score_audit
		WHERE tournament_slug = $1 AND actor = $2 AND reason = $3`, event.Slug, correction.Actor, correction.Reason).Scan(&edits); err != nil || edits != 3 {
		t.Fatalf("expected two corrections and one downstream reset in audit, got %d: %v", edits, err)
	}
}
