# Darts League

`darts-league` is a full-stack darts league app for running a single active season with public registration, public fixtures/standings, and a restricted admin workflow for season control and score entry.

## Stack

- Backend: Go
- Database: Postgres
- Frontend: React + TypeScript
- Testing: Go tests, Vitest, Playwright
- Timezone logic: `Europe/London`

## Current MVP Features

- Public player registration before the season starts
- Optional player nickname, with nickname-first public display
- Single round-robin season generation
- Public standings table with `P`, `W`, `L`, `LF`, `LA`, `LD`, `Pts`
- Public week unlocking logic
- Locked future weeks with placeholder/funny hidden fixture names
- Admin-only login at `/admin`
- Admin player deletion before season start
- Admin result entry, editing, undo, and audit history
- Fixed match format: `501`, first to `3` legs

## Repository Layout

- `backend/` - Go API and Postgres store
- `frontend/` - React app
- `docker-compose.yml` - local container stack
- `Makefile` - common local commands
- `AGENTS.md` - project/product rules for coding agents
- `TODO.md` - implementation tracking notes

## Running Locally

There are two supported ways to run the project locally.

### Option 1: Full Docker stack

Recommended if you want the app, backend, and Postgres all running in containers.

Start everything:

```bash
make up
```

Open the app:

```text
http://localhost:4173
```

Useful container commands:

```bash
make logs
make logs-backend
make logs-frontend
make logs-db
make down
```

Notes:

- The backend is exposed on `http://localhost:8080`
- Postgres is exposed on `localhost:5432`
- `docker-compose.yml` currently sets `APP_NOW` so the app simulates a later unlocked week for demo purposes

### Option 2: Mixed host development

Use Docker only for Postgres, but run backend/frontend directly on your machine.

Start Postgres:

```bash
make db-up
```

Run the backend against Postgres:

```bash
make dev-backend-db
```

Run the frontend:

```bash
make dev-frontend
```

Then open:

```text
http://localhost:5173
```

If you want backend-only development without Postgres, you can also run:

```bash
make dev-backend
```

That path falls back to the in-memory store if Postgres is unavailable.

## Admin Access

There is no visible admin navigation in the public UI.

Go directly to:

```text
/admin
```

Default local credentials from `docker-compose.yml`:

- Username: `admin`
- Password: `change-me`

## Testing

Run backend and frontend unit tests:

```bash
make test
```

Run end-to-end tests:

```bash
make test-e2e
```

Or directly from the frontend directory:

```bash
npm run test:e2e
```

## Container Publishing

GitHub Actions publishes Docker images for the backend and frontend to GitHub Container Registry (`ghcr.io`).

- Pull requests opened from branches in this repository publish preview images tagged as `pr-<number>-<shortsha>` and `pr-<number>`
- Published GitHub releases publish clean versioned images tagged with the release tag
- Stable releases also publish a `latest` tag; prereleases do not

Image names:

- `ghcr.io/<owner>/darts-league-backend`
- `ghcr.io/<owner>/darts-league-frontend`

The workflow file lives at `.github/workflows/container-images.yml`.

## Product Rules

Some important locked rules in the current MVP:

- Single active season only
- Registration stays open until admin explicitly starts the season
- Admins can delete players only before season start
- Match scoring is win `2`, loss `0`
- Match format is fixed to `501`, first to `3` legs
- Display names must be unique per season, case-insensitive
- Public views prefer nickname when available
- Admins can edit and undo results, and all changes are audited

## Next Phase

Planned later work includes Slack webhook notifications for league updates.
