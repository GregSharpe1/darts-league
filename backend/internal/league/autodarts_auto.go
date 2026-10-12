package league

import (
	"context"
	"errors"
	"time"
)

func (s PendingResultService) autoApproveImport(ctx context.Context, outcome ImportOutcome) (ImportOutcome, error) {
	if !s.autoSubmit || outcome.Duplicate || outcome.Changed || outcome.Pending.Status != PendingResultStatusPending {
		return outcome, nil
	}
	store, ok := s.store.(ImportStore)
	if !ok {
		return outcome, ErrImportStoreRequired
	}
	record, err := store.GetImport(ctx, outcome.Pending.ID)
	if err != nil {
		return outcome, err
	}
	if record.Changed || record.Import.ReviewReason != "" || record.Import.SettingsEvidence != "source_reported" || record.Import.PlayedAt == nil {
		return outcome, nil
	}
	season, err := s.store.GetActiveSeason(ctx)
	if errors.Is(err, ErrSeasonNotFound) {
		return outcome, nil
	}
	if err != nil {
		return outcome, err
	}
	if season.Status != SeasonStatusStarted || season.StartedAt == nil {
		return outcome, nil
	}
	played, err := time.Parse(time.RFC3339Nano, *record.Import.PlayedAt)
	if err != nil || played.Before(*season.StartedAt) || played.After(s.now()) {
		return outcome, nil
	}
	suggestion, err := s.suggestImport(ctx, record)
	if err != nil {
		return outcome, err
	}
	if suggestion.FixtureID == nil {
		return outcome, nil
	}
	_, err = s.store.GetResultByFixture(ctx, *suggestion.FixtureID)
	if err == nil {
		return outcome, nil
	}
	if !errors.Is(err, ErrResultNotFound) {
		return outcome, err
	}
	_, err = s.approve(ctx, ApprovalRequest{PendingID: record.Pending.ID, SeasonID: season.ID, FixtureID: *suggestion.FixtureID, Mapping: suggestion.Mapping, Actor: "autodarts:auto-submit"})
	if err != nil {
		return outcome, err
	}
	outcome.Pending, err = s.store.GetPendingResult(ctx, outcome.Pending.ID)
	return outcome, err
}
