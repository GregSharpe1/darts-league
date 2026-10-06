package autodarts

import "encoding/json"

type Import struct {
	Payload          json.RawMessage
	Digest           string
	Source           string
	ExternalMatchID  string
	PlayedAt         *string
	SettingsEvidence string
	Players          []Player
	Detail           *Detail
	ReviewReason     string
}

type Payload struct {
	SchemaVersion   string   `json:"schema_version"`
	Source          string   `json:"source"`
	ExternalMatchID string   `json:"external_match_id"`
	PlayedAt        *string  `json:"playedAt"`
	Settings        Settings `json:"settings"`
	Completed       bool     `json:"completed"`
	Players         []Player `json:"players"`
	Detail          *Detail  `json:"detail"`
}
type Settings struct {
	BaseScore int    `json:"base_score"`
	LegsToWin int    `json:"legs_to_win"`
	Out       string `json:"out"`
}
type Player struct {
	ID          string  `json:"match_player_id"`
	AccountID   *string `json:"account_id"`
	DisplayName string  `json:"display_name"`
	LegsWon     int     `json:"legs_won"`
	Stats       *Stats  `json:"stats"`
}
type Stats struct {
	MatchAverage     *float64 `json:"match_average"`
	PointsScored     *int     `json:"points_scored"`
	DartsThrown      *int     `json:"darts_thrown"`
	CheckoutHits     *int     `json:"checkout_hits"`
	CheckoutAttempts *int     `json:"checkout_attempts"`
}

func (p Player) Average() *float64 {
	if p.Stats == nil {
		return nil
	}
	return p.Stats.MatchAverage
}

type Detail struct {
	Coverage string `json:"coverage"`
	Legs     []Leg  `json:"legs"`
}
type Leg struct {
	Number    int     `json:"number"`
	Completed bool    `json:"completed"`
	WinnerID  *string `json:"winner_id"`
	Visits    []Visit `json:"visits"`
}
type Visit struct {
	Number         int     `json:"number"`
	PlayerID       string  `json:"player_id"`
	StartRemaining int     `json:"start_remaining"`
	EndRemaining   int     `json:"end_remaining"`
	Bust           bool    `json:"bust"`
	Throws         []Throw `json:"throws"`
}
type Throw struct {
	Number    int       `json:"number"`
	Segment   Segment   `json:"segment"`
	EntryType string    `json:"entry_type"`
	Position  *Position `json:"position"`
}
type Segment struct {
	Bed    string `json:"bed"`
	Number int    `json:"number"`
}
type Position struct {
	X               float64 `json:"x"`
	Y               float64 `json:"y"`
	Units           *string `json:"units"`
	Origin          *string `json:"origin"`
	AxisOrientation *string `json:"axis_orientation"`
	Provenance      string  `json:"provenance"`
}
type legacyPayload struct {
	MatchID  string       `json:"matchId"`
	PlayedAt *string      `json:"playedAt" optional:"true"`
	Player1  legacyPlayer `json:"player1"`
	Player2  legacyPlayer `json:"player2"`
}
type legacyPlayer struct {
	Name         string   `json:"name"`
	LegsWon      int      `json:"legsWon"`
	MatchAverage *float64 `json:"matchAverage" optional:"true"`
}
