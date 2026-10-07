# Pending review (#43)

Implemented on `feat/issue-43-pending-review`. Real React route
`/admin/pending-results`, existing admin authentication, mocked local HTTP only.
No live #41 backend integration, relay authentication, deployment or physical
Autodarts verification is claimed.

## Behavior and contract

- Fetch the summary inbox and only its selected detail. No automatic polling;
  **Fetch new results** retains the existing manual import endpoint.
- Parse the detail DTO, preserving source match-player IDs internally, exact
  coordinates, nulls and optional extended metrics. Render labels as escaped text;
  never display account IDs, digest or the original payload.
- Source-reported matches use the existing `MatchSummary` / `MatchAnalysis` /
  `Heatmap`. Blocked imports show reported score plus a conflict warning, not
  validated analytics. Legacy records do not claim source-verified format.
- Explicit current-season, division, two-player and fixture selection. No name
  inference. Fixture-order incoming score is shown beside the existing result.
- Existing or changed imports require replacement consent and a reason. Legacy
  format and missing source date require separate attestation/explanation.
- Missing expected snapshots fail closed. Send the server's exact snapshot,
  including timestamp precision and nullable averages; do not construct one from
  the old result summary. Mapping and replacement consent reset on source reload.
- Conflict/validation responses stop approval until explicit reload and review.
  Synchronous double-click guard and parent busy state protect concurrent actions.
- Confirmation/rejection invalidate pending list/detail, admin/public fixtures,
  standings and season caches. Inactive caches become stale; they are not all
  immediately refetched. Visible success messages survive removal from the inbox.

Actual request examples (matching #41's strict handler):

```json
{
  "season_id": 4,
  "fixture_id": 90,
  "mapping": {"demo-player-1": 11, "demo-player-2": 12},
  "expected_result": null,
  "replace": false,
  "reason": "",
  "attest_format": false,
  "missing_date_reason": ""
}
```

Replacement uses the same fields, with `replace: true`, a nonempty `reason`, and
the exact GET fixture object, for example:

```json
{
  "id": 900,
  "updated_at": "2026-06-15T11:00:00.123456Z",
  "player_one_legs": 3,
  "player_two_legs": 2,
  "player_one_average": 54.125,
  "player_two_average": null,
  "winner_id": 12
}
```

Reject body: `{"reason":"Conflicting capture; request a corrected import"}`;
204 is success. Detail source-player fields are `match_player_id`, `account_id`,
`display_name`, `legs_won`, `stats` (not component `id`/`label`). The API adapter
does not send `player_one_id`, `player_two_id`, `replace_result` or
`format_attested` in confirm requests.

## Evidence

Real-page Playwright captures at **375 / 768 / 1280**:

| State | Screenshot family |
| --- | --- |
| Unmapped summary | `unmapped-{width}.png` |
| Selected opponent / leg 3 | `analysis-{width}.png` |
| Explicit mapping / fixture-order score | `mapped-{width}.png` |
| Existing result / replacement consent | `replacement-{width}.png` |
| Stale approval conflict | `conflict-{width}.png` |
| Missing stats and coordinates | `missing-coordinates-{width}.png` |
| Legacy / missing source date | `legacy-{width}.png` |
| Blocked source | `blocked-{width}.png` |

`detail-error-1280.png` records the retryable selected-detail error. The existing
issue-47 pending-state captures are refreshed by its route-consistency test.
Mock data is fictitious and visibly labelled synthetic QA. Its geometry comes
from the approved sanitized randomized reference; no live positions are inferred.

Approved design reference: `docs/mockups/autodarts/DESIGN.md`. Implementation
retains its 244px desktop rail, 900px stacking breakpoint, dark charcoal panels,
Barlow/Rajdhani fonts, coral controls, native labelled selects and shared match
components. Extra live approval safeguards replace preview-only actions.
Mapping follows the complete shared summary rather than splitting that component.
Native fixture select text may shorten on mobile; full names and scheduled time
are repeated in the visible comparison panel. Captures scroll to the page top
before full-page capture so the sticky header does not obscure the inbox.

## Repeat checks

Use Node 22 and the checked-in dependencies, from `frontend/`:

```sh
npm ci
npm test
npm run build
npm run lint
npm run dev -- --host 127.0.0.1 --port 4393
# Separate terminal, same Node version:
PLAYWRIGHT_USE_DOCKER=1 PLAYWRIGHT_BASE_URL=http://127.0.0.1:4393 \
  npx playwright test tests/pending-review.spec.js tests/ui-consistency.spec.js
```

`PLAYWRIGHT_USE_DOCKER=1` only skips Playwright's backend/server startup here;
these tests intercept API calls and use the local Vite server, not Docker.
Verified: 78 unit tests, production build, lint, 15 review browser scenarios and
3 full route-consistency scenarios. Tests include exact POST bodies, cache
invalidation, reverse fixture order, selected-only fetch, same-player prevention,
invalid pair, missing snapshot, replacement consent, stale reload, legacy/date
guards, missing data, escaped labels, rejection, keyboard activation, authorization
and synchronous double activation. Screenshots assert no horizontal page overflow.

Direct visual inspection covered mobile mapping, tablet analysis and desktop
replacement against the approved layout; no independent reviewer was spawned
(explicit task constraint). No Lighthouse or broad accessibility audit is claimed.

## Existing dependency warnings

`npm ci` under Node **22.12.0** succeeded, but `jsdom@29.0.0` and
`eslint-visitor-keys@5.0.1` declare Node 22.13+ support. The tests/build/lint above
did pass; use a current Node 22 patch for supported tooling in the integration run.

The unchanged lockfile reported **21 audit findings** (1 low, 5 moderate, 12 high,
3 critical). `npm audit --omit=dev` reported **2 high dependency entries**:
`react-router` and its dependent `react-router-dom`, with multiple advisories,
including [GHSA-49rj-9fvp-4h2h](https://github.com/advisories/GHSA-49rj-9fvp-4h2h)
and [GHSA-wrjc-x8rr-h8h6](https://github.com/advisories/GHSA-wrjc-x8rr-h8h6).
These are existing dependency warnings, not newly introduced dependencies or an
exploitability assessment. Some advisories concern SSR/RSC paths rather than this
client-side app, but applicability has not been audited. No `npm audit fix` or
lockfile changes were made: dependency remediation needs a separately scoped
upgrade and regression run before treating the dependency baseline as secure.
