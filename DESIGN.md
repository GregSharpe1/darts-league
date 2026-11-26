# Darts League design guide

Use this guide for public league pages, tournament pages, and admin tools. The site has a dark sports-club look with charcoal surfaces, warm white type, and coral-red actions. Keep public pages focused on entering a competition or finding the next fixture, score, or bracket. Keep admin pages focused on the task and its outcome.

## Foundations

Use the existing CSS variables and shared classes. Update the shared definition when a change applies across the site. Keep styles specific to a page or component near that feature. [Design tokens](frontend/src/index.css), [Shared layout and controls](frontend/src/App.css), [Public page styles](frontend/src/publicPages.css)

| Role | Existing styles |
| --- | --- |
| Canvas and panels | `--color-bg` (`#090909`), `--color-surface` (`#121314`), and `--color-surface-raised` (`#1a1b1e`). Use `--panel-background`, `--border-panel`, and `--shadow-panel` for shared cards. |
| Text and accent | `--color-text` (`#f6f1eb`) for readable foregrounds, `--color-muted` (`#a29d97`) for secondary details, and `--color-accent` (`#ff5b5b`) or `--color-accent-soft` (`#ffb0b0`) for emphasis. |
| Typography | Use Rajdhani (`--font-display`) for headings and prominent numbers. Use Barlow (`--font-body`) for copy, labels, and controls. Use letter-spaced uppercase for display headings or short labels. Keep longer text in sentence case. |
| Shape and scale | Use the shared `.page-frame` with its 1180px maximum. Use `--radius-panel` (20px) and `--radius-panel-small` (16px) for cards, `--radius-field` (10px) for inputs, and `--radius-pill` for compact navigation and actions. Match inputs and form actions with `--control-height` (3rem). |

Use one clear page heading followed by sections in reading order. Use `hero-panel` and `competition-hero` where the competition needs an introduction. Use the compact hero for data-first pages such as standings. Reuse the league club photo and tournament dartboard photo with their dark gradients so headings and labels stay readable. Keep the target-and-dart mark shared between the home link and favicon. [Header and page shell](frontend/src/App.tsx), [Favicon](frontend/public/favicon.png)

## Navigation

| Where | Header and page destinations |
| --- | --- |
| Home `/` | Brand link only in the header. The two main cards explain league matches and standalone tournaments and link to their respective directories. Event names on the home card stay white through link states. |
| Public tournaments `/tournaments/*` | Show only **League** in the header navigation. Keep sign-up, event details, and bracket actions on the tournament pages. |
| Public league pages | Offer **Tournaments** and show **Register** while league registration is open. Omit **Divisions** on the division directory itself. Use **Fixtures**, **Standings**, and **All Divisions** for division detail navigation. |
| Admin `/admin/*` | Keep admin destinations and session actions in the admin toolbar. Label actions by what they do and make the item affected clear in confirmations. |

## Page patterns

- **League directory and fixtures.** Explain that admins assign divisions. Give each division a clear path to fixtures and standings. Present the final scheduled week and catch-up deadline as labelled UK dates once the schedule exists. On the fixture board, clearly mark locked weeks and show their reveal time while preserving the public pairing-only view.
- **Standings and match data.** Keep scores, points, and averages aligned for comparison and use tabular numerals where possible. Prefer player nicknames in public display, with display names as the fallback. Distinguish missing statistics from a recorded zero. Keep standings in a semantic table with scrolling inside a labelled, keyboard-accessible region. Keep points in view on narrow layouts.
- **Tournament directory and bracket.** Show the event name, date, and registration state together. Offer **Sign up** for open events and **View bracket** for started events. Preserve the bracket's round order and allow horizontal scrolling inside the bracket with a visible hint on narrow screens.
- **League and tournament sign-up.** Reuse `PlayerSignupForm` and the split sign-up card. Show competition details over the image and put the labelled fields and primary action on a solid panel. Give field-level feedback and a visible submission outcome. Replace the entry form with the appropriate closed or started state when entries are unavailable.
- **Admin.** Reuse `admin-card`, `admin-toolbar`, and `toolbar-actions`. Keep controls beside the roster, fixture, or settings they affect. Use neutral secondary actions for navigation and cancellation. Confirm destructive or irreversible actions with the affected item named and the consequence stated.

Reference implementations for each pattern are the [division directory](frontend/src/pages/home/DivisionsPage.tsx), [fixture board](frontend/src/pages/home/DivisionHomePage.tsx), [standings](frontend/src/pages/standings/StandingsPage.tsx), [tournament list](frontend/src/pages/tournaments/TournamentList.tsx), [knockout bracket](frontend/src/pages/tournaments/TournamentBracket.tsx), [shared sign-up form](frontend/src/pages/register/PlayerSignupForm.tsx), and [league admin](frontend/src/pages/admin/AdminPage.tsx).

## Controls, states, and responsive behaviour

- Give each section a clear primary action where needed. Use `--action-background` for the coral gradient on primary actions. Use quiet outlined pills for secondary actions and a neutral raised surface with muted text for disabled controls. Use links for navigation and buttons for actions.
- Show loading, empty, error, and success states in context. Reuse `StateNotice` for panel messages and use explicit status text alongside colour. Keep labels visible, associate field errors with inputs, and announce changes that need immediate attention. [StateNotice](frontend/src/components/StateNotice.tsx)
- Preserve visible keyboard focus and the global reduced-motion behaviour. Make controls and scrollable content operable by keyboard. Check text contrast over hero images and across link, hover, and focus states.
- Follow existing breakpoints. The shared shell and grids stack at 960px, public competition and sign-up layouts stack at 760px, and general spacing tightens at 640px. Let long player and event names wrap. Keep horizontal scrolling inside content that needs it, such as standings and brackets.
- Check affected pages at narrow mobile, tablet, and desktop widths. The existing browser checks use 375px, 768px, and 1280px. Review the page with realistic data, keyboard navigation, and the relevant registration or fixture states. [Responsive browser checks](frontend/tests/ui-consistency.spec.js)
