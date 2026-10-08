package league

import (
	"context"
	"encoding/json"
	"time"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

func (s *MemoryStore) InsertImport(ctx context.Context, imported autodarts.Import, received time.Time) (ImportOutcome, error) {
	if err := ctx.Err(); err != nil {
		return ImportOutcome{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	changed := false
	for id, pending := range s.pendingByID {
		var record ImportRecord
		if data, ok := s.importsByID[id]; ok {
			if err := json.Unmarshal(data, &record); err != nil {
				return ImportOutcome{}, err
			}
		} else {
			var err error
			record, err = LegacyImportRecord(pending)
			if err != nil {
				return ImportOutcome{}, err
			}
		}
		if record.Import.Source != imported.Source || record.Import.ExternalMatchID != imported.ExternalMatchID {
			continue
		}
		if record.Import.Digest == imported.Digest {
			return ImportOutcome{Pending: clonePending(s.pendingByID[id]), Duplicate: true, Changed: record.Changed}, nil
		}
		changed = true
	}
	a, b := imported.Players[0], imported.Players[1]
	status := PendingResultStatusPending
	if imported.ReviewReason != "" {
		status = PendingResultStatusReviewBlocked
	}
	pending := PendingResult{ID: s.nextPendingID, ExternalMatchID: imported.ExternalMatchID,
		PlayerOneName: a.DisplayName, PlayerOneLegs: a.LegsWon, PlayerOneAverage: a.Average(),
		PlayerTwoName: b.DisplayName, PlayerTwoLegs: b.LegsWon, PlayerTwoAverage: b.Average(), Status: status, ReceivedAt: received}
	data, err := json.Marshal(ImportRecord{Import: imported, Changed: changed})
	if err != nil {
		return ImportOutcome{}, err
	}
	s.nextPendingID++
	s.importsByID[pending.ID] = data
	s.pendingByID[pending.ID] = clonePending(pending)
	return ImportOutcome{Pending: clonePending(pending), Changed: changed}, nil
}

func (s *MemoryStore) GetImport(ctx context.Context, id int64) (ImportRecord, error) {
	if err := ctx.Err(); err != nil {
		return ImportRecord{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	data, ok := s.importsByID[id]
	if !ok {
		if pending, exists := s.pendingByID[id]; exists {
			return LegacyImportRecord(clonePending(pending))
		}
		return ImportRecord{}, ErrPendingResultNotFound
	}
	var record ImportRecord
	if err := json.Unmarshal(data, &record); err != nil {
		return ImportRecord{}, err
	}
	record.Pending = clonePending(s.pendingByID[id])
	return record, nil
}

func clonePending(p PendingResult) PendingResult {
	if p.PlayerOneAverage != nil {
		value := *p.PlayerOneAverage
		p.PlayerOneAverage = &value
	}
	if p.PlayerTwoAverage != nil {
		value := *p.PlayerTwoAverage
		p.PlayerTwoAverage = &value
	}
	if p.ConfirmedAt != nil {
		value := *p.ConfirmedAt
		p.ConfirmedAt = &value
	}
	return p
}
