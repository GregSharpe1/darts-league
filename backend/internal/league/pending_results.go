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
	store   Store
	results ResultService
	now     func() time.Time
	notify  func(context.Context, PendingResult)
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

// Ingest records a match reported by an external scoring source as pending
// admin confirmation. Matches already ingested (by external match id) are
// rejected so repeated polling does not create duplicates.
func (s PendingResultService) Ingest(ctx context.Context, externalMatchID, playerOneName string, playerOneLegs int, playerOneAverage *float64, playerTwoName string, playerTwoLegs int, playerTwoAverage *float64) (PendingResult, error) {
	if externalMatchID != "" {
		exists, err := s.store.PendingResultExistsForExternalMatch(ctx, externalMatchID)
		if err != nil {
			return PendingResult{}, err
		}
		if exists {
			return PendingResult{}, ErrDuplicateExternalMatch
		}
	}

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

	created, err := s.store.CreatePendingResult(ctx, pending)
	if err != nil {
		return PendingResult{}, err
	}
	if s.notify != nil {
		s.notify(ctx, created)
	}
	return created, nil
}

func (s PendingResultService) List(ctx context.Context) ([]PendingResult, error) {
	return s.store.ListPendingResults(ctx, PendingResultStatusPending)
}

// Confirm maps a pending result onto the fixture scheduled for the selected
// players and records it as a league result using the same validation as a
// manually entered score.
func (s PendingResultService) Confirm(ctx context.Context, pendingID, playerOneID, playerTwoID int64, actor string) (Result, error) {
	pending, err := s.store.GetPendingResult(ctx, pendingID)
	if err != nil {
		return Result{}, err
	}
	if pending.Status != PendingResultStatusPending {
		return Result{}, ErrPendingResultNotPending
	}

	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return Result{}, err
	}
	fixtures, err := s.store.ListFixturesBySeason(ctx, season.ID)
	if err != nil {
		return Result{}, err
	}
	existingResults, err := s.store.ListResultsBySeason(ctx, season.ID)
	if err != nil {
		return Result{}, err
	}

	fixture, hasResult, ok := findFixtureForPlayers(fixtures, existingResults, playerOneID, playerTwoID)
	if !ok {
		return Result{}, ErrNoFixtureForPlayers
	}

	playerOneLegs, playerTwoLegs := pending.PlayerOneLegs, pending.PlayerTwoLegs
	playerOneAverage, playerTwoAverage := pending.PlayerOneAverage, pending.PlayerTwoAverage
	if fixture.PlayerOneID != playerOneID {
		playerOneLegs, playerTwoLegs = playerTwoLegs, playerOneLegs
		playerOneAverage, playerTwoAverage = playerTwoAverage, playerOneAverage
	}

	var result Result
	if hasResult {
		result, err = s.results.EditResult(ctx, fixture.ID, playerOneLegs, playerTwoLegs, playerOneAverage, playerTwoAverage, actor)
	} else {
		result, err = s.results.RecordResult(ctx, fixture.ID, playerOneLegs, playerTwoLegs, playerOneAverage, playerTwoAverage)
	}
	if err != nil {
		return Result{}, err
	}

	confirmedAt := s.now().UTC()
	pending.Status = PendingResultStatusConfirmed
	pending.ConfirmedAt = &confirmedAt
	pending.ConfirmedBy = actor
	if _, err := s.store.UpdatePendingResult(ctx, pending); err != nil {
		return Result{}, err
	}

	return result, nil
}

func (s PendingResultService) Reject(ctx context.Context, pendingID int64, actor string) error {
	pending, err := s.store.GetPendingResult(ctx, pendingID)
	if err != nil {
		return err
	}
	if pending.Status != PendingResultStatusPending {
		return ErrPendingResultNotPending
	}

	rejectedAt := s.now().UTC()
	pending.Status = PendingResultStatusRejected
	pending.ConfirmedAt = &rejectedAt
	pending.ConfirmedBy = actor
	_, err = s.store.UpdatePendingResult(ctx, pending)
	return err
}

// findFixtureForPlayers returns the fixture scheduled between the two
// players, preferring one without a recorded result yet.
func findFixtureForPlayers(fixtures []Fixture, results []Result, playerOneID, playerTwoID int64) (Fixture, bool, bool) {
	resultByFixtureID := make(map[int64]bool, len(results))
	for _, result := range results {
		resultByFixtureID[result.FixtureID] = true
	}

	var fallback Fixture
	fallbackFound := false
	for _, fixture := range fixtures {
		matches := (fixture.PlayerOneID == playerOneID && fixture.PlayerTwoID == playerTwoID) ||
			(fixture.PlayerOneID == playerTwoID && fixture.PlayerTwoID == playerOneID)
		if !matches {
			continue
		}
		if !resultByFixtureID[fixture.ID] {
			return fixture, false, true
		}
		if !fallbackFound {
			fallback = fixture
			fallbackFound = true
		}
	}

	return fallback, true, fallbackFound
}
