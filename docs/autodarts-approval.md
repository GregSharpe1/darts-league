# Transactional import approval (#41)

This implements explicit review, not public publication or relay authentication.
Source JSON remains immutable. Pending/review-blocked imports are visible to admins;
blocked evidence cannot be confirmed. Legacy evidence needs a format attestation,
and absent source dates need a recorded explanation. Selecting a season and fixture
is explicit: arrival time never chooses the league target.

## HTTP contract implemented for #43

All endpoints below retain existing admin authentication. Detail uses `no-store`.

- `GET /api/admin/pending-results`: existing lightweight `pending_results` list.
- `GET /api/admin/pending-results/{id}`: `pending_result` summary plus `source`,
  `digest`, `changed_import`, `settings_evidence`, `review_reason`, `played_at`,
  normalized source `players`/`detail`, filtered `original_payload`, nullable
  `season_id`/`fixture_id`/`result_id`, `source_active`, `mapping`, `approval`.
- Admin division fixtures add `season_id`, `player_one_id`, `player_two_id` and
  `expected_result` (null for unscored fixtures). The snapshot includes `id`,
  `updated_at`, both leg scores/averages and `winner_id`. Return it unchanged on
  approval; do not reconstruct a timestamp or substitute a current timestamp.
- Confirm accepts `season_id`, `fixture_id`, `mapping` from match-local source
  player IDs to league IDs, `expected_result`, `replace`, `reason`, `attest_format`
  and `missing_date_reason`. `expected_result` must be explicitly present, even null.
  The strict decoder rejects unknown fields. The response is the existing result
  summary. A score already present or changed import needs `replace: true` and a
  reason; stale snapshots fail with 409 rather than silently replacing anything.
- Reject accepts `{reason}` and responds 204. The rejection and actor are audited.

These exact names supersede the preliminary positional confirmation examples in
the initial design. `examples-v1.json` now uses the implemented ID-keyed request.
No new authentication is added. No public source-detail route is introduced here.

## Atomicity and data lifecycle

Postgres applies result, mapping/source-link, status and audit writes in one
transaction. A deliberately coarse league-write lock coordinates approval with
existing lifecycle writes; optimize lock granularity only if throughput requires it.
Memory uses snapshot/rollback transaction semantics and is still not a durable
queue acknowledgement boundary. Each result update advances its microsecond
comparison token, including under the app's frozen or backward test clock.

The same source match cannot silently move to another fixture/season or swap its
recorded mapping. Replacements deactivate the previous source link. Contradictory
manual edits/undo detach detailed evidence but retain original imports and audit
history; an unchanged save preserves linkage. `ImportByFixture`, `ImportByResult`
and `MappedImport` support the next public-detail work. Public callers must replace
source labels with league nickname/display labels and apply reveal checks: the
mapped internal object is not by itself an authorized public response.

## Verification and limits

Backend tests pass, including stale snapshot under a frozen clock, reversed
mapping, duplicate/stale approval, blocked/legacy evidence, replacement/rejection,
manual edits/undo, closure/target conflicts, rollback and audit metadata.
Targeted race tests and Postgres tests passed against a disposable loopback test
database using `TEST_DATABASE_URL`. No production database was accessed.
Tests do not authorize deployment or verify a physical dartboard adapter.

Known source dates are untrusted review context; dates before the selected season
start or after server time require a review reason. No source timestamp proves
fixture identity. No automatic retention/purge or real-account matching is
implemented. New relay authentication remains deferred by maintainer decision.
