package playerstats

import (
	"context"
	"errors"
	"slices"
	"strconv"
	"strings"
	"sync"

	"github.com/greg/darts-league/backend/internal/league"
)

var (
	ErrIdentityNotFound = errors.New("a player profile has changed; refresh and try again")
	ErrIdentityOverlap  = errors.New("both profiles already have an entry in the same competition")
	ErrIdentityDecision = errors.New("choose two different profiles and a valid decision")
)

type IdentityEntry struct {
	RequestedIdentityID int64  `json:"-"`
	IdentityID          int64  `json:"identity_id"`
	Kind                string `json:"kind"`
	EntryID             int64  `json:"entry_id"`
	Scope               string `json:"scope"`
	Competition         string `json:"competition"`
	DisplayName         string `json:"display_name"`
	Nickname            string `json:"nickname"`
}

type IdentityPair struct {
	Left  int64 `json:"left"`
	Right int64 `json:"right"`
}

func pair(a, b int64) IdentityPair { return IdentityPair{min(a, b), max(a, b)} }

type IdentitySnapshot struct {
	Entries   []IdentityEntry
	Separated map[IdentityPair]bool
}

type IdentityDecision struct {
	IdentityPair
	Action string `json:"action"`
	Actor  string `json:"-"`
}

type IdentityStore interface {
	IdentitySnapshot(context.Context) (IdentitySnapshot, error)
	DecideIdentity(context.Context, IdentityDecision) error
}

type IdentitySuggestion struct {
	IdentityPair
	Signals []string `json:"signals"`
	CanLink bool     `json:"can_link"`
}

type IdentityReview struct {
	Entries     []IdentityEntry      `json:"entries"`
	Suggestions []IdentitySuggestion `json:"suggestions"`
}

type entryKey struct {
	kind string
	id   int64
}

func (s IdentitySnapshot) index() map[entryKey]int64 {
	index := make(map[entryKey]int64, len(s.Entries))
	for _, entry := range s.Entries {
		index[entryKey{entry.Kind, entry.EntryID}] = entry.IdentityID
	}
	return index
}

type competitionKey struct{ kind, scope string }

func (s IdentitySnapshot) groups() map[int64]map[competitionKey]bool {
	groups := make(map[int64]map[competitionKey]bool)
	for _, entry := range s.Entries {
		if groups[entry.IdentityID] == nil {
			groups[entry.IdentityID] = make(map[competitionKey]bool)
		}
		groups[entry.IdentityID][competitionKey{entry.Kind, entry.Scope}] = true
	}
	return groups
}

func overlaps(a, b map[competitionKey]bool) bool {
	for scope := range a {
		if b[scope] {
			return true
		}
	}
	return false
}

func (s IdentitySnapshot) ValidateDecision(d IdentityDecision) error {
	if d.Left <= 0 || d.Right <= 0 || d.Left == d.Right || (d.Action != "link" && d.Action != "separate") {
		return ErrIdentityDecision
	}
	groups := s.groups()
	if groups[d.Left] == nil || groups[d.Right] == nil {
		return ErrIdentityNotFound
	}
	if d.Action == "link" && overlaps(groups[d.Left], groups[d.Right]) {
		return ErrIdentityOverlap
	}
	return nil
}

const selectedProfileSignal = "Selected existing profile at sign-up"

func (s IdentitySnapshot) Review() IdentityReview {
	groups := s.groups()
	byName := make(map[string][]IdentityEntry)
	for _, entry := range s.Entries {
		display := league.NormalizeDisplayName(entry.DisplayName)
		nickname := league.NormalizeDisplayName(entry.Nickname)
		byName[display] = append(byName[display], entry)
		if nickname != "" && nickname != display {
			byName[nickname] = append(byName[nickname], entry)
		}
	}
	matches := make(map[IdentityPair]map[string]bool)
	for _, entry := range s.Entries {
		key := pair(entry.IdentityID, entry.RequestedIdentityID)
		if entry.RequestedIdentityID != 0 && key.Left != key.Right && groups[entry.RequestedIdentityID] != nil && !s.Separated[key] {
			matches[key] = map[string]bool{selectedProfileSignal: true}
		}
	}
	for name, entries := range byName {
		for i, a := range entries {
			for _, b := range entries[i+1:] {
				key := pair(a.IdentityID, b.IdentityID)
				if key.Left == key.Right || s.Separated[key] {
					continue
				}
				if matches[key] == nil {
					matches[key] = make(map[string]bool)
				}
				signal := "Shared name or nickname"
				if name == league.NormalizeDisplayName(a.DisplayName) && name == league.NormalizeDisplayName(b.DisplayName) {
					signal = "Same display name"
				}
				matches[key][signal] = true
			}
		}
	}
	review := IdentityReview{Entries: s.Entries, Suggestions: []IdentitySuggestion{}}
	for key, signals := range matches {
		labels := make([]string, 0, len(signals))
		for label := range signals {
			labels = append(labels, label)
		}
		slices.Sort(labels)
		review.Suggestions = append(review.Suggestions, IdentitySuggestion{key, labels, !overlaps(groups[key.Left], groups[key.Right])})
	}
	slices.SortFunc(review.Suggestions, func(a, b IdentitySuggestion) int {
		selectedA, selectedB := slices.Contains(a.Signals, selectedProfileSignal), slices.Contains(b.Signals, selectedProfileSignal)
		if selectedA != selectedB {
			if selectedA {
				return -1
			}
			return 1
		}
		if a.Left != b.Left {
			return compareID(a.Left, b.Left)
		}
		return compareID(a.Right, b.Right)
	})
	return review
}

