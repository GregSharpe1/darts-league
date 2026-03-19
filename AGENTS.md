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
- Match format is fixed to `501`, first to `3` legs
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
- `game_variant` = `501`
- `legs_to_win` = `3`

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
- fixed-format result validation (`3-0`, `3-1`, `3-2` only)
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
