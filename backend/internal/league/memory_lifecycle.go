package league

import "context"

func (s *MemoryStore) CloseSeason(_ context.Context, seasonID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeSeason.ID != seasonID || s.activeSeason.Status != SeasonStatusStarted {
		return ErrSeasonTransition
	}
	fixtures := []Fixture{}
	results := []Result{}
	for _, fixture := range s.fixturesByID {
		if fixture.SeasonID == seasonID {
			fixtures = append(fixtures, fixture)
		}
	}
	for _, result := range s.resultsByID {
		results = append(results, result)
	}
	if len(fixtures) == 0 || remainingFixtures(fixtures, results) != 0 {
		return ErrSeasonIncomplete
	}
	s.activeSeason.Status = SeasonStatusCompleted
	s.seasonsByID[seasonID] = s.activeSeason
	return nil
}

func (s *MemoryStore) CreateNextSeason(_ context.Context, seasonID int64, season Season) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeSeason.ID != seasonID || s.activeSeason.Status != SeasonStatusCompleted {
		return ErrSeasonTransition
	}
	season.ID = s.nextSeasonID
	s.nextSeasonID++
	s.seasonsByID[season.ID] = season
	s.activeSeason = season
	return nil
}

// Caller holds the store mutex for the entire mutation.
func (s *MemoryStore) writableFixture(fixtureID int64) error {
	fixture, ok := s.fixturesByID[fixtureID]
	if !ok {
		return ErrFixtureNotFound
	}
	if s.seasonsByID[fixture.SeasonID].Status == SeasonStatusCompleted {
		return ErrSeasonCompleted
	}
	return nil
}
