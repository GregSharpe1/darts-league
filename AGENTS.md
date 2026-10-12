# AGENTS.md

This file captures the working context, product rules, and implementation guidance for agents contributing to `darts-league`.

## Project Summary

`darts-league` is a full-stack application for running a darts league with public registration, weekly fixture visibility, public standings, and a restricted admin workflow for season control and score entry.

## Core Stack

- Backend: Go
- Database: Postgres
- Frontend: React + TypeScript
- Testing approach: TDD-first
- Timezone: `Europe/London`

## MVP Requirements

- Multiple users can enter the league before the season starts
- Public frontend shows current league games left to play
- Public frontend shows current league points/standings
- Admin login is available at `/admin`
- Only admins can enter and edit scores

## Locked Product Decisions

- Use a single active season for MVP
- Registration stays open until an admin explicitly starts the season
- Admins can delete players before the season starts
- League format is single round-robin
- Match format is configured per season (`game_variant` `301`/`501`, `legs_to_win`), editable only until week one is released; default `501`, first to `3`
- Autodarts imports support only `501`, first to `2` or `3`, double out
- Scoring is win `2`, loss `0`
- Future public weeks are visible but locked
- Locked weeks still show pairings, but details should be visually obscured
- Admins see the full season immediately
- All result edits must be audited
- Display names must be unique within a season, case-insensitive
- Players can optionally provide a darts nickname
- Public views prefer nickname when available; otherwise show display name
- Standings columns should include `P`, `W`, `L`, `LF`, `LA`, `LD`, `Pts`
- Standings order is points, leg difference, legs for, alphabetical

## Data Model Guidance

Expected MVP tables:

- `seasons`
- `players`
- `fixtures`
- `results`
- `admin_audit_log`

Suggested player fields:

- `display_name` (required)
- `display_name_normalized` (required for uniqueness)
- `nickname` (optional)
- `registered_at`

Suggested fixture fields:

- `week_number`
- `scheduled_at`
- `game_variant` (copied from the season when fixtures are generated)
- `legs_to_win` (copied from the season when fixtures are generated)

## API Guidance

Public endpoints should cover:

- player registration
- season summary
- current/public fixtures
- standings

Admin endpoints should cover:

- login/logout
- player management before season start
- season start
- result entry/editing
- audit history

Important: public fixture endpoints must not leak future-week details beyond the intended locked-state payload.

## UX Guidance

- Visual direction should be heavily inspired by `autodarts.io`, but not copied directly
- Use a dark premium sports aesthetic with red accents and bold typography
- Avoid generic dashboard styling
- Future locked weeks should feel intentionally gated, not simply disabled
- Admin screens should prioritize clarity and control over visual drama
- Public player labels should prefer nickname; admin screens may show both nickname and display name

## Test Expectations

Write tests before or alongside implementation for:

- registration validation
- case-insensitive unique player names
- nickname fallback behavior
- round-robin fixture generation
- weekly unlock at Monday `09:00 Europe/London`
- DST edge cases
- result validation against the fixture's `legs_to_win` (winner exactly `legs_to_win`, loser fewer)
- standings calculations and ordering
- admin auth protection
- pre-season-only player deletion
- audit logging for result edits

## Delivery Notes

- Prefer incremental, vertical slices over large speculative builds
- Keep the MVP focused; Slack notifications are a later phase
- Make server-side time logic authoritative
- Treat season start as effectively irreversible for MVP once fixtures are generated
- Keep file edits ASCII unless a file already requires otherwise

## Autodarts Score Capture

- `score-scrape/scoreScrape.js` is loaded by the bookmarklet in `scripts/score-scrape.txt` from `main`; merging to `main` changes the live scraper immediately
- The `legs_to_win` rule is enforced in four places that must stay in sync: the scraper, `backend/internal/autodarts/parse.go`, `backend/internal/league/approval.go`, and `frontend/src/lib/pendingReview.ts`
- Approval validates against the fixture's own `legs_to_win`; never hardcode a leg count
- Scraper settings persist in the `autodarts_score_scrape_settings` cookie; users must clear it after the league format changes

## Repository Workflows

- Reusable repo-local workflow docs can live under `.agents/skills/`
- Use `.agents/skills/feature-pr-workflow/SKILL.md` when a change should go through the full inspect -> implement -> test -> branch -> commit -> push -> PR flow
- Use `.agents/skills/ui-pr-workflow/SKILL.md` for user-visible frontend changes that must include full frontend verification, tracked screenshots, and PR evidence
