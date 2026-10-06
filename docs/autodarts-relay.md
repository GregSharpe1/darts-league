# Autodarts relay: durable transport slice (#38)

**Not deployable yet. Authentication is intentionally deferred, not satisfied.**
The maintainer's issue comment and approved implementation scope retain the HTTP
API Gateway/SQS topology without adding producer or consumer credentials. Existing
admin authentication is unchanged. Public submission, receive and acknowledgement
routes remain unauthenticated: anyone with the endpoint can inject results, read
private payloads, consume visibility attempts or obtain receipts and delete queued
messages. CORS is only a browser restriction, not authorization. Rate limits do not
prevent targeted abuse. Do not deploy this change or claim the original auth
acceptance criteria complete. Scoped producer/consumer authentication remains a
release blocker; no server credential belongs in the scraper or logs.

## Integration blocker: durable storage

`backend/cmd/api/main.go` currently falls back to `league.MemoryStore` when Postgres
is unavailable. `PendingResultService` hides the store, so the transport cannot
prove that `Ingest` persisted to disk. A successful memory write must never cause
an SQS acknowledgement. Changing persistence or API startup is outside this slice.

`NewPoller` therefore **fails closed for receipt-bearing deliveries** until the
integration owner wires `WithDurableIngest(func(context.Context, Message) error)`.
It neither ingests nor acknowledges such messages without this callback, and
returns `ErrDurableIngestRequired`. Do not configure `RESULTS_ENDPOINT` against the
new reader until durable ingestion is wired; repeated unconfigured polls would
eventually redrive messages to the DLQ. The old receipt-less response format is
still ingestible for caller compatibility, but never acknowledged by this poller.
That compatibility does not make the old delete-on-GET reader safe.

The callback must:

1. Validate the current payload and stable external match identity.
2. Commit the import using durable storage before returning nil. Honor cancellation.
3. Return `league.ErrDuplicateExternalMatch` only after verifying a committed import
   with that external match ID. Database uniqueness is authoritative under races.
4. Return an error for failed/uncertain commits or invalid payloads; do not swallow it.

The current Postgres `pending_results.external_match_id` unique constraint supports
idempotent replay, and the existing duplicate check queries stored rows. Concurrent
insert conflicts may fail one attempt; the next delivery verifies the committed row.
Detailed payload persistence/validation belongs to the ingestion issue. This slice
only stores what that callback commits; it does not claim durable full-match detail.
The current parser still requires a nonblank `matchId` and both player names.

## HTTP contract

Keep `RESULTS_ENDPOINT=https://<relay-id>.execute-api.<region>.amazonaws.com/results`.
No real credentials or endpoints are required for offline tests.

- `POST /results`: unchanged direct API Gateway -> SQS submission.
- `GET /results`: at most 10 messages, no deletes, `Cache-Control: no-store`.
  Each message has `body` (raw JSON string), `messageId`, `receiptHandle`, and
  `attributes` (`SentTimestamp`, `ApproximateReceiveCount`). Malformed JSON is
  returned unchanged for ingestion validation rather than silently discarded.
- `POST /results/ack`: `{ "messages": [{ "messageId": "<id>",
  "receiptHandle": "<latest-receipt>" }] }`. At most 10 unique IDs/receipts;
  request <=32 KiB, IDs <=128 characters, receipts <=2048 characters.
  Returns `{ "acknowledged": ["<id>"], "failed": [] }`. SQS partial deletion
  returns HTTP 503 with per-message outcomes, not unconditional success.

Receipt handles refer to a delivery, not a stable import identity. Use a new receipt
on redelivery. The reader deletes only explicitly submitted receipts; it cannot
independently verify a database commit or bind the supplied ID to its receipt.
Until authentication is implemented, this boundary trusts the caller completely.
SQS standard queues are at-least-once: even a successful delete may be followed by
a duplicate delivery. Storage must remain idempotent.

## Failure and resource bounds

- The poller processes all deliveries, acknowledges only successful durable commits
  or verified duplicates, and joins processing/ack errors. `PollNow` reports partial
  failures to the existing admin route; scheduled polling reports failure too.
- Failed receives, broken/lost HTTP responses, storage errors, malformed deliveries
  and ambiguous acknowledgements do not cause speculative deletion. Other committed
  members of a partial batch can be acknowledged. Retry is on normal polling or
  manual fetch, not an unbounded tight loop.
