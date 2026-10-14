# Heatmap capture investigation - 2026-10-14

## Confirmed cause

The sandbox-1/user-gsharpe deployment runs backend/frontend v0.0.28.
Fixture 250 contains all three legs and all 191 coordinates, but only one dart
is plottable. The source response was read through the signed-in Autodarts browser;
no historical match was modified or submitted again.

| Leg | Raw entry types | Coordinates | Scraper-plottable |
|---|---|---:|---:|
| 1 | detected: 60, corrected: 1, manual_coords: 1 | 62 | 1 |
| 2 | detected: 61 | 61 | 0 |
| 3 | detected: 68 | 68 | 0 |

`score-scrape/scoreScrape.js` maps `auto`/`automatic`, but not the observed
`detected` or `corrected` values. Only `manual_coords` receives board-radius,
bull-origin, x-right-y-up geometry. The frontend's `boardPoint` correctly refuses
to plot unknown geometry. This is an adapter coverage gap, not leg truncation.
The current main-branch scraper and the inspected local script had identical SHA-256:
`70bd462f5e6f13ed3dd86733ddc38883291004e7180c6342cac617bf94b1cfa0`.

Representative raw source values (identities omitted):

```json
[
  {"entry":"detected","segment":{"name":"S1","number":1,"bed":"SingleInner","multiplier":1},"coords":{"x":0.1746658831675783,"y":0.4720249614785737}},
  {"entry":"corrected","segment":{"name":"S1","number":1,"bed":"SingleOuter","multiplier":1},"coords":{"x":0.32674351753930864,"y":0.7569859800263389}},
  {"entry":"manual_coords","segment":{"name":"D6","number":6,"bed":"Double","multiplier":2},"coords":{"x":0.9565339661984197,"y":-0.08826241868864222}},
  {"entry":"detected","segment":{"name":"S20","number":20,"bed":"SingleOuter","multiplier":1},"coords":{"x":-0.06225430746397956,"y":0.7697826296412336}},
  {"entry":"detected","segment":{"name":"S5","number":5,"bed":"SingleOuter","multiplier":1},"coords":{"x":-0.2698222806151241,"y":0.6049824538692758}}
]
```

These examples are consistent with the existing board geometry but do not alone
prove calibration, axis conventions for every entry mode, or how corrections are
generated. A fix should verify those semantics, add explicit entry mappings and
tests, and address previously stored immutable imports through an explicit policy.
No application fix or data migration was made during the initial investigation.

## Follow-up branch fix

The scraper now declares board geometry for `detected` and `corrected`, as well
as `manual_coords`. Detected entries map to automatic provenance; corrected
entries retain unknown provenance rather than guessing the correction mechanism.
Unknown modes and incomplete coordinates remain non-plottable. Stored imports
are unchanged; re-capture and explicit admin replacement are required to repair
an existing result's active source.

Coordinate convention was additionally checked against public Autodarts client
release `3001763c520674c519d0787d2ec5a52319a6ad3a`:

- `https://play.autodarts.com/assets/MatchStatsContent-sssLPKJ4.js` collects
  coordinates across all turns without filtering entry types, and renders them
  with `x*RADIUS+SIZE/2` and `-y*RADIUS+SIZE/2`.
- `https://play.autodarts.com/assets/clients-Km8QNp-R.js` defines `NORM=170`,
  `DOUBLE_OUTER=170/NORM`, `BOARD=225/NORM`, `SIZE=1000` and
  `RADIUS=SIZE/(2*BOARD)`. Thus one coordinate unit is the outer double radius,
  with bull origin, right-positive X and up-positive Y.

The intercepted-XHR regression initially failed with plottable counts `[1,0,0]`
instead of `[15,18,15]` for mixed detected/corrected darts across three legs.
It checks exact coordinate preservation, independent provenance, and missing
coordinates; it never sends data to real services.

## Operator-authorized game

A real browser-controlled test was completed with a clearly labelled guest.
It exists in the operator's Autodarts history, but was never sent to the relay or
league. It is synthetic gameplay, not measured physical dart performance.

- Source time: 2026-10-14T10:18:17.358094Z through 2026-10-14T10:26:24.510769Z.
- X01, 501, Straight In, Double Out, first to two; result 2-1.
- Leg winners: player 1, player 2, player 1; finishes 52, 50, 52.
- Mixed inner/outer singles, triples, doubles, outer bull, inner bull, misses;
  deliberate leg-3 bust at 52 with T20.
- The source padded that bust with two `fill_miss` throws, both with absent
  coordinates (encoded as null in the fixed-length evidence tuples).

| Leg | Visits | Stored throws | manual_coords | fill_miss | Plottable |
|---|---:|---:|---:|---:|---:|
| 1 | 7 | 20 | 20 | 0 | 20 |
| 2 | 7 | 19 | 19 | 0 | 19 |
| 3 | 9 | 26 | 24 | 2 | 24 |

The sanitized per-visit capture is in `../autodarts-fixtures/heatmap-capture-2026-10-14.json`.
Its tuple field names are defined inside the file. It is evidence, not a relay
payload. Exact present coordinates are retained; identities are replaced.
Absent coordinate properties are explicitly represented as null in the tuples.

## Verification and procedure lessons

- Replayed both actual stats responses through the existing scraper's `project`
  function: test game plottable counts `[20,19,24]`; affected match `[1,0,0]`.
- Inspected the affected deployed match in Chromium: later-leg coordinates are
  reported as recorded but unsupported, matching database and API counts.
- Existing offline intercepted-XHR scraper browser test passed during investigation.
- Plain textbox `fill('2')` produced 11 in the custom legs field. Keyboard
  select-all, Backspace, `2`, Tab produced 2; verified before starting the game.
- An early helper timed out waiting for **Next** after checkout, when the UI
  transitions to **Next Leg**. A later action raced the asynchronous UI update;
  inspected the score before continuing, rather than duplicating the throw.
- Waiting for both the HTTP response and visible score update allowed the
  remainder of the match to complete. HTTP success alone is insufficient.

No physical-board throwing session was performed. Manual browser gameplay cannot
reproduce hardware-generated `detected` entries; those were verified separately
from the affected historical response. No credentials, cookies or headers were retained.
