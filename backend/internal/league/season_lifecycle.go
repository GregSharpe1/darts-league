package league

import (
	"context"
	"errors"
)

var (
	ErrSeasonTransition = errors.New("season has changed or does not allow this action")
	ErrSeasonIncomplete = errors.New("every fixture across all divisions must have a result before closing")
	ErrSeasonCompleted  = errors.New("completed season is read-only")
)

func (s SeasonService) CloseSeason(ctx context.Context, seasonID int64) (SeasonSummary, error) {
	if err := s.store.CloseSeason(ctx, seasonID); err != nil {
		return SeasonSummary{}, err
	}
	return s.Summary(ctx)
}

func (s SeasonService) CreateNextSeason(ctx context.Context, seasonID int64, name string) (SeasonSummary, error) {
	if err := ValidateSeasonName(name); err != nil {
		return SeasonSummary{}, err
	}
	if err := s.store.CreateNextSeason(ctx, seasonID, NewSeason(NormalizeSeasonName(name))); err != nil {
		return SeasonSummary{}, err
	}
	return s.Summary(ctx)
}

func remainingFixtures(fixtures []Fixture, results []Result) int {
	recorded := make(map[int64]bool, len(results))
	for _, result := range results {
		recorded[result.FixtureID] = true
	}
	remaining := 0
	for _, fixture := range fixtures {
		if !recorded[fixture.ID] {
			remaining++
		}
	}
	return remaining
}

func (s ResultService) requireWritableFixture(ctx context.Context, fixtureID int64) error {
	fixture, err := s.store.GetFixture(ctx, fixtureID)
	if err != nil {
		return err
	}
	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return err
	}
	if season.Status == SeasonStatusCompleted || fixture.SeasonID != season.ID {
		return ErrSeasonCompleted
	}
	return nil
}
