package league

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

type MatchDetail struct {
	SchemaVersion string               `json:"schema_version"`
	FixtureID     int64                `json:"fixture_id"`
	SeasonID      int64                `json:"season_id"`
	PlayedAt      *string              `json:"playedAt"`
	Players       [2]MatchDetailPlayer `json:"players"`
	Detail        *autodarts.Detail    `json:"detail"`
	Coverage      MatchCoverage        `json:"coverage"`
}

type MatchDetailPlayer struct {
	LeaguePlayerID int64             `json:"league_player_id"`
	PreferredName  string            `json:"preferred_name"`
	LegsWon        int               `json:"legs_won"`
	Stats          *MatchDetailStats `json:"stats"`
}

// Explicit nullable API fields keep unavailable statistics distinct from zero.
type MatchDetailStats struct {
	MatchAverage     *float64 `json:"match_average"`
	PointsScored     *int     `json:"points_scored"`
	DartsThrown      *int     `json:"darts_thrown"`
	CheckoutHits     *int     `json:"checkout_hits"`
	CheckoutAttempts *int     `json:"checkout_attempts"`
	FirstNineAverage *float64 `json:"first_nine_average"`
	AverageUntil170  *float64 `json:"average_until_170"`
	HighestFinish    *int     `json:"highest_finish"`
	Total180         *int     `json:"total_180"`
	Less60           *int     `json:"less_60"`
	Plus60           *int     `json:"plus_60"`
	Plus100          *int     `json:"plus_100"`
	Plus140          *int     `json:"plus_140"`
	Plus170          *int     `json:"plus_170"`
}

type MatchCoverage struct {
	RecordedThrows   int      `json:"recorded_throws"`
	KnownPositions   int      `json:"known_positions"`
	PositionFraction *float64 `json:"position_fraction"`
}

func (s FixtureService) MatchDetail(ctx context.Context, fixtureID int64, admin bool) (MatchDetail, error) {
	var response MatchDetail
	// Share the approval/lifecycle lock so a correction cannot mix old evidence with a new result.
	err := transact(ctx, s.store, func(store Store) error {
		var err error
		response, err = s.matchDetail(ctx, store, fixtureID, admin)
		return err
	})
	return response, err
}

func (s FixtureService) matchDetail(ctx context.Context, store Store, fixtureID int64, admin bool) (MatchDetail, error) {
	f, err := store.GetFixture(ctx, fixtureID)
	if err != nil {
		return MatchDetail{}, err
	}
	loc, err := time.LoadLocation("Europe/London")
	if err != nil {
		return MatchDetail{}, err
	}
	if !admin && s.now().Before(GroupFixturesByWeek([]Fixture{f}, loc)[0].RevealAt) {
		return MatchDetail{}, ErrFixtureNotFound
	}
	r, err := store.GetResultByFixture(ctx, fixtureID)
	if errors.Is(err, ErrResultNotFound) {
		return MatchDetail{}, ErrFixtureNotFound
	}
	if err != nil {
		return MatchDetail{}, err
	}
	players, err := store.ListPlayersBySeason(ctx, f.SeasonID)
	if err != nil {
		return MatchDetail{}, err
	}
	names := make(map[int64]string, len(players))
	for _, p := range players {
		names[p.ID] = p.PreferredName()
	}
	response := MatchDetail{SchemaVersion: "autodarts.api.v1", FixtureID: f.ID, SeasonID: f.SeasonID,
		Players: [2]MatchDetailPlayer{
			{LeaguePlayerID: f.PlayerOneID, PreferredName: names[f.PlayerOneID], LegsWon: r.PlayerOneLegs},
			{LeaguePlayerID: f.PlayerTwoID, PreferredName: names[f.PlayerTwoID], LegsWon: r.PlayerTwoLegs},
		},
	}
	for i, avg := range []*float64{r.PlayerOneAverage, r.PlayerTwoAverage} {
		if avg != nil {
			response.Players[i].Stats = &MatchDetailStats{MatchAverage: avg}
		}
	}
	imports, ok := store.(ApprovalStore)
	if !ok {
		return response, nil
	}
	source, err := imports.ImportByResult(ctx, r.ID)
	if errors.Is(err, ErrPendingResultNotFound) {
		return response, nil
	}
	if err != nil {
		return MatchDetail{}, err
	}
	if !source.Active || source.Pending.Status != PendingResultStatusConfirmed || source.FixtureID == nil || *source.FixtureID != f.ID || source.SeasonID == nil || *source.SeasonID != f.SeasonID || source.ResultID == nil || *source.ResultID != r.ID {
		return response, nil
	}
	mapped, err := source.MappedImport()
	if err != nil {
		return MatchDetail{}, err
	}
	response.PlayedAt, response.Detail = mapped.PlayedAt, mapped.Detail
	for i, player := range response.Players {
		for _, p := range mapped.Players {
			if p.ID == strconv.FormatInt(player.LeaguePlayerID, 10) && p.Stats != nil {
				s := p.Stats
				response.Players[i].Stats = &MatchDetailStats{
					MatchAverage: s.MatchAverage, PointsScored: s.PointsScored, DartsThrown: s.DartsThrown,
					CheckoutHits: s.CheckoutHits, CheckoutAttempts: s.CheckoutAttempts,
					FirstNineAverage: s.FirstNineAverage, AverageUntil170: s.AverageUntil170,
					HighestFinish: s.HighestFinish, Total180: s.Total180, Less60: s.Less60,
					Plus60: s.Plus60, Plus100: s.Plus100, Plus140: s.Plus140, Plus170: s.Plus170,
				}
			}
		}
	}
	if response.Detail != nil {
		for _, leg := range response.Detail.Legs {
			for _, visit := range leg.Visits {
				for _, dart := range visit.Throws {
					response.Coverage.RecordedThrows++
					if dart.Position != nil {
						response.Coverage.KnownPositions++
					}
				}
			}
		}
	}
	if response.Coverage.RecordedThrows > 0 {
		fraction := float64(response.Coverage.KnownPositions) / float64(response.Coverage.RecordedThrows)
		response.Coverage.PositionFraction = &fraction
	}
	return response, nil
}
