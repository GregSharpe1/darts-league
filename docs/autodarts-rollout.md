# Autodarts integration verification and staged rollout (#48)

## Release boundary

This work is local implementation and verification, **not deployment approval**.
New producer/relay authentication was explicitly deferred. The relay therefore
remains unauthenticated: receiving/acknowledging queue messages and submitting
source claims are not protected by the admin session. Input validation, content
digests and admin approval do not authenticate a scoring source. Do not expose
this iteration as a production-ready trusted ingestion service.

The pre-existing frontend lockfile audit reported 21 findings, including three
critical findings involving browser-test tooling. Production exploitability was
not established; neither a clean security audit nor a dependency remediation is
claimed. Keep development/test servers on loopback. Review affected dependency
usage and patch or explicitly accept mitigations before a deployment decision.

Physical automatically scored dart geometry remains unverified. The captured
manual browser coordinate adapter is supported; synthetic fixtures test rendering
but do not establish physical accuracy. Unknown positions/geometry stay unknown.

## Verified local flow

The real-browser flow uses an intercepted, fictitious Autodarts response and the
actual scraper, local HTTP relay wire adapter, actual Go application, unique
Postgres schema, production frontend build and existing admin session:

1. Capture all three legs and 48 recorded throws through the real scraper.
2. Poll the local relay and durably store the import before acknowledgement.
3. Select season, league, source-player mapping and fixture in the real review UI.
4. Record a source-date review note, inspect the 27-dart winner heatmap and approve.
5. Verify the public match API still returns 404 before weekly reveal, with no
   hidden contribution to player metrics or coordinates.
6. Replay the identical source: it remains reviewed and does not create a new inbox item.
7. Restart the test backend against the same schema with time advanced past reveal.
8. Read the public match and heatmap, preserving optional metrics while stripping
   source identity/profile fields.
9. Open the real player profile: verify metric coverage and switch between manual
   and automatic entry filters without fabricating points.
10. Manually correct the result: old match detail and player throw statistics
    detach. Undo: public match detail is 404 and the player's played count returns to zero.

`frontend/tests/autodarts/live-flow.mjs` records browser evidence under
`docs/pr-screenshots/issue-48/`. No real Autodarts account, deployed relay or Slack
channel is contacted. The test disables Slack explicitly, accepts only a loopback
database URL and creates a random private schema. Cleanup drops **only that
test-created schema** and removes its own temporary backend executable. Never set
the test database variable to production, even if forwarded to localhost.

The Go HTTP/Postgres flow additionally simulates a failed acknowledgement followed
by redelivery and proves a single durable import survives. It caught and now guards
the existing public-standings leak before reveal. Unit tests cover both London DST
transitions at the exact reveal boundary; the standings ranking algorithm itself
and mean-of-known-match averages are unchanged.

## Repeat checks

Use Node 22 satisfying the existing dependencies (the default workstation Node
20.13 is too old). Install with the same Node runtime to obtain native optional
bindings. Keep the checked-in lockfile; do not run an automatic audit fix as part
of these tests.

```sh
# Repository root, after installing frontend dependencies:
node score-scrape/scraper.test.mjs
node docs/autodarts/check-v1.mjs
node docs/autodarts-fixtures/verify.mjs

# backend/; TEST_DATABASE_URL must point to disposable local Postgres:
go test -race -shuffle=on -count=1 ./...

# frontend/:
npm test
npm run build
npm run lint
node tests/match/verify.mjs
node tests/matches/verify.mjs
node src/pages/players/browser/verify.mjs
npx playwright test tests/pending-review.spec.js tests/ui-consistency.spec.js
npm run test:e2e
node tests/autodarts/live-flow.mjs
```

Without `TEST_DATABASE_URL`, Go database tests skip; do not report those skips as
database verification. The browser full-flow script requires it and fails rather
than silently using the in-memory fallback. Build the frontend immediately before
running production-preview harnesses. The player-page harness is included above;
it checks public/admin profiles and entry-type filters at all three breakpoints.

## Coordinated cutover (future authorized deployment only)

Final local verification passed: the full Go race/shuffle suite with disposable
Postgres, 130 frontend unit tests, production build/lint, both existing league
end-to-end scenarios, 18 review/layout browser scenarios, and all component,
match-route and player-route browser harnesses (24/36/39 responsive captures).
The real Postgres/browser flow also passed through player profiles, entry filters,
duplicate delivery, hidden-before-reveal metrics and edit/undo detachment.
These are local test results, not a deployment or independent security approval.

1. Resolve the authentication/security decision, review dependency findings,
   take and test a database backup, and rehearse against a copy without real
   notification endpoints. Verify Postgres is used, not the development fallback.
2. Pause producer submission, scheduled polling and manual polling for the
   cutover. Do not run the old delete-on-read reader alongside the new consumer.
   Observe queue retention and backlogs; do not purge messages to simplify upgrade.
3. Apply/rehearse the runtime embedded migrations: legacy pending rows are
   retained/backfilled; source+external ID+digest replaces ID-only deduplication;
   source originals are immutable; review/result links and audit metadata are added.
4. Deploy the new receive/ack reader and backend as a coordinated version pair.
   Enable the new frontend with them: old positional confirmation requests are
   intentionally incompatible with explicit mappings and expected snapshots.
5. Exercise an isolated known import, including failed delivery and duplicate
   handling, admin approval, public gating and edit/undo. Check the actual `.com`
   producer origin, bounded payloads, queue/DLQ metrics and receipt handling.
6. Update the scraper. It defaults to sandbox: enabling live submissions is an
   explicit operator action documented in `score-scrape/SCRAPER.md`. Legacy
   summary payloads remain accepted; no arbitrary sunset/purge timer is applied.
7. Resume polling/producers only after durability and UI/API compatibility checks.

## Rollback and operational diagnosis

- A safe immediate fallback is to stop imports and use compatible admin manual
  entry; preserve queued messages, detailed imports and audit history.
- Do not blindly downgrade to the old reader or schema. The old reader deletes
  on receipt, old code assumes different uniqueness, and the new immutable-source
  trigger rejects writes that were formerly possible. Rehearse a version-specific
  recovery; never drop new columns/detail to make old code start.
- A committed import followed by failed acknowledgement can be redelivered safely;
  exact replay returns the existing reviewed/rejected/pending row, not a new result.
- Changed content creates a separate review requiring explicit replacement; it
  does not mean latest-arriving data is more trustworthy.
- Invalid/oversized payloads remain unacknowledged for bounded retry/native DLQ
  redrive. Diagnose sanitized error categories and message/queue counts rather than
  logging raw bodies, receipts, profiles or credentials. No automatic deletion of
  application imports is implemented.
- Capture missing-coordinate cases as unavailable evidence, not zero accuracy or
  synthetic dots. A manual result remains usable without detailed statistics.

No pushes, deployments, GitHub issue closure, live board matches or dependency
upgrades are part of this verification. Screenshot/browser checks are not a full
Lighthouse, independent visual or accessibility certification.