func compareID(a, b int64) int {
	if a < b {
		return -1
	}
	if a > b {
		return 1
	}
	return 0
}

// MemoryIdentities uses entry IDs to supply stable identities for the in-memory
// backend. The source returns a fresh roster on each read.
type MemoryIdentities struct {
	Source    func(context.Context) ([]IdentityEntry, error)
	mu        sync.Mutex
	linked    map[int64]int64
	separated map[IdentityPair]bool
}

func (m *MemoryIdentities) snapshot(ctx context.Context) (IdentitySnapshot, error) {
	entries, err := m.Source(ctx)
	if err != nil {
		return IdentitySnapshot{}, err
	}
	for i, entry := range entries {
		id := entry.EntryID * 2
		if entry.Kind == "tournament" {
			id++
		}
		if entry.IdentityID > 0 {
			id = entry.IdentityID
		}
		if linked := m.linked[id]; linked != 0 {
			id = linked
		}
		entries[i].IdentityID = id
		if linked := m.linked[entry.RequestedIdentityID]; linked != 0 {
			entries[i].RequestedIdentityID = linked
		}
	}
	return IdentitySnapshot{Entries: entries, Separated: m.separated}, nil
}

func (m *MemoryIdentities) IdentitySnapshot(ctx context.Context) (IdentitySnapshot, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot, err := m.snapshot(ctx)
	if err != nil {
		return IdentitySnapshot{}, err
	}
	separated := make(map[IdentityPair]bool, len(snapshot.Separated))
	for key := range snapshot.Separated {
		separated[key] = true
	}
	snapshot.Separated = separated
	return snapshot, nil
}

func (m *MemoryIdentities) DecideIdentity(ctx context.Context, d IdentityDecision) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	snapshot, err := m.snapshot(ctx)
	if err != nil {
		return err
	}
	if err := snapshot.ValidateDecision(d); err != nil {
		return err
	}
	key := pair(d.Left, d.Right)
	if m.separated == nil {
		m.separated = make(map[IdentityPair]bool)
	}
	if d.Action == "separate" {
		m.separated[key] = true
		return nil
	}
	if m.linked == nil {
		m.linked = make(map[int64]int64)
	}
	for id, linked := range m.linked {
		if linked == key.Right {
			m.linked[id] = key.Left
		}
	}
	m.linked[key.Right] = key.Left
	for previous := range m.separated {
		if previous.Left == key.Right || previous.Right == key.Right {
			delete(m.separated, previous)
			other := previous.Left
			if other == key.Right {
				other = previous.Right
			}
			if other != key.Left {
				m.separated[pair(key.Left, other)] = true
			}
		}
	}
	return nil
}

func (s Service) IdentityEntries(ctx context.Context) ([]IdentityEntry, error) {
	entries := []IdentityEntry{}
	seasons, err := s.League.ListSeasons(ctx)
	if err != nil {
		return nil, err
	}
	for _, season := range seasons {
		players, err := s.League.ListPlayersBySeason(ctx, season.ID)
		if err != nil {
			return nil, err
		}
		for _, p := range players {
			entries = append(entries, IdentityEntry{Kind: "league", EntryID: p.ID, IdentityID: p.ReturningIdentityID, Scope: strconv.FormatInt(season.ID, 10), Competition: season.Name, DisplayName: p.DisplayName, Nickname: p.Nickname})
		}
	}
	events, err := s.Tournament.ListTournaments(ctx)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		roster, err := s.Tournament.ListTournamentEntries(ctx, event.Slug)
		if err != nil {
			return nil, err
		}
		for _, entry := range roster {
			entries = append(entries, IdentityEntry{Kind: "tournament", EntryID: entry.ID, IdentityID: entry.ReturningIdentityID, Scope: event.Slug, Competition: event.Name, DisplayName: entry.DisplayName, Nickname: strings.TrimSpace(entry.Nickname)})
		}
	}
	return entries, nil
}
