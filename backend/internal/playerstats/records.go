package playerstats

import (
	"context"
	"errors"
	"slices"
	"strings"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/tournament"
)

type PlayerRecord struct {
	Profile      Profile `json:"profile"`
	Rank         int     `json:"rank"`
	Value        float64 `json:"value"`
	KnownMatches int     `json:"known_matches"`
	Played       int     `json:"played"`
}

type recordMetric struct {
	value   float64
	matches int
}

func (m *recordMetric) add(value int) {
	m.value += float64(value)
	m.matches++
}

func (m *recordMetric) maximum(value float64) {
	m.value = max(m.value, value)
	m.matches++
}

func (m *recordMetric) minimum(value int) {
	if m.matches == 0 || float64(value) < m.value {
		m.value = float64(value)
	}
	m.matches++
}

type recordAccumulator struct {
	profile                                            Profile
	played                                             int
	wins, maximums, checkouts, bullseyes, average, leg recordMetric
}

type tournamentRecordStore interface {
	TournamentRecordImport(context.Context, string, int) (league.ImportRecord, error)
}

func (s Service) Records(ctx context.Context) (map[string][]PlayerRecord, error) {
	index, err := s.identityIndex(ctx)
	if err != nil {
		return nil, err
	}
	events, err := s.publishedEvents(ctx)
	if err != nil {
		return nil, err
	}
	profiles, err := s.directoryFromEvents(ctx, index, events)
	if err != nil {
		return nil, err
	}
	players := make(map[int64]*recordAccumulator, len(profiles))
	for _, profile := range profiles {
		players[profile.IdentityID] = &recordAccumulator{profile: profile}
	}

	seasons, err := s.League.ListSeasons(ctx)
	if err != nil {
		return nil, err
	}
	now := s.Now()
	for _, season := range seasons {
		if season.RegistrationOpen() {
			continue
		}
		err := league.VisitPublicRecordMatches(ctx, s.League, season.ID, now, func(fixture league.Fixture, result league.Result, source *league.ImportRecord) {
			players[index[entryKey{"league", fixture.PlayerOneID}]].addMatch(fixture.PlayerOneID, fixture.PlayerOneID == result.WinnerID, result.PlayerOneAverage, source)
			players[index[entryKey{"league", fixture.PlayerTwoID}]].addMatch(fixture.PlayerTwoID, fixture.PlayerTwoID == result.WinnerID, result.PlayerTwoAverage, source)
		})
		if err != nil {
			return nil, err
		}
	}

	imports, _ := s.Tournament.(tournamentRecordStore)
	for _, event := range events {
		for _, round := range event.Bracket {
			for _, match := range round {
				if match.Legs == nil || match.WinnerID == 0 || match.Slots[0].Bye || match.Slots[1].Bye {
					continue
				}
				var source *league.ImportRecord
				if imports != nil {
					record, err := imports.TournamentRecordImport(ctx, event.Slug, match.Number)
					switch {
					case errors.Is(err, league.ErrPendingResultNotFound):
					case err != nil:
						return nil, err
					case tournamentEvidenceMatches(record, event.Slug, match):
						source = &record
					}
				}
				for _, slot := range match.Slots {
					players[index[entryKey{"tournament", slot.EntryID}]].addMatch(slot.EntryID, slot.EntryID == match.WinnerID, nil, source)
				}
			}
		}
	}

	return map[string][]PlayerRecord{
		"wins":           rankRecords(players, false, func(p *recordAccumulator) recordMetric { return p.wins }),
		"total_180":      rankRecords(players, false, func(p *recordAccumulator) recordMetric { return p.maximums }),
		"highest_finish": rankRecords(players, false, func(p *recordAccumulator) recordMetric { return p.checkouts }),
		"bullseyes":      rankRecords(players, false, func(p *recordAccumulator) recordMetric { return p.bullseyes }),
		"match_average":  rankRecords(players, false, func(p *recordAccumulator) recordMetric { return p.average }),
		"best_leg":       rankRecords(players, true, func(p *recordAccumulator) recordMetric { return p.leg }),
	}, nil
}

func tournamentEvidenceMatches(source league.ImportRecord, slug string, match tournament.BracketMatch) bool {
	if !source.Active || source.Pending.Status != league.PendingResultStatusConfirmed || source.Import.ReviewReason != "" ||
		source.Approval == nil || source.Approval.TournamentSlug != slug || source.Approval.MatchNumber != match.Number ||
		len(source.Import.Players) != 2 || len(source.Mapping) != 2 || match.Legs == nil {
		return false
	}
	first, second := source.Import.Players[0], source.Import.Players[1]
	left, right := source.Mapping[first.ID], source.Mapping[second.ID]
	if left == 0 || right == 0 || left == right {
		return false
	}
	if left != match.Slots[0].EntryID {
		first, second, left, right = second, first, right, left
	}
	return left == match.Slots[0].EntryID && right == match.Slots[1].EntryID &&
		first.LegsWon == match.Legs[0] && second.LegsWon == match.Legs[1] &&
		((left == match.WinnerID && first.LegsWon > second.LegsWon) || (right == match.WinnerID && second.LegsWon > first.LegsWon))
}

func rankRecords(players map[int64]*recordAccumulator, ascending bool, metric func(*recordAccumulator) recordMetric) []PlayerRecord {
	rows := []PlayerRecord{}
	for _, player := range players {
		value := metric(player)
		if value.matches > 0 && value.value > 0 {
			rows = append(rows, PlayerRecord{Profile: player.profile, Value: value.value, KnownMatches: value.matches, Played: player.played})
		}
	}
	slices.SortFunc(rows, func(a, b PlayerRecord) int {
		if a.Value != b.Value {
			if (a.Value < b.Value) == ascending {
				return -1
			}
			return 1
		}
		if name := strings.Compare(strings.ToLower(a.Profile.Name), strings.ToLower(b.Profile.Name)); name != 0 {
			return name
		}
		return compareID(a.Profile.IdentityID, b.Profile.IdentityID)
	})
	rank := 0
	for i := range rows {
		if i == 0 || rows[i].Value != rows[i-1].Value {
			rank = i + 1
		}
		if rank > 3 {
			return rows[:i]
		}
		rows[i].Rank = rank
	}
	return rows
}
