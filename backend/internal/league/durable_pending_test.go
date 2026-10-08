package league

import (
	"context"
	"errors"
	"testing"
	"time"
)

type durablePendingFake struct {
	Store
	err error
}

func (s *durablePendingFake) CreatePendingResultDurably(_ context.Context, pending PendingResult) (PendingResult, error) {
	return pending, s.err
}

func TestDurableIngestRefusesMemoryFallback(t *testing.T) {
	store := NewMemoryStore()
	service := NewPendingResultService(store, NewResultService(store))
	if err := service.IngestDurable(context.Background(), PendingResult{ExternalMatchID: "id"}); !errors.Is(err, ErrDurablePendingStoreRequired) {
		t.Fatalf("memory accepted durable ingestion: %v", err)
	}
	items, err := service.List(context.Background())
	if err != nil || len(items) != 0 {
		t.Fatalf("memory mutated: %v", err)
	}
}

func TestDurableIngestNotifiesOnlyFreshCommit(t *testing.T) {
	storageFailure := errors.New("unavailable")
	for _, result := range []error{nil, ErrDuplicateExternalMatch, storageFailure, context.DeadlineExceeded} {
		store := &durablePendingFake{Store: NewMemoryStore(), err: result}
		calls := 0
		now := time.Date(2026, 10, 1, 10, 0, 0, 0, time.UTC)
		service := NewPendingResultServiceWithNow(store, NewResultService(store), func() time.Time { return now }).WithNotifier(func(_ context.Context, p PendingResult) {
			calls++
			if p.Status != PendingResultStatusPending || !p.ReceivedAt.Equal(now) {
				t.Fatal("missing pending metadata")
			}
		})
		err := service.IngestDurable(context.Background(), PendingResult{ExternalMatchID: "id"})
		if !errors.Is(err, result) {
			t.Fatalf("lost storage result: %v", err)
		}
		want := 0
		if result == nil {
			want = 1
		}
		if calls != want {
			t.Fatalf("notifications=%d want=%d", calls, want)
		}
	}
}
