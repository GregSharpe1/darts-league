# Sanitized Autodarts reference fixtures

These are allowlisted design/test projections, **not raw upstream API payloads**
and not a proposed production ingestion schema. No accounts, profiles, host
identifiers, authenticated URLs, credentials or original match/player IDs are
included. Names and every retained ID are fictitious. Source timestamps are
omitted. Fixtures are ready from a clean checkout without a private capture.

| File | Provenance and purpose |
| --- | --- |
| `manual.json` | Observed completed browser test using manual coordinate/score entry. Exact source dart positions and leg/visit/dart relationships; not measured physical accuracy. |
| `randomized.json` | User-supplied synthetic randomized positions projected exactly, without new random numbers. Same statistics and relationships as manual. |
| `missing-coordinate.json` | Synthetic derivative of randomized; every x/y is null. Segment and statistic evidence remains available; no invented dots. |
| `score-only.json` | Synthetic score-only derivative; no darts or stats fields, rather than fabricated zero stats. |
| `reversed-order.json` | Synthetic derivative reversing players, match/leg statistics, legs and darts. IDs and positions unchanged. Consumers join on IDs, not array positions. |
| `dart-hashes.json` | SHA-256 of compact JSON dart arrays, freezing all positions and relationships for offline regression checks. |

Morgan Ember (`demo-player-1`) wins 3-0 over Casey Vale (`demo-player-2`).
Winner: 27 darts, 167 three-dart average, six 180s, T20 x21 / T19 x3 / D12 x3.
Each winning leg is 180, 180, 141 (T20, T19, D12), nine darts. Opponent: 21 M20
misses, split 6/9/6 across legs. Statistics remain keyed by remapped player ID;
leg 2 retains the originally opposite statistics order in the base fixtures.

Match ID is `demo-match-001`. Game IDs are `demo-leg-N`; visit IDs append
`-visit-V`; dart IDs append `-dart-D`. Leg numbers are 1-based; visit and dart
indices are the source's 0-based indices (visit is a turn within the leg, not
that player's personal visit count). Every dart retains playerId, gameId and
visitId. No original-to-demo ID lookup table is distributed.

## Offline verification

```sh
node docs/autodarts-fixtures/verify.mjs
```

Node standard library only. Checks score, player IDs, uniqueness, per-player and
per-leg dart/segment/visit totals, averages, checkouts, 180s, coordinate integrity,
cross-fixture relationships, absent score-only detail and identifying-field
exclusion. Hashes detect drift relative to the reviewed projection, not physical
accuracy. Base projections were additionally compared directly to both supplied
private captures locally, matching every coordinate and index.

`prepare.mjs` is an **optional maintainer-only** allowlist projector. It accepts
two explicit private local input filenames (manual and supplied randomized
captures), never obtains credentials or performs network requests. It overwrites
these fixture JSON files and their hashes; review the resulting differences
before accepting new baselines. It is not run by ordinary tests and no private
input filename is a repository dependency.

## Opt-in physical-board acceptance test (NOT VERIFIED)

This procedure is documented, not executed. Manual browser entry and synthetic
coordinates do not demonstrate automatically scored physical-board support.

1. Obtain explicit participant consent and operator approval. Use an isolated
   non-league test match and a real configured Autodarts board. Completing it
   changes the external test account's match history; never create a live match
   from CI, and do not approve it into league standings.
2. Sign in interactively on the operator's own machine. Do not export sessions,
   cookies, tokens, HAR headers, profiles or board host details to this repository.
3. Play one completed automatically scored 501, first-to-three match. Record a
   private local observation log of score, leg, turn, dart and physical region;
   include distinct regions, misses, and a checkout. Identify any manual score
   corrections separately rather than treating them as automatic detections.
4. Inspect the completed match's per-dart response locally. Confirm source
   entry provenance actually indicates automatic scoring, player/leg/turn links,
   score totals, coordinate presence, numeric ranges and axis orientation against
   the observed physical board. Do not require it to resemble the nine-dart test.
5. Check missing/null positions produce an unavailable state, not (0,0) or
   segment-center guesses. Check repeated fetches preserve IDs/positions and
   out-of-order arrays do not swap identities. This reference projects x to
   `250 + x*191`, y to `250 - y*191`; that orientation/scale remains unverified
   for a real automatically scored source until this procedure passes.
6. If sharing evidence, create a new allowlisted projection with fictitious
   identifiers and names, remove unrelated metadata, and manually review it for
   privacy. Do not replace these deterministic baselines with raw captures or
   run the nine-dart-specific projector on an arbitrary physical match.
7. Record pass/fail, board/client software versions in non-identifying form,
   automatic vs corrected darts, missing-coordinate behavior and sanitized
   evidence links. A maintainer must review that evidence before signing off
   real-coordinate support. Until then the status stays **not verified**.
