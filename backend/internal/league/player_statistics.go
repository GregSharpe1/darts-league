package league

import (
	"context"
	"errors"
	"reflect"
	"sort"
	"time"
)

func (s ResultService) PublicPlayerStatistics(ctx context.Context, seasonID, playerID int64) (PlayerStatistics, error) {
	return s.playerStatistics(ctx, seasonID, playerID, true)
}

// AdminPlayerStatistics must only be called after the existing admin auth boundary.
func (s ResultService) AdminPlayerStatistics(ctx context.Context, seasonID, playerID int64) (PlayerStatistics, error) {
	return s.playerStatistics(ctx, seasonID, playerID, false)
}

func (s ResultService) playerStatistics(ctx context.Context, seasonID, playerID int64, public bool) (PlayerStatistics, error) {
	if seasonID <= 0 || playerID <= 0 {
		return PlayerStatistics{}, ErrPlayerNotFound
	}
	var result PlayerStatistics
	// Reuse the approval/lifecycle transaction lock so result and source reads agree.
	err := transact(ctx, s.store, func(tx Store) error {
		var err error
		scoped := NewResultServiceWithNow(tx, s.now)
		result, err = scoped.readPlayerStatistics(ctx, seasonID, playerID, public)
		return err
	})
	if err != nil {
		return PlayerStatistics{}, err
	}
	return result, nil
}

func (s ResultService) readPlayerStatistics(ctx context.Context, seasonID, playerID int64, public bool) (PlayerStatistics, error) {
	imports, ok := s.store.(ApprovalStore)
	if !ok {
		return PlayerStatistics{}, ErrImportStoreRequired
	}
	players, err := s.store.ListPlayersBySeason(ctx, seasonID)
	if err != nil {
		return PlayerStatistics{}, err
	}
	names := make(map[int64]string, len(players))
	for _, p := range players {
		names[p.ID] = p.PreferredName()
	}
	name, found := names[playerID]
	if !found {
		return PlayerStatistics{}, ErrPlayerNotFound
	}
	fixtures, err := s.store.ListFixturesBySeason(ctx, seasonID)
	if err != nil {
		return PlayerStatistics{}, err
	}
	results, err := s.store.ListResultsBySeason(ctx, seasonID)
	if err != nil {
		return PlayerStatistics{}, err
	}
	byFixture := make(map[int64]Result, len(results))
	for _, r := range results {
		byFixture[r.FixtureID] = r
	}
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		return PlayerStatistics{}, err
	}
	now := s.now()
	currentWeek := CurrentPublicWeek(fixtures, now, loc)
	got := PlayerStatistics{SchemaVersion: "autodarts.api.v1", SeasonID: seasonID, PlayerID: playerID, PreferredName: name, History: []PlayerStatisticsMatch{}, Throws: []PlayerStatisticsThrow{}}
	acc := playerStatisticsAccumulator{result: &got}
	for _, week := range GroupFixturesByWeek(fixtures, loc) {
		if public && (week.WeekNumber > currentWeek || now.Before(week.RevealAt)) {
			continue
		}
		for _, f := range week.Fixtures {
			if f.SeasonID != seasonID || (f.PlayerOneID != playerID && f.PlayerTwoID != playerID) {
				continue
			}
			// Gate each fixture too, even if a malformed schedule shares a week number.
			if public && CurrentPublicWeek([]Fixture{f}, now, loc) < f.WeekNumber {
				continue
			}
			r, exists := byFixture[f.ID]
			if !exists {
				continue
			}
			source, sourceErr := imports.ImportByFixture(ctx, f.ID)
			if sourceErr != nil && !errors.Is(sourceErr, ErrPendingResultNotFound) {
				return PlayerStatistics{}, sourceErr
			}
			if sourceErr == nil && !statisticsSourceMatches(source, f, r) {
				continue
			}
			history := PlayerStatisticsMatch{FixtureID: f.ID, WeekNumber: f.WeekNumber, ScheduledAt: f.ScheduledAt, OpponentID: f.PlayerTwoID, Won: r.WinnerID == playerID, LegsFor: r.PlayerOneLegs, LegsAgainst: r.PlayerTwoLegs, MatchAverage: r.PlayerOneAverage, DetailCoverage: "none"}
			if f.PlayerTwoID == playerID {
				history.OpponentID, history.LegsFor, history.LegsAgainst, history.MatchAverage = f.PlayerOneID, r.PlayerTwoLegs, r.PlayerOneLegs, r.PlayerTwoAverage
			}
			history.OpponentName = names[history.OpponentID]
			if sourceErr == nil {
				acc.addImport(source, f.ID)
				if source.Import.Detail != nil {
					history.DetailCoverage = source.Import.Detail.Coverage
				}
			}
			acc.addMatch(history)
		}
	}
	acc.finish()
	sort.Slice(got.History, func(i, j int) bool {
		if got.History[i].ScheduledAt.Equal(got.History[j].ScheduledAt) {
			return got.History[i].FixtureID < got.History[j].FixtureID
		}
		return got.History[i].ScheduledAt.After(got.History[j].ScheduledAt)
	})
	sort.Slice(got.Throws, func(i, j int) bool {
		a, b := got.Throws[i], got.Throws[j]
		if a.FixtureID != b.FixtureID {
			return a.FixtureID < b.FixtureID
		}
		if a.LegNumber != b.LegNumber {
			return a.LegNumber < b.LegNumber
		}
		if a.VisitNumber != b.VisitNumber {
			return a.VisitNumber < b.VisitNumber
		}
		return a.ThrowNumber < b.ThrowNumber
	})
	return got, nil
}

func statisticsSourceMatches(source ImportRecord, f Fixture, r Result) bool {
	if !source.Active || source.Pending.Status != PendingResultStatusConfirmed || source.Import.ReviewReason != "" || source.ResultID == nil || *source.ResultID != r.ID || source.FixtureID == nil || *source.FixtureID != f.ID || source.SeasonID == nil || *source.SeasonID != f.SeasonID {
		return false
	}
	if len(source.Import.Players) != 2 || len(source.Mapping) != 2 {
		return false
	}
	a, b := source.Import.Players[0], source.Import.Players[1]
	if source.Mapping[a.ID] == f.PlayerTwoID {
		a, b = b, a
	}
	if source.Mapping[a.ID] != f.PlayerOneID || source.Mapping[b.ID] != f.PlayerTwoID {
		return false
	}
	winner, err := WinnerIDForFixture(f, a.LegsWon, b.LegsWon)
	return err == nil && r.WinnerID == winner && r.PlayerOneLegs == a.LegsWon && r.PlayerTwoLegs == b.LegsWon && reflect.DeepEqual(r.PlayerOneAverage, a.Average()) && reflect.DeepEqual(r.PlayerTwoAverage, b.Average())
}
