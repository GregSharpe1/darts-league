package league

import "github.com/greg/darts-league/backend/internal/autodarts"

type playerStatisticsAccumulator struct {
	result              *PlayerStatistics
	averages, firstNine float64
	points, darts       int
}

func (a *playerStatisticsAccumulator) addMatch(m PlayerStatisticsMatch) {
	r := a.result
	r.Played++
	r.Coverage.EligibleMatches++
	r.LegsFor += m.LegsFor
	r.LegsAgainst += m.LegsAgainst
	if m.Won {
		r.Won++
		r.Points += 2
	} else {
		r.Lost++
	}
	if m.MatchAverage != nil {
		a.averages += *m.MatchAverage
		r.Coverage.MatchAverageMean++
	}
	r.History = append(r.History, m)
}

func (a *playerStatisticsAccumulator) addImport(record ImportRecord, fixtureID int64) {
	r := a.result
	var sourceID string
	var stats *autodarts.Stats
	for _, player := range record.Import.Players {
		if record.Mapping[player.ID] == r.PlayerID {
			sourceID, stats = player.ID, player.Stats
		}
	}
	var high *int
	if stats != nil {
		if stats.FirstNineAverage != nil {
			a.firstNine += *stats.FirstNineAverage
			r.Coverage.FirstNineMatchAverageMean++
		}
		if stats.CheckoutHits != nil && stats.CheckoutAttempts != nil {
			statisticsSum(&r.CheckoutHits, *stats.CheckoutHits)
			statisticsSum(&r.CheckoutAttempts, *stats.CheckoutAttempts)
			r.Coverage.Checkout++
		}
		if stats.Total180 != nil {
			statisticsSum(&r.Total180, *stats.Total180)
			r.Coverage.Total180++
		}
		high = stats.HighestFinish
	}
	var best *int
	detail := record.Import.Detail
	throwsBefore, positionsBefore := len(r.Throws), r.Coverage.KnownPositions
	points, darts := 0, 0
	if detail != nil {
		r.Coverage.MatchesWithDetail++
		for _, leg := range detail.Legs {
			legDarts := 0
			for _, visit := range leg.Visits {
				if visit.PlayerID != sourceID {
					continue
				}
				points += visit.StartRemaining - visit.EndRemaining
				darts += len(visit.Throws)
				legDarts += len(visit.Throws)
				if leg.Completed && leg.WinnerID != nil && *leg.WinnerID == sourceID && !visit.Bust && visit.EndRemaining == 0 {
					statisticsMax(&high, visit.StartRemaining)
				}
				for _, dart := range visit.Throws {
					position := dart.Position
					if position != nil {
						copy := *position
						if dart.EntryType == "manual" {
							copy.Provenance = "manual"
						}
						position = &copy
						r.Coverage.KnownPositions++
					}
					r.Throws = append(r.Throws, PlayerStatisticsThrow{FixtureID: fixtureID, LegNumber: leg.Number, VisitNumber: visit.Number, ThrowNumber: dart.Number, Segment: dart.Segment, EntryType: dart.EntryType, Position: position})
				}
			}
			if detail.Coverage == "complete" && leg.Completed && leg.WinnerID != nil && *leg.WinnerID == sourceID && legDarts > 0 && (best == nil || legDarts < *best) {
				value := legDarts
				best = &value
			}
		}
		// Only complete, ingestion-validated history establishes full match points/darts.
		if detail.Coverage == "complete" && darts > 0 {
			a.points += points
			a.darts += darts
			r.Coverage.DartWeightedAverage++
		}
	}
	if len(r.Throws) > throwsBefore {
		r.Coverage.MatchesWithRecordedThrows++
	}
	if r.Coverage.KnownPositions > positionsBefore {
		r.Coverage.MatchesWithPositions++
	}
	if best != nil {
		r.Coverage.BestLegDarts++
		if r.BestLegDarts == nil || *best < *r.BestLegDarts {
			r.BestLegDarts = best
		}
	}
	if high != nil {
		statisticsMax(&r.HighestFinish, *high)
		r.Coverage.HighestFinish++
	}
}

func (a *playerStatisticsAccumulator) finish() {
	r := a.result
	r.LegDifference = r.LegsFor - r.LegsAgainst
	r.MatchAverageMean = statisticsRatio(a.averages, r.Coverage.MatchAverageMean)
	r.FirstNineMatchAverageMean = statisticsRatio(a.firstNine, r.Coverage.FirstNineMatchAverageMean)
	r.DartWeightedAverage = statisticsRatio(3*float64(a.points), a.darts)
	if r.CheckoutAttempts != nil {
		r.CheckoutPercentage = statisticsRatio(100*float64(*r.CheckoutHits), *r.CheckoutAttempts)
	}
	r.Coverage.RecordedThrows = len(r.Throws)
	r.Coverage.PositionFraction = statisticsRatio(float64(r.Coverage.KnownPositions), r.Coverage.RecordedThrows)
}

func statisticsRatio(numerator float64, denominator int) *float64 {
	if denominator == 0 {
		return nil
	}
	value := numerator / float64(denominator)
	return &value
}

func statisticsSum(target **int, value int) {
	if *target == nil {
		*target = new(int)
	}
	**target += value
}

func statisticsMax(target **int, value int) {
	if *target == nil || value > **target {
		copy := value
		*target = &copy
	}
}
