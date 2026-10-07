package httpapi

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/resultsrelay"
	"github.com/greg/darts-league/backend/internal/store/postgres"
	"github.com/jackc/pgx/v5"
)

func flowDatabase(t *testing.T) *postgres.Store {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL must identify a disposable local Postgres")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	var nonce [8]byte
	if _, err := rand.Read(nonce[:]); err != nil {
		t.Fatal(err)
	}
	name := "autodarts_flow_" + hex.EncodeToString(nonce[:])
	schema := pgx.Identifier{name}.Sanitize()
	if _, err := conn.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := conn.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE"); err != nil {
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
	query := u.Query()
	query.Set("search_path", name)
	u.RawQuery = query.Encode()
	store, err := postgres.Open(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	return store
}

func TestAutodartsRelayApprovalAndPublicReveal(t *testing.T) {
	ctx := context.Background()
	store := flowDatabase(t)
	now := time.Date(2026, 6, 14, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	if _, err := store.UpsertSeason(ctx, league.NewSeason("Integration test only")); err != nil {
		t.Fatal(err)
	}
	seasons := league.NewSeasonServiceWithNow(store, clock)
	registration := league.NewRegistrationServiceWithNow(store, clock)
	divisions, err := seasons.ProvisionDivisions(ctx, 1)
	if err != nil {
		t.Fatal(err)
	}
	var players []league.Player
	for _, name := range []string{"Morgan Ember", "Casey Vale"} {
		player, err := registration.RegisterPlayer(ctx, league.Player{DisplayName: name})
		if err != nil {
			t.Fatal(err)
		}
		if _, err := registration.AssignPlayer(ctx, player.ID, &divisions[0].ID); err != nil {
			t.Fatal(err)
		}
		players = append(players, player)
	}
	if _, err := seasons.StartSeason(ctx); err != nil {
		t.Fatal(err)
	}
	fixtures, err := store.ListFixturesByDivision(ctx, divisions[0].ID)
	if err != nil || len(fixtures) != 1 {
		t.Fatalf("fixtures: %v %v", fixtures, err)
	}
	results := league.NewResultServiceWithNow(store, clock)
	pending := league.NewPendingResultServiceWithNow(store, results, clock)
	data, err := os.ReadFile("../../../docs/autodarts/examples-v1.json")
	if err != nil {
		t.Fatal(err)
	}
	var examples map[string]json.RawMessage
	if err := json.Unmarshal(data, &examples); err != nil {
		t.Fatal(err)
	}
	var failAck atomic.Bool
	var acknowledgements atomic.Int64
	failAck.Store(true)
	relay := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			writeJSON(w, http.StatusOK, map[string]any{"messages": []map[string]string{{"messageId": "delivery-1", "receiptHandle": "receipt-1", "body": string(examples["producer"])}}})
			return
		}
		stored, err := pending.List(ctx)
		if err != nil || len(stored) != 1 {
			t.Errorf("acknowledgement before durable insert: %v %v", stored, err)
		}
		acknowledgements.Add(1)
		if failAck.Load() {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"acknowledged": []string{"delivery-1"}, "failed": []string{}})
	}))
	t.Cleanup(relay.Close)
	client, err := resultsrelay.NewClient(relay.URL + "/results")
	if err != nil {
		t.Fatal(err)
	}
	poller := resultsrelay.NewPoller(client, pending, time.UTC, time.Minute, log.New(io.Discard, "", 0)).WithDurableIngest(func(ctx context.Context, message resultsrelay.Message) error {
		_, err := pending.IngestDurablePayload(ctx, []byte(message.Body))
		return err
	})
	if err := poller.PollNow(ctx); err == nil {
		t.Fatal("failed acknowledgement was reported successful")
	}
	failAck.Store(false)
	if err := poller.PollNow(ctx); err != nil {
		t.Fatal(err)
	}
	imports, err := pending.List(ctx)
	if err != nil || len(imports) != 1 || acknowledgements.Load() != 2 {
		t.Fatalf("redelivery did not preserve exactly one import: %v %v", imports, err)
	}
	auth := NewAuthHandlerWithNow("admin", "test-password", "test-session-key", clock)
	mux := http.NewServeMux()
	auth.RegisterRoutes(mux)
	NewPendingResultHandler(pending).RegisterRoutes(mux, auth.RequireAdmin)
	NewResultHandler(results).RegisterRoutes(mux, auth.RequireAdmin)
	var cookie *http.Cookie
	send := func(method, path, body string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
		if cookie != nil {
			req.AddCookie(cookie)
		}
		rec := httptest.NewRecorder()
		mux.ServeHTTP(rec, req)
		return rec
	}
	login := send(http.MethodPost, "/api/admin/login", `{"username":"admin","password":"test-password"}`)
	if login.Code != http.StatusOK || len(login.Result().Cookies()) == 0 {
		t.Fatalf("login: %d", login.Code)
	}
	cookie = login.Result().Cookies()[0]
	fixture := fixtures[0]
	body, err := json.Marshal(league.ApprovalRequest{SeasonID: fixture.SeasonID, FixtureID: fixture.ID, Mapping: map[string]int64{"seat-a": players[0].ID, "seat-b": players[1].ID}, Reason: "Source clock and fixture reviewed"})
	if err != nil {
		t.Fatal(err)
	}
	approval := send(http.MethodPost, fmt.Sprintf("/api/admin/pending-results/%d/confirm", imports[0].ID), string(body))
	if approval.Code != http.StatusOK {
		t.Fatalf("approval: %d %s", approval.Code, approval.Body.String())
	}
	standingsPath := "/api/divisions/" + divisions[0].Slug + "/standings"
	readStandings := func() []standingRowResponse {
		rec := send(http.MethodGet, standingsPath, "")
		var response struct {
			Standings []standingRowResponse `json:"standings"`
		}
		if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil || rec.Code != http.StatusOK {
			t.Fatalf("standings: %d %s %v", rec.Code, rec.Body.String(), err)
		}
		return response.Standings
	}
	for _, row := range readStandings() {
		if row.Played != 0 || row.Points != 0 || row.Average != nil {
			t.Fatalf("unrevealed result leaked into standings: %+v", row)
		}
	}
	now = time.Date(2026, 6, 15, 8, 0, 0, 0, time.UTC)
	visible := readStandings()
	if len(visible) != 2 || visible[0].Played != 1 || visible[0].Points != 2 {
		t.Fatalf("revealed result missing: %+v", visible)
	}
}
