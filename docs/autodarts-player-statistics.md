# Coverage-aware player statistics (#45)

Service-only implementation; no API route, ranking change, frontend, new relay
authentication, schema migration, caching or deployment. Integrators should use:

```go
func (s ResultService) PublicPlayerStatistics(ctx context.Context, seasonID, playerID int64) (PlayerStatistics, error)
func (s ResultService) AdminPlayerStatistics(ctx context.Context, seasonID, playerID int64) (PlayerStatistics, error)
```

Construct with existing `NewResultService` (server time) or
`NewResultServiceWithNow` (injected server clock). IDs are explicit league IDs;
never resolve account IDs, source labels or returning names across seasons.
Historical season/player pairs remain readable after rollover. Invalid or absent
pairs return `ErrPlayerNotFound`; store/context errors propagate. The public
method must be used for public endpoints. The admin method does **not** authenticate
its caller: keep it behind existing admin authentication. Responses should be
`Cache-Control: no-store`. Do not serialize `ImportRecord` alongside this DTO.

## Eligibility and consistency

- Start with current results and fixtures in the requested season, involving the
  requested registered player. Eligible matches means **played**, not scheduled.
- Public results, including manual results, must pass `CurrentPublicWeek` and
  `GroupFixturesByWeek` plus individual-fixture reveal checks, with server time
  and `Europe/London`. Monday 09:00, spring/autumn DST, and completed seasons all
  retain the existing fixture reveal semantics. Hidden matches contribute
  nothing, including to coverage denominators, history or coordinates.
- `ImportByFixture` selects the active source. Require confirmed status, no
  review conflict, matching season/fixture/current result ID, the two explicit
  player mappings, scores, winner and averages. An inconsistent active link
  fails closed for the entire match; it is not reclassified as manual evidence.
- Pending/rejected/blocked changes never displace the current confirmed source.
  Replacement contributes once from the new source. Manual edits detach source
  detail; supplied manual averages still count. Undo removes the match. A later
  manual result never automatically reattaches a superseded source.
- Read via the existing transaction/approval locks, so a result and source cannot
  be read across a concurrent correction. No new store interface or SQL is added.
  This intentionally inherits the coarse league lock and season-wide reads;
  optimize only if measurements justify a dedicated read snapshot/query.
- Synthetic fixtures are test-only data, not a new production flag or source
  trust mechanism. No synthetic coordinates are generated. Relay authentication
  and verified physical-board geometry remain deferred, not solved by approval.

## Metric definitions

All nullable metrics serialize unknown as JSON `null`; known zero stays zero.
Metric coverage is the number of eligible matches with that metric's evidence.
It is **not** permission to extrapolate to missing matches.

| JSON field | Definition |
| --- | --- |
| `played`, `won`, `lost`, `legs_for`, `legs_against`, `leg_difference`, `points` | Current eligible league summaries; win 2, loss 0. No ranking changes. |
| `match_average_mean` | Arithmetic mean of known current result averages, including manual values; preserves `BuildStandings` semantics. |
| `dart_weighted_average` | `3 * sum(validated points) / sum(actual darts)` over complete ingestion-validated match histories only. Points are visit start minus end; busts add zero points but count actual darts, and checkouts are not padded. Missing supplied totals may be derived from complete validated detail. Summary-only/partial points/darts are deliberately excluded, even if numerically present: bounds checks alone do not prove complete scoring evidence. |
| `first_nine_match_average_mean` | Arithmetic mean of known supplied **per-match** first-nine averages. Explicitly not dart-weighted, not mean per leg, and not reconstructed from partial visits: exact first-nine denominators are unavailable. |
| `checkout_hits`, `checkout_attempts`, `checkout_percentage` | Sum only pairs where both hits and attempts are known; percentage is `100 * sum(hits) / sum(attempts)`, never mean of percentages. Known zero attempts retains counts but yields null percentage. Known zero hits with positive attempts yields 0. Attempts/intent are never inferred from segments. |
| `total_180` | Sum known supplied match totals, not partial visit counts. Null when no match reports it. Zero counts as covered. |
| `best_leg_darts` | Minimum actual darts in a won completed leg from a complete validated match history. Conservative: partial histories do not establish a full-leg dart denominator. |
| `highest_finish` | Maximum known supplied match high finish or witnessed checkout visit's start remaining in a completed won leg. Partial detail can establish an observed finish, not missing match totals. This is a maximum over known evidence, not a claim to cover all finishes. |

