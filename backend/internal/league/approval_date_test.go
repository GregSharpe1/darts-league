package league

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestApprovalRequiresReasonForSourceDateOutsideSeasonWindow(t *testing.T) {
	for _, scenario := range []string{"before_start", "future"} {
		t.Run(scenario, func(t *testing.T) {
			store, service, req := approvalSetup(t)
			ctx := context.Background()
			season, err := store.GetActiveSeason(ctx)
			if err != nil {
				t.Fatal(err)
			}
			start := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
			if scenario == "before_start" {
				season.StartedAt = &start
			} else {
				service.now = func() time.Time { return time.Date(2026, 6, 14, 0, 0, 0, 0, time.UTC) }
			}
			if _, err := store.UpsertSeason(ctx, season); err != nil {
				t.Fatal(err)
			}
			req.Reason = ""
			if _, err := service.Approve(ctx, req); !errors.Is(err, ErrImportAttestation) {
				t.Fatalf("unreviewed %s date: %v", scenario, err)
			}
			req.Reason = "Source clock checked; this is the selected fixture."
			if _, err := service.Approve(ctx, req); err != nil {
				t.Fatal(err)
			}
		})
	}
}
