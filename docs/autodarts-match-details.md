# Recorded match details (#44)

## Implemented routes and authorization

- `GET /api/fixtures/{fixtureID}/autodarts`
- `GET /api/admin/fixtures/{fixtureID}/autodarts` (existing admin session required)
- Frontend: `/matches/:fixtureId` and `/admin/matches/:fixtureId`.

Both HTTP routes set `Cache-Control: no-store`, including error and unauthorized
responses. Public missing, invalid, unrecorded and unrevealed fixture IDs return
the same 404 body: `{"error":{"code":"not_found","message":"Match not found."}}`.
Admin authentication precedes ID lookup; authenticated admins can read a current
result before reveal. Internal failures return a sanitized 503, not source errors.

Reveal is the fixture's scheduled calendar day at 09:00 Europe/London, using the
existing `GroupFixturesByWeek` reveal calculation and the injected server clock.
Generated fixture days are Mondays. Equality is revealed; BST/GMT transitions do
not use fixed UTC offsets. Closure does not bypass reveal. Numeric fixture lookup
and `ListPlayersBySeason(fixture.SeasonID)` avoid active-season/division lookup,
so revealed historical results remain readable after rollover.

`FixtureService.MatchDetail(ctx, fixtureID, admin)` reads fixture, result, players
and active source inside the existing transaction/lifecycle lock. Only a confirmed,
active `ImportByResult(currentResult.ID)` with matching fixture, season and result
links supplies imported evidence. `MappedImport` remaps every visit and winner;
its whole `Import` value is NEVER serialized. The response contains an allowlisted
player projection, normalized detail and coverage. Source labels, account IDs,
external match IDs, digest, payload and approval metadata are excluded for both
public and admin match views. Admin source review remains a different API.

No eligible source means summary 200, `playedAt:null`, `detail:null`. Manual
averages explicitly present on the current league result remain available; other
statistics are unknown. Contradictory manual edits detach source; undo gives 404;
re-recording does not reattach it. An unchanged save preserves linkage. Explicit
replacement selects only the new active source. Legacy linked imports can supply
their known summary statistics without invented detail or dates.

## Exact response shape

```ts
{
  schema_version: 'autodarts.api.v1',
  fixture_id: number,
  season_id: number,
  playedAt: string | null,
  players: [
    { league_player_id: number, preferred_name: string, legs_won: number, stats: Stats | null },
    { league_player_id: number, preferred_name: string, legs_won: number, stats: Stats | null }
  ],
  detail: { coverage: 'partial' | 'complete', legs: Leg[] } | null,
  coverage: {
    recorded_throws: number,
    known_positions: number,
    position_fraction: number | null
  }
}
```

Players are in fixture order, with league nickname then display-name fallback.
Scores always come from the current league result. A non-null `Stats` object has
all 14 fields below; each is a number or explicit null:

`match_average`, `points_scored`, `darts_thrown`, `checkout_hits`,
`checkout_attempts`, `first_nine_average`, `average_until_170`, `highest_finish`,
`total_180`, `less_60`, `plus_60`, `plus_100`, `plus_140`, `plus_170`.

Unknown/omitted optional source fields normalize to null in this API (immutable
stored source JSON/digest is unchanged). Zero remains zero; source averages and
statistics are not recomputed from partial throws. `playedAt` is the original
source timestamp, never the fixture, receipt or result-entry time.

`Leg`, `Visit`, `Throw`, `Segment`, `Position` retain the normalized field names
and nullable fields in [contract-v1.md](autodarts/contract-v1.md). The difference
is identity: **every `winner_id` and `player_id` is a decimal string league player
ID**, referring to one of the two numeric `league_player_id` values. Coordinates,
units/origin/axes and entry/position provenance remain unchanged. Unknown geometry
is readable but not automatically treated as board-normalized data.

Coverage counts positions among recorded throws, including non-plottable geometry.
`position_fraction = known_positions / recorded_throws`, or null when there are
no recorded throws. It is not coverage of all darts played, nor board-plottable
coverage; the shared heatmap separately reports plottable positions.

