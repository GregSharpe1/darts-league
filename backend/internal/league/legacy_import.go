package league

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
	"github.com/greg/darts-league/backend/internal/autodarts"
)

func LegacyImportRecord(p PendingResult) (ImportRecord, error) {
	a := autodarts.Player{ID: "legacy-1", DisplayName: p.PlayerOneName, LegsWon: p.PlayerOneLegs}
	b := autodarts.Player{ID: "legacy-2", DisplayName: p.PlayerTwoName, LegsWon: p.PlayerTwoLegs}
	if p.PlayerOneAverage != nil {
		a.Stats = &autodarts.Stats{MatchAverage: p.PlayerOneAverage}
	}
	if p.PlayerTwoAverage != nil {
		b.Stats = &autodarts.Stats{MatchAverage: p.PlayerTwoAverage}
	}
	type summaryPlayer struct {
		Name    string   `json:"name"`
		Legs    int      `json:"legsWon"`
		Average *float64 `json:"matchAverage"`
	}
	payload, err := json.Marshal(struct {
		MatchID string        `json:"matchId"`
		One     summaryPlayer `json:"player1"`
		Two     summaryPlayer `json:"player2"`
	}{p.ExternalMatchID, summaryPlayer{p.PlayerOneName, p.PlayerOneLegs, p.PlayerOneAverage}, summaryPlayer{p.PlayerTwoName, p.PlayerTwoLegs, p.PlayerTwoAverage}})
	if err != nil {
		return ImportRecord{}, err
	}
	payload, err = jsoncanonicalizer.Transform(payload)
	if err != nil {
		return ImportRecord{}, err
	}
	digest := sha256.Sum256(payload)
	return ImportRecord{Pending: p, Import: autodarts.Import{Payload: payload, Digest: hex.EncodeToString(digest[:]), Source: "autodarts", ExternalMatchID: p.ExternalMatchID, SettingsEvidence: "legacy_stored_summary", Players: []autodarts.Player{a, b}}}, nil
}
