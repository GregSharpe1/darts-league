package league

import (
	"context"
	"errors"
	"testing"
)

func TestTransactionRollbackIsolatesPointerValues(t *testing.T) {
	store := NewMemoryStore()
	ctx := context.Background()
	if _, err := store.CreateFixtures(ctx, []Fixture{{SeasonID: 1, PlayerOneID: 11, PlayerTwoID: 22}}); err != nil {
		t.Fatal(err)
	}
	average := 60.12
	if _, err := store.CreateResult(ctx, Result{FixtureID: 1, PlayerTwoAverage: &average}); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("abort")
	err := store.Transaction(ctx, func(tx Store) error {
		result, err := tx.GetResultByFixture(ctx, 1)
		if err != nil {
			return err
		}
		*result.PlayerTwoAverage = 0
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	result, err := store.GetResultByFixture(ctx, 1)
	if err != nil || *result.PlayerTwoAverage != 60.12 {
		t.Fatalf("rollback leaked pointer write: %+v %v", result, err)
	}
}
