package league

import (
	"context"
	"errors"
	"time"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

const (
	missingCompetitionReason = "Competition type is missing. Choose league or tournament in Pending Results."
	noTournamentMatchReason  = "No eligible unplayed tournament match matches both players and the reported format."
	multipleTournamentReason = "Multiple eligible tournament matches. Choose the event and bracket match in Pending Results."
	autoSubmitDisabledReason = "Auto-submit is disabled."
)

func (p PendingResult) ReviewExplanation() string {
	if p.PendingReason != "" {
		return p.PendingReason
	}
	if p.CompetitionType == "" && (p.Status == PendingResultStatusPending || p.Status == PendingResultStatusReviewBlocked) {
		return missingCompetitionReason
	}
	return ""
}

func (s PendingResultService) validPlayedAt(imported autodarts.Import, earliest time.Time) bool {
	if imported.PlayedAt == nil {
		return false
	}
	played, err := time.Parse(time.RFC3339Nano, *imported.PlayedAt)
	return err == nil && !played.Before(earliest) && !played.After(s.now())
}

func (s PendingResultService) routeImport(ctx context.Context, outcome ImportOutcome) (ImportOutcome, error) {
	if outcome.Duplicate {
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
	reason, err := s.autoApproveImport(ctx, record, &outcome)
	if err != nil {
		return outcome, err
	}
	if reason == "" {
		outcome.Pending, err = s.store.GetPendingResult(ctx, outcome.Pending.ID)
	} else {
		outcome.Pending.PendingReason = reason
		outcome.Pending, err = s.store.UpdatePendingResult(ctx, outcome.Pending)
	}
	return outcome, err
}

func (s PendingResultService) autoApproveImport(ctx context.Context, record ImportRecord, outcome *ImportOutcome) (string, error) {
	if record.Import.ReviewReason != "" {
		return "Source conflict: " + record.Import.ReviewReason, nil
	}
	if record.Import.CompetitionType == "" {
		return missingCompetitionReason, nil
	}
	if record.Import.CompetitionType == autodarts.TournamentMatch {
		if s.autoSubmit {
			if store, ok := s.store.(TournamentImportStore); ok {
				if err := store.LockTournamentImports(ctx); err != nil {
					return "", err
				}
			}
		}
		candidates, reason, err := s.tournamentCandidates(ctx, record)
		if reason != "" || err != nil {
			return reason, err
		}
		switch {
		case len(candidates) == 0:
			return noTournamentMatchReason, nil
		case len(candidates) > 1:
			return multipleTournamentReason, nil
		case len(candidates[0].Mappings) != 1:
			return "Tournament player names are ambiguous. Verify the bracket entrants in Pending Results.", nil
		}
		if reason := s.autoSubmitBlocker(record); reason != "" {
			return reason, nil
		}
		candidate := candidates[0]
		_, err = s.approveTournament(ctx, ApprovalRequest{
			PendingID: record.Pending.ID, TournamentSlug: candidate.TournamentSlug, MatchNumber: candidate.MatchNumber,
			Mapping: candidate.Mappings[0], Actor: "autodarts:auto-submit",
		}, record)
		return "", err
	}

	if reason := s.autoSubmitBlocker(record); reason != "" {
		return reason, nil
	}
	return s.autoApproveLeague(ctx, record, outcome)
}

func (s PendingResultService) autoSubmitBlocker(record ImportRecord) string {
	switch {
	case record.Changed:
		return "A different version of this source match already exists."
	case !s.autoSubmit:
		return autoSubmitDisabledReason
	case record.Import.SettingsEvidence != "source_reported":
		return "Match format requires manual verification."
	case !s.validPlayedAt(record.Import, time.Time{}):
		return "The source match date is missing or ahead of the server clock."
	default:
		return ""
	}
}

func (s PendingResultService) autoApproveLeague(ctx context.Context, record ImportRecord, outcome *ImportOutcome) (string, error) {
	season, err := s.store.GetActiveSeason(ctx)
	if errors.Is(err, ErrSeasonNotFound) {
		return "There is no active league season.", nil
	}
	if err != nil {
		return "", err
	}
	if season.Status != SeasonStatusStarted || season.StartedAt == nil {
		return "The league season is not started and writable.", nil
	}
	if !s.validPlayedAt(record.Import, *season.StartedAt) {
		return "The source match date is outside the started league season.", nil
	}
	suggestion, err := s.suggestImport(ctx, record)
	if err != nil {
		return "", err
	}
	if len(suggestion.Mapping) != 2 {
		return "Both Autodarts names must already be mapped to distinct league players.", nil
	}
	if suggestion.FixtureID == nil {
		return "No unique eligible league fixture matches both players and the reported format.", nil
	}
	_, err = s.store.GetResultByFixture(ctx, *suggestion.FixtureID)
	if err == nil {
		return "The matching league fixture already has a result.", nil
	}
	if !errors.Is(err, ErrResultNotFound) {
		return "", err
	}
	result, err := s.approve(ctx, ApprovalRequest{PendingID: record.Pending.ID, SeasonID: season.ID, FixtureID: *suggestion.FixtureID, Mapping: suggestion.Mapping, Actor: "autodarts:auto-submit"})
	if err != nil {
		return "", err
	}
	outcome.AutoSubmitted = &result
	return "", nil
}
