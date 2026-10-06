# Autodarts pending-results concepts

## 1. Identity
Extend the existing admin UI, not a new dashboard brand. Source contract:
`frontend/src/index.css`, `App.css`, `PendingResultsPage.tsx`, and
`PendingResultCard.tsx`. Large condensed headings, charcoal gradient panels,
warm white text, coral actions. Standalone HTML mockups only; no app integration.

## 2. Color
Reuse background #090909, surface #121314, raised #1a1b1e, text #f6f1eb,
muted #a29d97, accent #ff5b5b and soft accent #ffb0b0. Extend with
success #91d6b1, warning #edc783, subtle white 8% borders. Dartboard: charcoal,
cream #c8c1b6, muted green #355e51 and red #9b4347; heat: coral to amber #ffc66d
to pale yellow #fff0b0. Reserve heat colors for density, not player identity.

## 3. Typography
Existing Barlow body and Rajdhani display fonts, bundled locally with SIL OFL
licenses under `fonts/` so references need no external font requests.
12px uppercase metadata, 14px secondary text, 16px body, 24px panel headings,
48-64px page headings and 72px score. Tabular numerals for comparisons.

## 4. Layout
1180px content width from the app. 4px spacing unit. 24px panel padding,
16px gaps, 20px panel radii, pill controls. Desktop: 244px inbox rail plus
review pane; analytics: board plus statistics. Below 900px rail stacks;
below 640px all detail columns stack. No horizontal page overflow.

## 5. Primitives and states
Reuse app-shell, topbar, brand, eyebrow and panel material via existing CSS.
New local primitives: receipt row, selectable view pills, score hero, labelled
mapping select, paired metric row, segmented player/leg controls, SVG dartboard.
States: review, analytics, unmapped, preview-confirmed, preview-rejected,
coordinates unavailable (retains score, statistics and textual segment counts).
Approval is disabled for empty/identical player mapping. Actions are explicitly
preview-only. No mutations or calls to backend, Autodarts or relay.

## 6. Interaction
Review/analysis view switch; player and leg filters update board and counts;
mapping state toggle; native select mapping; preview confirmation/rejection.
No ornamental motion. Focus outlines and clear hover/selected states.
Tab/Shift-Tab traverses native buttons and selects; Enter/Space activates buttons,
arrow keys operate selects. Selected pills expose `aria-pressed`. The mobile
winning-route table is a labelled, keyboard-focusable horizontal scroll region.
Below 640px a scroll hint makes its deliberately offscreen columns discoverable.
No keyboard trap; preview actions announce their outcome through a status region.

## 7. Accessibility and data fidelity
Admin needs to verify identity before publishing, not infer identity from a
heatmap. Visible field labels, native controls, 44px touch targets, textual
status and chart equivalents. Table captions identify scope. SVG has a title
and description; segment totals accompany heat visualization. Do not rely on
color alone. Use exact per-dart coordinates from the supplied randomized capture,
projected into `../../autodarts-fixtures/randomized.json` without
profile fields or source identifiers. Draw individual dots and overlapping heat kernels;
segment badges show totals, not exact-point counts. Never add runtime jitter.
27 winner darts: 21 T20, 3 T19, 3 D12. Opponent: 21 misses (6, 9, 6 by leg).
Clearly mark synthetic randomized positions and test-match statistics. A missing
coordinate state must draw no points, heat kernels or density legend; never
coerce null coordinates to zero or infer locations from scored segments.

## 8. Scope and handoff
Two views plus unmatched-player and no-coordinate states, at 1280/768/375px.
Mapping to a sample division/opponent is illustrative, not an inferred fixture.
No production claim for matching, statistics ingestion or persistence.
All browser assets are local to this checkout. No new dependencies, React tooling,
backend work, or production UI changes. Production accessibility/performance
audits remain implementation work; mockups get browser layout and interaction checks.
Accepted debt: static review totals intentionally describe only this 3-0 reference;
this is not a general match renderer. Score-only data is a fixture contract, not
a third UI mode. Physical automatically scored coordinates are not verified.
