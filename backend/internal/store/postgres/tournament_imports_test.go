package postgres

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/greg/darts-league/backend/internal/autodarts"
	"github.com/greg/darts-league/backend/internal/league"
	"github.com/greg/darts-league/backend/internal/notifications"
	"github.com/greg/darts-league/backend/internal/resultsrelay"
	"github.com/greg/darts-league/backend/internal/tournament"
)

func publishedCup(t *testing.T, store *Store, slug string, legs int32, names ...string) (tournament.Event, map[int64]string) {
	t.Helper()
	ctx := context.Background()
	service := tournament.Service{Store: store, Now: time.Now}
	if _, err := service.Create(ctx, strings.ReplaceAll(slug, "-", " "), "2026-06-16", "", tournament.MatchFormat{GameVariant: "501", LegsToWin: legs}); err != nil {
		t.Fatal(err)
	}
	if _, err := store.SetTournamentRegistration(ctx, slug, true); err != nil {
		t.Fatal(err)
	}
	byID := make(map[int64]string, len(names))
	for _, name := range names {
		entry, err := service.Register(ctx, slug, name, "Flight")
		if err != nil {
			t.Fatal(err)
		}
		byID[entry.ID] = name
	}
	if _, err := store.SetTournamentRegistration(ctx, slug, false); err != nil {
		t.Fatal(err)
	}
	event, err := store.StartTournament(ctx, slug)
	if err != nil {
		t.Fatal(err)
	}
	return event, byID
}

func cupPayload(id, one, two string, legs int) autodarts.Payload {
	played := "2026-06-16T11:00:00Z"
	return autodarts.Payload{
		CompetitionType: autodarts.TournamentMatch, SchemaVersion: "autodarts.import.v1", Source: "autodarts",
		ExternalMatchID: id, PlayedAt: &played, Settings: autodarts.Settings{BaseScore: 501, LegsToWin: legs, Out: "double"},
		Completed: true, Players: []autodarts.Player{{ID: "a", DisplayName: one, LegsWon: legs}, {ID: "b", DisplayName: two, LegsWon: 1}},
	}
}

func relayBody(t *testing.T, payload autodarts.Payload) []byte {
	t.Helper()
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatal(err)
	}
	return body
}

func deliverCup(t *testing.T, service league.PendingResultService, payload autodarts.Payload) league.ImportOutcome {
	t.Helper()
	outcome, err := service.IngestDurablePayload(context.Background(), relayBody(t, payload))
	if err != nil {
		t.Fatal(err)
	}
	return outcome
}

