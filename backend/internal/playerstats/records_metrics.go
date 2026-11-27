package playerstats

import (
	"math"

	"github.com/greg/darts-league/backend/internal/autodarts"
	"github.com/greg/darts-league/backend/internal/league"
)

func (p *recordAccumulator) addMatch(entryID int64, won bool, average *float64, source *league.ImportRecord) {
	if p == nil {
		return
	}
	p.played++
	if won {
		p.wins.add(1)
	} else {
		p.wins.add(0)
	}
	if average == nil && source != nil {
		for _, player := range source.Import.Players {
			if source.Mapping[player.ID] == entryID {
				average = player.Average()
				break
			}
		}
	}
	if average != nil && !math.IsNaN(*average) && !math.IsInf(*average, 0) {
		// Rank at the same two-decimal precision shown on the leaderboard.
		p.average.maximum(math.Round(*average*100) / 100)
	}
	if source == nil {
		return
	}
	for _, player := range source.Import.Players {
		if source.Mapping[player.ID] == entryID {
			p.addRecordedDarts(player, source.Import.Detail)
			return
		}
	}
}

func (p *recordAccumulator) addRecordedDarts(player autodarts.Player, detail *autodarts.Detail) {
	var maximums *int
	finish, bulls, darts, visits180, fastest := 0, 0, 0, 0, 0
	if player.Stats != nil {
		maximums = player.Stats.Total180
		if player.Stats.HighestFinish != nil {
			finish = *player.Stats.HighestFinish
		}
	}
	if detail != nil {
		for _, leg := range detail.Legs {
			won := leg.Completed && leg.WinnerID != nil && *leg.WinnerID == player.ID
			legDarts := 0
			for _, visit := range leg.Visits {
				if visit.PlayerID != player.ID {
					continue
				}
				legDarts += len(visit.Throws)
				treble20 := 0
				for _, dart := range visit.Throws {
					switch {
					case dart.Segment.Bed == "inner_bull":
						bulls++
					case dart.Segment.Bed == "triple" && dart.Segment.Number == 20:
						treble20++
					}
				}
				if treble20 == 3 && !visit.Bust && visit.StartRemaining-visit.EndRemaining == 180 {
					visits180++
				}
				if won && !visit.Bust && visit.EndRemaining == 0 {
					finish = max(finish, visit.StartRemaining)
				}
			}
			darts += legDarts
			if detail.Coverage == "complete" && won && legDarts > 0 && (fastest == 0 || legDarts < fastest) {
				fastest = legDarts
			}
		}
	}
	switch {
	case maximums != nil:
		p.maximums.add(*maximums)
	case detail != nil && detail.Coverage == "complete" && darts > 0:
		p.maximums.add(visits180)
	}
	if darts > 0 {
		p.bullseyes.add(bulls)
	}
	if finish > 0 {
		p.checkouts.maximum(float64(finish))
	}
	if fastest > 0 {
		p.leg.minimum(fastest)
	}
}
