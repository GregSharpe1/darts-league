# Autodarts v1 minimum implementation contract

Scope/approval: [decision record](README.md). API extensions below are target
contracts, not claims about deployed routes. No new authentication in this work.

## Payload and filtering

POST the JSON payload to the existing `/results` relay. Queue success means
accepted for delivery, **not** stored, approved or published. Keep legacy summaries.
The backend independently validates and projects the allowlisted payload before
storing JSONB; producers filter before sending. Never persist arbitrary upstream
profiles, emails, avatars, cookies, headers, device identifiers or URLs. Unknown
keys are excluded from stored content and digest, not copied into an error log.

Limits: 128 KiB UTF-8 body before parsing; depth 12; exactly two players; at most
five legs, 200 visits per leg, three throws per visit (3,000 throws total). Reject
invalid JSON, duplicate keys, nonfinite numbers, excessive counts/size; never
truncate. IDs: 1-128 ASCII letters/digits/underscore/hyphen. Labels: 1-80 Unicode
characters without control characters. Numeric counts are nonnegative integers.

Versioned allowlist (all fields required except optional extensions below):

| Field | Shape |
| --- | --- |
| `schema_version` | `autodarts.import.v1`; unknown versions rejected |
| `source`, `external_match_id` | `autodarts`, nonempty source match ID |
| `playedAt` | Original source RFC3339 timestamp with offset, or null if unknown; never substitute scrape/receive time |
| `settings`, `completed` | `{base_score:501,legs_to_win:3,out:"double"}`, `true` |
| `players` | Two `{match_player_id,account_id,display_name,legs_won,stats}` objects; unique match-local IDs, nullable account IDs, valid 3-0/3-1/3-2 in either order |
| `stats` | Null or `{match_average,points_scored,darts_thrown,checkout_hits,checkout_attempts}`; every member nullable |
| `detail` | Null or `{coverage,legs:[...]}`; coverage is `partial` or `complete` |

Average: finite 0-180. Points <=5010, darts/attempts <=3000, checkout hits <=3;
hits <= attempts <= darts when known. Zero darts requires null average and zero
or unknown points. Zero attempts yields null percentage. Unknown is never zero.
Source match average is the supplied three-dart average, not a recomputation from
possibly incomplete totals. Names and account IDs are unverified claims, never
stable league identity or automatic cross-season mapping keys; guests map manually.

Optional nullable `stats` extensions (match scope only):

| Field | Source `matchStats` field | Bounds |
| --- | --- | --- |
| `first_nine_average` | `first9Average` | Finite 0-180 |
| `average_until_170` | `averageUntil170` | Finite 0-180 |
| `highest_finish` | `checkoutPoints` | Integer 0-170 |
| `total_180` | `total180` | Integer 0-1000 |
| `less_60`, `plus_60`, `plus_100`, `plus_140`, `plus_170` | `less60`, `plus60`, `plus100`, `plus140`, `plus170` | Each integer 0-1000 |

Join source rows by `playerId`, never array position. Scoring bands are separate
source categories, not cumulative counts; do not sum thresholds or infer attempts.
The count bound follows the maximum 1000 recorded visits; partial detail is not
used to recompute these source match metrics. Source `score=0` is not total points:
keep `points_scored` unknown unless a complete derivation has been verified.

Absent or null means unavailable; zero is a known source value, never a fallback.
Missing extensions remain absent from filtered canonical JSON, so existing v1
canonical bytes and digests are unchanged. Explicit null is retained in stored
filtered JSON and participates in its digest. Typed output uses nullable pointers
with `omitempty` and optional decoder tags: nil extensions omit, known zero stays.
Reading the existing filtered JSONB into the typed payload retains known values;
no new database columns are required. UI shows source match metrics on whole-match
scope only; absent per-leg metrics display Not available, not match values or
partial visit totals. Existing golden examples remain the unextended v1 vectors.

