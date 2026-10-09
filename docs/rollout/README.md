# Incremental Autodarts rollout

Draft integration PR #49 is reference only. Review and merge these stages in order:

| Stage | PR | Instructions | Compatibility |
| --- | --- | --- | --- |
| 1 | [#50](https://github.com/GregSharpe1/darts-league/pull/50) | [UI foundations](stage-01.md) | Frontend only; old backend works |
| 2 | [#51](https://github.com/GregSharpe1/darts-league/pull/51) | [Durable legacy relay](stage-02.md) | Reader/backend pair; old schema and UI |
| 3 | [#52](https://github.com/GregSharpe1/darts-league/pull/52) | [Expand storage](stage-03.md) | Additive only; rollback to stage 2 remains valid |
| 4 | [#53](https://github.com/GregSharpe1/darts-league/pull/53) | [Admin cutover](stage-04.md) | Explicit non-backward-compatible stopped-writer cutover |
| 5 | [#54](https://github.com/GregSharpe1/darts-league/pull/54) | [Match analysis](stage-05.md) | Stage-4 schema; read endpoints and producer activation |
| 6 | [#55](https://github.com/GregSharpe1/darts-league/pull/55) | [Player analytics](stage-06.md) | Read-only; no new migration |

Each PR after stage 1 is based on its predecessor to isolate its review diff. While
the stack is active, merge with merge commits rather than squash/rebase. After a
predecessor merges, retarget its successor to `main`; do not merge the successor
into the old feature branch. Keep predecessor branches until successors are
retargeted. Downstream drafts must not become ready before dependencies and CI pass.

Merging and deploying are different decisions. No deployment was performed while
splitting the PR. Keep detailed capture sandboxed through stage 4. Pausing ingestion
uses `RESULTS_ENDPOINT` unset; it does not reverse a committed schema cutover.
After stage 4, never run a stage-3-or-older backend against that database. Recovery
requires a compatible image pair or backup restoration plus reconciliation of all
post-backup writes and acknowledged deliveries.

Relay authentication is deferred and remains a public-production gate. CORS is not
authentication. Automatic physical-board geometry is unverified. Synthetic/manual
browser evidence does not prove physical accuracy. Require the real Postgres CI
tests before database rollout; local tests without `TEST_DATABASE_URL` skip them.

Full UI captures are uploaded as CI artifacts with bounded retention. Git keeps a
small representative set. Production frontend typechecking excludes test-only
fixtures outside its Docker context; CI separately typechecks the full test tree.
