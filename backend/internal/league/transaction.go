package league

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
)

var ErrTransactionRequired = errors.New("transaction store required")

type TransactionStore interface {
	Transaction(context.Context, func(Store) error) error
}

func transact(ctx context.Context, store Store, fn func(Store) error) error {
	tx, ok := store.(TransactionStore)
	if !ok {
		return ErrTransactionRequired
	}
	return tx.Transaction(ctx, fn)
}

func (s *MemoryStore) Transaction(ctx context.Context, fn func(Store) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// ponytail: serialize the small in-memory league; use per-season locks if throughput matters.
	s.mu.Lock()
	defer s.mu.Unlock()
	values := struct {
		Season  Season
		Seasons map[int64]Season
		Players map[int64]Player
		Results map[int64]Result
		Audits  map[int64]AuditLogEntry
		Pending map[int64]PendingResult
	}{s.activeSeason, s.seasonsByID, s.playersByID, s.resultsByID, s.auditByID, s.pendingByID}
	data, err := json.Marshal(values)
	if err != nil {
		return err
	}
	// Unmarshal into fresh maps so rollback cannot leak through pointer-valued fields.
	values.Season = Season{}
	values.Seasons, values.Players, values.Results, values.Audits, values.Pending = nil, nil, nil, nil, nil
	if err := json.Unmarshal(data, &values); err != nil {
		return err
	}
	tx := &MemoryStore{
		activeSeason: values.Season, seasonsByID: values.Seasons,
		divisionsByID: maps.Clone(s.divisionsByID), playersByID: values.Players,
		fixturesByID: maps.Clone(s.fixturesByID), resultsByID: values.Results,
		auditByID: values.Audits, pendingByID: values.Pending, importsByID: maps.Clone(s.importsByID),
		nextDivisionID: s.nextDivisionID, nextPlayerID: s.nextPlayerID, nextFixtureID: s.nextFixtureID,
		nextResultID: s.nextResultID, nextAuditID: s.nextAuditID, nextSeasonID: s.nextSeasonID, nextPendingID: s.nextPendingID,
	}
	if err := fn(tx); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	s.activeSeason, s.seasonsByID = tx.activeSeason, tx.seasonsByID
	s.divisionsByID, s.playersByID, s.fixturesByID = tx.divisionsByID, tx.playersByID, tx.fixturesByID
	s.resultsByID, s.auditByID, s.pendingByID, s.importsByID = tx.resultsByID, tx.auditByID, tx.pendingByID, tx.importsByID
	s.nextDivisionID, s.nextPlayerID, s.nextFixtureID = tx.nextDivisionID, tx.nextPlayerID, tx.nextFixtureID
	s.nextResultID, s.nextAuditID, s.nextSeasonID, s.nextPendingID = tx.nextResultID, tx.nextAuditID, tx.nextSeasonID, tx.nextPendingID
	return nil
}
