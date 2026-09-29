package notifications

import (
	"context"
	"fmt"
	"log"
	"strings"

	"github.com/greg/darts-league/backend/internal/league"
)

func NewPendingResultNotifier(poster MessagePoster, channelID, publicBaseURL string) func(context.Context, league.PendingResult) {
	channelID = strings.TrimSpace(channelID)
	publicBaseURL = strings.TrimRight(strings.TrimSpace(publicBaseURL), "/")
	return func(ctx context.Context, pending league.PendingResult) {
		if poster == nil || channelID == "" {
			return
		}
		text := fmt.Sprintf("New score awaiting confirmation\n%s %d-%d %s",
			escapePlayerLabel(pending.PlayerOneName), pending.PlayerOneLegs,
			pending.PlayerTwoLegs, escapePlayerLabel(pending.PlayerTwoName))
		if publicBaseURL != "" {
			text += fmt.Sprintf("\n<%s/admin/pending-results|Review pending results>", publicBaseURL)
		}
		if err := poster.PostMessage(ctx, channelID, text); err != nil && !errorsIsDisabled(err) {
			log.Printf("slack pending result notification failed: pending_id=%d: %v", pending.ID, err)
		}
	}
}
