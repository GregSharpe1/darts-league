package league

import (
	"context"
	"errors"
	"testing"
)

type rollbackIngestStore struct{ *MemoryStore }

var errIngestRollback = errors.New("injected commit failure")

func (s rollbackIngestStore) Transaction(ctx context.Context, fn func(Store) error) error {
	return s.MemoryStore.Transaction(ctx, func(tx Store) error {
		if err := fn(tx); err != nil {
			return err
		}
		return errIngestRollback
	})
}

func TestLegacyIngestRollbackDoesNotNotify(t *testing.T) {
	store := rollbackIngestStore{NewMemoryStore()}
	notified := false
	s := NewPendingResultService(store, NewResultService(store)).WithNotifier(func(context.Context, PendingResult) { notified = true })
	_, err := s.Ingest(context.Background(), "legacy", "A", 3, nil, "B", 1, nil)
	if !errors.Is(err, errIngestRollback) || notified {
		t.Fatalf("error=%v notified=%v", err, notified)
	}
	rows, err := store.ListPendingResults(context.Background(), PendingResultStatusPending)
	if err != nil || len(rows) != 0 {
		t.Fatalf("rollback %+v %v", rows, err)
	}
}
