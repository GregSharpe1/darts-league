package league

import (
	"context"
	"errors"
	"time"
)

// publicStatisticsFixtures enforces both the week's reveal and each fixture's
// reveal, even when older schedules contain inconsistent dates within a week.
func publicStatisticsFixtures(fixtures []Fixture, now time.Time, loc *time.Location) map[int64]bool {
	visible := make(map[int64]bool, len(fixtures))
	currentWeek := CurrentPublicWeek(fixtures, now, loc)
	for _, week := range GroupFixturesByWeek(fixtures, loc) {
		if week.WeekNumber > currentWeek || now.Before(week.RevealAt) {
			continue
		}
		for _, fixture := range week.Fixtures {
			if CurrentPublicWeek([]Fixture{fixture}, now, loc) >= fixture.WeekNumber {
				visible[fixture.ID] = true
			}
		}
	}
	return visible
}

// VisitPublicRecordMatches streams publicly revealed league scores and matches
// them to active import evidence using the same checks as player history.
func VisitPublicRecordMatches(ctx context.Context, store Store, seasonID int64, now time.Time, visit func(Fixture, Result, *ImportRecord)) error {
	imports, ok := store.(ApprovalStore)
	if !ok {
		return ErrImportStoreRequired
	}
	fixtures, err := store.ListFixturesBySeason(ctx, seasonID)
	if err != nil {
		return err
	}
	results, err := store.ListResultsBySeason(ctx, seasonID)
	if err != nil {
		return err
	}
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		return err
	}
	visible := publicStatisticsFixtures(fixtures, now, loc)
	resultsByFixture := make(map[int64]Result, len(results))
	for _, result := range results {
		resultsByFixture[result.FixtureID] = result
	}
	for _, fixture := range fixtures {
		result, exists := resultsByFixture[fixture.ID]
		if !exists || !visible[fixture.ID] || fixture.SeasonID != seasonID {
			continue
		}
		source, err := imports.ImportByFixture(ctx, fixture.ID)
		switch {
		case errors.Is(err, ErrPendingResultNotFound):
			visit(fixture, result, nil)
		case err != nil:
			return err
		case statisticsSourceMatches(source, fixture, result):
			visit(fixture, result, &source)
		}
	}
	return nil
}
