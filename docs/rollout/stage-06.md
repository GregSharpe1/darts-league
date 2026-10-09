# Stage 6: read-only player statistics

Depends on stage 5. Adds public/admin player-season endpoints, coverage-aware
aggregation, player profiles, entry-type heatmap filters and stable-ID navigation
from standings/match pages. No migration, producer update or new write API.

## Deploy

Deploy the backend first, then the frontend. Use immutable CI image tags and keep
the existing database/auth/relay values. The stage-5 frontend tolerates the added
`player_id` response field. Do not link to profiles until their backend is ready.
Keep `APP_NOW` empty. Public reads exclude hidden/pending/superseded evidence;
admin reads still require the existing session. No source/account IDs are public.

Verify a player with detailed matches, one with only manual/score-only results,
missing metrics versus true zero, entry filters, historical season scope and
edit/undo detachment. Denominators are evidence coverage, not invented all-dart
totals. Manual points are not proof of physical-board accuracy; unsupported
automatic geometry stays unplottable. Relay authentication remains deferred.

## Rollback

Restore the stage-5 frontend first to remove profile links, then the stage-5
backend. No database rollback is needed. Preserve the stage-4 schema boundary;
rolling further back requires its documented coordinated recovery procedure.

## Checks

`go test -race -shuffle=on -count=1 ./...` from backend; database tests need a
disposable `TEST_DATABASE_URL` and otherwise skip. Frontend: `npm test`,
`npm run test:typecheck`, `npm run build`, `npm run lint`,
`node tests/matches/verify.mjs`, and `node src/pages/players/browser/verify.mjs`.
The browser harness covers 375/768/1280, null/zero/manual/partial/error states,
keyboard filters, history links, overflow and sanitized errors. Full captures
are ignored and uploaded by CI rather than committed as an exhaustive PNG set.