func TestTournamentImportsCompleteEventWithClosedLeague(t *testing.T) {
	ctx := context.Background()
	store, importer, leagueBody, fixture := autoPG(t)
	var payload autodarts.Payload
	if err := json.Unmarshal(leagueBody, &payload); err != nil {
		t.Fatal(err)
	}
	event, roster := publishedCup(t, store, "spring-cup", 2, payload.Players[0].DisplayName, payload.Players[1].DisplayName, "Casey Guest")
	// Separate games between the same league pair. A tournament submission must
	// stay pending until that pair can play, even while a league fixture is open.
	blocked := cupPayload("before-compatible-match", payload.Players[0].DisplayName, payload.Players[1].DisplayName, 3)
	if out := deliverCup(t, importer, blocked); out.Pending.Status != league.PendingResultStatusPending || !strings.Contains(out.Pending.PendingReason, "No eligible") {
		t.Fatal(out)
	}
	if _, err := store.GetResultByFixture(ctx, fixture.FixtureID); !errors.Is(err, league.ErrResultNotFound) {
		t.Fatalf("tournament import wrote a league result: %v", err)
	}
	// With compatible destinations in both competitions, changing a source's
	// declared competition cannot make it score twice.
	leaguePair := []string{payload.Players[0].DisplayName, payload.Players[1].DisplayName}
	publishedCup(t, store, "parallel-cup", 3, leaguePair...)
	tournamentFirst := cupPayload("shared-cross", leaguePair[0], leaguePair[1], 3)
	if out := deliverCup(t, importer, tournamentFirst); out.Pending.Status != league.PendingResultStatusConfirmed {
		t.Fatal(out)
	}
	tournamentFirst.CompetitionType = autodarts.LeagueMatch
	cross := deliverCup(t, importer, tournamentFirst)
	leagueRequest := fixture
	leagueRequest.PendingID, leagueRequest.Replace = cross.Pending.ID, true
	// cupPayload identifies source seats as a and b, in producer order.
	leagueRequest.Mapping = map[string]int64{"a": fixture.Mapping["seat-a"], "b": fixture.Mapping["seat-b"]}
	if _, err := importer.Approve(ctx, leagueRequest); !errors.Is(err, league.ErrApprovalConflict) {
		t.Fatalf("tournament source reused in the league: %v", err)
	}
	if out, err := importer.IngestDurablePayload(ctx, leagueBody); err != nil || out.Pending.Status != league.PendingResultStatusConfirmed {
		t.Fatalf("league result: %+v, %v", out, err)
	}
	unscored, _ := publishedCup(t, store, "other-cup", 3, leaguePair...)
	payload.CompetitionType = autodarts.TournamentMatch
	cross = deliverCup(t, importer, payload)
	if cross.Pending.Status != league.PendingResultStatusPending || !strings.Contains(cross.Pending.PendingReason, "already applied") {
		t.Fatal(cross)
	}
	detail, err := importer.ImportDetail(ctx, cross.Pending.ID)
	if err != nil || len(detail.TournamentCandidates) != 0 {
		t.Fatalf("league source offered a tournament destination in %s: %+v, %v", unscored.Slug, detail, err)
	}
	if err := store.CloseSeason(ctx, fixture.SeasonID); err != nil {
		t.Fatal(err)
	}
	var recorded int
	samePair := false
	for roundIndex, round := range event.Bracket {
		for matchIndex, match := range round {
			if match.Slots[0].Bye || match.Slots[1].Bye {
				continue
			}
			current, err := store.GetTournament(ctx, event.Slug)
			if err != nil {
				t.Fatal(err)
			}
			match = current.Bracket[roundIndex][matchIndex]
			one, two := match.Slots[1].EntryID, match.Slots[0].EntryID
			if roster[one] == "Casey Guest" {
				one, two = two, one
			}
			samePair = samePair || roster[two] != "Casey Guest"
			first := cupPayload("cup-"+strconv.Itoa(recorded), roster[one], roster[two], 2)
			out := deliverCup(t, importer, first)
			if out.Pending.Status != league.PendingResultStatusConfirmed || out.Pending.ConfirmedBy != "autodarts:auto-submit" {
				t.Fatalf("approval while league closed: %+v", out)
			}
			record, err := store.GetImport(ctx, out.Pending.ID)
			if err != nil || record.Approval == nil || record.Approval.Mapping["a"] != one ||
				record.Approval.TournamentSlug != event.Slug || record.Approval.MatchNumber != match.Number {
				t.Fatalf("missing tournament audit: %+v, %v", record, err)
			}
			if replay := deliverCup(t, importer, first); !replay.Duplicate || replay.Pending.ID != out.Pending.ID {
				t.Fatalf("replay: %+v", replay)
			}
			cross := first
			cross.CompetitionType = autodarts.LeagueMatch
			if copy := deliverCup(t, importer, cross); !copy.Changed || copy.Pending.Status != league.PendingResultStatusPending {
				t.Fatalf("cross-competition replay: %+v", copy)
			}
			persisted, err := store.GetTournament(ctx, event.Slug)
			if err != nil {
				t.Fatal(err)
			}
			actual := persisted.Bracket[roundIndex][matchIndex]
			want := [2]int{1, 2}
			if actual.Slots[0].EntryID == one {
				want = [2]int{2, 1}
			}
			if actual.WinnerID != one || actual.Legs == nil || *actual.Legs != want {
				t.Fatalf("score and entrant identity: %+v", actual)
			}
			recorded++
		}
	}
	var resultCount int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM tournament_import_results WHERE tournament_slug=$1`, event.Slug).Scan(&resultCount); err != nil || recorded != 2 || resultCount != 2 || !samePair {
		t.Fatalf("expected exactly two persisted scores, recorded=%d count=%d same_pair=%t err=%v", recorded, resultCount, samePair, err)
	}
}

type cupSlack struct{ posts []string }

func (s *cupSlack) PostMessage(_ context.Context, channel, msg string) error {
	s.posts = append(s.posts, channel+" "+msg)
	return nil
}

func TestTournamentImportAmbiguousReviewAndConcurrency(t *testing.T) {
	ctx := context.Background()
	store := approvalPG(t)
	now := func() time.Time { return time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC) }
	slack := &cupSlack{}
	auto := league.NewPendingResultServiceWithNow(store, league.NewResultService(store), now).WithAutoSubmit(true).
		WithNotifier(notifications.NewPendingResultNotifier(slack, "TEST", "https://example.invalid"))
	for _, slug := range []string{"east-cup", "west-cup"} {
		publishedCup(t, store, slug, 3, "Alice", "Bob")
	}
	ambiguous := cupPayload("ambiguous", "Alice", "Bob", 3)
	out := deliverCup(t, auto, ambiguous)
	detail, err := auto.ImportDetail(ctx, out.Pending.ID)
	if err != nil || detail.Pending.CompetitionType != autodarts.TournamentMatch || len(detail.TournamentCandidates) != 2 {
		t.Fatalf("ambiguous result: %+v, %v", detail, err)
	}
	target := detail.TournamentCandidates[1]
	req := league.ApprovalRequest{PendingID: out.Pending.ID, TournamentSlug: target.TournamentSlug, MatchNumber: target.MatchNumber, Mapping: target.Mappings[0], Actor: "reviewer"}
	var wg sync.WaitGroup
	approved := make(chan league.TournamentImportResult, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			result, err := auto.ApproveTournament(ctx, req)
			if err == nil {
				approved <- result
			} else if !errors.Is(err, league.ErrPendingResultNotPending) {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
	close(approved)
	if len(approved) != 1 {
		t.Fatalf("concurrent approvals: %d", len(approved))
	}
	alternate := deliverCup(t, auto.WithAutoSubmit(false), cupPayload("disabled", "Alice", "Bob", 3))
	detail, err = auto.ImportDetail(ctx, alternate.Pending.ID)
	if err != nil || len(detail.TournamentCandidates) != 1 || detail.TournamentCandidates[0].TournamentSlug == target.TournamentSlug {
		t.Fatalf("unique manual suggestion: %+v, %v", detail, err)
	}
	deliverCup(t, auto, cupPayload("no-match", "Alice", "Bob", 2))
	legacy := cupPayload("legacy", "Alice", "Bob", 3)
	legacy.CompetitionType = ""
	oldBody := relayBody(t, legacy)
	relay := &importRelay{messages: []resultsrelay.Message{{MessageID: "old", ReceiptHandle: "durable-old", Body: string(oldBody)}}}
	acks := 0
	relay.ack = func(messages []resultsrelay.Message) error {
		acks += len(messages)
		return nil
	}
	poller := resultsrelay.NewPoller(relay, auto, time.UTC, time.Minute, log.New(io.Discard, "", 0)).WithDurableIngest(func(ctx context.Context, message resultsrelay.Message) error {
		_, err := auto.IngestDurablePayload(ctx, []byte(message.Body))
		return err
	})
	if err := poller.PollNow(ctx); err != nil || acks != 1 {
		t.Fatalf("read old durable receipt: acks=%d, %v", acks, err)
	}
	stored, err := store.ImportsBySource(ctx, "autodarts", "legacy")
	if err != nil || len(stored) != 1 || stored[0].Pending.Status != league.PendingResultStatusPending ||
		stored[0].Import.CompetitionType != "" || strings.Contains(string(stored[0].Import.Payload), "competition_type") {
		t.Fatalf("legacy record: %+v, %v", stored, err)
	}
	wants := []string{"Multiple eligible tournament matches", "Auto-submit is disabled", "No eligible unplayed tournament match", "Competition type is missing"}
	if len(slack.posts) != len(wants) {
		t.Fatalf("Slack posts: %v", slack.posts)
	}
	for i, want := range wants {
		msg := slack.posts[i]
		if !strings.Contains(msg, want) || !strings.Contains(msg, "Alice") || !strings.Contains(msg, "Bob") ||
			!strings.Contains(msg, "/admin/pending-results") || (i < 3 && !strings.Contains(msg, "Tournament match")) {
			t.Errorf("notification %d: %q", i, msg)
		}
	}
}

func TestTournamentImportConcurrentDeliveryAndRollback(t *testing.T) {
	ctx := context.Background()
	store := approvalPG(t)
	event, _ := publishedCup(t, store, "concurrent-cup", 2, "Alice", "Bob")
	service := league.NewPendingResultService(store, league.NewResultService(store)).WithAutoSubmit(true)
	payload := cupPayload("shared-source", "Alice", "Bob", 2)
	body := relayBody(t, payload)
	_, err := store.pool.Exec(ctx, `CREATE FUNCTION reject_tournament_approval() RETURNS trigger LANGUAGE plpgsql AS $$
		BEGIN RAISE EXCEPTION 'injected audit failure'; END $$;
		CREATE TRIGGER reject_tournament_approval BEFORE UPDATE ON pending_results
		FOR EACH ROW WHEN (NEW.status = 'confirmed') EXECUTE FUNCTION reject_tournament_approval()`)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.IngestDurablePayload(ctx, body); err == nil {
		t.Fatal("injected approval error was ignored")
	}
	if current, err := store.GetTournament(ctx, event.Slug); err != nil || current.Bracket[0][0].WinnerID != 0 {
		t.Fatalf("rollback left a winner: %+v, %v", current, err)
	}
	if history, err := store.ImportsBySource(ctx, "autodarts", payload.ExternalMatchID); err != nil || len(history) != 0 {
		t.Fatalf("rollback left an import: %+v, %v", history, err)
	}
	if _, err := store.pool.Exec(ctx, `DROP TRIGGER reject_tournament_approval ON pending_results`); err != nil {
		t.Fatal(err)
	}
	results := make(chan league.ImportOutcome, 8)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			outcome, err := service.IngestDurablePayload(ctx, body)
			if err != nil {
				t.Error(err)
				return
			}
			results <- outcome
		}()
	}
	wg.Wait()
	close(results)
	var newCount int
	for outcome := range results {
		if outcome.Pending.Status != league.PendingResultStatusConfirmed {
			t.Errorf("delivery did not report the committed result: %+v", outcome)
		}
		if !outcome.Duplicate {
			newCount++
		}
	}
	var rows int
	if err := store.pool.QueryRow(ctx, `SELECT count(*) FROM tournament_import_results WHERE tournament_slug=$1`, event.Slug).Scan(&rows); err != nil || rows != 1 || newCount != 1 {
		t.Fatalf("one source claim and approval expected: rows=%d new=%d err=%v", rows, newCount, err)
	}
}