Each leg: `{number,completed,winner_id,visits}`; unique number 1-5, winner a
match-player ID when completed, otherwise null. Visits:
`{number,player_id,start_remaining,end_remaining,bust,throws}`; ascending positive
number <=200, remaining 0-501 (start >=2), 1-3 throws. Throw:
`{number,segment,entry_type,position}`; contiguous number 1-3. Segment:
`{bed,number}` with miss/0, single|double|triple/1-20, outer_bull|inner_bull/25.
Entry type: automatic/manual/unknown, independent of coordinate presence.

Position is null or `{x,y,units,origin,axis_orientation,provenance}`. x/y are finite
source numbers **unchanged**; units/origin/axis_orientation are bounded labels or
null when unknown. Provenance is automatic/manual/unknown. Unknown source geometry
does not prevent storage; it prevents treating values as normalized board points.
Do not infer units, flip axes, fabricate segment centers or replace missing points
with (0,0). Physical normalization is deferred until a verified adapter exists.
Synthetic examples are test-only, not live performance evidence. Real manual
coordinates may be stored with manual provenance and their stated source units.

Structural invalidity prevents ingestion. Semantically conflicting detail is
retained filtered with `review_blocked` and a reason, not silently repaired or
used in aggregates. Validate local transitions, bust rollback, no throws after
checkout, double-out, player references, leg winners and complete-detail totals.
Complete coverage also requires full turn/remaining continuity from 501 and
reconciliation with summary scores. Partial coverage must not claim full totals.
Conflicting detail needs a corrected import or separate audited manual result.

Legacy: `{matchId,playedAt?,player1,player2}`, each player
`{name,legsWon,matchAverage?}`. Missing/null average and timestamp become null;
create match-local `legacy-1`/`legacy-2`, null account IDs and null detail/totals.
Record `settings_evidence:legacy_unverified`; confirmation requires explicit
`format_attested:true`. No invented settings/completion evidence, no sunset date.

## Durable ingestion and replay

Keep HTTP/SQS; #38 owns the receive/ack wire extension to existing GET `/results`
and its acknowledgement operation. Preserve `{messages:[{body:...}]}` semantics
with per-delivery acknowledgement data. No producer identity/binding envelope or
new relay namespace is required. Ack only after durable import, known duplicate
or durable sanitized invalid-message disposition. Storage failure means no ack.
An uncertain ack may redeliver; never undo a committed import. Do not run the old
delete-on-read consumer concurrently. Queue retention is not application retention.

Use `(source,external_match_id)` to group imports across seasons. Store immutable
filtered JSONB, digest, original playedAt, received_at, status, review reason and
nullable season/fixture/result links alongside existing pending-result data.
Use one pending-result row per distinct content digest and a unique constraint on
`(source,external_match_id,digest)`; no revision-head/event infrastructure needed.
Digest: SHA-256 of RFC 8785 canonical JSON of the filtered payload: recursively
sorted keys, compact UTF-8, array order preserved, ECMAScript number encoding,
-0 becomes 0. Do not include receive times, mapping or review metadata. Legacy
hashes its filtered legacy shape.
Canonicalization must agree across languages; examples lock vectors. Digest is
content identity, not authenticity. Changing playedAt or player order is a change.

- Exact duplicate: return the existing row regardless of status, do not notify
  again, reopen, rebind or replace anything. Enforce this under concurrent receive.
- Changed digest: new pending row linked by the same source identity; warn
  `changed_import`. It never updates an existing original or current result.
  Arrival order does not establish source truth; explicit review chooses content.
- Rejected identical bytes stay rejected; changed content can be reviewed.
  An approved import's target cannot silently move to another season/fixture.
  A conflicting target for the same source identity returns 409 `target_conflict`.
- Keep all originals, including rejected/blocked/superseded data. No automatic
  purge, expiration, reopening or legacy removal.

## Approval and correction

Extend current confirm request with `season_id`, `fixture_id`, `replace_result`,
`expected_result` and `reason`, retaining `player_one_id`/`player_two_id` as the
league players corresponding to the **source order**. `expected_result` is null
for an empty fixture, otherwise the last-read `{id,player_one_legs,player_two_legs,
player_one_average,player_two_average}` snapshot. Compare atomically under lock;
no new result-version service is needed. Concurrent changed state returns 409.

