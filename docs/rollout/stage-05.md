# Stage 5: publish recorded matches and enable detailed capture

Requires the completed stage-4 database/backend/frontend cutover. Adds no new
database migration or player aggregate endpoints. Deploy the backend first (new
read-only match endpoints), then the frontend with match links. Both use the
existing stage-4 import/approval contract. Do not enable the new scraper first.

Public match lookup returns only revealed, currently linked approved evidence;
admin lookup requires the existing session. Missing/manual detail renders a
score-only summary. Source/account identities and original payloads are excluded.
Unknown coordinates remain unknown. Player-profile links wait for stage 6.

## Activation

1. Keep the producer sandboxed and preserve `RESULTS_ENDPOINT` while updating
   the app images. Verify admin auth, a score-only match, recorded detail, an
   unrevealed direct URL (404), edits and undo. Keep `APP_NOW` empty in deployments.
2. The updated `score-scrape/scoreScrape.js` defaults to sandbox. Refresh the
   Autodarts tab to remove the old hook before installing it; do not run both.
   Configure the existing relay `SubmissionUrl`, ending in `/results`.
3. Only after access controls and controlled end-to-end checks are approved,
   set `window.__scoreScrapeSandbox = false` before installing the script to
   permit real submission. This does not authenticate the producer: the deferred
   relay-authentication gate still applies. No live physical-board accuracy is
   claimed, and automatic coordinate geometry remains unverified.
4. Verify commit/ack/replay, source mapping, public reveal and score-only fallback
   on the actual version pair before normal use. Stage 6 is not required.

## Rollback

Disable detailed submissions or return the producer to sandbox. The stage-4
consumer can retain both already-received detailed imports and legacy summaries.
The app may return to the stage-4 image pair without a database downgrade; only
the public analysis pages are lost. Never roll back behind the stage-4 schema
boundary or restore the old delete-on-read reader.

## Checks and evidence

Backend: `go test -race -shuffle=on -count=1 ./...`; database tests require
disposable `TEST_DATABASE_URL` and must pass in CI before rollout.
Frontend: `npm test`, `npm run test:typecheck`, `npm run build`, `npm run lint`,
`node tests/matches/verify.mjs`. Scraper: `node --test score-scrape/scraper.test.mjs`
from the repository root. The browser checks cover detailed, unknown-coordinate,
score-only, error/loading, admin auth and link eligibility at 375/768/1280.
Full screenshots are ignored locally and uploaded as CI artifacts; no exhaustive
PNG collection is added to this PR.
