package league

import (
	"context"
	"encoding/json"
	"testing"
)

func TestPlayerStatisticsSourceLifecycle(t *testing.T) {
	for _, action := range []string{"pending change", "rejected change", "blocked change", "replacement", "unchanged edit", "manual edit", "undo", "undo then manual"} {
		t.Run(action, func(t *testing.T) {
			// Given a real approved source followed by a lifecycle operation.
			ctx := context.Background()
			store, service := statisticsSetup(t)
			payload := statisticsPayload(false)
			original := statisticsApprove(t, store, payload)
			wantPlayed, wantDetail, wantThrows := 1, 1, 27
			switch action {
			case "pending change", "rejected change", "blocked change", "replacement":
				payload = statisticsPayload(true)
				payload.ExternalMatchID = "stats-1"
				if action == "blocked change" {
					payload.Detail.Legs[0].Visits[0].EndRemaining = 500
				}
				body, err := json.Marshal(payload)
				if err != nil {
					t.Fatal(err)
				}
				pending := NewPendingResultService(store, service)
				outcome, err := pending.IngestPayload(ctx, body)
				if err != nil {
					t.Fatal(err)
				}
				if action == "rejected change" {
					if err := pending.RejectWithReason(ctx, outcome.Pending.ID, "reviewer", "wrong source"); err != nil {
						t.Fatal(err)
					}
				}
				if action == "replacement" {
					_, err = pending.Approve(ctx, ApprovalRequest{PendingID: outcome.Pending.ID, SeasonID: 1, FixtureID: original.FixtureID, Mapping: map[string]int64{"a": 1, "b": 2}, ExpectedResult: ExpectedFromResult(original), Replace: true, Reason: "correction", Actor: "reviewer", MissingDateReason: "unknown"})
					if err != nil {
						t.Fatal(err)
					}
					wantThrows = 36
				}
			case "unchanged edit":
				if _, err := service.EditResult(ctx, original.FixtureID, 3, 0, nil, nil, "admin"); err != nil {
					t.Fatal(err)
				}
			case "manual edit":
				if _, err := service.EditResult(ctx, original.FixtureID, 3, 1, nil, nil, "admin"); err != nil {
					t.Fatal(err)
				}
				wantDetail, wantThrows = 0, 0
			case "undo", "undo then manual":
				if err := service.DeleteResult(ctx, original.FixtureID, "admin"); err != nil {
					t.Fatal(err)
				}
				wantPlayed, wantDetail, wantThrows = 0, 0, 0
				if action == "undo then manual" {
					if _, err := service.RecordResult(ctx, original.FixtureID, 3, 0, nil, nil); err != nil {
						t.Fatal(err)
					}
					wantPlayed = 1
				}
			}
			// When reading again without a cache.
			got, err := service.PublicPlayerStatistics(ctx, 1, 1)
			// Then only the current eligible result and active source contribute once.
			if err != nil {
				t.Fatal(err)
			}
			if got.Played != wantPlayed || len(got.History) != wantPlayed || got.Coverage.MatchesWithDetail != wantDetail || len(got.Throws) != wantThrows {
				t.Fatalf("lifecycle %s: %+v", action, got)
			}
		})
	}
}

func TestPlayerStatisticsRejectsStaleSourceLinks(t *testing.T) {
	for _, change := range []string{"result id", "score", "average", "pending", "rejected", "blocked"} {
		t.Run(change, func(t *testing.T) {
			// Given a linked record that no longer agrees with the current result/status.
			ctx := context.Background()
			store, service := statisticsSetup(t)
			r := statisticsApprove(t, store, statisticsPayload(false))
			source, err := store.ImportByFixture(ctx, r.FixtureID)
			if err != nil {
				t.Fatal(err)
			}
			switch change {
			case "result id":
				id := int64(999)
				source.ResultID = &id
			case "score":
				r.PlayerTwoLegs = 1
			case "average":
				average := 42.0
				r.PlayerOneAverage = &average
			case "pending":
				source.Pending.Status = PendingResultStatusPending
			case "rejected":
				source.Pending.Status = PendingResultStatusRejected
			case "blocked":
				source.Pending.Status = PendingResultStatusReviewBlocked
			}
			if _, err := store.UpdateResult(ctx, r); err != nil {
				t.Fatal(err)
			}
			if err := store.SaveImportApproval(ctx, source); err != nil {
				t.Fatal(err)
			}
			// When reading an inconsistent source/result pair.
			got, err := service.PublicPlayerStatistics(ctx, 1, 1)
			// Then no source-derived summary, detail or coverage leaks.
			if err != nil {
				t.Fatal(err)
			}
			if got.Played != 0 || got.Coverage.EligibleMatches != 0 || len(got.Throws) != 0 || len(got.History) != 0 {
				t.Fatalf("stale: %+v", got)
			}
		})
	}
}
