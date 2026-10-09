package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/store/postgres"
	"github.com/jackc/pgx/v5"
)

func matchPostgres(t *testing.T) league.Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := pgx.Identifier{"match_detail_" + strconv.FormatInt(time.Now().UnixNano(), 10)}.Sanitize()
	if _, err := conn.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// Deletes only the disposable schema created above; no shared tables are touched.
		if _, err := conn.Exec(ctx, `DROP SCHEMA `+schema+` CASCADE`); err != nil {
			t.Error(err)
		}
		if err := conn.Close(ctx); err != nil {
			t.Error(err)
		}
	})
	u, err := url.Parse(dsn)
	if err != nil {
		t.Fatal(err)
	}
	q := u.Query()
	q.Set("search_path", schema)
	u.RawQuery = q.Encode()
	store, err := postgres.Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	return store
}

func TestMatchDetailHTTPSourceLifecycle(t *testing.T) {
	for _, engine := range []string{"memory", "postgres"} {
		t.Run(engine, func(t *testing.T) {
			var store league.Store = league.NewMemoryStore()
			if engine == "postgres" {
				store = matchPostgres(t)
			}
			ctx := context.Background()
			if _, err := store.EnsureActiveSeason(ctx, league.NewSeason("Detail test")); err != nil {
				t.Fatal(err)
			}
			now := time.Date(2026, 3, 25, 12, 0, 0, 0, time.UTC)
			clock := func() time.Time { return now }
			seasons := league.NewSeasonServiceWithNow(store, clock)
			reg := league.NewRegistrationServiceWithNow(store, clock)
			divisions, err := seasons.ProvisionDivisions(ctx, 1)
			if err != nil {
				t.Fatal(err)
			}
			for _, p := range []league.Player{{DisplayName: "Alice", Nickname: "Arrow"}, {DisplayName: "Bob"}} {
				p, err := reg.RegisterPlayer(ctx, p)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := reg.AssignPlayer(ctx, p.ID, &divisions[0].ID); err != nil {
					t.Fatal(err)
				}
			}
			season, err := seasons.StartSeason(ctx)
			if err != nil {
				t.Fatal(err)
			}
			fixtures, err := store.ListFixturesBySeason(ctx, season.ID)
			if err != nil {
				t.Fatal(err)
			}
			f := fixtures[0]
			results := league.NewResultServiceWithNow(store, clock)
			pending := league.NewPendingResultServiceWithNow(store, results, clock)
			raw, err := os.ReadFile("../../../docs/autodarts/examples-v1.json")
			if err != nil {
				t.Fatal(err)
			}
			var examples map[string]json.RawMessage
			if err := json.Unmarshal(raw, &examples); err != nil {
				t.Fatal(err)
			}
			outcome, err := pending.IngestPayload(ctx, examples["producer"])
			if err != nil {
				t.Fatal(err)
			}
			req := league.ApprovalRequest{PendingID: outcome.Pending.ID, SeasonID: f.SeasonID, FixtureID: f.ID, Mapping: map[string]int64{"seat-a": f.PlayerTwoID, "seat-b": f.PlayerOneID}, Actor: "admin", Reason: "verified date", MissingDateReason: "not provided"}
			r, err := pending.Approve(ctx, req)
			if err != nil {
				t.Fatal(err)
			}
			auth := NewAuthHandlerWithNow("admin", "secret", "test-only", clock)
			mux := http.NewServeMux()
			NewMatchDetailHandler(league.NewFixtureServiceWithNow(store, clock)).RegisterRoutes(mux, auth.RequireAdmin)
			get := func(admin bool, want int) league.MatchDetail {
				t.Helper()
				prefix := "/api/"
				if admin {
					prefix += "admin/"
				}
				req := httptest.NewRequest("GET", fmt.Sprintf("%sfixtures/%d/autodarts", prefix, f.ID), nil)
				if admin {
					req.AddCookie(&http.Cookie{Name: adminSessionCookieName, Value: auth.signSession("admin", now.Add(time.Hour))})
				}
				w := httptest.NewRecorder()
				mux.ServeHTTP(w, req)
				if w.Code != want || w.Header().Get("Cache-Control") != "no-store" {
					t.Fatalf("HTTP %d: %s", w.Code, w.Body.String())
				}
				var got league.MatchDetail
				if want == 200 {
					if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
						t.Fatal(err)
					}
					for _, secret := range []string{"seat-a", "seat-b", "account_id", "fictional-match", "digest", "display_name", "source_payload"} {
						if strings.Contains(w.Body.String(), secret) {
							t.Fatalf("leaked %s", secret)
						}
					}
				}
				return got
			}
			get(false, 404)
			if get(true, 200).Detail == nil {
				t.Fatal("admin lost pre-reveal evidence")
			}
			now = time.Date(2026, 3, 30, 8, 0, 0, 0, time.UTC)
			got := get(false, 200)
			if got.Players[0].PreferredName != "Arrow" || got.Players[1].PreferredName != "Bob" || got.Players[1].LegsWon != 3 || got.Detail.Legs[0].Visits[0].PlayerID != strconv.FormatInt(f.PlayerTwoID, 10) {
				t.Fatalf("mapped identity: %+v", got)
			}
			if _, err := results.EditResult(ctx, f.ID, r.PlayerOneLegs, r.PlayerTwoLegs, r.PlayerOneAverage, r.PlayerTwoAverage, "admin"); err != nil {
				t.Fatal(err)
			}
			if get(false, 200).Detail == nil {
				t.Fatal("unchanged save detached source")
			}
			if _, err := results.EditResult(ctx, f.ID, 3, 0, nil, nil, "admin"); err != nil {
				t.Fatal(err)
			}
			for _, admin := range []bool{false, true} {
				if got := get(admin, 200); got.Detail != nil || got.Players[1].Stats != nil || got.PlayedAt != nil {
					t.Fatal("manual correction retained source")
				}
			}
			if err := results.DeleteResult(ctx, f.ID, "admin"); err != nil {
				t.Fatal(err)
			}
			get(false, 404)
			get(true, 404)
			r, err = results.RecordResult(ctx, f.ID, 3, 1, nil, nil)
			if err != nil {
				t.Fatal(err)
			}
			if get(false, 200).Detail != nil {
				t.Fatal("undo/record reattached old source")
			}
			changed, err := pending.IngestPayload(ctx, []byte(strings.ReplaceAll(string(examples["producer"]), "60.12", "61.12")))
			if err != nil {
				t.Fatal(err)
			}
			req.PendingID, req.Replace, req.ExpectedResult = changed.Pending.ID, true, league.ExpectedFromResult(r)
			if _, err := pending.Approve(ctx, req); err != nil {
				t.Fatal(err)
			}
			if got := get(false, 200); got.Detail == nil || *got.Players[1].Stats.MatchAverage != 61.12 {
				t.Fatal("replacement not current")
			}
			if err := store.CloseSeason(ctx, season.ID); err != nil {
				t.Fatal(err)
			}
			if err := store.CreateNextSeason(ctx, season.ID, league.NewSeason("Next")); err != nil {
				t.Fatal(err)
			}
			if got := get(false, 200); got.SeasonID != season.ID || got.Players[0].PreferredName != "Arrow" || got.Detail == nil {
				t.Fatal("rollover lost historical match")
			}
		})
	}
}
