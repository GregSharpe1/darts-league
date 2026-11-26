package league

import (
	"context"
	"maps"
	"strings"
	"time"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

type TournamentImportCandidate struct {
	TournamentSlug string             `json:"tournament_slug"`
	TournamentName string             `json:"tournament_name"`
	MatchNumber    int                `json:"match_number"`
	Players        [2]string          `json:"players"`
	Mappings       []map[string]int64 `json:"mappings"`
}

type TournamentImportResult struct {
	TournamentSlug string `json:"tournament_slug"`
	MatchNumber    int    `json:"match_number"`
	Legs           [2]int `json:"legs"`
	WinnerID       int64  `json:"winner_id"`
}

type TournamentImportStore interface {
	// Write callers lock publication while finding and scoring candidates.
	LockTournamentImports(context.Context) error
	TournamentImportCandidates(context.Context, autodarts.Import) ([]TournamentImportCandidate, error)
	SaveTournamentImport(context.Context, ImportRecord, ApprovalRequest) (TournamentImportResult, error)
}

func (s PendingResultService) tournamentCandidates(ctx context.Context, record ImportRecord) ([]TournamentImportCandidate, string, error) {
	store, ok := s.store.(TournamentImportStore)
	if !ok {
		return nil, "Tournament imports require the durable store.", nil
	}
	if record.Import.ExternalMatchID == "" {
		return nil, "The source match has no Autodarts match ID.", nil
	}
	imports, ok := s.store.(ApprovalStore)
	if !ok {
		return nil, "", ErrImportStoreRequired
	}
	history, err := imports.ImportsBySource(ctx, record.Import.Source, record.Import.ExternalMatchID)
	if err != nil {
		return nil, "", err
	}
	for _, previous := range history {
		if previous.Pending.Status == PendingResultStatusConfirmed {
			return nil, "This source match was already applied.", nil
		}
	}
	candidates, err := store.TournamentImportCandidates(ctx, record.Import)
	return candidates, "", err
}

func (s PendingResultService) ApproveTournament(ctx context.Context, req ApprovalRequest) (TournamentImportResult, error) {
	var result TournamentImportResult
	err := transact(ctx, s.store, func(tx Store) error {
		imports, ok := tx.(ApprovalStore)
		if !ok {
			return ErrImportStoreRequired
		}
		record, err := imports.GetImport(ctx, req.PendingID)
		if err != nil {
			return err
		}
		scoped := s
		scoped.store = tx
		result, err = scoped.approveTournament(ctx, req, record)
		return err
	})
	return result, err
}

func (s PendingResultService) approveTournament(ctx context.Context, req ApprovalRequest, record ImportRecord) (TournamentImportResult, error) {
	var result TournamentImportResult
	store, ok := s.store.(TournamentImportStore)
	if !ok {
		return result, ErrImportStoreRequired
	}
	if record.Pending.Status == PendingResultStatusReviewBlocked || record.Import.ReviewReason != "" {
		return result, ErrImportReviewBlocked
	}
	if record.Pending.Status != PendingResultStatusPending {
		return result, ErrPendingResultNotPending
	}
	if record.Import.CompetitionType == autodarts.LeagueMatch || req.SeasonID != 0 || req.FixtureID != 0 ||
		req.ExpectedResult != nil || req.TournamentSlug == "" || req.MatchNumber <= 0 || strings.TrimSpace(req.Actor) == "" {
		return result, ErrApprovalConflict
	}
	if (record.Import.SettingsEvidence != "source_reported" && !req.AttestFormat) ||
		(record.Import.PlayedAt == nil && strings.TrimSpace(req.MissingDateReason) == "") ||
		(record.Import.PlayedAt != nil && !s.validPlayedAt(record.Import, time.Time{}) && strings.TrimSpace(req.Reason) == "") {
		return result, ErrImportAttestation
	}
	if record.Changed && (!req.Replace || strings.TrimSpace(req.Reason) == "") {
		return result, ErrReplacementRequired
	}
	if err := store.LockTournamentImports(ctx); err != nil {
		return result, err
	}
	candidates, _, err := s.tournamentCandidates(ctx, record)
	if err != nil {
		return result, err
	}
	for _, candidate := range candidates {
		if candidate.TournamentSlug != req.TournamentSlug || candidate.MatchNumber != req.MatchNumber {
			continue
		}
		for _, mapping := range candidate.Mappings {
			if !maps.Equal(mapping, req.Mapping) {
				continue
			}
			now := s.now().UTC()
			record.Pending.Status, record.Pending.ConfirmedAt, record.Pending.ConfirmedBy = PendingResultStatusConfirmed, &now, req.Actor
			record.Approval = &ImportAudit{
				PendingID: req.PendingID, Source: record.Import.Source, ExternalMatchID: record.Import.ExternalMatchID,
				Digest: record.Import.Digest, TournamentSlug: req.TournamentSlug, MatchNumber: req.MatchNumber,
				Mapping: maps.Clone(req.Mapping), Reason: strings.TrimSpace(req.Reason), AttestFormat: req.AttestFormat,
				MissingDateReason: strings.TrimSpace(req.MissingDateReason),
			}
			return store.SaveTournamentImport(ctx, record, req)
		}
	}
	return result, ErrApprovalConflict
}
