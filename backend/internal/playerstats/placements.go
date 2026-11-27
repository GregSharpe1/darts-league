package playerstats

import (
	"context"
	"fmt"
	"slices"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/tournament"
)

type Service struct {
	League     league.Store
	Tournament tournament.Store
	Identities IdentityStore
	Now        func() time.Time
}

type Profile struct {
	Name           string `json:"name"`
	DisplayName    string `json:"display_name,omitempty"`
	Competition    string `json:"competition"`
	SeasonID       int64  `json:"season_id,omitempty"`
	PlayerID       int64  `json:"player_id,omitempty"`
	TournamentSlug string `json:"tournament_slug,omitempty"`
	EntryID        int64  `json:"entry_id,omitempty"`
}

type Placement struct {
	Position       int    `json:"position"`
	Competition    string `json:"competition"`
	Division       string `json:"division,omitempty"`
	TournamentSlug string `json:"tournament_slug,omitempty"`
	Tied           bool   `json:"tied,omitempty"`
}

type Placements struct {
	Name       string     `json:"name"`
	League     *Placement `json:"league"`
	Tournament *Placement `json:"tournament"`
}

type publishedEvent struct {
	tournament.Event
	entries []tournament.Entry
}

func (s Service) publishedEvents(ctx context.Context) ([]publishedEvent, error) {
	events, err := s.Tournament.ListTournaments(ctx)
	if err != nil {
		return nil, err
	}
	var published []publishedEvent
	for _, event := range events {
		if len(event.Bracket) == 0 {
			continue
		}
		entries, err := s.Tournament.ListTournamentEntries(ctx, event.Slug)
		if err != nil {
			return nil, err
		}
		// Legacy draws contain only labels. Resolve unambiguous entries in a copy
		// because the memory store shares the published bracket's backing array.
		event.Bracket = slices.Clone(event.Bracket)
		for i, round := range event.Bracket {
			event.Bracket[i] = slices.Clone(round)
		}
		event.Bracket.ResolvePublishedEntries(entries)
		publishedIDs := make(map[int64]bool)
		for _, match := range event.Bracket[0] {
			for _, slot := range match.Slots {
				publishedIDs[slot.EntryID] = !slot.Bye
			}
		}
		entries = slices.DeleteFunc(entries, func(entry tournament.Entry) bool { return !publishedIDs[entry.ID] })
		published = append(published, publishedEvent{Event: event, entries: entries})
	}
	return published, nil
}

func entryName(entry tournament.Entry) string {
	if entry.Nickname != "" {
		return entry.Nickname
	}
	return entry.DisplayName
}

