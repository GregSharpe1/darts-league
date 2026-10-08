package league

import (
	"context"
	"errors"
)

var ErrDurablePendingStoreRequired = errors.New("durable pending result store required")
var ErrLegacyContentConflict = errors.New("legacy match content differs from persisted result")

// IngestDurable accepts a validated legacy result. Only stores with an explicit
// commit boundary may participate; the development memory fallback cannot.
func (s PendingResultService) IngestDurable(ctx context.Context, pending PendingResult) error {
	store, ok := s.store.(interface {
		CreatePendingResultDurably(context.Context, PendingResult) (PendingResult, error)
	})
	if !ok {
		return ErrDurablePendingStoreRequired
	}
	pending.Status = PendingResultStatusPending
	pending.ReceivedAt = s.now().UTC()
	created, err := store.CreatePendingResultDurably(ctx, pending)
	if err != nil {
		return err
	}
	if s.notify != nil {
		s.notify(ctx, created)
	}
	return nil
}
