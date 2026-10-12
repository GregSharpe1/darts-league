package league

import (
	"context"
	"errors"
	"maps"
	"strings"
)

type ImportSuggestion struct {
	SeasonID   *int64           `json:"season_id"`
	DivisionID *int64           `json:"division_id"`
	Mapping    map[string]int64 `json:"mapping"`
	FixtureID  *int64           `json:"fixture_id"`
}

func (s PendingResultService) suggestImport(ctx context.Context, record ImportRecord) (ImportSuggestion, error) {
	suggestion := ImportSuggestion{Mapping: map[string]int64{}}
	season, err := s.store.GetActiveSeason(ctx)
	if errors.Is(err, ErrSeasonNotFound) {
		return suggestion, nil
	}
	if err != nil {
		return suggestion, err
	}
	suggestion.SeasonID = &season.ID
	store, ok := s.store.(AutodartsMappingStore)
	if !ok {
		return suggestion, ErrImportStoreRequired
	}
	mappings, err := store.ListAutodartsMappings(ctx, season.ID)
	if err != nil {
		return suggestion, err
	}
	players, err := s.store.ListPlayersBySeason(ctx, season.ID)
	if err != nil {
		return suggestion, err
	}
	byName := map[string]Player{}
	byID := map[int64]Player{}
	for _, player := range players {
		byID[player.ID] = player
	}
	for _, mapping := range mappings {
		if player, ok := byID[mapping.PlayerID]; ok {
			byName[mapping.Username] = player
		}
	}
	if len(record.Import.Players) != 2 {
		return suggestion, nil
	}
	a, b := record.Import.Players[0], record.Import.Players[1]
	if strings.TrimSpace(a.DisplayName) == strings.TrimSpace(b.DisplayName) {
		return suggestion, nil
	}
	commonDivision := true
	for _, source := range record.Import.Players {
		player, found := byName[strings.TrimSpace(source.DisplayName)]
		if !found {
			continue
		}
		suggestion.Mapping[source.ID] = player.ID
		if player.DivisionID == nil || player.Status != PlayerStatusAssigned {
			commonDivision = false
		} else if suggestion.DivisionID == nil {
			suggestion.DivisionID = player.DivisionID
		} else if *suggestion.DivisionID != *player.DivisionID {
			commonDivision = false
		}
	}
	if !commonDivision {
		suggestion.DivisionID = nil
	}
	if record.Import.ExternalMatchID != "" {
		historyStore, ok := s.store.(ApprovalStore)
		if !ok {
			return suggestion, ErrImportStoreRequired
		}
		history, err := historyStore.ImportsBySource(ctx, record.Import.Source, record.Import.ExternalMatchID)
		if err != nil {
			return suggestion, err
		}
		for _, prior := range history {
			if prior.Pending.Status == PendingResultStatusConfirmed && prior.FixtureID == nil {
				return ImportSuggestion{SeasonID: &season.ID, Mapping: map[string]int64{}}, nil
			}
			if prior.FixtureID != nil && (prior.SeasonID == nil || *prior.SeasonID != season.ID || !maps.Equal(prior.Mapping, suggestion.Mapping)) {
				return ImportSuggestion{SeasonID: &season.ID, Mapping: map[string]int64{}}, nil
			}
		}
	}
	if len(suggestion.Mapping) != 2 || suggestion.DivisionID == nil {
		return suggestion, nil
	}
	one, two := suggestion.Mapping[a.ID], suggestion.Mapping[b.ID]
	if one == two {
		return suggestion, nil
	}
	fixtures, err := s.store.ListFixturesBySeason(ctx, season.ID)
	if err != nil {
		return suggestion, err
	}
	for _, fixture := range fixtures {
		if fixture.DivisionID != *suggestion.DivisionID || fixture.GameVariant != "501" || (fixture.LegsToWin != 2 && fixture.LegsToWin != 3) {
			continue
		}
		if !((fixture.PlayerOneID == one && fixture.PlayerTwoID == two) || (fixture.PlayerOneID == two && fixture.PlayerTwoID == one)) {
			continue
		}
		if ValidateResultScore(a.LegsWon, b.LegsWon, fixture.LegsToWin) != nil {
			continue
		}
		if suggestion.FixtureID != nil {
			suggestion.FixtureID = nil
			return suggestion, nil
		}
		id := fixture.ID
		suggestion.FixtureID = &id
	}
	return suggestion, nil
}
