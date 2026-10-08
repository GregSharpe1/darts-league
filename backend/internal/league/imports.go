package league

import (
	"context"
	"errors"
	"time"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

var ErrDurableStoreRequired = errors.New("durable import store required")
var ErrImportStoreRequired = errors.New("detailed import store required")

const PendingResultStatusReviewBlocked PendingResultStatus = "review_blocked"

type ImportOutcome struct {
	Pending   PendingResult
	Duplicate bool
	Changed   bool
}

// ImportRecord is an admin/service DTO, never a public projection. Source IDs
// join players, leg winners and visits; player array order is not identity.
type ImportRecord struct {
	Pending   PendingResult
	Import    autodarts.Import
	Changed   bool
	SeasonID  *int64
	FixtureID *int64
	ResultID  *int64
	Active    bool
	Mapping   map[string]int64
	Approval  *ImportAudit
}

type ImportStore interface {
	InsertImport(context.Context, autodarts.Import, time.Time) (ImportOutcome, error)
	GetImport(context.Context, int64) (ImportRecord, error)
}

type ApprovalStore interface {
	ImportStore
	ImportsBySource(context.Context, string, string) ([]ImportRecord, error)
	ImportByFixture(context.Context, int64) (ImportRecord, error)
	ImportByResult(context.Context, int64) (ImportRecord, error)
	SaveImportApproval(context.Context, ImportRecord) error
}

// DurableImports is a capability declaration, not implied by detailed storage.
// MemoryStore deliberately implements only ImportStore.
type DurableImportStore interface {
	ImportStore
	DurableImports()
}

func (s PendingResultService) IngestDurablePayload(ctx context.Context, body []byte) (ImportOutcome, error) {
	if _, ok := s.store.(DurableImportStore); !ok {
		return ImportOutcome{}, ErrDurableStoreRequired
	}
	return s.IngestPayload(ctx, body)
}

func (s PendingResultService) IngestPayload(ctx context.Context, body []byte) (ImportOutcome, error) {
	store, ok := s.store.(ImportStore)
	if !ok {
		return ImportOutcome{}, ErrImportStoreRequired
	}
	parsed, err := autodarts.Parse(body)
	if err != nil {
		return ImportOutcome{}, err
	}
	outcome, err := store.InsertImport(ctx, parsed, s.now().UTC())
	if err == nil && !outcome.Duplicate && s.notify != nil {
		s.notify(ctx, outcome.Pending)
	}
	return outcome, err
}

func (s PendingResultService) ImportDetail(ctx context.Context, id int64) (ImportRecord, error) {
	store, ok := s.store.(ImportStore)
	if !ok {
		return ImportRecord{}, ErrImportStoreRequired
	}
	return store.GetImport(ctx, id)
}
