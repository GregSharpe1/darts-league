# Stage 04: admin review contract cutover

This is a **non-backward-compatible cutover**, not a rolling upgrade. Stage 03
only expanded storage; stage 04 activates new writers, approval requests and
immutable import evidence. Deploy the matching backend and frontend images.
Do not run old backends, old admin clients or old queue consumers concurrently
with this stage. Image/deployment selection activates the contract; there is
no additional feature-flag framework.

## Prerequisites

- Integrate stages 01 (UI foundations), 02 (receive/ack transport) and 03
  (additive schema). Resolve any stage-02 legacy durable adapter in favor of
  the final `resultsrelay` implementation and `IngestDurablePayload` wiring.
- Test against a disposable Postgres using `TEST_DATABASE_URL`: all backend
  tests including race, expansion compatibility, migration from legacy and
  expanded databases, approval rollback/concurrency and relay redelivery.
  A run without this variable skips DB tests and is not migration validation.
- Build and retain both immutable image versions, a database backup with a
  tested restore path, and queue retention/redrive configuration sufficient
  for the maintenance window. Record pending imports, results and audit counts.
- Keep the detailed production producer sandboxed until stage 05. This stage
  does not change `score-scrape/scoreScrape.js` or activate detailed production
  captures. Legacy summaries remain accepted, but require format attestation
  and a reason when source date is missing before approval.
- Authentication/relay infrastructure is not delivered or resolved here.
  Existing admin authentication remains mandatory; separately verify relay
  access controls and deployment credentials before enabling ingestion.

## Ordered cutover

1. Announce maintenance; block admin writes and close existing admin tabs.
   Unset `RESULTS_ENDPOINT`, pause scheduled/manual consumers and producer
   delivery as needed. Drain in-flight operations, then stop **all old backend
   writers and old frontend instances**, including jobs and replicas. Preserve
   queued messages; do not purge or acknowledge them as a deployment shortcut.
2. Take the final consistent backup after writes stop. Record its timestamp and
   queue checkpoint. Confirm no old process can reconnect or be restarted by
   the deployment controller. An automatic rollback to an old image is unsafe.
3. Start one stage-04 backend with `RESULTS_ENDPOINT` unset and maintenance
   access only. Startup obtains an advisory transaction lock and executes:
   `initialSchema`, `importExpansionSchema`, contract `imports.sql`,
   `approval.sql`, then legacy backfill, all in one transaction. Expansion files
   remain unchanged. Contract SQL removes external-ID-only uniqueness, installs
   digest/active-result indices and immutable-original/target triggers, and
   permits fixture-less rejection audits. Backfill happens only now, after old
   writers stop; it labels stored summaries as reconstructed evidence, never
   inventing source timestamps, throw detail or confirmed fixture associations.
4. Verify migration success and actual Postgres use. The existing application
   can fall back to memory on a database error: `/healthz` alone is insufficient.
   Do not reopen writes if logs report fallback. Check reconstructed rows,
   statuses, averages and counts, immutable evidence and repeat startup.
5. Deploy the matching stage-04 frontend image before reopening admin access;
   ensure old assets/tabs are reloaded. Smoke-test authenticated list/detail,
   explicit player-to-fixture mapping, expected-result conflicts, replacement
   reason, legacy format/date attestation, rejection and audit records. Check
   that unrevealed fixtures do not contribute to public standings.
6. Once both images and durable storage are verified, reopen admin access.
   Configure `RESULTS_ENDPOINT` only when stage-02 receive/ack transport is
   ready and access controls verified. Receipted messages are acknowledged
   only after durable commit or persisted duplicate detection. Memory fallback
   must never acknowledge them. Failed validation, persistence or acknowledgement
   retains messages for retry/redrive. Verify redelivery does not duplicate an
   import. Detailed production producer activation still waits for stage 05.

## Rollback boundary

Before the migration commits, its transaction can roll back without activating
the contract. Keep writers stopped while investigating and verify the actual
database state before deciding what can restart.

After commit, **do not roll back to stage 03 or older backend/frontend images**
against this database. Old conflict targets and mutation/approval semantics are
incompatible. Unsetting `RESULTS_ENDPOINT` pauses ingestion; it does not undo
the schema, backfill or approvals. Safe rollback is either:

- a known-good stage-04-compatible backend/frontend pair retaining this
  contract; or
- stop all writers again, preserve post-backup imports/results/audits and queue
  state, restore the pre-cutover backup, and explicitly reconcile every write
  and delivery since that backup before reopening. Acknowledged messages may
  no longer be in the queue; restoring the database alone loses them. Never
  replay blindly or assume the old external-ID uniqueness can be reinstated
  after multiple digest versions exist.

## Build and test boundaries

`frontend/Dockerfile` builds with only `frontend/` as its context. Production
`tsconfig.app.json` excludes test/spec/browser harness files; no parent `docs/`
or private captures are copied into the image. In a full repository checkout,
run `npm run test:typecheck`, `npm test`, `npm run build` and `npm run lint`.
CI runs the separate test typecheck before publishing images, so exclusion
from the production compiler does not hide test type errors.

Admin browser regression: `npx playwright test tests/pending-review.spec.js`.
Shared source-analysis regression: `node tests/match/verify.mjs`. Both use
sanitized fixtures; local fonts include their licenses. Exhaustive screenshots
go to ignored `frontend/test-results/`; only representative screenshots belong
in `docs/pr-screenshots/feat-autodarts-04-admin-cutover/` (maximum four).

No public match/player detail routes or statistics features are included.
