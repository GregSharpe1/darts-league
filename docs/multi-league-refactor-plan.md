# Multi-League Refactor Plan

## Goal

Refactor `darts-league` from a single global active season into a system that can support multiple concurrent leagues, each with its own current season, registration window, fixtures, standings, admin workflow, and notifications.

## Current State

The codebase is already season-oriented in several places, but the application still behaves as if there is only one global league:

- Backend services resolve work through a single `GetActiveSeason()` store call.
- Postgres has `seasons`, `players`, `fixtures`, `results`, and `admin_audit_log`, but no `leagues` table.
- HTTP routes are global and do not carry league context.
- Frontend routes, API hooks, and React Query keys are all global.
- Bootstrapping and notifications assume one shared active season.

## Main Blockers

### Backend and domain

- `backend/internal/league/service.go` defines the core store API around `EnsureActiveSeason()` and `GetActiveSeason()`.
- Registration, season summary, season start, fixtures, results, standings, and audit flows all derive state from the single active season.
- `backend/internal/league/memory_store.go` stores one `activeSeason` in memory.
- `backend/cmd/api/main.go` bootstraps a single `"MVP Season"` automatically.

### Database

- `backend/internal/store/postgres/schema.sql` has no league-level entity.
- `backend/internal/store/postgres/store.go` loads the active season with `ORDER BY id DESC LIMIT 1`, which becomes incorrect as soon as multiple concurrent leagues exist.
- Existing foreign keys do not fully enforce that fixtures and results only reference players from the same season.

### API and frontend

- `backend/internal/httpapi/season.go`, `backend/internal/httpapi/registration.go`, and `backend/internal/httpapi/results.go` expose global endpoints such as `/api/season`, `/api/fixtures`, and `/api/standings`.
- `frontend/src/App.tsx` assumes one public league context and one admin control surface.
- `frontend/src/lib/api.ts` uses global endpoints and static React Query keys like `['season']`, `['fixtures']`, and `['standings']`.

### Notifications

- `backend/internal/notifications/service.go` resolves weekly data from the single active season.
- Slack delivery logic will need an explicit league-scoping strategy.

## Target Architecture

### Core model

- Add a first-class `leagues` entity.
- Make `seasons` belong to a league.
- Keep season-specific rules, but scope them per league.
- Allow multiple leagues to run concurrently.
- Enforce at most one active season per league.

### Routing and API shape

Prefer stable public slugs for league addressing.

Examples:

- Public routes:
  - `/leagues/:leagueSlug`
  - `/leagues/:leagueSlug/standings`
  - `/leagues/:leagueSlug/register`
- API routes:
  - `/api/leagues/:leagueSlug/season`
  - `/api/leagues/:leagueSlug/fixtures`
  - `/api/leagues/:leagueSlug/standings`
  - `/api/admin/leagues/:leagueSlug/players`
  - `/api/admin/leagues/:leagueSlug/season`

## Recommended Refactor Phases

### Phase 1 - Lock product and domain decisions

- Confirm that leagues should be addressed by stable URL slug.
- Decide whether admin auth remains deployment-wide or becomes league-scoped.
- Decide whether league creation is admin-only and whether it is exposed in the MVP UI.
- Preserve the existing rule of a single active season, but redefine it as a per-league rule.

### Phase 2 - Introduce league data model

- Create a `leagues` table with fields such as:
  - `id`
  - `name`
  - `slug`
  - `created_at`
- Add `league_id` to `seasons`.
- Add indexes and constraints:
  - unique `slug`
  - unique active season per league, if represented by status
- Backfill existing data into a default league.
- Make `league_id` non-null after backfill.

### Phase 3 - Tighten relational integrity

- Review fixture and result foreign keys so cross-season or cross-league references are impossible.
- Consider composite constraints or validation that guarantees:
  - fixture players belong to the same season as the fixture
  - result winner belongs to the same season as the fixture result
- Add migration checks for any inconsistent existing data before enforcing stricter constraints.

### Phase 4 - Refactor store interfaces

- Replace singleton store methods with explicit scoped methods.
- Likely additions:
  - `GetLeagueBySlug(ctx, slug)`
  - `ListLeagues(ctx)`
  - `CreateLeague(ctx, league)`
  - `GetActiveSeasonByLeague(ctx, leagueID)`
  - `GetSeason(ctx, seasonID)`
- Remove or deprecate `EnsureActiveSeason()` and `GetActiveSeason()`.
- Update the in-memory store to support multiple leagues and seasons.

### Phase 5 - Refactor domain services

- Change service entrypoints so they require explicit league scope rather than discovering a global season.
- Refactor registration, season, fixture, and result services to resolve the active season for a given league.
- Keep existing business rules intact:
  - registration open until season start
  - delete players only before season start
  - standings and result validation remain season-specific

### Phase 6 - Redesign HTTP handlers

- Update handlers to read league slug from the route.
- Convert global endpoints into league-scoped endpoints.
- Add league discovery endpoints for frontend navigation and admin selection.
- If rollout risk needs to be reduced, temporarily maintain compatibility aliases that map old global routes to a default league.

### Phase 7 - Refactor frontend routing and API client

- Add league-scoped routes in `frontend/src/App.tsx`.
- Introduce a league landing page or selector if more than one league is visible to public users.
- Update hooks in `frontend/src/lib/api.ts` so requests and React Query keys include `leagueSlug`.
- Update page copy that currently refers to the global "active season".
- Scope admin workflows to a selected league.

### Phase 8 - Notifications and background jobs

- Refactor weekly notifications to either:
  - iterate all eligible leagues, or
  - target a configured league explicitly
- Decide whether Slack channels are shared across all leagues or configured per league.
- Ensure scheduled jobs cannot accidentally post the wrong league's fixtures or standings.

### Phase 9 - Testing and rollout safety

- Rewrite backend tests that assume singleton season access.
- Add new coverage for:
  - two leagues with concurrent active seasons
  - registration uniqueness within league season scope
  - no cross-league leakage in public endpoints
  - no cross-league leakage in admin endpoints
  - notification jobs selecting the correct league
- Update frontend tests for scoped routes and request paths.
- Add migration tests or validation scripts for production-like backfill scenarios.

## Migration Strategy

Use an incremental rollout instead of a single large rewrite.

1. Add `leagues` and `league_id` to the schema.
2. Create one default league and backfill all existing seasons into it.
3. Refactor store and service layers to accept league scope.
4. Ship league-scoped API endpoints.
5. Update the frontend to use league-scoped routes and query keys.
6. Remove singleton compatibility paths once the new flow is stable.

## Risks

- Broad surface-area changes across backend services and handlers.
- Existing tests are tightly coupled to singleton season behavior.
- Query cache collisions will happen if frontend keys are not scoped carefully.
- Notifications could be misrouted during transition if league selection is implicit.
- Data migration ordering matters, especially before adding stricter constraints.

## Recommended First Slice

The safest first implementation slice is:

1. Add the `leagues` table and `seasons.league_id` migration.
2. Backfill a default league.
3. Introduce league-aware store methods while keeping the old ones temporarily.
4. Add league discovery and league-scoped read-only public endpoints.
5. Move the frontend public pages to slug-based routes.

This gives the project a clean foundation before touching admin mutations, notifications, and compatibility cleanup.

## Definition of Done

This refactor is complete when:

- Multiple leagues can exist concurrently.
- Each league has its own active season and independent public/admin views.
- Registration, fixtures, standings, results, and audit history are scoped correctly.
- Notifications target the intended league.
- Old singletons are removed from the core service/store design.
- Tests cover at least one true multi-league concurrency scenario end to end.
