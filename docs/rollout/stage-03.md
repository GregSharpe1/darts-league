# Stage 3: expand storage without activating detailed imports

Deploy after the durable legacy relay stage. This stage is deliberately not the
schema from the original integration PR copied unchanged.

## What changes

- Backend startup adds nullable import evidence/linkage columns, audit metadata
  columns and an empty normalized source-player table in an advisory-locked
  transaction. Repeated startup is safe.
- The detailed payload validator and sanitized synthetic corpus are available for
  the next stage, but are not wired into ingestion or exposed by new HTTP routes.
- Existing producer, polling and admin request/response behavior is unchanged.

## Compatibility boundary

The old `pending_results_external_match_id_key` uniqueness constraint remains.
No content-digest unique index, immutable trigger or source backfill is installed.
No existing columns or constraints are removed/relaxed. Existing summaries are
not rewritten and new legacy writes still leave source payload/digest null.
Legacy `ON CONFLICT (external_match_id)` statements, status changes and result
edits continue to work. This is the rollback window, not the approval cutover.

`TestImportExpansionPreservesLegacyWriters` uses real Postgres to create legacy
rows before expansion, repeat expansion, insert/replay with the old conflict
target, update an old row, and assert no source rewriting or immutable trigger.
It requires a disposable `TEST_DATABASE_URL`; without one it skips, not passes.
The repository CI supplies Postgres for this test.

## Deploy and rollback

Back up and rehearse against a database copy. Deploy the backend image with its
existing database/auth Secrets. The application DB role needs DDL privileges.
Check logs for real Postgres use: in-memory fallback is not a valid migration or
durability result. The frontend image does not need to change for this stage.

Keep detailed production submissions disabled: the stage-2 consumer still accepts
only legacy summaries. Keep the new scraper undeployed/sandboxed. Existing legacy
ingestion can continue with the stage-2 reader/backend pair. Authentication remains
a separate production release gate.

Rollback uses the stage-2 backend image with the expanded columns/table left in
place; do not drop columns or restore a backup merely to roll back this stage.
Only stage 4, after old writers are stopped, will remove the old conflict target,
backfill source evidence and enforce immutable originals/targets.

## Checks

From `backend`: `go test ./...` and `go test -race -shuffle=on -count=1 ./...`.
With disposable Postgres, run
`go test -count=1 ./internal/store/postgres -run TestImportExpansionPreservesLegacyWriters`.
No live board or relay is needed for the synthetic validator tests.
