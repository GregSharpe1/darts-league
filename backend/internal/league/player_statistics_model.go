package league

import (
	"time"

	"github.com/greg/darts-league/backend/internal/autodarts"
)

type PlayerStatistics struct {
	SchemaVersion             string                   `json:"schema_version"`
	SeasonID                  int64                    `json:"season_id"`
	PlayerID                  int64                    `json:"player_id"`
	PreferredName             string                   `json:"preferred_name"`
	Played                    int                      `json:"played"`
	Won                       int                      `json:"won"`
	Lost                      int                      `json:"lost"`
	LegsFor                   int                      `json:"legs_for"`
	LegsAgainst               int                      `json:"legs_against"`
	LegDifference             int                      `json:"leg_difference"`
	Points                    int                      `json:"points"`
	MatchAverageMean          *float64                 `json:"match_average_mean"`
	DartWeightedAverage       *float64                 `json:"dart_weighted_average"`
	FirstNineMatchAverageMean *float64                 `json:"first_nine_match_average_mean"`
	CheckoutHits              *int                     `json:"checkout_hits"`
	CheckoutAttempts          *int                     `json:"checkout_attempts"`
	CheckoutPercentage        *float64                 `json:"checkout_percentage"`
	Total180                  *int                     `json:"total_180"`
	BestLegDarts              *int                     `json:"best_leg_darts"`
	HighestFinish             *int                     `json:"highest_finish"`
	Coverage                  PlayerStatisticsCoverage `json:"coverage"`
	History                   []PlayerStatisticsMatch  `json:"history"`
	Throws                    []PlayerStatisticsThrow  `json:"throws"`
}

// Metric counts are known matches, not darts, legs, or all scheduled fixtures.
type PlayerStatisticsCoverage struct {
	EligibleMatches           int      `json:"eligible_matches"`
	MatchesWithDetail         int      `json:"matches_with_detail"`
	MatchAverageMean          int      `json:"match_average_mean"`
	DartWeightedAverage       int      `json:"dart_weighted_average"`
	FirstNineMatchAverageMean int      `json:"first_nine_match_average_mean"`
	Checkout                  int      `json:"checkout"`
	Total180                  int      `json:"total_180"`
	BestLegDarts              int      `json:"best_leg_darts"`
	HighestFinish             int      `json:"highest_finish"`
	MatchesWithRecordedThrows int      `json:"matches_with_recorded_throws"`
	MatchesWithPositions      int      `json:"matches_with_positions"`
	RecordedThrows            int      `json:"recorded_throws"`
	KnownPositions            int      `json:"known_positions"`
	PlottablePositions        int      `json:"plottable_positions"`
	PositionFraction          *float64 `json:"position_fraction"`
}

type PlayerStatisticsMatch struct {
	FixtureID      int64     `json:"fixture_id"`
	WeekNumber     int       `json:"week_number"`
	ScheduledAt    time.Time `json:"scheduled_at"`
	OpponentID     int64     `json:"opponent_id"`
	OpponentName   string    `json:"opponent_name"`
	Won            bool      `json:"won"`
	LegsFor        int       `json:"legs_for"`
	LegsAgainst    int       `json:"legs_against"`
	MatchAverage   *float64  `json:"match_average"`
	DetailCoverage string    `json:"detail_coverage"`
}

// Positions retain source geometry; no physical-board adapter is verified yet.
type PlayerStatisticsThrow struct {
	FixtureID   int64               `json:"fixture_id"`
	LegNumber   int                 `json:"leg_number"`
	VisitNumber int                 `json:"visit_number"`
	ThrowNumber int                 `json:"throw_number"`
	Segment     autodarts.Segment   `json:"segment"`
	EntryType   string              `json:"entry_type"`
	Position    *autodarts.Position `json:"position"`
	Plottable   bool                `json:"plottable"`
}
