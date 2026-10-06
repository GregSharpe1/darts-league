package autodarts

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"reflect"
	"regexp"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/cyberphone/json-canonicalization/go/src/webpki.org/jsoncanonicalizer"
)

var identifier = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
var timestamp = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T([01]\d|2[0-3]):[0-5]\d:[0-5]\d(\.\d+)?(Z|[+-]([01]\d|2[0-3]):[0-5]\d)$`)

func Parse(body []byte) (Import, error) {
	var result Import
	if err := checkedJSON(body); err != nil {
		return result, err
	}
	// JSON integer values may use exponent/decimal notation. Normalize numbers
	// with the contract's ECMAScript rules before decoding bounded integer fields.
	body, err := jsoncanonicalizer.Transform(body)
	if err != nil {
		return result, ErrInvalidJSON
	}
	var envelope map[string]json.RawMessage
	if err := json.Unmarshal(body, &envelope); err != nil || envelope == nil {
		return result, ErrInvalidPayload
	}
	var filtered []byte
	if version, ok := envelope["schema_version"]; ok {
		var v string
		if json.Unmarshal(version, &v) != nil || v != "autodarts.import.v1" {
			return result, ErrUnsupportedVersion
		}
		filtered, err = project(body, reflect.TypeFor[Payload]())
		if err != nil {
			return result, err
		}
		var p Payload
		if err := json.Unmarshal(filtered, &p); err != nil {
			return result, ErrInvalidPayload
		}
		if p.Source != "autodarts" || p.Settings != (Settings{501, 3, "double"}) || !p.Completed {
			return result, ErrInvalidPayload
		}
		result = Import{Source: p.Source, ExternalMatchID: p.ExternalMatchID, PlayedAt: p.PlayedAt, Players: p.Players, Detail: p.Detail, SettingsEvidence: "source_reported"}
	} else {
		filtered, err = project(body, reflect.TypeFor[legacyPayload]())
		if err != nil {
			return result, err
		}
		var p legacyPayload
		if err := json.Unmarshal(filtered, &p); err != nil {
			return result, ErrInvalidPayload
		}
		result = Import{Source: "autodarts", ExternalMatchID: p.MatchID, PlayedAt: p.PlayedAt, SettingsEvidence: "legacy_unverified"}
		for i, player := range []legacyPlayer{p.Player1, p.Player2} {
			id := "legacy-1"
			if i == 1 {
				id = "legacy-2"
			}
			mapped := Player{ID: id, DisplayName: player.Name, LegsWon: player.LegsWon}
			if player.MatchAverage != nil {
				mapped.Stats = &Stats{MatchAverage: player.MatchAverage}
			}
			result.Players = append(result.Players, mapped)
		}
	}
	if err := result.validate(); err != nil {
		return Import{}, err
	}
	canonical, err := jsoncanonicalizer.Transform(filtered)
	if err != nil {
		return Import{}, ErrInvalidJSON
	}
	digest := sha256.Sum256(canonical)
	result.Payload = canonical
	result.Digest = hex.EncodeToString(digest[:])
	result.ReviewReason = result.detailConflict()
	return result, nil
}

func label(s string) bool {
	if n := utf8.RuneCountInString(s); n < 1 || n > 80 {
		return false
	}
	for _, r := range s {
		if unicode.IsControl(r) {
			return false
		}
	}
	return true
}
func count(n *int, max int) bool { return n == nil || (*n >= 0 && *n <= max) }
func ordered(a, b *int) bool     { return a == nil || b == nil || *a <= *b }
func (p Import) validate() error {
	if !identifier.MatchString(p.ExternalMatchID) || len(p.Players) != 2 {
		return ErrInvalidPayload
	}
	if p.PlayedAt != nil {
		if !timestamp.MatchString(*p.PlayedAt) {
			return ErrInvalidPayload
		}
		if _, err := time.Parse(time.RFC3339Nano, *p.PlayedAt); err != nil {
			return ErrInvalidPayload
		}
	}
	a, b := p.Players[0], p.Players[1]
	if a.ID == b.ID || !((a.LegsWon == 3 && b.LegsWon >= 0 && b.LegsWon < 3) || (b.LegsWon == 3 && a.LegsWon >= 0 && a.LegsWon < 3)) {
		return ErrInvalidPayload
	}
	for _, player := range p.Players {
		if !identifier.MatchString(player.ID) || !label(player.DisplayName) || (player.AccountID != nil && !identifier.MatchString(*player.AccountID)) {
			return ErrInvalidPayload
		}
		s := player.Stats
		if s == nil {
			continue
		}
		if s.MatchAverage != nil && (*s.MatchAverage < 0 || *s.MatchAverage > 180) {
			return ErrInvalidPayload
		}
		if !count(s.PointsScored, 5010) || !count(s.DartsThrown, 3000) || !count(s.CheckoutHits, 3) || !count(s.CheckoutAttempts, 3000) || !ordered(s.CheckoutHits, s.CheckoutAttempts) || !ordered(s.CheckoutAttempts, s.DartsThrown) {
			return ErrInvalidPayload
		}
		if s.DartsThrown != nil && *s.DartsThrown == 0 && (s.MatchAverage != nil || (s.PointsScored != nil && *s.PointsScored != 0)) {
			return ErrInvalidPayload
		}
	}
	return validateDetail(p.Detail)
}
