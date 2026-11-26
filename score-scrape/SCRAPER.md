# Autodarts score capture

Install `scoreScrape.js` in the completed-match browser tab. The green indicator
shows that capture is installed. Set the HTTPS relay endpoint on first use.
Existing settings cookies retain the endpoint. Clear
`autodarts_score_scrape_settings` to change it.

The scraper supports **501 double out, first to 2 or 3 legs** for both league and
tournament imports. Each captured match asks **Was this a league match or a
tournament match?** Choose **League match** or **Tournament match**, then check
the competition, players, score, format and original time in **Submit result**.
The backend finds the fixture or bracket match. Two games between the same
players need separate Autodarts match IDs.

**Sandbox is the default.** Confirmations preview the result. Production
submission requires deliberately setting `window.__scoreScrapeSandbox = false`
before installing the script in a fresh tab. Keep sandbox enabled for synthetic
tests. Recognized top-level synthetic/provenance markers block submission.

An accepted submission is queued for delivery. Automatic approval and admin
review happen in the backend. If a POST fails, **Retry** sends the same bytes,
including the selected competition, for backend deduplication. Closing or
cancelling allows a later stats response to prompt again. The scraper copies
no credentials, headers, profiles or host metadata.

## Observed source mapping

Only GET stats XHRs from `https://api.autodarts.com/as/v0/matches/{id}/stats`
(or the same path at `api.autodarts.io`) are considered. Successful JSON/text
responses must have the matching ID, X01/501/Double, targetLegs=2 or 3, no sets
(targetSets null/0), finishedAt, two distinct players and a matching winner with
2-0/2-1 or 3-0/3-1/3-2 scores. Other app responses are untouched.

- Original `createdAt` becomes `playedAt`, preserving its offset and precision;
  missing time remains null, never capture time. `finishedAt` proves completion.
- `players` and aligned `scores` preserve source order. Match statistics join
  exclusively on `matchStats.playerId`; no positional fallback. `userId` is an
  optional unverified account claim, never the nested user profile.
- `average`, `dartsThrown`, `checkoutsHit`, `checkouts` map to the corresponding
  contract metrics; missing values remain null. Source `score` was zero even for
  the nine-dart winner in the observed capture, so it is **not** total points:
  `points_scored` stays null, including when only partial visits are available.
- Optional match stats map `first9Average` -> `first_nine_average`,
  `averageUntil170` -> `average_until_170`, `checkoutPoints` -> `highest_finish`,
  `total180` -> `total_180`, and `less60`/`plus60`/`plus100`/`plus140`/`plus170`
  -> `less_60`/`plus_60`/`plus_100`/`plus_140`/`plus_170`. Missing keys are omitted;
  explicit null remains null, and zero is a known zero. Averages are finite
  0-180, highest finish is integer 0-170, counts are integers 0-1000. Bands are
  separate source categories, not cumulative thresholds. No attempts or totals
  are inferred, and match values are not copied into per-leg statistics.
- `games[].leg`, `turns[].turn`, `throws[].throw` are zero-based; output adds one.
  `winnerPlayerId` and `playerId` retain match-local identity. `turn.score` is end
  remaining; start is score+points, or score for a bust rollback. All games and
  turns are sorted by source indices; duplicate indices reject rather than drop.
- Source `Triple`, `Double`, `Single`, `Outside` map explicitly; `Outside/M20`
  becomes contract miss/0. Single/Double 25 and OuterBull/InnerBull map to bulls.
  The observed `SingleOuter` and `SingleInner` beds map to `single`, preserving their number and coordinates.
  Unknown beds reject rather than silently dropping darts.
- `manual_coords`, `manual_segment`, `manual` map to manual;
  `detected`/`auto`/`automatic` map to automatic. `corrected` retains unknown
  provenance: the entry value alone does not establish who or what corrected it.
  Other entry values, including source-generated `fill_miss`, remain unknown.
- `coords.x/y` are unchanged. Missing/incomplete pairs become null, never (0,0).
  `manual_coords`, `detected` and `corrected` use the outer double-ring radius
  as one unit, the bull as origin, x right and y up. Declare that supported
  `board-radius` geometry independently of entry provenance. The observed raw
  responses and upstream renderer are documented in
  `docs/autodarts/heatmap-investigation-2026-10-14.md`.
  The captured T20 radius is about 0.60; the D12 radius is about 0.97;
  misses may exceed 1. No values are rescaled or inferred from the segment.
  Other entry modes, including legacy `auto`/`automatic`, retain unknown geometry
  until separately verified; plotting is not evidence of physical accuracy.
- Detail coverage is conservatively partial even when all source games appear
  present. The producer does not claim total reconciliation; backend #39 owns
  semantic conflict detection. Invalid structure rejects; detail is not truncated.

The historical response establishes the `detected`/`corrected` entry names and
the upstream renderer establishes their coordinate convention. Physical-board
calibration and arbitrary correction history still need operator-approved
real-board acceptance; the new browser-click test is not a hardware test.

## Previously imported matches

Older receipts without a competition type still arrive in Pending Results.
An admin must choose the intended competition for manual approval. Existing
scores and source evidence stay linked to their original destination. Reload
Autodarts before installing a newer scraper in a tab where it is already active.
See the [Pending Results guide](../README.md#pending-results-and-tournament-imports).

## Offline checks

Use the existing frontend Playwright dependency, or an isolated installation:

```sh
node --test score-scrape/scraper.test.mjs
# Alternatively point to an already installed module, without production deps:
PLAYWRIGHT_MODULE=/path/to/node_modules/playwright node --test score-scrape/scraper.test.mjs
go -C backend test ./internal/autodarts
node docs/autodarts-fixtures/verify.mjs
```

The browser test intercepts **every** request, fulfills a fictitious app/API,
captures relay POST bodies, and never contacts live services. Sanitized fixture
positions are replayed only in that isolated context and compared exactly.
