# Shared application presentation (#47)

Extend the approved Autodarts references without changing routes or behavior.
Use the existing Barlow body/Rajdhani display typography, charcoal panel gradient,
coral primary actions and warm white/muted text. Match components retain scoped
styles; page integration remains in #43/#44.

Shared tokens: panel radius 20px (16px on narrow screens), field radius 10px,
pill actions, minimum touch height 44px, white 8% borders, existing panel material
and primary-action gradient. Operational page titles cap at 64px; public hero
headings keep their existing scale. Forms inherit body typography. Focus is a
visible coral 2px outline. Disabled actions have a raised neutral surface and
not-allowed cursor. Respect reduced-motion requests globally.

Preserve registration, authentication, setup, fixture scoring, audit, reveal and
season lifecycle behavior. No new dependencies or API changes. Route screenshots
and keyboard/layout checks use deterministic fictitious API data; the existing
real-server smoke test remains the functional workflow check.

Run with the existing frontend dependencies under supported Node 22:

```sh
cd frontend
npx playwright test tests/ui-consistency.spec.js
npm test
npm run build
npm run lint
npm run test:e2e
```

The visual test captures every existing route at 375/768/1280px, plus pending
empty/error/loading and closed-season states. Screenshots live in
`docs/pr-screenshots/issue-47/`. These checks are not a full accessibility audit
or evidence for new routes introduced by other issues; recheck those on integration.

## Verification result

- Presentation test passed at 375/768/1280px, including native-select typography
  and focus, disabled actions, loading/empty/error views, no page overflow and
  reduced motion. Error checks allow the existing query retry delay.
- 53 frontend unit tests, production build and lint passed under Node 22.
- Existing real-server registration/season/scoring smoke and duplicate-fixture
  regression tests passed. Those use isolated in-memory application state.
- No route, API, registration or season/scoring behavior was changed.
