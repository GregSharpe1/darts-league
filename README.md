# Darts League

`darts-league` is a full-stack darts league app for running a single active season with admin-managed divisions, public fixtures/standings, and a restricted admin workflow for season control and score entry.

## Stack

- Backend: Go
- Database: Postgres
- Frontend: React + TypeScript
- Testing: Go tests, Vitest, Playwright
- Timezone logic: `Europe/London`

## Current MVP Features

- Public player registration into a shared waitlist before the season starts
- Optional player nickname, with nickname-first public display
- Multi-division season management inside one active season
- Single round-robin fixture generation per division
- Public standings table with `P`, `W`, `L`, `LF`, `LA`, `LD`, `Pts`
- Public week unlocking logic
- Locked future weeks with placeholder/funny hidden fixture names
- Admin-only login at `/admin`
- Admin player deletion before season start
- Admin player assignment and reassignment across divisions until week one is released
- Admin result entry, editing, undo, and audit history
- Admin league closure once every division's fixtures have results, with read-only final boards
- New-season registration without deleting historical players, divisions, results, or audits
- Fixed match format: `501`, first to `3` legs
- Frontend and backend version reporting sourced from container image builds
- Slack app notifications for admin signups and weekly league updates

## Repository Layout

- `backend/` - Go API and Postgres store
- `frontend/` - React app
- `.agents/skills/` - repo-local agent workflow and skill docs
- `docs/` - operational guides, including [Slack notifications](docs/notifications.md)
- `docker-compose.yml` - local container stack
- `Makefile` - common local commands
- `AGENTS.md` - project/product rules for coding agents
- `TODO.md` - implementation tracking notes

## Agent Workflow

This repo includes a reusable workflow doc at `.agents/skills/feature-pr-workflow/SKILL.md` for feature work that should follow the full cycle of inspection, implementation, verification, branch creation, commit, push, and PR creation.

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
- Set `INSTANCE_NAME` to label the deployment in the UI, for example `Cardiff Office - Darts League`
- `docker-compose.yml` currently sets `APP_NOW` so the app simulates a later unlocked week for demo purposes
- The footer shows `Frontend local | Backend local` when you run the full Docker stack

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

In mixed host development, both services default their version label to `dev` unless you override `APP_VERSION` before starting them.

## Admin Access

There is no visible admin navigation in the public UI.

Go directly to:

```text
/admin
```

Default local credentials from `docker-compose.yml`:

- Username: `admin`
- Password: `change-me`

Admin workflow notes:

- Registration is global; players do not pick a division themselves
- Admins create divisions, assign players, and then start the season
- After season start and before the first Monday `09:00 Europe/London` release, admins can still edit league settings, rename divisions, change division count, move players between divisions, and update Slack channel IDs
- Recreating divisions in that pre-release window resets player assignments back to the waitlist and regenerates fixtures
- After the first weekly release, central season setup locks and only division scoring remains editable

### Closing a league and opening the next season

1. Record every scheduled match result in every division. The admin page shows the remaining match count and enables **Close league** once all results are recorded.
2. Confirm **Close league**. All divisions and scores become read-only; closure cannot be undone. Final standings and revealed match results stay on the public division boards, and admins can still read scores and audit history. Weekly Slack posts stop. Existing future-week reveal rules still apply.
3. When ready, enter the **Next league name** and confirm **Open next season registration**. Public boards switch immediately to the new season, before its fixtures start. Its roster and divisions start empty; returning players can register again. Configure divisions and assignments, then use the existing **Start season** action.

Historical data stays stored after the switch; there is no archive browser or automatic player carry-over. Restarting the backend does not replace a completed season. Persistent retention requires Postgres; the development in-memory fallback does not survive process restarts.

## Instance Naming

Use `INSTANCE_NAME` to label a deployment in the UI and browser title.

- Example: `Cardiff Office - Darts League`
- If unset, the app falls back to `Darts League`

## Version Reporting

The public footer shows both the frontend and backend versions.

- The frontend version is baked into the frontend container image at build time
- The backend version is baked into the backend container image at build time
- The backend also exposes its version at `GET /api/version`

Example response:

```json
{"version":"v0.0.6"}
```

Both services use `APP_VERSION` during image builds. Local Docker Compose sets this to `local`, while mixed host development falls back to `dev` unless you override it.

## Slack Integration

The backend supports an optional Slack app integration.

See the [notifications guide](docs/notifications.md) for setup, channel routing,
message examples, safe testing, troubleshooting, and known limitations.

Quick setup:

1. Go to https://api.slack.com/apps and create a new app for your workspace.
2. Under `OAuth & Permissions`, add the `chat:write` bot scope.
3. Install the app to your workspace and copy the bot token (`xoxb-...`).
4. Invite the app to both Slack channels you want to use.
5. Copy the channel IDs for the public and admin channels from Slack.

If Slack returns `channel_not_found`, the usual causes are:

- the value is a channel name instead of a channel ID
- the app has not been invited to the channel yet
- the admin channel is private and the bot is not a member
- for public channels, the app may also need `chat:write.public` if it is not invited

Set these environment variables to enable it:

- `SLACK_BOT_TOKEN`
- `SLACK_PUBLIC_CHANNEL_ID`
- `SLACK_ADMIN_CHANNEL_ID`

Set `PUBLIC_BASE_URL` to the public frontend URL (for example,
`https://darts.example.com`) to include "View the standings here." in both weekly
messages, with only "here" linked to that division's standings page. The link is
omitted when this variable is unset. Weekly headers include the division name.

Behavior:

- successful player registrations post to the admin Slack channel
- With Helm notifications enabled, Monday `09:00 Europe/London` posts this week's fixtures per division
- Friday `09:00 Europe/London` posts this week's results plus cumulative standings per division
- Weekly messages use each division's configured channel, falling back to `SLACK_PUBLIC_CHANNEL_ID`
- Weekly scheduling is disabled by default; enable `backend.notifications.enabled` in Helm (the API server and Docker Compose do not schedule these commands)

For manual delivery, run from `backend/` with the database and Slack environment
configured. **These commands send real messages; there is no dry-run mode and
rerunning them can produce duplicates.** Use test channels for testing.

```bash
go run ./cmd/api notify weekly-fixtures
go run ./cmd/api notify weekly-summary
```

See the [testing and manual delivery section](docs/notifications.md#testing-and-manual-delivery)
for automated tests and Docker Compose configuration.

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

- Pull requests opened from branches in this repository publish preview images tagged as `pr-<number>-<shortsha>` and `pr-<number>` only for components changed under `backend/` or `frontend/`
- Published GitHub releases publish clean versioned images tagged with the release tag
- Stable releases also publish a `latest` tag; prereleases do not
- The precise image version shown in the UI and returned by `GET /api/version` is baked in during the image build and matches the release tag or `pr-<number>-<shortsha>` preview tag

Image names:

- `ghcr.io/<owner>/darts-league-backend`
- `ghcr.io/<owner>/darts-league-frontend`

The workflow file lives at `.github/workflows/container-images.yml`.

## Product Rules

Some important locked rules in the current MVP:

- Single active season only
- Registration stays open until admin explicitly starts the season
- Admins can delete players only before season start
- One season can contain multiple admin-managed divisions
- Match scoring is win `2`, loss `0`
- Match format is fixed to `501`, first to `3` legs
- Display names are unique within each season, case-insensitive; returning players may reuse their names in a new season
- Public views prefer nickname when available
- Admins can edit and undo results, and all changes are audited

## Next Phase

Planned later work includes Slack delivery idempotency and optional manual resend controls.
