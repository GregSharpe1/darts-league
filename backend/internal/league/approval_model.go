package league

import (
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

var (
	ErrApprovalConflict    = errors.New("approval target or expected result has changed")
	ErrReplacementRequired = errors.New("explicit replacement and reason required")
	ErrImportReviewBlocked = errors.New("import detail conflicts with summary and cannot be approved")
	ErrImportAttestation   = errors.New("legacy format attestation and missing-date reason required")
	ErrInvalidMapping      = errors.New("map both source players to distinct fixture players")
)

type ExpectedResult struct {
	ID        int64     `json:"id"`
	UpdatedAt time.Time `json:"updated_at"`
	ResultSnapshot
}

func ExpectedFromResult(r Result) *ExpectedResult {
	return &ExpectedResult{ID: r.ID, UpdatedAt: r.UpdatedAt, ResultSnapshot: *SnapshotFromResult(r)}
}

type ApprovalRequest struct {
	PendingID         int64            `json:"-"`
	Actor             string           `json:"-"`
	SeasonID          int64            `json:"season_id"`
	FixtureID         int64            `json:"fixture_id"`
	Mapping           map[string]int64 `json:"mapping"`
	ExpectedResult    *ExpectedResult  `json:"expected_result"`
	Replace           bool             `json:"replace"`
	Reason            string           `json:"reason"`
	AttestFormat      bool             `json:"attest_format"`
	MissingDateReason string           `json:"missing_date_reason"`
}

type ImportAudit struct {
	ResultID          int64            `json:"result_id,omitempty"`
	PendingID         int64            `json:"pending_id"`
	Source            string           `json:"source"`
	ExternalMatchID   string           `json:"external_match_id"`
	Digest            string           `json:"digest"`
	Mapping           map[string]int64 `json:"mapping,omitempty"`
	Reason            string           `json:"reason"`
	AttestFormat      bool             `json:"attest_format"`
	MissingDateReason string           `json:"missing_date_reason,omitempty"`
	PreviousPendingID *int64           `json:"previous_pending_id,omitempty"`
}

// MappedImport returns a detached projection; source originals are never rewritten.
// Numeric league IDs are strings here to retain the normalized match reference shape.
func (r ImportRecord) MappedImport() (autodarts.Import, error) {
	b, err := json.Marshal(r.Import)
	if err != nil {
		return autodarts.Import{}, err
	}
	var mapped autodarts.Import
	if err := json.Unmarshal(b, &mapped); err != nil {
		return mapped, err
	}
	mapped.Payload = nil
	for i := range mapped.Players {
		id, ok := r.Mapping[mapped.Players[i].ID]
		if !ok {
			return autodarts.Import{}, ErrInvalidMapping
		}
		mapped.Players[i].ID = strconv.FormatInt(id, 10)
		mapped.Players[i].AccountID = nil
	}
	if mapped.Detail != nil {
		for i := range mapped.Detail.Legs {
			leg := &mapped.Detail.Legs[i]
			if leg.WinnerID != nil {
				id, ok := r.Mapping[*leg.WinnerID]
				if !ok {
					return autodarts.Import{}, ErrInvalidMapping
				}
				winner := strconv.FormatInt(id, 10)
				leg.WinnerID = &winner
			}
			for j := range leg.Visits {
				id, ok := r.Mapping[leg.Visits[j].PlayerID]
				if !ok {
					return autodarts.Import{}, ErrInvalidMapping
				}
				leg.Visits[j].PlayerID = strconv.FormatInt(id, 10)
			}
		}
	}
	return mapped, nil
}
