package league

import (
	"context"
	"errors"
	"time"
)

type PendingResultStatus string

const (
	PendingResultStatusPending   PendingResultStatus = "pending"
	PendingResultStatusConfirmed PendingResultStatus = "confirmed"
	PendingResultStatusRejected  PendingResultStatus = "rejected"
)

var (
	ErrPendingResultNotFound   = errors.New("pending result not found")
	ErrPendingResultNotPending = errors.New("pending result has already been processed")
	ErrDuplicateExternalMatch  = errors.New("match has already been received")
	ErrNoFixtureForPlayers     = errors.New("no fixture found for the selected players")
)

// PendingResult is an externally reported match awaiting admin confirmation
// before it is recorded against a league fixture.
type PendingResult struct {
	ID               int64
	ExternalMatchID  string
	PlayerOneName    string
	PlayerOneLegs    int
	PlayerOneAverage *float64
	PlayerTwoName    string
	PlayerTwoLegs    int
	PlayerTwoAverage *float64
	Status           PendingResultStatus
	ReceivedAt       time.Time
	ConfirmedAt      *time.Time
	ConfirmedBy      string
}

type PendingResultService struct {
	autoSubmit bool
	store      Store
	results    ResultService
	now        func() time.Time
	notify     func(context.Context, PendingResult)
	// notifyAutoSubmit runs only after an automatic approval has committed.
	notifyAutoSubmit func(context.Context, Result)
}

func NewPendingResultService(store Store, results ResultService) PendingResultService {
	return NewPendingResultServiceWithNow(store, results, time.Now)
}

func NewPendingResultServiceWithNow(store Store, results ResultService, now func() time.Time) PendingResultService {
	return PendingResultService{store: store, results: results, now: now}
}

func (s PendingResultService) WithNotifier(notify func(context.Context, PendingResult)) PendingResultService {
	s.notify = notify
	return s
}

func (s PendingResultService) WithAutoSubmitNotifier(notify func(context.Context, Result)) PendingResultService {
	s.notifyAutoSubmit = notify
	return s
}

func (s PendingResultService) WithAutoSubmit(enabled bool) PendingResultService {
	s.autoSubmit = enabled
	return s
}

// Ingest records a match reported by an external scoring source as pending
// admin confirmation. Matches already ingested (by external match id) are
// rejected so repeated polling does not create duplicates.
func (s PendingResultService) Ingest(ctx context.Context, externalMatchID, playerOneName string, playerOneLegs int, playerOneAverage *float64, playerTwoName string, playerTwoLegs int, playerTwoAverage *float64) (PendingResult, error) {
	pending := PendingResult{
		ExternalMatchID:  externalMatchID,
		PlayerOneName:    playerOneName,
		PlayerOneLegs:    playerOneLegs,
		PlayerOneAverage: playerOneAverage,
		PlayerTwoName:    playerTwoName,
		PlayerTwoLegs:    playerTwoLegs,
		PlayerTwoAverage: playerTwoAverage,
		Status:           PendingResultStatusPending,
		ReceivedAt:       s.now().UTC(),
	}

	var created PendingResult
	err := transact(ctx, s.store, func(tx Store) error {
		if externalMatchID != "" {
			exists, err := tx.PendingResultExistsForExternalMatch(ctx, externalMatchID)
			if err != nil {
				return err
			}
			if exists {
				return ErrDuplicateExternalMatch
			}
		}
		var err error
		created, err = tx.CreatePendingResult(ctx, pending)
		return err
	})
	if err != nil {
		return PendingResult{}, err
	}
	if s.notify != nil {
		s.notify(ctx, created)
	}
	return created, nil
}

func (s PendingResultService) List(ctx context.Context) ([]PendingResult, error) {
	rows, err := s.store.ListPendingResults(ctx, PendingResultStatusPending)
	if err != nil {
		return nil, err
	}
	blocked, err := s.store.ListPendingResults(ctx, PendingResultStatusReviewBlocked)
	return append(rows, blocked...), err
}

func (s PendingResultService) Reject(ctx context.Context, pendingID int64, actor string) error {
	return s.RejectWithReason(ctx, pendingID, actor, "")
}