This implemented shape supersedes the preliminary `public_detail` and
`public_summary_only` examples in `examples-v1.json`: it adds `season_id` and
`playedAt`, nests averages in nullable `stats`, retains every known statistic, and
uses consistent string detail references rather than numeric ones. Source model
JSON is not a public response contract. The shared component README's #43/#44
ownership sentence is reversed: #43 owns pending review; #44 owns match publication.

## Frontend adapter and lifecycle

`frontend/src/lib/matchApi.ts` owns the narrow unknown-to-typed boundary:
`parseMatch(unknown): FixtureMatch` returns `{fixtureId, seasonId, match, coverage}`.
It validates version, safe numeric IDs, two distinct players, reference membership,
bounded stats/detail, finite coordinates, provenance, timestamp and coverage.
Unknown keys are dropped, not passed through. The `MatchData` adapter converts
league IDs with `String(id)`, names to `label`, preserves scores/stats/detail and
sets `synthetic:false` for the production API. Test data is explicitly synthetic.
Semantic scoring validation remains the ingestion/approval boundary.

`fetchMatch` selects the public/admin endpoint, includes existing session cookies,
requests no-store, supports cancellation and a 15-second timeout, and never echoes
server error bodies. There are no new dependencies or duplicated generic request
wrappers. `MatchPage` reuses `MatchAnalysis` (which includes `MatchSummary`) and
the shared loading/error/missing-evidence states; 404 and admin 401 have explicit
page states. React Query has zero retention/stale time, always refetches on mount
and focus, and hides old content while refetching or after an error. There is no
client-side authorization/reveal calculation and no background live-update claim.

Active and completed public division lists link recorded, server-unlocked results
only. Admin score cards link recorded results, including future and read-only
scores. Existing save/undo forms remain in place. Admin route detection covers
`/admin/*`, so detail pages do not show public registration/division navigation.

## Checks and handoff

- Go unit/HTTP tests: guessed IDs, session auth, no-store, before/exact/after reveal,
  both London DST boundaries, nicknames/order, all mapped references/coordinates,
  source privacy, optional stats/nulls, legacy/manual fallbacks, edit/unchanged save,
  undo/re-record, replacement and historical rollover.
- HTTP source lifecycle runs against memory and a unique disposable Postgres
  schema when `TEST_DATABASE_URL` is supplied; only that test-owned schema is
  removed at cleanup. Full backend race suite is also run.
- Frontend unit tests exercise parser rejection, privacy, fetch options, page
  states and link eligibility. Node 22 `npm test`, `npm run build`, `npm run lint`.
- After build: `node tests/matches/verify.mjs` from `frontend/` runs the real app
  routes/components in Playwright with intercepted local synthetic API responses
  and locally fulfilled fonts; no Autodarts, relay or live backend calls. It tests
  filters, retry, admin navigation, result links and overflow at 375/768/1280.
  36 captures plus SHA-256 manifest: `frontend/tests/matches/screenshots/`.
  The first capture run failed on a test-only font URL rewrite; corrected before
  final captures. Review found default-blue result links; scoped link styling
  restores the app's coral palette and 44px targets.

#45 can reuse the eligibility policy and DTO fields, not internal source labels.
Aggregate queries still need their own revealed/current-source selection and
known/eligible match coverage; calling this transaction per fixture would take
the deliberately coarse league lock repeatedly. No aggregate behavior changed.
#46 can use the two APIs/routes and the tests above as integration seams; coordinate
rendering remains conditional on declared, supported geometry.

Limits: no live physical adapter or live relay-to-browser end-to-end claim; no
deployment, push or issue closure. Relay authentication remains deferred and the
relay remains unauthenticated; review does not prove origin, ownership or accuracy.
No retention/purge changes. No independent subagent visual review (scoped worker
prohibits nesting), full Lighthouse audit or React instrumentation was run. Existing
Node 22.12.0 ran tests/build/lint successfully but npm warned that jsdom and
eslint-visitor-keys request Node >=22.13; upgrading the workstation runtime is
outside this change. Existing component/table typography and horizontal mobile
table scrolling are retained; browser checks are not a full accessibility audit.
