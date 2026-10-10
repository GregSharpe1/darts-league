# Stage 02: durable legacy relay

For deployment use the [Terraform relay handoff](../../score-scrape/terraform/README.md).
The obsolete SAM deployment entrypoints have been removed without changing any
existing AWS stack. Terraform does not automatically migrate an old queue backlog.
This document records the historical stage-2 legacy backend contract; stage 4
replaces that adapter with content-versioned ingestion and explicit review.

## Scope and release gates

This stage is independently usable with the current Postgres schema and old
admin/frontend. It stores legacy summaries in `pending_results`; it does not
approve results or change standings. Existing admin confirmation, rejection,
result validation and `ResultStore` interfaces remain unchanged. No migration,
backfill, trigger or index is added. The old `scoreScrape.js` producer continues
unchanged. Detailed/v1 producers remain sandbox-only and undeployed.

**Public rollout is gated. Authentication is explicitly deferred.** The relay
routes still use `NONE` authorization. Anyone with network access can
submit fabricated results, receive private match data/receipt handles, or delete
deliveries through `/results/ack`. CORS is not authentication: restricting the
browser origin does not stop non-browser clients. Do not expose this as an
approved public deployment until an access-control decision and its verification
are complete. No new identity system is introduced in this stage.

Real-Postgres durability/concurrency tests must also pass against a disposable
`TEST_DATABASE_URL` before rollout. That variable was absent during local
verification; these tests were skipped, not passed. No database infrastructure,
parent/production database access, deployment, push or PR was performed.

## Delivery contract

- `GET /results` receives at most ten SQS messages without deleting them. Each
  contains `messageId`, `receiptHandle` and the raw legacy `body`. Oversized bodies
  carry a rejection marker and stay unacknowledged for native redrive.
- Backend parses at the untrusted boundary, before duplicate lookup: max 256 KiB
  UTF-8 body, nonblank match ID (max 128 bytes, no control characters), names using
  the existing 60-character/normalized-spacing convention, no control characters,
  present integer nonnegative legs bounded by Postgres INTEGER, and a decisive
  score. Missing/null averages stay null; numeric averages must be finite and in
  0..180. Actual fixture first-to settings remain enforced by admin confirmation.
  Unknown fields (including version/detailed fields), malformed JSON, missing IDs
  or incomplete receipt identities are never acknowledged, even for an existing ID.
- `PendingResultService.IngestDurable(ctx, PendingResult) error` calls the optional
  store capability `CreatePendingResultDurably(ctx, PendingResult)
  (PendingResult, error)`. Only the real Postgres store implements it in production.
- Postgres uses an explicit READ COMMITTED transaction with `INSERT ... ON
  CONFLICT (external_match_id) DO NOTHING`. A conflict is followed by a separate
  lookup comparing the committed names, scores and nullable averages. Only a successful commit returns success or
  `ErrDuplicateExternalMatch`. Pending, confirmed and rejected duplicates all
  qualify; their existing contents/status are not overwritten. A failed lookup,
  write, cancellation or uncertain commit is an error, not a duplicate.
- Reusing an ID with different content is not an acknowledged duplicate. It stays
  queued for retry/DLQ handling without overwriting the stored summary. Preserve
  that message for review/redrive after stage 4 adds content-versioned imports.
- `Poller.WithDurableIngest` now accepts
  `func(context.Context, league.PendingResult) error` (validated legacy data, not
  a raw transport `Message`). `cmd/api` wires the service only for `*postgres.Store`.
  The service independently refuses memory fallback even if wired accidentally.
- `POST /results/ack` deletes only supplied receipt identities after persistence.
  Batch partial failures are surfaced; already committed messages can replay
  safely with new receipts. Poison/uncommitted messages are never deleted by GET.
- Existing notifier is invoked only for the fresh committed insert, never a
  duplicate. Notification remains best effort, not an outbox: a process crash
  after commit can lose the notification, but not the pending result.
- Receiptless responses remain supported for local development and are never
  explicitly acknowledged. **The old deployed destructive reader is unsafe**:
  backend receiptless compatibility cannot undo deletion performed by that reader.

HTTP requests time out after 12 seconds; a backend poll has a 45-second deadline.
The Lambda has a 10-second timeout and an 8-second SDK abort signal. The source
queue visibility timeout is 180 seconds, retention four days, max receive count
five. The encrypted DLQ retains messages for 14 days. Terraform protects both
queues with `prevent_destroy`; see the handoff for its limits. CloudWatch alarms cover visible DLQ messages and source
age over three days; operators must connect alarm actions to their alerting
destination (the configuration does not choose one). Logs contain counts and generic
failure categories, never bodies, receipt handles or SDK error details.

## Paired deployment procedure (not executed)

1. Satisfy the authentication/public-exposure and disposable-Postgres test gates
   above. Check existing queue backlog, retention deadlines and alarm routing.
2. Pause **all** scheduled and manual polling consumers. Set `RESULTS_ENDPOINT`
   empty and restart backend consumers; empty keeps polling disabled. Stop any
   separate old readers. Keep the legacy producer unchanged: messages can queue
   during the short maintenance window, within source retention.
3. Follow the Terraform handoff to deploy the reader, receive/ack routes and queues
   while polling stays paused. Preserve existing state for an already-deployed
   Terraform relay. Confirm `allowed_origin` is `https://play.autodarts.com`.
4. Verify the backend is using Postgres (not fallback memory), schema matches its
   deployed application stage, and both relay routes are the paired versions.
   Test fresh delivery, lost acknowledgement/redelivery, pending visibility in the
   old admin, and rejected duplicate behavior in the approved test environment.
5. Only after those checks restore `RESULTS_ENDPOINT` to the paired `/results`
   endpoint and resume polling. Observe committed/acknowledged/failure counts and
   DLQ/source-age alarms. Default scheduled polls are weekdays 09:00-17:00 local.

Rollback: pause polling first and leave `RESULTS_ENDPOINT` empty. Preserve queue,
DLQ and pending rows; never switch back to destructive GET while messages remain.
Do not purge or blindly redrive the DLQ. Diagnose malformed/detailed payloads and
only redrive reviewed deliveries after the appropriate compatible stage is active.

## Verification and Stage 3 handoff

Local checks: `go test ./...` and `go test -race -shuffle=on -count=1 ./...` from
`backend/`; `node --test` from `score-scrape/reader/`. Infrastructure checks and
Node 22 packaging instructions are in the Terraform handoff. The HTTP/service
tests include simulated storage failure, lost ack, partial ack, response bounds,
timeouts, malformed/detailed input, null averages and memory refusal. Simulated
storage tests are not evidence of real Postgres durability. Two new real-Postgres
tests cover concurrent replays, one fresh notification, rejected/confirmed replay,
cancellation, and no ack after storage failure; both were explicitly skipped here.
The reader runtime is Node 22. Local mocked checks are not a live AWS validation.

Use only a disposable test database: existing full-suite database tests reset
tables. The new relay-specific tests create and drop their own random schema.
Run the current backend tests with that disposable URL to exercise the deployed
stage's real commit/uniqueness boundaries.

Stage 4 replaces the legacy-only parser and typed callback
when introducing detailed persistence; do not silently drop detailed fields into
this legacy adapter. Keep the commit-before-ack, persisted-duplicate verification,
memory refusal and nullable-average guarantees. Reconcile the callback signature
when stacking later reference commits: earlier integration code expected raw
`Message`. No detailed import types, detailed producer activation, new approval
routes or detailed UI behavior have shipped here.