func (s Service) Directory(ctx context.Context) ([]Profile, error) {
	seasons, err := s.League.ListSeasons(ctx)
	if err != nil {
		return nil, err
	}
	index, err := s.identityIndex(ctx)
	if err != nil {
		return nil, err
	}
	profiles := []Profile{}
	seen := make(map[int64]bool)
	for _, season := range seasons {
		if season.RegistrationOpen() {
			continue
		}
		players, err := s.League.ListPlayersBySeason(ctx, season.ID)
		if err != nil {
			return nil, err
		}
		for _, player := range players {
			if player.Status != league.PlayerStatusAssigned || player.DivisionID == nil {
				continue
			}
			key := index[entryKey{"league", player.ID}]
			if key == 0 {
				continue
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			profiles = append(profiles, Profile{Name: player.PreferredName(), DisplayName: player.DisplayName, Competition: season.Name, SeasonID: season.ID, PlayerID: player.ID})
		}
	}
	events, err := s.publishedEvents(ctx)
	if err != nil {
		return nil, err
	}
	for _, event := range events {
		for _, entry := range event.entries {
			key := index[entryKey{"tournament", entry.ID}]
			if key == 0 {
				continue
			}
			if seen[key] {
				continue
			}
			seen[key] = true
			profiles = append(profiles, Profile{Name: entryName(entry), Competition: event.Name, TournamentSlug: event.Slug, EntryID: entry.ID})
		}
	}
	return profiles, nil
}

func (s Service) ForLeaguePlayer(ctx context.Context, seasonID, playerID int64) (Placements, error) {
	players, err := s.League.ListPlayersBySeason(ctx, seasonID)
	if err != nil {
		return Placements{}, err
	}
	for _, player := range players {
		if player.ID == playerID {
			events, err := s.publishedEvents(ctx)
			if err != nil {
				return Placements{}, err
			}
			return s.best(ctx, entryKey{"league", player.ID}, player.PreferredName(), events)
		}
	}
	return Placements{}, league.ErrPlayerNotFound
}

func (s Service) ForTournamentPlayer(ctx context.Context, slug string, entryID int64) (Placements, error) {
	events, err := s.publishedEvents(ctx)
	if err != nil {
		return Placements{}, err
	}
	for _, event := range events {
		if event.Slug != slug {
			continue
		}
		for _, entry := range event.entries {
			if entry.ID == entryID {
				return s.best(ctx, entryKey{"tournament", entry.ID}, entryName(entry), events)
			}
		}
	}
	return Placements{}, league.ErrPlayerNotFound
}

func (s Service) identityIndex(ctx context.Context) (map[entryKey]int64, error) {
	snapshot, err := s.Identities.IdentitySnapshot(ctx)
	if err != nil {
		return nil, err
	}
	return snapshot.index(), nil
}

func (s Service) best(ctx context.Context, entry entryKey, name string, events []publishedEvent) (Placements, error) {
	index, err := s.identityIndex(ctx)
	if err != nil {
		return Placements{}, err
	}
	identityID := index[entry]
	if identityID == 0 {
		return Placements{}, league.ErrPlayerNotFound
	}
	bestLeague, err := s.bestLeague(ctx, index, identityID)
	if err != nil {
		return Placements{}, err
	}
	best := Placements{Name: name, League: bestLeague}
	for _, event := range events {
		for _, entry := range event.entries {
			if index[entryKey{"tournament", entry.ID}] != identityID {
				continue
			}
			position, tied := tournamentFinish(event.Bracket, entry.ID)
			if position > 0 && (best.Tournament == nil || position < best.Tournament.Position) {
				best.Tournament = &Placement{Position: position, Competition: event.Name, TournamentSlug: event.Slug, Tied: tied}
			}
		}
	}
	return best, nil
}

func tournamentFinish(bracket tournament.Bracket, entryID int64) (int, bool) {
	for i, round := range bracket {
		for _, match := range round {
			if match.WinnerID == 0 || match.Legs == nil {
				continue
			}
			if match.WinnerID == entryID && i == len(bracket)-1 {
				return 1, false
			}
			if match.WinnerID != entryID && (match.Slots[0].EntryID == entryID || match.Slots[1].EntryID == entryID) {
				eliminations := 0
				for _, candidate := range round {
					if !candidate.Slots[0].Bye && !candidate.Slots[1].Bye {
						eliminations++
					}
				}
				return len(round) + 1, eliminations > 1
			}
		}
	}
	return 0, false
}

func (s Service) bestLeague(ctx context.Context, index map[entryKey]int64, identityID int64) (*Placement, error) {
	seasons, err := s.League.ListSeasons(ctx)
	if err != nil {
		return nil, err
	}
	var best *Placement
	now := s.Now()
	for _, season := range seasons {
		if season.Status != league.SeasonStatusCompleted {
			continue
		}
		players, err := s.League.ListPlayersBySeason(ctx, season.ID)
		if err != nil {
			return nil, err
		}
		for _, player := range players {
			if player.Status != league.PlayerStatusAssigned || player.DivisionID == nil || index[entryKey{"league", player.ID}] != identityID {
				continue
			}
			finish, err := s.leagueFinish(ctx, season, player, players, now)
			if err != nil {
				return nil, err
			}
			if finish != nil && (best == nil || finish.Position < best.Position) {
				best = finish
			}
		}
	}
	return best, nil
}

func (s Service) leagueFinish(ctx context.Context, season league.Season, player league.Player, players []league.Player, now time.Time) (*Placement, error) {
	divisionID := *player.DivisionID
	fixtures, err := s.League.ListFixturesByDivision(ctx, divisionID)
	if err != nil {
		return nil, err
	}
	loc, err := time.LoadLocation(season.Timezone)
	if err != nil {
		return nil, err
	}
	// Closing a season can precede a future week's scheduled public reveal.
	for _, fixture := range fixtures {
		if league.CurrentPublicWeek([]league.Fixture{fixture}, now, loc) < fixture.WeekNumber {
			return nil, nil
		}
	}
	results, err := s.League.ListResultsByDivision(ctx, divisionID)
	if err != nil {
		return nil, err
	}
	rows := league.DivisionStandings(divisionID, players, fixtures, results, now, loc)
	for i, row := range rows {
		if row.PlayerID != player.ID || row.Played == 0 {
			continue
		}
		divisions, err := s.League.ListDivisionsBySeason(ctx, season.ID)
		if err != nil {
			return nil, err
		}
		for _, division := range divisions {
			if division.ID == divisionID {
				return &Placement{Position: i + 1, Competition: season.Name, Division: division.Name}, nil
			}
		}
		return nil, fmt.Errorf("division %d missing from season %d", divisionID, season.ID)
	}
	return nil, nil
}
