package resultsrelay

import (
	"encoding/json"
	"io"
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/greg/darts-league/backend/internal/league"
)

type legacyPlayer struct {
	Name         string   `json:"name"`
	LegsWon      *int     `json:"legsWon"`
	MatchAverage *float64 `json:"matchAverage"`
}

func parseLegacy(body string) (league.PendingResult, error) {
	invalid := league.PendingResult{}
	if len(body) > 256*1024 || !utf8.ValidString(body) {
		return invalid, ErrInvalidDelivery
	}
	var payload struct {
		MatchID string       `json:"matchId"`
		Player1 legacyPlayer `json:"player1"`
		Player2 legacyPlayer `json:"player2"`
	}
	decoder := json.NewDecoder(strings.NewReader(body))
	// Legacy has no version or detailed fields: fail closed until activation.
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return invalid, ErrInvalidDelivery
	}
	if err := decoder.Decode(new(json.RawMessage)); err != io.EOF {
		return invalid, ErrInvalidDelivery
	}
	if strings.TrimSpace(payload.MatchID) == "" || len(payload.MatchID) > 128 || strings.ContainsFunc(payload.MatchID, unicode.IsControl) {
		return invalid, ErrInvalidDelivery
	}
	for _, player := range []legacyPlayer{payload.Player1, payload.Player2} {
		name := strings.Join(strings.Fields(player.Name), " ")
		if name == "" || utf8.RuneCountInString(name) > league.MaxDisplayNameLength || strings.ContainsFunc(player.Name, unicode.IsControl) ||
			player.LegsWon == nil || *player.LegsWon < 0 || *player.LegsWon > math.MaxInt32 {
			return invalid, ErrInvalidDelivery
		}
		if average := player.MatchAverage; average != nil && (math.IsNaN(*average) || math.IsInf(*average, 0) || *average < 0 || *average > 180) {
			return invalid, ErrInvalidDelivery
		}
	}
	// The fixture's configured first-to target is enforced by existing admin confirmation.
	if err := league.ValidateResultScore(*payload.Player1.LegsWon, *payload.Player2.LegsWon, max(*payload.Player1.LegsWon, *payload.Player2.LegsWon)); err != nil {
		return invalid, ErrInvalidDelivery
	}
	return league.PendingResult{
		ExternalMatchID: payload.MatchID,
		PlayerOneName:   strings.Join(strings.Fields(payload.Player1.Name), " "), PlayerOneLegs: *payload.Player1.LegsWon, PlayerOneAverage: payload.Player1.MatchAverage,
		PlayerTwoName: strings.Join(strings.Fields(payload.Player2.Name), " "), PlayerTwoLegs: *payload.Player2.LegsWon, PlayerTwoAverage: payload.Player2.MatchAverage,
	}, nil
}
