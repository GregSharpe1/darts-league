# Autodarts references (prepared locally)

Adapted from the maintainer's supplied static concepts, not a reconstruction of
Autodarts. Uses existing Barlow/Rajdhani, charcoal/coral admin styling. Every
match identity is fictitious: Morgan Ember, Casey Vale and `demo-*` IDs.
No application routes, backend changes, new dependencies, submissions or
persistence. Preview actions only change local explanatory text.

## Open from a clean checkout

From the repository root, serve the checkout:

```sh
python3 -m http.server 8765 --bind 127.0.0.1
```

Open <http://127.0.0.1:8765/docs/mockups/autodarts/>.
Use `?view=analysis` for analysis and
`?view=analysis&coordinates=missing` for the no-coordinate state.
The unmatched-state button shows mapping-required review. The reference reuses
`frontend/src/index.css` and `App.css`, so serve the repository root, not this
directory. Fonts and licenses are bundled under `fonts/`; no external requests,
login, private captures or live game are required.

## Reference matrix

Each link family has desktop (1280), tablet (768), and mobile (375) captures in
`screenshots/`. These are generated from sanitized DOM content, never copied
from the original unsanitized screenshots.

| State | Desktop | Tablet | Mobile |
| --- | --- | --- | --- |
| Inbox/review | [1280](screenshots/review-1280.png) | [768](screenshots/review-768.png) | [375](screenshots/review-375.png) |
| Mapping required | [1280](screenshots/unmatched-1280.png) | [768](screenshots/unmatched-768.png) | [375](screenshots/unmatched-375.png) |
| Analysis | [1280](screenshots/analysis-1280.png) | [768](screenshots/analysis-768.png) | [375](screenshots/analysis-375.png) |
| No coordinates | [1280](screenshots/no-coordinate-1280.png) | [768](screenshots/no-coordinate-768.png) | [375](screenshots/no-coordinate-375.png) |
| Opponent / leg 3 | [1280](screenshots/opponent-leg3-1280.png) | [768](screenshots/opponent-leg3-768.png) | [375](screenshots/opponent-leg3-375.png) |
| Confirmed preview | [1280](screenshots/confirmed-1280.png) | [768](screenshots/confirmed-768.png) | [375](screenshots/confirmed-375.png) |
| Rejected preview | [1280](screenshots/rejected-1280.png) | [768](screenshots/rejected-768.png) | [375](screenshots/rejected-375.png) |

[DESIGN.md](DESIGN.md) specifies tokens, responsive geometry, keyboard behavior,
chart equivalents, unavailable-coordinate behavior and accepted prototype debt.
[Fixtures](../../autodarts-fixtures/README.md) describe provenance, ID remapping,
offline checks and the unverified physical-board acceptance procedure.

## Repeat checks

```sh
node docs/autodarts-fixtures/verify.mjs
# With the existing frontend dependencies and Playwright Chromium installed,
# and the local server above running:
node docs/mockups/autodarts/verify.mjs
```

A clean development checkout can install the existing lockfile with
`npm ci --prefix frontend`, then install its Chromium browser with
`frontend/node_modules/.bin/playwright install chromium`. This adds no dependency
to the application. Installation may require network; running the reference
checks does not. `MOCKUP_BASE_URL` can select another loopback port if 8765 is in
use; `PLAYWRIGHT_MODULE` optionally selects an already-installed Playwright module.
Neither variable is required in a normal checkout.

The browser check blocks non-local requests, verifies the sanitized winner name
before taking screenshots, checks 1280/768/375 layouts, all player/leg counts and
exact SVG positions, mapping guards, preview actions, keyboard focus and the
no-coordinate state. Screenshots are regenerated in place. This is browser
prototype QA, not a claim of a full production accessibility/Lighthouse audit or
independent review. See [QA.md](QA.md) for the local review record.
