# Player-season statistics (#46)

Public route: `/seasons/:seasonId/players/:playerId`.
Existing-session admin route: `/admin/seasons/:seasonId/players/:playerId`.
Both fetch the corresponding `/api/.../statistics` endpoint with explicit league
IDs, credentials, cancellation, a 15-second timeout, and `cache: no-store`.
The API serializes the existing filtered `league.PlayerStatistics` projection.
Admin authentication runs before ID parsing or lookup; all handler responses,
including authentication and lookup failures, have `Cache-Control: no-store`.
Errors never echo store or source payloads.

The parser rejects malformed identities, metrics, histories, throws, and
inconsistent coverage. The page refetches on mount/focus and hides old data during
refetch. It never calculates public reveal eligibility. Historical URLs retain
their explicit season/player identity; there is no archive browser or name-based
identity resolution. Standings without the additive `player_id` remain readable
without invented links. Match pages link only returned league player IDs.

Metric coverage is per metric, not shared detail coverage. Unknown is not zero;
zero checkout attempts is not a zero checkout percentage. First-nine values are
explicitly means of known per-match averages, not per-leg or dart-weighted.

`PlayerThrows` adapts only server-marked `plottable` positions into the shared
Heatmap. Unverified known coordinates remain in the typed data and coverage text,
but their plot input is null. The local CSS hides the shared plot-input caption
because it cannot distinguish withheld coordinates from absent coordinates;
adjacent original-evidence counts explain both honestly. Segments remain readable
for every selected entry type. No physical-board accuracy is claimed.

## Repeat checks

From `frontend`, with the checked-in dependencies and Node 22:

```sh
export PATH=/home/greg/.npm/_npx/52027bd8fc0022aa/node_modules/node/bin:$PATH
npm test
npm run build
npm run lint
node src/pages/players/browser/verify.mjs
```

From `backend`: `go test ./...` and `go test -race -shuffle=on -count=1 ./...`.
Postgres integration tests require `TEST_DATABASE_URL`; this slice's HTTP tests
exercise the actual ResultService with MemoryStore, ingestion and approval.

Browser evidence uses the production app, intercepted fictional API responses,
and local fonts. It exercises both real player routes, public standings navigation,
history URLs, keyboard entry filters, retry, zero/unknown/partial/manual/empty
states and sanitized failures at 375/768/1280 pixels. It checks horizontal
overflow, font loading, plotted point counts and page errors. The 39 screenshots
and SHA-256 manifest live under `browser/screenshots/`. These are offline route
checks, not live database E2E, a Lighthouse audit, or independent design review.
Parent integration owns the assembled Postgres/browser E2E. Relay authentication
and verified automatic geometry remain deferred; admin access still requires the
existing session. No global CSS or shared heatmap internals were changed.
