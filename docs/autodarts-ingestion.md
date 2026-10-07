# Durable Autodarts ingestion (#39)

The backend now validates legacy and `autodarts.import.v1` messages and commits
filtered imports before the #38 relay acknowledges their receipts. This is not
confirmation/publication, producer authentication, or deployment approval.

## Boundary and evidence

`internal/autodarts.Parse` enforces the [v1 contract](autodarts/contract-v1.md):
128 KiB before parsing, depth 12, duplicate-key rejection (including escaped
equivalents), valid Unicode, finite numbers, required fields and nullability,
two players, five legs, 200 visits per leg and three throws per visit. Discarded
unknown content is also checked; a container is capped at 3,000 entries. Nothing
is truncated. IDs/labels/counts, settings, score format and statistics are checked.
Errors are fixed codes, never echoed source content.

Typed DTO fields form the storage allowlist. Missing legacy averages remain
unknown; optional legacy fields retain their original presence for hashing.
`playedAt` retains its original offset/string, never receipt time. Unknown source
geometry and real manual positions remain unchanged and unnormalized.

Structural failures are not inserted or acknowledged; native SQS retry/DLQ
handling remains #38's policy. There is no invalid-payload archive. Semantic
conflicts are retained as `review_blocked` with a stable reason: invalid player
references/winners, score transitions/busts, darts after a terminal dart,
turn/remaining continuity, incomplete coverage, legs after match completion or
summary reconciliation. Partial histories do not infer missing turns/totals;
complete histories reconcile wins, known points/darts/checkout hits. Supplied
match averages are not recalculated, and checkout attempts are not inferred.

## Canonical content identity

SHA-256 is computed over RFC 8785 canonical **filtered** JSON. The pinned small
`github.com/cyberphone/json-canonicalization` Go package supplies ECMAScript
number encoding, UTF-16 key sorting and Unicode validation. `encoding/json`
alone does not supply RFC 8785 (notably number formats, escaping and key order).
The dependency avoids a second, subtly incompatible canonicalizer. Counts with
equivalent JSON number spellings, including exponents, normalize identically.
Tests compare Node's contract algorithm with Go for exponent thresholds,
subnormal/max-finite numbers, negative zero, Unicode ordering and string escaping;
the contract's legacy digest is pinned independently.

The digest proves content identity, not origin or authenticity. No source
account/name is treated as a league identity.

## Storage and migration

`postgres.Open` runs the existing embedded schema followed by embedded
`internal/store/postgres/imports.sql` and the Go backfill, in one transaction
under a migration advisory lock. This follows the repository's runtime-schema
mechanism; there is no separate migration runner to invoke.

- Remove the old external-match-only uniqueness constraint. Add unique
  `(source, external_match_id, digest)` and nullable season/fixture/result links.
- Keep one pending row per digest, filtered JSONB, original timestamp,
  received time, evidence, review reason and changed-content warning.
- Add `import_players`, keyed by pending row and match-local player ID, with
  source order, nullable statistics and nullable league-player/fixture-slot
  mappings. Leg winners and visits reference these same source IDs in JSONB.
  A reversed mapping must map IDs and statistics, not reverse throw arrays.
- Existing summaries are reconstructed from their stored columns and marked
  `legacy_stored_summary`. Their original upstream bytes, omitted-vs-null
  fields and source time cannot be recovered. Preserve scores, known averages,
  statuses, review actor/time and IDs; do not invent detail/settings evidence or
  infer fixture/season binding. New legacy messages use `legacy_unverified`.
  Both evidence types require format attestation in #41.
- Existing null/invalid historical source IDs and historical formats are retained,
  not rejected by a new-input validator during migration. Reconstructed evidence
  gets its own canonical digest. A later source payload with different field
  presence is changed content, not a claim that lost original bytes were matched.
- An update trigger protects stored originals and their summary columns after
  digest assignment. Review/link metadata is separate. No retention, purge,
  reopening, result replacement or automatic source-link attachment is added.

`InsertImport` serializes each source identity using a transaction-scoped
advisory lock, checks duplicates, inserts pending data and normalized players,
then commits. The unique index is the final concurrent replay guard. Different
digests become separate rows with `changed_import`; exact replay returns the
existing row regardless of status. It never alters an original or its target.
Notifications occur only after a fresh successful commit, not on replay. They
are best-effort, not a transactional notification outbox.

## Service handoff to #41

- `PendingResultService.IngestDurablePayload(ctx, body) (ImportOutcome, error)`:
  validated durable boundary; `Pending`, `Duplicate`, `Changed` outcome fields.
- `IngestPayload` also supports non-durable in-memory development imports. It is
  not sufficient evidence for queue acknowledgement.
- `PendingResultService.ImportDetail(ctx, id) (ImportRecord, error)`:
  internal/admin-only source evidence, typed players/detail, pending summary,
  changed warning, nullable season/fixture/result IDs and active-link flag.
- `ImportStore`: `InsertImport(ctx, autodarts.Import, receivedAt)` and
  `GetImport(ctx, id)`. `InsertImport` accepts already-parsed internal values;
  use the service parser for untrusted input. `DurableImportStore` additionally
  requires `DurableImports()`, implemented only by Postgres, never MemoryStore.
- Existing `PendingResult` list DTO stays lightweight. `ImportRecord` is **not**
  safe for public serialization: it contains unverified source labels/account IDs.

`main.startResultPoller` wires the durable callback only for the real Postgres
store. Receipt-bearing messages bypass the old legacy-only precheck. Memory
fallback cannot acknowledge them and `IngestDurablePayload` fails with
`ErrDurableStoreRequired`. Receipt-less legacy transport cannot be acknowledged;
when Postgres is configured it still uses the same durable parser/store.
The receipt-less development path also uses detailed ingestion: it preserves
typed evidence and `review_blocked` rather than flattening contradictory detail
into an approvable summary. Memory reads return copies of stored evidence.

## Deliberate exclusions and release gate

#41 still owns explicit target selection, format/date attestation, transactional
confirmation/rejection/replacement/audit, source-target conflict enforcement,
mapping all detail references, detachment/supersession on manual changes, and
admin/public API projections and reveal gating. Link columns are preparation,
not an implemented approval protocol. Existing confirmation routes have not
been redesigned here: **do not deploy this ingestion slice alone**.

Producer/relay authentication remains explicitly deferred. Unauthenticated
senders can submit untrusted claims and queue traffic; validation, digests and
manual review do not authenticate origin or authorize account ownership.
Physical coordinate normalization is also unverified/deferred. No frontend,
producer, relay infrastructure or shared contract/reference files were changed.

## Verification

Run `go test ./...` from `backend`. Set `TEST_DATABASE_URL` only to a disposable
local Postgres to run the integration tests (never a production database).
Run `go test -race -count=1 ./internal/autodarts ./internal/resultsrelay
./internal/league ./internal/store/postgres` as one command for the targeted race
suite. Node is used for cross-language canonicalization tests when installed.

Tests cover concurrent replay, changed content, rejected duplicates, immutable
originals, committed receipt acknowledgement, lost acknowledgements/redelivery,
storage failure/no acknowledgement, memory refusal, schema idempotence and legacy
backfill. Boot tests also start from the old pending table, including a null
historical external ID, and reopen it twice through `postgres.Open`.
Sanitized #37 fixtures are explicitly adapted into v1 test payloads,
joining source IDs and preserving coordinates; they are never decoded as the
production schema. Manual/randomized/missing-coordinate/score-only/reversed-order
fixtures and complete-detail totals/continuity are exercised.
