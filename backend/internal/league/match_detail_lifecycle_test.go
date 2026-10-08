package league

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMatchDetailCurrentSourceLifecycle(t *testing.T) {
	for _, action := range []string{"unchanged", "edit", "undo", "undo-record", "replace", "rollover", "manual", "legacy"} {
		t.Run(action, func(t *testing.T) {
			ctx := context.Background()
			store, pending, req := matchDetailSetup(t)
			results := NewResultService(store)
			if action == "manual" {
				if _, err := results.RecordResult(ctx, 1, 3, 1, nil, nil); err != nil {
					t.Fatal(err)
				}
			} else {
				if action == "legacy" {
					p, err := store.CreatePendingResult(ctx, PendingResult{ExternalMatchID: "old", PlayerOneName: "private-one", PlayerTwoName: "private-two", PlayerOneLegs: 3, PlayerTwoLegs: 1, Status: PendingResultStatusPending})
					if err != nil {
						t.Fatal(err)
					}
					req.PendingID, req.AttestFormat = p.ID, true
					req.Mapping = map[string]int64{"legacy-1": 22, "legacy-2": 11}
				}
				r, err := pending.Approve(ctx, req)
				if err != nil {
					t.Fatal(err)
				}
				switch action {
				case "unchanged":
					_, err = results.EditResult(ctx, 1, r.PlayerOneLegs, r.PlayerTwoLegs, r.PlayerOneAverage, r.PlayerTwoAverage, "admin")
				case "edit":
					_, err = results.EditResult(ctx, 1, 3, 0, nil, nil, "manual")
				case "undo", "undo-record":
					err = results.DeleteResult(ctx, 1, "admin")
					if err == nil && action == "undo-record" {
						_, err = results.RecordResult(ctx, 1, 3, 1, nil, nil)
					}
				case "replace":
					replacement := changedImport(t, pending, req)
					replacement.Replace, replacement.ExpectedResult = true, ExpectedFromResult(r)
					_, err = pending.Approve(ctx, replacement)
				case "rollover":
					err = store.CloseSeason(ctx, req.SeasonID)
					if err == nil {
						err = store.CreateNextSeason(ctx, req.SeasonID, NewSeason("Next"))
					}
				}
				if err != nil {
					t.Fatal(err)
				}
			}
			service := NewFixtureServiceWithNow(store, func() time.Time { return time.Date(2026, 4, 1, 12, 0, 0, 0, time.UTC) })
			for _, admin := range []bool{false, true} {
				got, err := service.MatchDetail(ctx, 1, admin)
				if action == "undo" {
					if !errors.Is(err, ErrFixtureNotFound) {
						t.Fatalf("undone result visible: %v", err)
					}
					continue
				}
				if err != nil {
					t.Fatal(err)
				}
				wantDetail := action == "unchanged" || action == "replace" || action == "rollover"
				if (got.Detail != nil) != wantDetail {
					t.Fatalf("stale/missing detail: %+v", got)
				}
				if got.Players[0].PreferredName != "The Arrow" || got.SeasonID != req.SeasonID {
					t.Fatalf("historical identity lost: %+v", got)
				}
				if action == "edit" && (got.Players[0].LegsWon != 3 || got.Players[1].LegsWon != 0 || got.Players[1].Stats != nil) {
					t.Fatalf("stale source summary: %+v", got.Players)
				}
				if action == "replace" && *got.Players[1].Stats.MatchAverage != 61.12 {
					t.Fatal("superseded source average")
				}
			}
		})
	}
}

func TestMatchDetailPreservesAllowedStatsAndEveryReference(t *testing.T) {
	ctx := context.Background()
	store, pending, req := matchDetailSetup(t)
	r, err := store.GetImport(ctx, req.PendingID)
	if err != nil {
		t.Fatal(err)
	}
	stats := r.Import.Players[0].Stats
	zero, finish, first9 := 0, 120, 72.5
	stats.FirstNineAverage, stats.AverageUntil170 = &first9, &first9
	stats.HighestFinish, stats.Total180 = &finish, &zero
	stats.Less60, stats.Plus60, stats.Plus100, stats.Plus140, stats.Plus170 = &zero, &zero, &zero, &zero, &zero
	private := "private-account"
	r.Import.Players[0].AccountID = &private
	r.Import.Players[0].DisplayName = "private-source-label"
	data, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	store.importsByID[req.PendingID] = data
	if _, err := pending.Approve(ctx, req); err != nil {
		t.Fatal(err)
	}
	got, err := NewFixtureService(store).MatchDetail(ctx, 1, true)
	if err != nil {
		t.Fatal(err)
	}
	if *got.Players[1].Stats.FirstNineAverage != first9 || *got.Players[1].Stats.Total180 != 0 || *got.Players[1].Stats.HighestFinish != finish {
		t.Fatalf("lost optional stats: %+v", got.Players[1].Stats)
	}
	for i, leg := range got.Detail.Legs {
		original := r.Import.Detail.Legs[i]
		if original.WinnerID != nil && *leg.WinnerID != strconv.FormatInt(req.Mapping[*original.WinnerID], 10) {
			t.Fatal("winner not remapped")
		}
		for j, visit := range leg.Visits {
			before := original.Visits[j]
			if visit.PlayerID != strconv.FormatInt(req.Mapping[before.PlayerID], 10) || !reflect.DeepEqual(visit.Throws, before.Throws) {
				t.Fatal("visit/coordinates changed")
			}
		}
	}
	encoded, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"private-source-label", "private-account", r.Import.Digest, r.Import.ExternalMatchID} {
		if secret != "" && strings.Contains(string(encoded), secret) {
			t.Fatalf("source privacy leak: %s", secret)
		}
	}
	var public struct {
		Players []struct {
			Stats map[string]*float64 `json:"stats"`
		} `json:"players"`
	}
	if err := json.Unmarshal(encoded, &public); err != nil {
		t.Fatal(err)
	}
	if len(public.Players[1].Stats) != 14 {
		t.Fatalf("stats allowlist lost fields: %s", encoded)
	}
}
