package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/resultsrelay"
)

// NewAutoSubmitNotifier posts to the admin channel after an automatic
// Autodarts approval has committed. Failures are logged, never retried.
func NewAutoSubmitNotifier(store league.Store, poster MessagePoster, channelID, publicBaseURL string) func(context.Context, league.Result) {
	channelID = strings.TrimSpace(channelID)
	publicBaseURL = strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	return func(ctx context.Context, result league.Result) {
		if poster == nil || channelID == "" {
			return
		}
		one, two, division := "Player one", "Player two", "Unknown division"
		if fixture, err := store.GetFixture(ctx, result.FixtureID); err == nil {
			if players, err := store.ListPlayersBySeason(ctx, fixture.SeasonID); err == nil {
				for _, p := range players {
					if p.ID == fixture.PlayerOneID {
						one = p.AdminLabel()
					}
					if p.ID == fixture.PlayerTwoID {
						two = p.AdminLabel()
					}
				}
			}
			if divisions, err := store.ListDivisionsBySeason(ctx, fixture.SeasonID); err == nil {
				for _, d := range divisions {
					if d.ID == fixture.DivisionID {
						division = d.Name
					}
				}
			}
		}
		text := fmt.Sprintf("Autodarts result recorded automatically\n- %s %d-%d %s\n- Division: %s",
			escapePlayerLabel(one), result.PlayerOneLegs, result.PlayerTwoLegs, escapePlayerLabel(two), escapePlayerLabel(division))
		if publicBaseURL != "" {
			text += fmt.Sprintf("\n<%s/admin/matches/%d|View match>", publicBaseURL, result.FixtureID)
		}
		if err := poster.PostMessage(ctx, channelID, text); err != nil && !errorsIsDisabled(err) {
			log.Printf("slack auto-submit notification failed: fixture_id=%d: %v", result.FixtureID, err)
		}
	}
}

// NewRelayFailureNotifier posts a payload-free alert to the admin channel when
// a relay message fails ingestion or automatic approval.
func NewRelayFailureNotifier(poster MessagePoster, channelID, publicBaseURL string) func(context.Context, resultsrelay.Message, string) {
	channelID = strings.TrimSpace(channelID)
	publicBaseURL = strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	return func(ctx context.Context, message resultsrelay.Message, reason string) {
		if poster == nil || channelID == "" {
			return
		}
		text := "Autodarts import failed\n- Reason: " + reason
		if id := externalMatchID(message.Body); id != "" {
			text += "\n- Autodarts match: " + escapePlayerLabel(id)
		}
		if message.MessageID != "" {
			text += "\n- Relay message: " + escapePlayerLabel(message.MessageID)
		}
		if publicBaseURL != "" {
			text += fmt.Sprintf("\n<%s/admin/pending-results|Open pending results>", publicBaseURL)
		}
		if err := poster.PostMessage(ctx, channelID, text); err != nil && !errorsIsDisabled(err) {
			log.Printf("slack relay failure notification failed: message_id=%q: %v", message.MessageID, err)
		}
	}
}

func externalMatchID(body string) string {
	var ids struct {
		External string `json:"external_match_id"`
		Legacy   string `json:"matchId"`
	}
	if json.Unmarshal([]byte(body), &ids) != nil {
		return ""
	}
	id := strings.TrimSpace(ids.External + ids.Legacy)
	if len(id) > 100 {
		return ""
	}
	return id
}
