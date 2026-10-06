# Autodarts implementation decisions - #36

Maintainer choices **approved**, 2026-10-06, GregSharpe1:
[issue comment](https://github.com/GregSharpe1/darts-league/issues/36#issuecomment-6019959456).
This supersedes the earlier unapproved infrastructure proposal. The wire/API
details in [contract-v1.md](contract-v1.md) are implementation conventions for
those choices, not a claim that each field was individually approved or shipped.

| Decision | Status and scope |
| --- | --- |
| Transport | Approved: current HTTP relay/SQS topology; durable receive/ack after persistence (#38). Keep weekday 09:00-17:00 Europe/London, 15-minute polling and manual fetch. |
| Authentication | Explicitly deferred: **do not implement new producer/relay authentication now**. Preserve existing admin authentication. No new identity infrastructure or authenticated binding prerequisite. |
| Storage | Approved: filtered JSONB plus normalized source identity, status, fixture/result linkage and totals where queried. Preserve originals. |
| Review | Approved: explicit admin approval and replacement, never automatic score replacement on import. |
| Publication | Approved: only current confirmed detail after fixture reveal; the same gating applies to derived statistics. |
| Corrections | Approved: retain original imports, exclude contradictory imported detail/statistics after manual score edits or undo. |
| Averages | Approved: retain existing mean of known match averages; any later dart-weighted metric must have a distinct name. |
| Retention | Approved: **no automatic deletion** of application imports/detail; rejected and historical records stay stored until a policy is agreed. No legacy sunset timer. This does not change existing SQS delivery retention. |
| Delivery | Local tested commits authorized. No push, deployment or issue closure authorized. |

## Existing implementation and required changes

- `score-scrape/scoreScrape.js` emits `{matchId,player1,player2}`. Keep this
  summary adapter; missing averages are null, not zero. Require 501 first-to-3
  for league confirmation, even if upstream settings permit other games.
- `score-scrape/template.yaml` exposes POST/GET `/results` backed by SQS;
  `score-scrape/reader/index.js` currently deletes during reads. #38 must remove
  that destructive read and ack only after durable ingestion/duplicate handling.
- `backend/internal/resultsrelay/{client,poller}.go` remains the HTTP consumer;
  don't replace it with direct queue consumption or change its polling schedule.
- Extend `PendingResultService` and existing `/api/admin/pending-results` routes.
  Current active-season lookup, ID-only deduplication, implicit result overwrite
  and nontransactional confirm are not the target contract. Use explicit admin
  season/fixture mapping and atomic result/link/audit updates instead.
- Existing public fixture/standings lists stay lightweight. Apply reveal gating
  to aggregates too; this is an intentional correction, not a claim that current
  standings already filter every future result.

## Release limitations and remaining questions

The relay remains unauthenticated. Neither a producer-supplied ID, account ID,
timestamp nor a digest proves origin, ownership or accuracy. Untrusted submissions
and relay access remain security risks; admin review and validation limit effects
but do not solve authentication. Authentication design is deferred work, not a
completed acceptance criterion. Do not deploy this iteration.

The physical-board 501 coordinate adapter is **unverified**. Manual/synthetic
fixtures can exercise storage, mapping and presentation now; they are not proof
of hardware accuracy. Preserve optional source coordinates and their units even
when geometry is unknown; board normalization/physical heatmaps need real adapter
evidence later. No such evidence is needed merely to store raw coordinates.

No implementation-wide product sign-off blocker remains. Narrow deferred choices:
future relay authentication design, long-term deletion/backup retention policy,
and verified physical coordinate scale/orientation. None blocks summary/manual
imports or raw optional coordinate storage.

## Check

`examples-v1.json` contains fictitious test-only examples. Run:

```sh
node docs/autodarts/check-v1.mjs
```

This standard-library checker guards examples, limits, local scoring, replay,
mapping, nulls and publication policy. It does not test production persistence,
transport, complete X01 reconciliation or a physical adapter. Dependent workers
must test their own implementations; see the contract for those requirements.
