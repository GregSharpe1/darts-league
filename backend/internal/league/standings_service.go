package league

import (
	"context"
	"time"
)

func (s ResultService) Standings(ctx context.Context, divisionSlug string) ([]StandingRow, error) {
	season, err := s.store.GetActiveSeason(ctx)
	if err != nil {
		return nil, err
	}
	division, err := s.store.GetDivisionBySlug(ctx, season.ID, divisionSlug)
	if err != nil {
		return nil, err
	}
	players, err := s.store.ListPlayersBySeason(ctx, season.ID)
	if err != nil {
		return nil, err
	}
	fixtures, err := s.store.ListFixturesByDivision(ctx, division.ID)
	if err != nil {
		return nil, err
	}
	results, err := s.store.ListResultsByDivision(ctx, division.ID)
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(season.Timezone)
	if err != nil {
		return nil, err
	}
	visible := make(map[int64]bool)
	now := s.now()
	for _, week := range GroupFixturesByWeek(fixtures, loc) {
		if !now.Before(week.RevealAt) {
			for _, fixture := range week.Fixtures {
				visible[fixture.ID] = true
			}
		}
	}
	publicResults := make([]Result, 0, len(results))
	for _, result := range results {
		if visible[result.FixtureID] {
			publicResults = append(publicResults, result)
		}
	}
	divisionPlayers := make([]Player, 0)
	for _, player := range players {
		if player.DivisionID != nil && *player.DivisionID == division.ID && player.Status == PlayerStatusAssigned {
			divisionPlayers = append(divisionPlayers, player)
		}
	}
	return BuildStandings(divisionPlayers, fixtures, publicResults), nil
}
