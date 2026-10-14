---
name: autodarts-game-capture
description: Use when an operator requests an Autodarts test game, raw match stats capture, or investigation of missing heatmaps, per-leg coordinates, or scraper entry types.
---

# Autodarts game capture

Play through the real UI; retain sanitized response evidence, not authentication data.
Read `AGENTS.md`, `score-scrape/SCRAPER.md` and the Playwright skill first.

## Safety and sign-in

- Obtain authorization for a test match: it creates Autodarts history and can affect account statistics. Use a clearly labelled guest, not an invited real opponent or tournament.
- Open a dedicated Chrome profile with localhost-only CDP. Let the operator sign in; stop until they confirm. Never request passwords, inspect tokens, export cookies/storage, or record a sign-in HAR/trace.
- Do not run the production bookmarklet: it disables sandbox. Never submit test results to the relay or league. Mark retained test fixtures `synthetic: true` and replay with all external requests blocked.

```sh
profile=$(mktemp -d /tmp/autodarts-capture.XXXXXX)
google-chrome --user-data-dir="$profile" --remote-debugging-address=127.0.0.1 \
  --remote-debugging-port=9222 --no-first-run https://play.autodarts.com/
```

Attach Playwright using `chromium.connectOverCDP('http://127.0.0.1:9222')`.
Use existing tooling; a disposable Playwright container with host networking can
reach this loopback endpoint when the host lacks Node. Do not expose CDP externally.

## Configure and play

1. **Play Now**, **Add Player**, search/create `Heatmap Capture Test`, **Add**.
2. **Edit settings**: 501, Legs, first to 2, Straight In, Double Out, bull-off Off.
3. The **First to Legs** custom textbox needs click, `ControlOrMeta+A`, Backspace,
   `2`, Tab. Plain `fill('2')` was observed to produce 11. Assert its value is `2`,
   click **Save & Close**, and verify **First to 2 Legs** before **Start Game**.
4. Click the board; wait for the throw POST response **and the visible score/dart
   state update** before another action. HTTP success alone precedes UI updates.
   After **Next**, wait for the next turn. Never blindly retry a timed-out throw:
   inspect the current score first to avoid duplicating an accepted dart.
5. After checkout, wait for **Next Leg**; after the deciding checkout, **Finish**.
   A disabled **Next** is a transition, not a failed throw. Starters alternate.

Use `getByRole('img', {name:'Dartboard', exact:true})`. Inspect current SVG geometry
before calculating clicks; the observed viewBox is `0 0 1000 1000`, centre `(500,500)`.
Clockwise sectors: `20,1,18,4,13,6,10,15,2,17,3,19,7,16,8,11,14,9,12,5`.
For sector index i, angle = i*pi/10; SVG point = `(500+r*sin(angle),500-r*cos(angle))`.
Scale by the board's current width/1000 and height/1000 for locator-relative clicks.

| Target | SVG radius |
|---|---:|
| Inner single / outer single | 130 / 290 |
| Triple / double | 226.667 / 366.667 |
| Outer bull / inner bull | 27 / 0 |
| Miss | 420 |

Example three-leg plan (two players, alternate visits):
- Leg 1: player 1 wins using `T20 T19 S18` (135), `T20 T20 T20` (180),
  `T20 T18 S20` (134), `S20 D16` (52). Opponent mixes low singles, bulls, doubles, misses.
- Leg 2: player 2 starts and wins using `T19 T19 T19` (171), twice `T20 T20 S20`
  (140 each), then inner bull (50). Opponent stays below checkout.
- Leg 3: repeat player 1's route, but bust at 52 with T20, then finish S20 D16
  on the next visit. Check remaining scores and winner after every visit.

## Capture and compare

Attach a response listener **before Finish**, restricted to successful GETs at
`https://api.autodarts.com/as/v0/matches/{id}/stats` (also `.io`). Capture the body
only. Opening/reloading `/history/matches/{id}` triggers the stats request again.

Allowlist match settings, scores, completion, statistics, and games/turns/throws.
Preserve exact `entry`, `coords`, segment, turn order and bust fields. Replace
player/match identities consistently; omit host, account IDs and nested profiles.
Do not keep raw private responses, headers or browser state in the repository.
Existing affected matches may be read with authorization; never edit or resubmit them.

For each leg report: throws, coordinate pairs, raw entry frequencies, and scraper
board-plottable positions. Exercise the actual `project` code from
`score-scrape/scoreScrape.js` offline, not a reimplementation. No outbound POSTs.
Compare with the league API/UI separately; recorded coordinates are not necessarily plottable.

Observed distinctions:
- `manual_coords`: browser clicks, supported geometry.
- `detected` / `corrected`: use supported board geometry; detected is automatic,
  corrected retains unknown provenance. Before the heatmap fix both lost geometry.
  Do not infer hardware calibration from manual tests.
- `fill_miss`: source-generated padding after a bust, with absent coordinates;
  never invent locations for these darts.

## Completion checks

- Verify first-to-two result (prefer 2-1 to cover three legs), per-leg counts and
  double-out finishes against the captured response.
- Retain sanitized evidence with provenance and reproduction results. Explain
  failed attempts and any hardware/geometry validation still missing.
- Stop capture listeners and disposable tooling. Remove temporary helpers.
  Tell the operator before closing their dedicated browser/deleting its signed-in
  profile; do not close unrelated browser sessions. Never commit automatically.

Reference evidence: `docs/autodarts/heatmap-investigation-2026-10-14.md`.
