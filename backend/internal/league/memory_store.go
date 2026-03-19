package league

import (
	"context"
	"errors"
	"sync"
)

var ErrPlayerNotFound = errors.New("player not found")
var ErrResultNotFound = errors.New("result not found")

type MemoryStore struct {
	mu            sync.RWMutex
	activeSeason  Season
	playersByID   map[int64]Player
	fixturesByID  map[int64]Fixture
	resultsByID   map[int64]Result
	auditByID     map[int64]AuditLogEntry
	nextPlayerID  int64
	nextFixtureID int64
	nextResultID  int64
	nextAuditID   int64
	nextSeasonID  int64
}

func NewMemoryStore() *MemoryStore {
	season := NewSeason("MVP Season")
	season.ID = 1

	return &MemoryStore{
		activeSeason:  season,
		playersByID:   make(map[int64]Player),
		fixturesByID:  make(map[int64]Fixture),
		resultsByID:   make(map[int64]Result),
		auditByID:     make(map[int64]AuditLogEntry),
		nextPlayerID:  1,
		nextFixtureID: 1,
		nextResultID:  1,
		nextAuditID:   1,
		nextSeasonID:  2,
	}
}

func (s *MemoryStore) EnsureActiveSeason(_ context.Context, season Season) (Season, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeSeason.ID != 0 {
		return s.activeSeason, nil
	}
	if season.ID == 0 {
		season.ID = s.nextSeasonID
		s.nextSeasonID++
	}
	s.activeSeason = season
	return season, nil
}

func (s *MemoryStore) GetActiveSeason(_ context.Context) (Season, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.activeSeason.ID == 0 {
		return Season{}, ErrSeasonNotFound
	}

	return s.activeSeason, nil
}

func (s *MemoryStore) ListPlayersBySeason(_ context.Context, seasonID int64) ([]Player, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	players := make([]Player, 0, len(s.playersByID))
	for _, player := range s.playersByID {
		if player.SeasonID == seasonID {
			players = append(players, player)
		}
	}

	return players, nil
}

func (s *MemoryStore) CreatePlayer(_ context.Context, player Player) (Player, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	player.ID = s.nextPlayerID
	s.nextPlayerID++
	s.playersByID[player.ID] = player

	return player, nil
}

func (s *MemoryStore) ListFixturesBySeason(_ context.Context, seasonID int64) ([]Fixture, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fixtures := make([]Fixture, 0, len(s.fixturesByID))
	for _, fixture := range s.fixturesByID {
		if fixture.SeasonID == seasonID {
			fixtures = append(fixtures, fixture)
		}
	}

	return fixtures, nil
}

func (s *MemoryStore) CreateFixtures(_ context.Context, fixtures []Fixture) ([]Fixture, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	created := make([]Fixture, len(fixtures))
	for i, fixture := range fixtures {
		fixture.ID = s.nextFixtureID
		s.nextFixtureID++
		s.fixturesByID[fixture.ID] = fixture
		created[i] = fixture
	}

	return created, nil
}

func (s *MemoryStore) GetFixture(_ context.Context, fixtureID int64) (Fixture, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	fixture, ok := s.fixturesByID[fixtureID]
	if !ok {
		return Fixture{}, ErrFixtureNotFound
	}
	return fixture, nil
}

func (s *MemoryStore) ListResultsBySeason(_ context.Context, seasonID int64) ([]Result, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	results := make([]Result, 0, len(s.resultsByID))
	for _, result := range s.resultsByID {
		fixture, ok := s.fixturesByID[result.FixtureID]
		if ok && fixture.SeasonID == seasonID {
			results = append(results, result)
		}
	}
	return results, nil
}

func (s *MemoryStore) CreateResult(_ context.Context, result Result) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	result.ID = s.nextResultID
	s.nextResultID++
	s.resultsByID[result.ID] = result
	return result, nil
}

func (s *MemoryStore) GetResultByFixture(_ context.Context, fixtureID int64) (Result, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, result := range s.resultsByID {
		if result.FixtureID == fixtureID {
			return result, nil
		}
	}
	return Result{}, ErrResultNotFound
}

func (s *MemoryStore) UpdateResult(_ context.Context, result Result) (Result, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.resultsByID[result.ID]; !ok {
		return Result{}, ErrResultNotFound
	}
	s.resultsByID[result.ID] = result
	return result, nil
}

func (s *MemoryStore) DeleteResultByFixture(_ context.Context, fixtureID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for id, result := range s.resultsByID {
		if result.FixtureID == fixtureID {
			delete(s.resultsByID, id)
			return nil
		}
	}
	return ErrResultNotFound
}

func (s *MemoryStore) CreateAuditLog(_ context.Context, entry AuditLogEntry) (AuditLogEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	entry.ID = s.nextAuditID
	s.nextAuditID++
	s.auditByID[entry.ID] = entry
	return entry, nil
}

func (s *MemoryStore) ListAuditLogsBySeason(_ context.Context, seasonID int64) ([]AuditLogEntry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entries := make([]AuditLogEntry, 0, len(s.auditByID))
	for _, entry := range s.auditByID {
		fixture, ok := s.fixturesByID[entry.FixtureID]
		if ok && fixture.SeasonID == seasonID {
			entries = append(entries, entry)
		}
	}
	return entries, nil
}

func (s *MemoryStore) DeletePlayer(_ context.Context, seasonID, playerID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	player, ok := s.playersByID[playerID]
	if !ok || player.SeasonID != seasonID {
		return ErrPlayerNotFound
	}

	delete(s.playersByID, playerID)
	return nil
}

func (s *MemoryStore) UpsertSeason(_ context.Context, season Season) (Season, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if season.ID == 0 {
		season.ID = s.nextSeasonID
		s.nextSeasonID++
	}

	s.activeSeason = season
	return season, nil
}
