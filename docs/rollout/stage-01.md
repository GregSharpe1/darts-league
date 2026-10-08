# Stage 01: UI foundations

Independently deployable frontend-only change based on `66650fc` (current
`origin/main` at implementation). Deploy the frontend app image against the
existing backend; no later Autodarts stage is required.

- Shared panel/control tokens, keyboard focus, disabled controls, heading sizing
  and reduced-motion styling, adapted from `47a9453`.
- Nullable response average types and the shared formatter fix from `fdb6204`.
  Null/omitted averages remain blank or placeholders; real zero remains numeric.
  Public results, admin score forms, pending results and audit history are covered.
- No new pages, components, endpoints, dependencies or authentication changes.
  No backend, database schema, relay, container or deployment configuration changes.
  Existing request payloads and old backend endpoints remain compatible.

## Verification

Using Node 22 and the existing locked dependencies:

- `npm test`: 38 tests passed (6 files), including 8 new average regressions.
  Before the fix, 4 new cases failed on null `toFixed` in the formatter, public
  results, score form and audit card.
- `npm run build` and `npm run lint`: passed.
- `npx playwright test tests/ui-consistency.spec.js`: 3 mocked browser cases
  passed at 375, 768 and 1280px against the production preview. Checks existing
  navigation, null versus zero display, select focus, disabled/enabled controls
  and reduced motion; no live API is used.
- `GIT_MASTER=1 npx playwright test tests/smoke.spec.ts`: 1 existing lifecycle
  scenario passed, including 8 registrations, division assignment, season start,
  scoring, closure and next-season registration, using the unchanged in-memory
  backend. No database or containers are needed.

For mock-only testing, run `npm run preview -- --host 127.0.0.1 --port 4281`
and set `PLAYWRIGHT_BASE_URL=http://127.0.0.1:4281 PLAYWRIGHT_USE_DOCKER=1`
on the Playwright command. The existing flag only skips managed web servers;
this command does not start Docker. Set `CAPTURE_UI_SCREENSHOTS=1` to refresh
the representative [screenshots](../pr-screenshots/feat-autodarts-01-ui-foundations/).
Other captures go to ignored `frontend/test-results/`. No screenshot sweep or
new design/performance tooling is included.

CI uploads the full UI capture set as `ui-evidence-<run ID>` with 14-day retention.
Only a small explicitly selected set is tracked in Git; generated feature capture
directories are ignored. Later stages add offline harnesses that the same CI step
detects and runs when present. This changes verification, not deployment behavior.

## Rollback

Redeploy the previous frontend app image. Rollback is straightforward: there are
no schema migrations, backend/relay updates, or persisted data transformations to
reverse. The old UI and backend endpoints continue to operate unchanged.
