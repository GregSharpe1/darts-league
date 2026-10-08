package autodarts

import "sort"

func provenance(s string) bool { return s == "automatic" || s == "manual" || s == "unknown" }

func segmentScore(s Segment) (int, bool) {
	switch s.Bed {
	case "miss":
		return 0, s.Number == 0
	case "single":
		return s.Number, s.Number >= 1 && s.Number <= 20
	case "double":
		return 2 * s.Number, s.Number >= 1 && s.Number <= 20
	case "triple":
		return 3 * s.Number, s.Number >= 1 && s.Number <= 20
	case "outer_bull":
		return 25, s.Number == 25
	case "inner_bull":
		return 50, s.Number == 25
	default:
		return 0, false
	}
}

func validateDetail(d *Detail) error {
	if d == nil {
		return nil
	}
	if (d.Coverage != "partial" && d.Coverage != "complete") || len(d.Legs) > 5 {
		return ErrInvalidPayload
	}
	numbers := map[int]bool{}
	for _, leg := range d.Legs {
		if leg.Number < 1 || leg.Number > 5 || numbers[leg.Number] || len(leg.Visits) > 200 {
			return ErrInvalidPayload
		}
		numbers[leg.Number] = true
		if leg.WinnerID != nil && !identifier.MatchString(*leg.WinnerID) {
			return ErrInvalidPayload
		}
		previous := 0
		for _, visit := range leg.Visits {
			if visit.Number <= previous || visit.Number > 200 || !identifier.MatchString(visit.PlayerID) || visit.StartRemaining < 2 || visit.StartRemaining > 501 || visit.EndRemaining < 0 || visit.EndRemaining > 501 || len(visit.Throws) < 1 || len(visit.Throws) > 3 {
				return ErrInvalidPayload
			}
			previous = visit.Number
			for i, dart := range visit.Throws {
				_, valid := segmentScore(dart.Segment)
				if dart.Number != i+1 || !valid || !provenance(dart.EntryType) {
					return ErrInvalidPayload
				}
				if p := dart.Position; p != nil {
					if !provenance(p.Provenance) {
						return ErrInvalidPayload
					}
					for _, s := range []*string{p.Units, p.Origin, p.AxisOrientation} {
						if s != nil && !label(*s) {
							return ErrInvalidPayload
						}
					}
				}
			}
		}
	}
	return nil
}

func (p Import) detailConflict() string {
	if p.Detail == nil {
		return ""
	}
	complete := p.Detail.Coverage == "complete"
	players := map[string]bool{p.Players[0].ID: true, p.Players[1].ID: true}
	wins, points, darts := map[string]int{}, map[string]int{}, map[string]int{}
	legs := append([]Leg(nil), p.Detail.Legs...)
	sort.Slice(legs, func(i, j int) bool { return legs[i].Number < legs[j].Number })
	for i, leg := range legs {
		if complete && (leg.Number != i+1 || !leg.Completed) {
			return "incomplete_coverage"
		}
		if leg.Completed != (leg.WinnerID != nil) || (leg.WinnerID != nil && !players[*leg.WinnerID]) {
			return "leg_winner_conflict"
		}
		remaining := map[string]int{}
		lastPlayer := ""
		lastNumber := 0
		checkedOut := ""
		for j, visit := range leg.Visits {
			if !players[visit.PlayerID] {
				return "unknown_player"
			}
			if checkedOut != "" {
				return "throws_after_checkout"
			}
			if (complete && visit.Number != j+1) || (visit.Number == lastNumber+1 && visit.PlayerID == lastPlayer) {
				return "turn_continuity"
			}
			if !complete && visit.Number > lastNumber+1 {
				remaining = map[string]int{}
			}
			prior, seen := remaining[visit.PlayerID]
			if (seen && prior != visit.StartRemaining) || (complete && !seen && visit.StartRemaining != 501) {
				return "remaining_continuity"
			}
			lastPlayer = visit.PlayerID
			lastNumber = visit.Number
			score := visit.StartRemaining
			bust := false
			for k, dart := range visit.Throws {
				value, _ := segmentScore(dart.Segment)
				score -= value
				double := dart.Segment.Bed == "double" || dart.Segment.Bed == "inner_bull"
				bust = score < 0 || score == 1 || (score == 0 && !double)
				if bust || score == 0 {
					if k != len(visit.Throws)-1 {
						return "throws_after_terminal_dart"
					}
					if !bust {
						checkedOut = visit.PlayerID
					}
				}
			}
			if bust {
				score = visit.StartRemaining
			}
			if bust != visit.Bust || score != visit.EndRemaining {
				return "score_transition"
			}
			if complete && !bust && score > 0 && len(visit.Throws) != 3 {
				return "incomplete_turn"
			}
			remaining[visit.PlayerID] = score
			points[visit.PlayerID] += visit.StartRemaining - score
			darts[visit.PlayerID] += len(visit.Throws)
		}
		if checkedOut != "" && (!leg.Completed || *leg.WinnerID != checkedOut) {
			return "leg_winner_conflict"
		}
		if complete && checkedOut == "" {
			return "missing_checkout"
		}
		if leg.Completed {
			wins[*leg.WinnerID]++
		}
		if complete && i < len(p.Detail.Legs)-1 && (wins[p.Players[0].ID] == 3 || wins[p.Players[1].ID] == 3) {
			return "legs_after_match_complete"
		}
	}
	for _, player := range p.Players {
		if wins[player.ID] > player.LegsWon || (complete && wins[player.ID] != player.LegsWon) {
			return "summary_legs_conflict"
		}
		if complete && player.Stats != nil {
			s := player.Stats
			if (s.PointsScored != nil && *s.PointsScored != points[player.ID]) || (s.DartsThrown != nil && *s.DartsThrown != darts[player.ID]) || (s.CheckoutHits != nil && *s.CheckoutHits != wins[player.ID]) {
				return "summary_totals_conflict"
			}
		}
	}
	return ""
}
