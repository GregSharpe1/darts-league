# Local reference QA

Deliverables prepared locally for issue #37. No publishing, account activity or
league mutations were performed. Only reference/fixture directories changed.

## Executed checks

- `node docs/autodarts-fixtures/verify.mjs`: passed all five fixtures, frozen
  position hashes, per-player/per-leg totals, visit sequences, ID links and
  cross-variant relationships. Node stdlib only; no private inputs or network.
- `node --check docs/mockups/autodarts/mockup.js`: passed.
- `node docs/mockups/autodarts/verify.mjs`: passed at 1280, 768 and 375px, using
  `MOCKUP_BASE_URL=http://127.0.0.1:8877` and an already-installed Playwright
  module selected via `PLAYWRIGHT_MODULE`. Non-local browser requests blocked.
  Local Barlow/Rajdhani loaded; no JS errors or page-level horizontal overflow.
- All player/leg filters: exact fixture-to-SVG coordinates, dart counts and
  textual segment totals. Missing-coordinate filters: no rendered dart points.
- Empty league, missing player and identical-player mappings block approval;
  preview confirm/reject only update local status. Keyboard focus outlines and
  mobile keyboard table scrolling passed.
- Independent local comparison against both supplied source captures: all 48
  coordinates, player/leg/visit/dart relationships, segments and match statistics
  preserved in each base projection. Original UUIDs and source player names
  absent from every fixture JSON and mockup HTML. Raw captures were not copied.
- All 21 screenshots have PNG signatures and the specified viewport widths.
  Worktree HEAD verified as `feat/issue-37-autodarts-references`.

An initial browser run encountered a pre-existing server on the default port
and failed at player selection. The check now asserts the fictitious winner
before capture. The final successful run used the correct worktree on port 8877
and regenerated every screenshot; the failed run's images are not retained.

## Visual review

Reviewed the three `screenshots/review-sheet-*.png` contact sheets covering all
seven states at each width, plus full-size desktop review/mapping-required,
mobile analysis and tablet no-coordinate captures. The original layout,
charcoal gradient cards, coral actions, condensed headings and explicit admin
mapping are preserved. Names, source ID and office text are fictitious.
The no-coordinate state visibly replaces the board with an unavailable message
while retaining readable statistics and segment totals.

Mobile winning-route columns intentionally scroll within their panel rather
than widening the page. Review identified the partially visible final header;
the final capture adds a scroll hint and a labelled, focusable scroll region,
with keyboard scrolling tested. Contact sheets are layout overviews, not evidence
for tiny glyph or exact-coordinate fidelity; browser assertions cover the latter.

## Boundaries

- These are static design references, not a production feature or persistence
  implementation. Summary copy is specific to this fixture.
- No independent subagent review was requested/permitted. No full screen-reader,
  WCAG certification, Lighthouse or production regression suite was run.
- Real automatically scored physical-board coordinates remain **not verified**.
  The opt-in procedure is documented in the fixture README, not executed.
