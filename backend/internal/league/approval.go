package league

import (
	"context"
	"errors"
	"maps"
	"reflect"
	"strings"
	"time"
)

func (s PendingResultService) Approve(ctx context.Context, req ApprovalRequest) (Result, error) {
	var result Result
	err := transact(ctx, s.store, func(tx Store) error {
		var err error
		scoped := s
		scoped.store = tx
		scoped.results = NewResultServiceWithNow(tx, s.now)
		result, err = scoped.approve(ctx, req)
		return err
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func (s PendingResultService) approve(ctx context.Context, req ApprovalRequest) (Result, error) {
	store, ok := s.store.(ApprovalStore)
	if !ok {
		return Result{}, ErrImportStoreRequired
	}
	record, err := store.GetImport(ctx, req.PendingID)
	if err != nil {
		return Result{}, err
	}
	if record.Pending.Status == PendingResultStatusReviewBlocked || record.Import.ReviewReason != "" {
		return Result{}, ErrImportReviewBlocked
	}
	if record.Pending.Status != PendingResultStatusPending {
		return Result{}, ErrPendingResultNotPending
	}
	if strings.TrimSpace(req.Actor) == "" || req.SeasonID <= 0 || req.FixtureID <= 0 {
		return Result{}, ErrApprovalConflict
	}
	if err := s.results.requireWritableFixture(ctx, req.FixtureID); err != nil {
		return Result{}, err
	}
	fixture, err := s.store.GetFixture(ctx, req.FixtureID)
	if err != nil {
		return Result{}, err
	}
	if fixture.SeasonID != req.SeasonID || fixture.GameVariant != "501" || fixture.LegsToWin != 3 {
		return Result{}, ErrApprovalConflict
	}
	if len(record.Import.Players) != 2 || len(req.Mapping) != 2 {
		return Result{}, ErrInvalidMapping
	}
	a, b := record.Import.Players[0], record.Import.Players[1]
	one, two := req.Mapping[a.ID], req.Mapping[b.ID]
	if !((one == fixture.PlayerOneID && two == fixture.PlayerTwoID) || (one == fixture.PlayerTwoID && two == fixture.PlayerOneID)) || one == two {
		return Result{}, ErrInvalidMapping
	}
	if record.Import.SettingsEvidence != "source_reported" && !req.AttestFormat {
		return Result{}, ErrImportAttestation
	}
	if record.Import.PlayedAt == nil && strings.TrimSpace(req.MissingDateReason) == "" {
		return Result{}, ErrImportAttestation
	}
	if record.Import.PlayedAt != nil {
		played, err := time.Parse(time.RFC3339Nano, *record.Import.PlayedAt)
		if err != nil {
			return Result{}, ErrImportAttestation
		}
		season, err := s.store.GetActiveSeason(ctx)
		if err != nil {
			return Result{}, err
		}
		outside := played.After(s.now()) || (season.StartedAt != nil && played.Before(*season.StartedAt))
		if outside && strings.TrimSpace(req.Reason) == "" {
			return Result{}, ErrImportAttestation
		}
	}
	existing, err := s.store.GetResultByFixture(ctx, fixture.ID)
	if err != nil && !errors.Is(err, ErrResultNotFound) {
		return Result{}, err
	}
	var old *ResultSnapshot
	if err == nil {
		old = SnapshotFromResult(existing)
		if req.ExpectedResult == nil || req.ExpectedResult.ID != existing.ID || !req.ExpectedResult.UpdatedAt.Equal(existing.UpdatedAt) || !reflect.DeepEqual(req.ExpectedResult.ResultSnapshot, *old) {
			return Result{}, ErrApprovalConflict
		}
	} else if req.ExpectedResult != nil {
		return Result{}, ErrApprovalConflict
	}
	history := []ImportRecord{record}
	if record.Import.ExternalMatchID != "" {
		history, err = store.ImportsBySource(ctx, record.Import.Source, record.Import.ExternalMatchID)
		if err != nil {
			return Result{}, err
		}
	}
	changed := record.Changed
	for _, previous := range history {
		if previous.Pending.ID != record.Pending.ID && previous.Import.Digest != record.Import.Digest {
			changed = true
		}
		if previous.Pending.Status == PendingResultStatusConfirmed && previous.FixtureID == nil {
			return Result{}, ErrApprovalConflict
		}
		if previous.FixtureID != nil && (*previous.FixtureID != fixture.ID || previous.SeasonID == nil || *previous.SeasonID != req.SeasonID || !maps.Equal(previous.Mapping, req.Mapping)) {
			return Result{}, ErrApprovalConflict
		}
	}
	if (old != nil || changed) && (!req.Replace || strings.TrimSpace(req.Reason) == "") {
		return Result{}, ErrReplacementRequired
	}
	if one != fixture.PlayerOneID {
		a, b = b, a
	}
	if err := ValidateResultScore(a.LegsWon, b.LegsWon, 3); err != nil {
		return Result{}, err
	}
	meta := &ImportAudit{PendingID: req.PendingID, Source: record.Import.Source, ExternalMatchID: record.Import.ExternalMatchID, Digest: record.Import.Digest, Mapping: maps.Clone(req.Mapping), Reason: strings.TrimSpace(req.Reason), AttestFormat: req.AttestFormat, MissingDateReason: strings.TrimSpace(req.MissingDateReason)}
	previous, err := store.ImportByFixture(ctx, fixture.ID)
	if err == nil {
		meta.PreviousPendingID = &previous.Pending.ID
		previous.Active = false
		if err := store.SaveImportApproval(ctx, previous); err != nil {
			return Result{}, err
		}
	} else if !errors.Is(err, ErrPendingResultNotFound) {
		return Result{}, err
	}
	now := s.now().UTC()
	result := Result{FixtureID: fixture.ID, PlayerOneLegs: a.LegsWon, PlayerTwoLegs: b.LegsWon, PlayerOneAverage: a.Average(), PlayerTwoAverage: b.Average(), EnteredAt: now, UpdatedAt: now}
	result.WinnerID, err = WinnerIDForFixture(fixture, a.LegsWon, b.LegsWon)
	if err != nil {
		return Result{}, err
	}
	if old == nil {
		result, err = s.store.CreateResult(ctx, result)
	} else {
		result.ID, result.EnteredAt = existing.ID, existing.EnteredAt
		result.UpdatedAt = nextResultTime(now, existing.UpdatedAt)
		result, err = s.store.UpdateResult(ctx, result)
	}
	if err != nil {
		return Result{}, err
	}
	record.Pending.Status, record.Pending.ConfirmedAt, record.Pending.ConfirmedBy = PendingResultStatusConfirmed, &now, req.Actor
	meta.ResultID = result.ID
	record.SeasonID, record.FixtureID, record.ResultID, record.Active = &fixture.SeasonID, &fixture.ID, &result.ID, true
	record.Mapping, record.Approval = maps.Clone(req.Mapping), meta
	if err := store.SaveImportApproval(ctx, record); err != nil {
		return Result{}, err
	}
	_, err = s.store.CreateAuditLog(ctx, AuditLogEntry{SeasonID: fixture.SeasonID, FixtureID: fixture.ID, Action: "import_confirmed", Actor: req.Actor, OldResult: old, NewResult: SnapshotFromResult(result), Import: meta, CreatedAt: now})
	return result, err
}
