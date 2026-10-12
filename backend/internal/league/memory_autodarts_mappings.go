package league

import (
	"context"
	"sort"
)

func (s *MemoryStore) ListAutodartsMappings(_ context.Context, seasonID int64) ([]AutodartsMapping, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rows := []AutodartsMapping{}
	for id, username := range s.autodartsUsernames {
		if s.playersByID[id].SeasonID == seasonID {
			rows = append(rows, AutodartsMapping{PlayerID: id, Username: username})
		}
	}
	sort.Slice(rows, func(i, j int) bool { return rows[i].PlayerID < rows[j].PlayerID })
	return rows, nil
}

func (s *MemoryStore) SetAutodartsUsername(_ context.Context, seasonID, playerID int64, username *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	p, ok := s.playersByID[playerID]
	if !ok || p.SeasonID != seasonID {
		return ErrPlayerNotFound
	}
	if username == nil {
		delete(s.autodartsUsernames, playerID)
		return nil
	}
	for id, existing := range s.autodartsUsernames {
		if id != playerID && s.playersByID[id].SeasonID == seasonID && existing == *username {
			return ErrAutodartsUsernameConflict
		}
	}
	s.autodartsUsernames[playerID] = *username
	return nil
}
