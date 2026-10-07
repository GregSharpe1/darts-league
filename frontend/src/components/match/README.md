# Match components (#42)

Props-only React components; no fetching, mutations, routing, approval controls,
or API envelope parsing. Import from `src/components/match`. Component imports
load `match.css`; no global stylesheet changes or extra dependencies are needed.
Existing Barlow/Rajdhani fonts and `--color-*`, `--font-*`, `--shadow-panel` tokens
are reused. The isolated harness bundles the already approved local font files.

## Exports and normalized input

```tsx
import { MatchAnalysis, MatchSummary, Heatmap, MatchStatus } from './components/match'
import type { MatchData } from './components/match'

// The page owns fetching, authorization, DTO parsing, and invalidation.
<MatchAnalysis status="loading" />
<MatchAnalysis status="error" onRetry={refetch} />
<MatchAnalysis key={fixtureId} status="ready" match={normalizedMatch} />
<MatchSummary match={normalizedMatch} />
<Heatmap throws={selectedThrows} label="Player / Leg 1" synthetic={false} />
<MatchStatus status="loading" />
```

`types.ts` exports readonly `MatchData`, `MatchPlayer`, `MatchStats`, `MatchLeg`,
`MatchVisit`, `MatchThrow`, `Position`, `Segment`, `Provenance`, and
`MatchAnalysisProps`. Component prop interfaces are also exported:

- `MatchAnalysisProps`: `{status:'loading'}` | `{status:'error',onRetry?:()=>void}` |
  `{status:'ready',match:MatchData}`.
- `MatchSummaryProps`: `{match:MatchData}`.
- `HeatmapProps`: `{throws:readonly MatchThrow[],label:string,synthetic:boolean}`.
- `MatchStatusProps`: the loading/error variants above; error text is fixed and
  sanitized, not an echoed server or source error.
- `MatchData`: `{players:readonly [MatchPlayer,MatchPlayer],playedAt:string|null,
  detail:{coverage:'partial'|'complete',legs:readonly MatchLeg[]}|null,synthetic:boolean}`.
- `MatchPlayer`: `{id:string,label:string,legs_won:number,stats:MatchStats|null}`.
- `MatchStats`: nullable `match_average`, `points_scored`, `darts_thrown`,
  `checkout_hits`, `checkout_attempts` numbers. Source average is never recomputed.
  Optional nullable extensions: `first_nine_average`, `average_until_170`,
  `highest_finish`, `total_180`, `less_60`, `plus_60`, `plus_100`, `plus_140`,
  `plus_170`. First-nine average, 180s and highest finish display in the summary
  and whole-match analysis; per-leg values are unavailable. Missing/null is not
  zero. Source bands are separate categories, not cumulative thresholds.
- Legs, visits, throws, position fields and segment beds follow
  `docs/autodarts/contract-v1.md` without renaming or coercing raw numbers.

The page's validated DTO adapter chooses player IDs/labels. For public views use
league IDs and nickname fallback only, never source/account IDs or raw labels.
For admin review, match-local IDs can be retained. Remap **every** `player_id`
and `winner_id` with the players, never just reorder names. To show fixture order,
order the two player objects by mapped IDs; statistics remain attached to their
player objects. Source array order is not identity. Labels render as text.

Only supply detail eligible for that view: #43 owns current-source/reveal gating;
#44 owns blocked-import review, warnings and approval. This library is not an
authorization or semantic-validation boundary. Do not wrap review-blocked source
data as validated `MatchData` without a separate conflict presentation. Use
`key={fixtureId}` (or pending import ID) to reset filters between matches. The
initial player is the winner, independent of source order; later selection uses ID.

## Coordinate contract

Plotting supports precisely `units:'board-radius'`, `origin:'bull'`,
`axis_orientation:'x-right-y-up'`. The unchanged source coordinate is rendered at
`cx=250+x*191`, `cy=250-y*191`, matching the approved reference board. All other
declarations and null positions are unavailable for plotting; segment counts,
visit routes, raw coordinates and provenance remain readable. Mixed data plots
only supported recorded positions. Coverage distinguishes recorded positions,
recorded throws and board-plottable positions, not all match darts. A real (0,0)
point plots; a missing point never becomes (0,0). Off-board points are included
in SVG bounds, not clamped. Overlapping kernels illustrate recorded positions,
not a fitted probability distribution; individual points have no runtime jitter.

**No physical Autodarts adapter is verified here.** Never tag unknown raw units
with these labels to make a board appear. The test-only fixture converter declares
this geometry solely for the already approved synthetic randomized reference.
It preserves all 48 supplied x/y values. Source manual/automatic entry and position
provenance are separate; neither is inferred from the existence of coordinates.

## Scope and visual contract

Follows `docs/mockups/autodarts/DESIGN.md`: charcoal gradient cards, coral accents,
condensed display type, 20px radii, 24px padding, paired score hero, pill player
controls, board/statistics columns, and the reference's board rings/palette.
Under 640px columns stack; visit tables intentionally scroll in a labelled,
keyboard-focusable region with a visible mobile hint. Native selects/buttons have
44px targets and focus outlines. Exact raw values are available in a disclosure.

Intentional generalizations of the static reference: no inbox/admin page shell;
no hardcoded nine-dart routes, first-nine averages or 180 counts; generic per-visit
rows support arbitrary 3-0/3-1/3-2 data. Partial counts are explicitly not match
totals; checkout attempts are never inferred. Unknown values are not zero, and
zero attempts displays an inapplicable percentage. Summary-only, loading/error,
empty selection, unsupported and absent coordinates are explicit states.

## Repeat verification

From `frontend/`, with Node 22 and the checked-in lockfile:

```sh
npm ci
npm test
npm run build
npm run lint
node tests/match/verify.mjs
```

The browser check uses installed Playwright Chromium (install with the existing
`npx playwright install chromium` if absent). It builds the **real** components
through Vite's separate HTML entry into ignored `node_modules/.cache/match-harness`,
then serves on an ephemeral loopback port. No backend, login or remote requests.
For interactive inspection: `npm run dev`, then `/tests/match/index.html`.

Screenshots: `frontend/tests/match/screenshots/`, SHA-256 manifest `hashes.json`.
Eight states at 375/768/1280: analysis, unknown geometry, summary-only, partial 3-2,
80-character label stress, loading, error, opponent/leg 3. Browser checks assert
no page overflow, exact positions/counts, fonts, keyboard filters, keyboard table
scroll, retry, and no browser/HTTP errors. Unit tests additionally cover all 48
exact source positions, reversed source order, all segment/player/leg counts,
mixed/missing positions, actual zero position, busts, unfinished legs, long unsafe
labels, unknown stats and zero checkout attempts.

Evidence is component/browser QA, not a full Lighthouse audit, independent design
review, live adapter test, or #43/#44 integration test. No React instrumentation,
chart framework, dependency, global CSS, app route or shared contract was changed.
