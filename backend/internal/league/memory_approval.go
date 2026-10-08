package league

import (
	"context"
	"encoding/json"
	"reflect"
)

func (s *MemoryStore) ImportsBySource(ctx context.Context, source, external string) ([]ImportRecord, error) {
	return s.findImports(ctx, func(r ImportRecord) bool { return r.Import.Source == source && r.Import.ExternalMatchID == external })
}

func (s *MemoryStore) findImports(ctx context.Context, match func(ImportRecord) bool) ([]ImportRecord, error) {
	s.mu.RLock()
	ids := make([]int64, 0, len(s.pendingByID))
	for id := range s.pendingByID {
		ids = append(ids, id)
	}
	s.mu.RUnlock()
	records := []ImportRecord{}
	for _, id := range ids {
		r, err := s.GetImport(ctx, id)
		if err != nil {
			return nil, err
		}
		if match(r) {
			records = append(records, r)
		}
	}
	return records, nil
}

func (s *MemoryStore) ImportByFixture(ctx context.Context, id int64) (ImportRecord, error) {
	rows, err := s.findImports(ctx, func(r ImportRecord) bool { return r.Active && r.FixtureID != nil && *r.FixtureID == id })
	if err != nil {
		return ImportRecord{}, err
	}
	if len(rows) == 0 {
		return ImportRecord{}, ErrPendingResultNotFound
	}
	return rows[0], nil
}

func (s *MemoryStore) ImportByResult(ctx context.Context, id int64) (ImportRecord, error) {
	rows, err := s.findImports(ctx, func(r ImportRecord) bool { return r.Active && r.ResultID != nil && *r.ResultID == id })
	if err != nil {
		return ImportRecord{}, err
	}
	if len(rows) == 0 {
		return ImportRecord{}, ErrPendingResultNotFound
	}
	return rows[0], nil
}

func (s *MemoryStore) SaveImportApproval(ctx context.Context, r ImportRecord) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	pending, ok := s.pendingByID[r.Pending.ID]
	if !ok {
		return ErrPendingResultNotFound
	}
	var original ImportRecord
	if b, ok := s.importsByID[r.Pending.ID]; ok {
		if err := json.Unmarshal(b, &original); err != nil {
			return err
		}
	} else {
		var err error
		original, err = LegacyImportRecord(pending)
		if err != nil {
			return err
		}
	}
	if original.SeasonID != nil && s.seasonsByID[*original.SeasonID].Status == SeasonStatusCompleted {
		return ErrSeasonCompleted
	}
	if r.SeasonID != nil && s.seasonsByID[*r.SeasonID].Status == SeasonStatusCompleted {
		return ErrSeasonCompleted
	}
	if original.FixtureID != nil && (!reflect.DeepEqual(original.FixtureID, r.FixtureID) || !reflect.DeepEqual(original.Mapping, r.Mapping)) {
		return ErrApprovalConflict
	}
	original.SeasonID, original.FixtureID, original.ResultID, original.Active = r.SeasonID, r.FixtureID, r.ResultID, r.Active
	original.Mapping, original.Approval = r.Mapping, r.Approval
	b, err := json.Marshal(original)
	if err != nil {
		return err
	}
	pending.Status, pending.ConfirmedAt, pending.ConfirmedBy = r.Pending.Status, r.Pending.ConfirmedAt, r.Pending.ConfirmedBy
	s.importsByID[pending.ID], s.pendingByID[pending.ID] = b, clonePending(pending)
	return nil
}
