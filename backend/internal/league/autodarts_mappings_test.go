package league

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestMappingsValidateAndSerialize(t *testing.T) {
	ctx := context.Background()
	store := NewMemoryStore()
	a, _ := store.CreatePlayer(ctx, Player{SeasonID: 1, DisplayName: "One"})
	b, _ := store.CreatePlayer(ctx, Player{SeasonID: 1, DisplayName: "Two"})
	s := NewPendingResultService(store, NewResultService(store))
	if got, err := s.SaveAutodartsMapping(ctx, 1, a.ID, "  exact Name  "); err != nil || got.Username != "exact Name" {
		t.Fatalf("save: %+v %v", got, err)
	}
	if _, err := s.SaveAutodartsMapping(ctx, 1, b.ID, "exact Name"); !errors.Is(err, ErrAutodartsUsernameConflict) {
		t.Fatal(err)
	}
	for _, name := range []string{"", " \t ", "bad\x00name", strings.Repeat("x", 81)} {
		if _, err := s.SaveAutodartsMapping(ctx, 1, a.ID, name); !errors.Is(err, ErrAutodartsUsernameInvalid) {
			t.Fatalf("%q: %v", name, err)
		}
	}
	if _, err := s.SaveAutodartsMapping(ctx, 2, a.ID, "other"); !errors.Is(err, ErrSeasonTransition) {
		t.Fatal(err)
	}
	if _, err := s.SaveAutodartsMapping(ctx, 1, 999, "other"); !errors.Is(err, ErrPlayerNotFound) {
		t.Fatal(err)
	}
	if _, err := s.SaveAutodartsMapping(ctx, 1, b.ID, "Exact Name"); err != nil {
		t.Fatal(err)
	}
	failure := errors.New("rollback")
	err := store.Transaction(ctx, func(tx Store) error {
		if err := tx.(AutodartsMappingStore).SetAutodartsUsername(ctx, 1, a.ID, nil); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	view, err := s.AutodartsMappings(ctx)
	if err != nil || len(view.Mappings) != 2 || view.AutoSubmitEnabled {
		t.Fatalf("view: %+v %v", view, err)
	}
	if err := s.ClearAutodartsMapping(ctx, 1, a.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.ClearAutodartsMapping(ctx, 1, a.ID); err != nil {
		t.Fatal(err)
	}
	if err := store.DeletePlayer(ctx, 1, b.ID); err != nil {
		t.Fatal(err)
	}
	view, err = s.AutodartsMappings(ctx)
	if err != nil || len(view.Mappings) != 0 {
		t.Fatalf("clear/delete: %+v %v", view, err)
	}
}
