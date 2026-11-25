package tournament

import (
	"context"
	"slices"
	"sort"
	"sync"

	"github.com/greg/darts-league/backend/internal/league"
)

type MemoryStore struct {
	mu          sync.RWMutex
	events      map[string]Event
	entries     map[string][]Entry
	nextEntryID int64
}

func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		events:      make(map[string]Event),
		entries:     make(map[string][]Entry),
		nextEntryID: 1,
	}
}

func (s *MemoryStore) CreateTournament(_ context.Context, event Event) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.events[event.Slug]; exists {
		return Event{}, ErrDuplicateSlug
	}
	s.events[event.Slug] = event
	return event, nil
}

func (s *MemoryStore) GetTournament(_ context.Context, slug string) (Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	event, exists := s.events[slug]
	if !exists {
		return Event{}, ErrNotFound
	}
	return event, nil
}

func (s *MemoryStore) ListTournaments(_ context.Context) ([]Event, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	events := make([]Event, 0, len(s.events))
	for _, event := range s.events {
		events = append(events, event)
	}
	sort.Slice(events, func(i, j int) bool {
		if events[i].StartsOn == events[j].StartsOn {
			return events[i].Slug < events[j].Slug
		}
		return events[i].StartsOn > events[j].StartsOn
	})
	return events, nil
}

func (s *MemoryStore) SetTournamentRegistration(_ context.Context, slug string, open bool) (Event, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event, exists := s.events[slug]
	if !exists {
		return Event{}, ErrNotFound
	}
	event.RegistrationOpen = open
	s.events[slug] = event
	return event, nil
}

func (s *MemoryStore) CreateTournamentEntry(_ context.Context, slug string, entry Entry) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	event, exists := s.events[slug]
	if !exists {
		return Entry{}, ErrNotFound
	}
	if !event.RegistrationOpen {
		return Entry{}, league.ErrRegistrationClosed
	}
	name := league.NormalizeDisplayName(entry.DisplayName)
	for _, existing := range s.entries[slug] {
		if league.NormalizeDisplayName(existing.DisplayName) == name {
			return Entry{}, league.ErrDuplicatePlayerName
		}
	}
	entry.ID = s.nextEntryID
	s.nextEntryID++
	s.entries[slug] = append(s.entries[slug], entry)
	return entry, nil
}

func (s *MemoryStore) ListTournamentEntries(_ context.Context, slug string) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, exists := s.events[slug]; !exists {
		return nil, ErrNotFound
	}
	return append([]Entry{}, s.entries[slug]...), nil
}

func (s *MemoryStore) DeleteTournamentEntry(_ context.Context, slug string, id int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	entries := s.entries[slug]
	index := slices.IndexFunc(entries, func(entry Entry) bool { return entry.ID == id })
	if index < 0 {
		return ErrEntryNotFound
	}
	s.entries[slug] = slices.Delete(entries, index, index+1)
	return nil
}