`coverage` contains integer `eligible_matches`, `matches_with_detail`, and
known-match counts `match_average_mean`, `dart_weighted_average`,
`first_nine_match_average_mean`, `checkout`, `total_180`, `best_leg_darts`,
`highest_finish`. `matches_with_detail` includes partial recordings and is not
interchangeable with any metric's coverage count.

## History and heatmap handoff

Top-level identity fields: `schema_version: "autodarts.api.v1"`, `season_id`,
`player_id`, `preferred_name` (league nickname fallback).

`history` is newest scheduled date first, ties by fixture ID. Each record contains
only `fixture_id`, `week_number`, `scheduled_at` (RFC3339), `opponent_id`,
`opponent_name` (league nickname fallback), `won`, `legs_for`, `legs_against`,
nullable `match_average`, and `detail_coverage` (`none`, `partial`, `complete`).
No source match IDs, account IDs, source player IDs/names or raw payloads appear.

`throws` combines only this player's recorded throws, including bust darts,
joined by source ID mapping rather than array order. Records contain `fixture_id`,
`leg_number`, `visit_number`, `throw_number`, `segment: {bed, number}`,
`entry_type`, nullable `position`, and `plottable`. Sort order is fixture/leg/
visit/throw. Position is `{x,y,units,origin,axis_orientation,provenance}`; nullable
geometry stays unknown. Coordinates retain their original values. Manual entry
forces manual position provenance without altering the stored source original.

Coordinate coverage adds `matches_with_recorded_throws`, `matches_with_positions`,
`recorded_throws`, `known_positions`, `plottable_positions`, nullable
`position_fraction = known_positions / recorded_throws`. This denominator is
recorded throws, **not** all match darts. No verified geometry adapter exists:
all positions currently remain `plottable:false` and plottable coverage is zero,
even when source geometry labels are present. A future verified adapter may
normalize them explicitly; do not place unknown points or segment centers on a
board. Both arrays serialize as `[]`, not null, when empty.

## Example public JSON (manual result, no invented detail)

```json
{
  "schema_version": "autodarts.api.v1",
  "season_id": 1, "player_id": 1, "preferred_name": "League One",
  "played": 1, "won": 1, "lost": 0,
  "legs_for": 3, "legs_against": 1, "leg_difference": 2, "points": 2,
  "match_average_mean": 42, "dart_weighted_average": null,
  "first_nine_match_average_mean": null,
  "checkout_hits": null, "checkout_attempts": null, "checkout_percentage": null,
  "total_180": null, "best_leg_darts": null, "highest_finish": null,
  "coverage": {
    "eligible_matches": 1, "matches_with_detail": 0,
    "match_average_mean": 1, "dart_weighted_average": 0,
    "first_nine_match_average_mean": 0, "checkout": 0, "total_180": 0,
    "best_leg_darts": 0, "highest_finish": 0,
    "matches_with_recorded_throws": 0, "matches_with_positions": 0,
    "recorded_throws": 0, "known_positions": 0, "plottable_positions": 0,
    "position_fraction": null
  },
  "history": [{
    "fixture_id": 1, "week_number": 1, "scheduled_at": "2026-03-23T09:00:00Z",
    "opponent_id": 2, "opponent_name": "League Two", "won": true,
    "legs_for": 3, "legs_against": 1, "match_average": 42,
    "detail_coverage": "none"
  }],
  "throws": []
}
```

## Checks and integration boundary

Run from `backend`: `go test ./...` and
`go test -race -shuffle=on -count=1 ./internal/league`.
Statistics tests use real MemoryStore ingestion and transactional approval,
covering reversed player order, unequal match dart counts, ratios/zero/null,
partial/unfinished detail, busts, manual summaries/positions, source replacement,
stale links, edits/undo, both DST boundaries, closure and returning-player scope.
No new query requires Postgres-specific statistics tests. The existing Postgres
suite needs a disposable `TEST_DATABASE_URL` to exercise its integration tests.

The endpoint/UI workers own routes and presentation. Parent integration still
owns reveal filtering in existing public standings; this slice does not modify
`BuildStandings`, `service.go`, existing store contracts or ranking behavior.
