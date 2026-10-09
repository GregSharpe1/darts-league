package league

import (
	"context"
	"errors"
	"reflect"
	"time"
)

func (s ResultService) RecordResult(ctx context.Context, fixtureID int64, one, two int, oneAverage, twoAverage *float64) (Result, error) {
	var result Result
	err := transact(ctx, s.store, func(tx Store) error {
		var err error
		result, err = NewResultServiceWithNow(tx, s.now).recordResult(ctx, fixtureID, one, two, oneAverage, twoAverage)
		return err
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func nextResultTime(now, previous time.Time) time.Time {
	// Postgres stores microseconds; every write must change the comparison token even with a frozen/backward clock.
	if now.Sub(previous) < time.Microsecond {
		return previous.Add(time.Microsecond)
	}
	return now
}

func (s ResultService) EditResult(ctx context.Context, fixtureID int64, one, two int, oneAverage, twoAverage *float64, actor string) (Result, error) {
	var result Result
	err := transact(ctx, s.store, func(tx Store) error {
		var err error
		result, err = NewResultServiceWithNow(tx, s.now).editResult(ctx, fixtureID, one, two, oneAverage, twoAverage, actor)
		return err
	})
	if err != nil {
		return Result{}, err
	}
	return result, nil
}

func (s ResultService) DeleteResult(ctx context.Context, fixtureID int64, actor string) error {
	return transact(ctx, s.store, func(tx Store) error {
		return NewResultServiceWithNow(tx, s.now).deleteResult(ctx, fixtureID, actor)
	})
}

func detachImport(ctx context.Context, store Store, old Result, updated *Result) (*ImportAudit, error) {
	imports, ok := store.(ApprovalStore)
	if !ok {
		return nil, ErrImportStoreRequired
	}
	r, err := imports.ImportByResult(ctx, old.ID)
	if errors.Is(err, ErrPendingResultNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if updated != nil && reflect.DeepEqual(SnapshotFromResult(old), SnapshotFromResult(*updated)) {
		return nil, nil
	}
	r.Active = false
	if updated == nil {
		r.ResultID = nil
	}
	meta := &ImportAudit{ResultID: old.ID, PendingID: r.Pending.ID, Source: r.Import.Source, ExternalMatchID: r.Import.ExternalMatchID, Digest: r.Import.Digest, Mapping: r.Mapping, Reason: "manual result change detached source statistics"}
	return meta, imports.SaveImportApproval(ctx, r)
}