Always require explicit season/fixture selection; never infer current season from
arrival time. Show original playedAt and warn when absent or outside the selected
season's known dates; require a reason acknowledging that mismatch. playedAt is
untrusted context, not authenticated binding. Missing legacy time cannot prevent
manual approval, but cannot silently select a new season either. Validate fixture
season and both distinct players, read-only closed seasons and fixed score format.
Reverse scores, averages and **all** detail references together when source order
differs from fixture order. Mapping means admin selection, not account ownership.

An existing result requires `replace_result:true`, matching `expected_result` and
nonempty reason; otherwise return 409. Empty fixtures require false/null. Commit
result, pending status, active source link and audit (actor, old/new values, source
IDs and reason) in one transaction. Reject remains explicit and audited. States:
pending -> confirmed/rejected; semantic conflicts -> review_blocked. Replacing or
conflicting manual score/average edits/undo detach the former active source link
and mark it superseded, preserving its JSONB. Exclude detached imported stats and
detail, including source averages unless explicitly supplied as manual values;
unchanged saves keep links. Undo must not automatically reattach source.

## Statistics and public projection

Keep points 2/0, tie-breaks and the existing arithmetic mean of **known** match
averages, not a dart-weighted mean or sum divided by played count. Expose coverage
as known/eligible match counts for new analytics. Later dart-weighted analytics
must be distinctly labelled and need complete validated points/darts. Bust darts
count but bust visits score zero; checkout counts actual throws, not three padded
darts. Unfinished/partial legs do not establish complete match totals. Unknown
checkout attempts stay null; don't infer intent from a missed segment.

Imported public detail/statistics require a current confirmed result/source link
AND fixture reveal <= server now (Monday 09:00 Europe/London, DST-aware). Reveal
also gates manual results in public aggregates/counts, not only page buttons.
Apply the same rules to heatmaps. Hidden and absent detail both 404.
Revealed summary-only results return 200 with `detail:null`. After undo a result
is absent; after manual correction expose the current summary, not detached detail.
Historical confirmed/revealed fixture IDs remain readable after season rollover.
Public projection uses league IDs/nickname fallback, never account/source IDs or
raw source labels. Coordinate coverage counts recorded positions/recorded throws,
not all match darts; unknown-geometry positions are not board-plottable coverage.

## API extensions on existing paths

Existing admin authentication remains required. Add `schema_version:autodarts.api.v1`
to new DTOs, preserve existing list fields/envelope and existing result responses.

| Method/path | Response / request |
| --- | --- |
| GET /api/admin/pending-results | `{pending_results:[summary...]}`; current default pending list, no nested source/legs/throws |
| GET /api/admin/pending-results/{pendingID} | `id`, `status`, `source_payload`, normalized `detail`, `playedAt`, `settings_evidence`, `warnings`, `season_id`, `fixture_id`; admin-only |
| POST /api/admin/pending-results/{pendingID}/confirm | Explicit mapping/replacement above -> 200 existing result DTO |
| POST /api/admin/pending-results/{pendingID}/reject | `{reason}` -> 204 |
| GET /api/fixtures/{fixtureID}/autodarts | Versioned normalized players/detail/coverage -> 200, or indistinguishable 404 |

Admin/public fixture lists never embed throws. Detail responses use no-store;
invalidate any derived data after confirm, replace, edit, undo and reveal. Render
labels as text. Keep existing `{error:{code,message}}` HTTP error shape; never
echo source data. Relevant codes: 400 `invalid_json`,
401 existing admin unauthenticated, 404 `not_found`, 409 `replacement_required` /
`stale_result` / `target_conflict` / `already_reviewed` / `season_closed`, 413
`payload_too_large`, 422 `unsupported_version` / `invalid_payload` /
`detail_conflict` / `mapping_mismatch` / `format_attestation_required` /
`played_at_review_required`, 503 `temporarily_unavailable`. Relay transport status
and ack serialization belong to #38; don't fabricate synchronous import decisions
in the queue submission response.