- Weekday 09:00-17:00 polling and its existing interval (default 15 minutes), timezone
  selection and manual fetch are unchanged.
- Submission payloads are limited by SQS `MaximumMessageSize: 262144` (256 KiB).
  Legacy oversize messages receive an empty body with `rejection: payload_too_large`.
  Response-budget overflow uses `rejection: response_limit`. Neither is acknowledged;
  the original message remains in SQS for retry/redrive.
- Receive responses are <=2 MiB, including JSON escaping. This also keeps the Lambda
  proxy wrapper below its 6 MiB response ceiling. Go bounds reads to 2 MiB for receive
  and 32 KiB for ack, rejects excessive counts, malformed envelopes and redirects,
  and verifies every expected ack ID. It never sends result bodies in acknowledgements.
- SDK operation deadline 8 seconds, maximum 2 SDK attempts; Lambda/API integration
  timeout 10 seconds; Go HTTP timeout 12 seconds; complete poll context 45 seconds;
  SQS visibility 180 seconds. The durable callback must honor the context. A commit
  racing cancellation may redeliver and must be verified as a duplicate.
- Existing API throttles remain 2 requests/second with burst 5 (shared by routes).
  These are best-effort service throttles, not producer quotas or authentication.
- Allowed browser origin defaults to `https://play.autodarts.com`, controlled by the
  `AllowedOrigin` stack parameter. No `.io` compatibility origin or wildcard remains.
  API Gateway owns CORS; the Lambda does not override it.

## Quarantine, retention and observability

Native SQS redrive moves repeatedly unacknowledged messages to the encrypted DLQ
after `maxReceiveCount: 5`. This includes permanent validation failures, storage
outages and responses repeatedly exceeding the response budget. There is no poison
message deletion in application code. Source retention remains 4 days; DLQ retention
is 14 days (the SQS maximum), with CloudFormation deletion/replacement retention.
For standard queues, expiry retains the original enqueue timestamp: moving to the
DLQ does **not** promise 14 additional days. Five attempts is a receive-count bound,
not an elapsed-time guarantee, and outages may expire source messages first.

There is no new application data purge or automatic DLQ replay. Operators must
inspect/export quarantined originals to access-controlled durable storage before
SQS expiry, fix the cause, then deliberately redrive them. Do not purge a queue to
clear its alarm. An indefinite quarantine/retention policy and automated archival
remain decisions for the ingestion/retention work; bounded SQS retention is not
indefinite data preservation.

Reader logs contain receive/deferred counts, oldest delivered message age, ack
success/failure counts and generic SQS failure events. Poller logs contain batch
counts, not payloads, player names, receipts, URLs, callback error details or secrets.
An ack failure leaves the poller's confirmed-ack count at zero even if SQS partially
deleted the batch; the reader logs the exact successful count.

CloudWatch alarms expose DLQ visible count >0 and source oldest-message age >3 days
(allowing the existing weekend pause before the 4-day expiry). Alarm notification
actions are deliberately not guessed: an operator must attach approved routing.
`DeadLetterQueueUrl` and `AcknowledgementUrl` are new stack outputs. Reader IAM only
receives/deletes on the source queue; producer integration only sends to that queue.

## Verification and remaining acceptance criteria

Offline commands:

```sh
cd score-scrape/reader
npm install --ignore-scripts --no-package-lock
node --test *.test.js
cd ../../backend
go test -race -shuffle=on -count=1 ./...
go vet ./...
```

Tests cover no delete on GET, explicit/partial ack, input/response bounds, malformed
redelivery, failed storage, lost ack, verified duplicate replay, HTTP receive/ack
integration, durable callback gating and infrastructure settings. The HTTP scenario
uses the real ingestion service with an in-memory test store, **not** a claim of
Postgres durability or an AWS DLQ integration test.

Before release: wire the durable callback only to verified persistent storage (no
memory fallback), add the detailed ingestion contract, test real Postgres commit/
crash/replay and AWS visibility/DLQ behavior, validate the SAM stack, resolve auth,
set alarm routing and approve quarantine archival/retention. Coordinate reader and
backend rollout while polling is disabled; never mix the old destructive reader
with new acknowledgement semantics. No deployment is included in this change.
