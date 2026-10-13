package postgres

import (
	"context"
	"errors"
	"github.com/greg/darts-league/backend/internal/league"
	"github.com/jackc/pgx/v5/pgxpool"
	"testing"
)

func TestAutodartsMappingMigrationPersistenceAndRollback(t *testing.T) {
	ctx := context.Background()
	store := approvalPG(t)
	_, req := approvalFixture(t, store)
	if _, err := store.pool.Exec(ctx, `ALTER TABLE players DROP COLUMN autodarts_username`); err != nil {
		t.Fatal(err)
	}
	dsn := store.pool.(*pgxpool.Pool).Config().ConnString()
	name := "private source"
	for i := range 2 {
		reopened, err := Open(ctx, dsn)
		if err != nil {
			t.Fatal(err)
		}
		t.Cleanup(reopened.Close)
		if i == 0 {
			if err := reopened.SetAutodartsUsername(ctx, req.SeasonID, req.Mapping["seat-a"], &name); err != nil {
				t.Fatal(err)
			}
		}
		rows, err := reopened.ListAutodartsMappings(ctx, req.SeasonID)
		if err != nil || len(rows) != 1 || rows[0].Username != name {
			t.Fatalf("restart: %+v %v", rows, err)
		}
	}
	if err := store.SetAutodartsUsername(ctx, req.SeasonID, req.Mapping["seat-b"], &name); !errors.Is(err, league.ErrAutodartsUsernameConflict) {
		t.Fatal(err)
	}
	if err := store.SetAutodartsUsername(ctx, 999, req.Mapping["seat-a"], &name); !errors.Is(err, league.ErrPlayerNotFound) {
		t.Fatal(err)
	}
	failure := errors.New("rollback")
	err := store.Transaction(ctx, func(tx league.Store) error {
		if err := tx.(league.AutodartsMappingStore).SetAutodartsUsername(ctx, req.SeasonID, req.Mapping["seat-a"], nil); err != nil {
			return err
		}
		return failure
	})
	if !errors.Is(err, failure) {
		t.Fatal(err)
	}
	rows, err := store.ListAutodartsMappings(ctx, req.SeasonID)
	if err != nil || len(rows) != 1 {
		t.Fatalf("rollback: %+v %v", rows, err)
	}
	if _, err := store.pool.Exec(ctx, `UPDATE seasons SET status='completed' WHERE id=$1`, req.SeasonID); err != nil {
		t.Fatal(err)
	}
	service := league.NewPendingResultService(store, league.NewResultService(store))
	if _, err := service.SaveAutodartsMapping(ctx, req.SeasonID, req.Mapping["seat-a"], name); err != nil {
		t.Fatal(err)
	}
	if err := store.CreateNextSeason(ctx, req.SeasonID, league.NewSeason("next")); err != nil {
		t.Fatal(err)
	}
	season, err := store.GetActiveSeason(ctx)
	if err != nil {
		t.Fatal(err)
	}
	p, err := store.CreatePlayer(ctx, league.Player{SeasonID: season.ID, DisplayName: "new"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveAutodartsMapping(ctx, season.ID, p.ID, name); err != nil {
		t.Fatal(err)
	}
	if _, err := service.SaveAutodartsMapping(ctx, req.SeasonID, req.Mapping["seat-a"], "stale"); !errors.Is(err, league.ErrSeasonTransition) {
		t.Fatal(err)
	}
	if _, err := service.SaveAutodartsMapping(ctx, season.ID, req.Mapping["seat-a"], "wrong target"); !errors.Is(err, league.ErrPlayerNotFound) {
		t.Fatal(err)
	}
	if err := store.DeletePlayer(ctx, season.ID, p.ID); err != nil {
		t.Fatal(err)
	}
	view, err := service.AutodartsMappings(ctx)
	if err != nil || len(view.Mappings) != 0 {
		t.Fatalf("new season: %+v %v", view, err)
	}
	rows, err = store.ListAutodartsMappings(ctx, req.SeasonID)
	if err != nil || len(rows) != 1 || rows[0].Username != name {
		t.Fatalf("history: %+v %v", rows, err)
	}
}
