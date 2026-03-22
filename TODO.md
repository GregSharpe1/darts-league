# Darts League Implementation TODO

This file tracks the agreed implementation plan and current progress for the `darts-league` application.

## Status Key

- [ ] Not started
- [~] In progress
- [x] Completed

## Product Decisions Locked In

- [x] Backend uses Go with Postgres
- [x] Frontend uses React with modern React standards
- [x] Development follows a TDD-first approach
- [x] Single active season for MVP
- [x] Public registration stays open until admin starts the season
- [x] Admin can delete registered players before season start
- [x] League format is single round-robin
- [x] Match format is `501`, first to `3` legs
- [x] Scoring is win `2`, loss `0`
- [x] Public UI prefers nickname when available, otherwise display name
- [x] Display name must be unique per season, case-insensitive
- [x] Public users see future weeks as locked, but pairings remain visible
- [x] Admin sees the full season schedule immediately
- [x] Admin can edit results and all edits are audited
- [x] Timezone is `Europe/London`
- [x] Standings include `P`, `W`, `L`, `LF`, `LA`, `LD`, `Pts`
- [x] Slack webhook posting weekly fixtures is a later-phase feature

## Phase 1 - Project Scaffold

- [x] Create repository structure (`backend/`, `frontend/`)
- [x] Initialize Go module and backend app skeleton
- [x] Initialize React + TypeScript frontend app
- [x] Add base config management for app/env settings
- [x] Add Postgres migration setup
- [x] Add Postgres-backed store implementation and backend wiring
- [x] Add backend test harness and first test command
- [x] Add frontend test harness and first test command
- [x] Add shared developer scripts/commands where useful

## Phase 2 - Registration and Season State

- [x] Model `seasons` table and season state transitions
- [x] Model `players` table with nickname support
- [x] Enforce case-insensitive unique display names per season
- [x] Implement public player registration endpoint
- [x] Implement registration validation tests
- [x] Implement admin player list endpoint/view
- [x] Implement pre-season player deletion endpoint
- [x] Implement tests preventing player deletion after season start

## Phase 3 - Fixtures and Weekly Visibility

- [x] Implement round-robin fixture generation logic
- [x] Implement week assignment logic for fixtures
- [x] Model `fixtures` table with `501` and `legs_to_win=3`
- [x] Implement admin season start action
- [x] Freeze registration once season starts
- [x] Implement public fixtures endpoint with locked future-week response shape
- [x] Implement current week resolver using `Europe/London`
- [x] Implement tests for Monday `09:00` unlock behavior
- [x] Implement DST edge-case tests for `Europe/London`

## Phase 4 - Results and Standings

- [x] Model `results` table
- [x] Validate only legal result scorelines (`3-0`, `3-1`, `3-2`)
- [x] Implement standings calculation logic
- [x] Apply standings ordering: points, leg difference, legs for, alphabetical
- [x] Implement public standings endpoint
- [x] Add domain tests for standings and result validation

## Phase 5 - Admin Auth and Audit

- [x] Implement env-based admin authentication
- [x] Protect `/admin` backend routes with secure session auth
- [x] Implement admin login/logout endpoints
- [x] Implement result entry endpoint
- [x] Implement result edit endpoint
- [x] Model `admin_audit_log` table
- [x] Record audit entries for every result change
- [x] Add integration tests for admin auth and result auditing

## Phase 6 - Frontend Public Experience

- [x] Build registration page
- [x] Build public fixtures page
- [x] Build standings page
- [x] Build nickname-first player presentation
- [x] Build locked future-week cards with visible pairings
- [x] Add countdown/reveal treatment for Monday unlock
- [x] Apply `autodarts.io`-inspired visual system without copying branding
- [x] Ensure responsive behavior on desktop and mobile

## Phase 7 - Frontend Admin Experience

- [x] Build `/admin` login flow
- [x] Build pre-season player management UI
- [x] Build season start UI
- [x] Build full-season fixture management UI for admins
- [x] Build result entry/editing UI
- [x] Build audit log view
- [x] Add frontend tests for admin gating and result flows

## Phase 8 - End-to-End Validation and Polish

- [x] Add end-to-end smoke test for core season flow
- [x] Verify public locked-week behavior does not leak unintended details
- [x] Verify standings update correctly after score entry/edit
- [x] Verify admin restrictions before/after season start
- [x] Polish loading, error, and empty states

## Phase 9 - Later Feature: Slack Webhook

- [x] Add scheduled Monday `09:00 Europe/London` Slack app job
- [x] Post weekly fixtures to Slack using the public Slack app channel
- [x] Post to a separate admin Slack channel when someone signs up, including their name and the time signed up
- [x] Post on Friday morning to the main Slack channel with a summary of the week's games and full standings
- [x] Add retry/error logging for failed Slack delivery
- [ ] Optionally add admin-triggered manual resend

## Immediate Next Steps

- [x] Set up project structure and toolchain
- [x] Write the first backend domain tests for season and player rules
- [x] Scaffold the frontend app and baseline routes

## Local Dev Notes

- [x] Docker Compose Postgres is available via `docker-compose.yml`
- [x] Start the database with `make db-up`
- [x] Run the backend against Postgres with `make dev-backend-db`
- [x] Run the frontend with `make dev-frontend`
- [x] Run the full stack in containers with `make up`
- [x] Open the app at `http://localhost:4173`
- [x] Follow container logs with `make logs` or service-specific log targets
