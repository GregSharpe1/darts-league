# Slack notifications

The backend sends signup notifications and weekly division updates through
Slack's `chat.postMessage` API. There is no scheduler inside the API server:
signup messages run during registration, while weekly messages are separate CLI
commands scheduled by the [Helm chart](../deploy/helm/darts-league/README.md).

## What gets sent

| Message | Trigger | Destination | Content |
| --- | --- | --- | --- |
| New player signup | Successful player registration | Admin channel | Player label, registration time, total registered players |
| Weekly fixtures | Monday 09:00 by default | Each division's channel, or public fallback | Current public week's pairings and division standings link |
| Weekly summary | Friday 09:00 by default | Each division's channel, or public fallback | Current week's results, leader, pending results, cumulative standings and link |

Each weekly command loads the active season and composes one message per
division. A division's `SlackPublicChannelID` takes precedence over
`SLACK_PUBLIC_CHANNEL_ID`. If several divisions use the same channel, it receives
separate messages, each with its division name in the header.

Weekly content uses the latest unlocked week for that division, not the next
week. Completed seasons and divisions outside the active season are skipped,
as are divisions with no fixtures or no unlocked week. Missing channel
configuration or a missing bot token also prevents delivery. The summary still
posts when there are no results: it says `No results recorded yet` and lists
unplayed fixtures under `Awaiting result`. That section is omitted when all
current-week fixtures have results.

Fixture and result labels use `Nickname (Display Name)` where a nickname exists,
otherwise just the display name. The leader and standings use nickname only,
falling back to display name. Standings are cumulative, not limited to the week
shown in the results section; see the division-scoping limitation below.

## Setup

### 1. Create and install the Slack app

1. Create an app at <https://api.slack.com/apps> for the target workspace.
2. Add the bot scope `chat:write` under **OAuth & Permissions**.
3. Install the app to the workspace and securely store its bot token.
4. Invite the app to the admin channel and every division/fallback channel it
   should post in, including private channels.
5. Copy channel IDs, not channel names, from Slack.

Do not commit bot tokens to source control, values files, or documentation.
Use a secret manager or Kubernetes Secret. Membership in the destination
channels is recommended; posting to public channels without joining may require
the additional `chat:write.public` scope.

### 2. Configure the backend

| Environment variable | Purpose |
| --- | --- |
| `SLACK_BOT_TOKEN` | Bot token; unset or blank disables Slack delivery |
| `SLACK_ADMIN_CHANNEL_ID` | Signup destination; blank skips signup messages |
| `SLACK_PUBLIC_CHANNEL_ID` | Weekly fallback destination; optional when divisions have their own channels |
| `PUBLIC_BASE_URL` | Public frontend URL, e.g. `https://darts.example.com`; blank omits standings links |
| `DATABASE_URL` | Same league database used by the API; required for weekly commands to see the real season |
| `APP_TIMEZONE` | Signup timestamp timezone; defaults to `Europe/London` |
| `APP_NOW` | Optional simulated clock for testing; leave unset in production |

Set division channel IDs in `/admin` before the first weekly release locks season
setup. The weekly commands use the stored season timezone for week selection and
message dates; changing the scheduler timezone does not change the season.

`PUBLIC_BASE_URL` must be an absolute frontend URL including `https://` (or
`http://` for local testing), not the backend API address. Surrounding whitespace
and trailing slashes are removed. For a division with slug `premier`, the message
contains this Slack markup immediately below the date/season line:

```text
View the standings <https://darts.example.com/divisions/premier/standings|here>.
```

Slack displays "View the standings here." with only **here** clickable. The URL
uses the division slug, not its display name. The application does not validate
the configured base URL, so check it before enabling delivery.

### 3. Enable scheduled delivery in Helm

Weekly CronJobs are disabled by default. Configure the database and images as
described in the [chart guide](../deploy/helm/darts-league/README.md), then supply:

```yaml
backend:
  env:
    slackAdminChannelId: CADMIN123
    slackPublicChannelId: CPUBLIC123
    publicBaseUrl: https://darts.example.com
  slack:
    existingSecret: darts-league-slack
    existingSecretKeys:
      botToken: SLACK_BOT_TOKEN
  notifications:
    enabled: true
    timeZone: Europe/London
    weeklyFixturesSchedule: "0 9 * * 1"
    weeklySummarySchedule: "0 9 * * 5"
```

The referenced Secret must exist in the release namespace and contain the bot
token under `SLACK_BOT_TOKEN`. Both CronJobs inherit the backend ConfigMap
(including `PUBLIC_BASE_URL`) and receive the database URL and token from Secrets.
Kubernetes must support CronJob `spec.timeZone` (stable in Kubernetes 1.27).
`Europe/London` keeps the schedules at 09:00 local time across GMT/BST changes.

Disabling `backend.notifications.enabled` only removes the weekly CronJobs. It
does not disable signup notifications or prevent manual weekly commands.

## Examples

These examples use fictional players in a single-division season. For ASCII
readability, emoji are shown as Slack aliases and decorative separators as
hyphens. The application sends emoji characters. The link markup is shown as
sent; Slack renders only its `here` label. Table spacing is illustrative.

### Signup (admin channel)

```text
New player signup
- Player: The Sharpshooter (Greg Sharpe)
- Signed up: Wed 01 Jul 2026 12:30 BST
- Total registered: 4
```

### Monday fixtures (division channel)

```text
:dart: Week 1 Fixtures - Division 1
:calendar: Mon 06 Jul 2026 19:30 BST - Summer League
View the standings <https://darts.example.com/divisions/division-1/standings|here>.

:trophy: The Sharpshooter (Greg Sharpe) vs Alex Taylor
:trophy: Double Trouble (Sam Jones) vs Jamie Smith
```

The date/time in this message is the first fixture's scheduled time, not the
09:00 notification delivery time.

### Friday results and standings (division channel)

````text
:mega: Week 1 Results + Standings - Division 1
:calendar: Fri 10 Jul 2026 - Summer League
View the standings <https://darts.example.com/divisions/division-1/standings|here>.

:white_check_mark: Results
- The Sharpshooter (Greg Sharpe) 3-1 Alex Taylor

:crown: Leader: The Sharpshooter on 2 pts

:hourglass_flowing_sand: Awaiting result
- Double Trouble (Sam Jones) vs Jamie Smith

:bar_chart: Standings
```
#   Player            P  W  L  LF  LA  LD Pts
1   The Sharpshooter  1  1  0   3   1  +2   2
2   Double Trouble    0  0  0   0   0   0   0
3   Jamie Smith       0  0  0   0   0   0   0
4   Alex Taylor       1  0  1   1   3  -2   0
```
````

`P/W/L` are played/won/lost, `LF/LA/LD` are legs for/against/difference, and `Pts`
is points (two per win). Ordering is points, leg difference, legs for, then name.

## Testing and manual delivery

Run automated tests without contacting Slack, from `backend/`:

```sh
go test -race -shuffle=on -count=1 ./internal/notifications ./internal/slack
```

**The following commands send real messages when credentials are configured.
There is no CLI dry-run mode.** Use a test database and test Slack channels for a
delivery test, including test division channel IDs: changing only the fallback
channel does not override stored division channels. Repeating a command can
duplicate messages.

After securely supplying the environment variables, run from `backend/`:

```sh
go run ./cmd/api notify weekly-fixtures
go run ./cmd/api notify weekly-summary
```

Or, with the same configuration supplied to the backend container, from the
repository root:

```sh
docker compose exec backend /usr/local/bin/darts-league-api notify weekly-fixtures
docker compose exec backend /usr/local/bin/darts-league-api notify weekly-summary
```

The checked-in Docker Compose file explicitly sets Slack variables to empty and
does not pass `PUBLIC_BASE_URL`. Supply them through a local Compose override;
exporting host variables alone does not change that container configuration.
Compose does not schedule the weekly commands.

## Troubleshooting and current limitations

- **No message:** Check token, destination IDs, bot membership, database, fixtures
  and the current public week. The CLI can exit successfully with
  `notification command completed without sending a message` when delivery is
  disabled or there is no eligible content. Check the CronJob configuration too;
  its presence/enabled values do not prove a live deployment has run it.
- **`channel_not_found`:** Use a channel ID and invite the app to that channel.
  Check division overrides as well as the fallback channel.
- **No standings link:** Set `PUBLIC_BASE_URL` / `backend.env.publicBaseUrl` for
  the notification process, not just the frontend.
- **Wrong season data:** The CLI must use the real database. The backend currently
  falls back to an in-memory store when Postgres cannot be opened; inspect logs
  rather than treating a successful exit as proof of delivery.
- **Delivery failures:** Signup failures are logged without undoing registration.
  A weekly send error stops the command; earlier divisions may already have been
  posted. The Slack client retries transport/read failures, HTTP 429 and HTTP 5xx
  up to twice, immediately, without backoff or `Retry-After` handling.
- **Duplicate or stale messages:** There is no delivery ledger or deduplication.
  Manual reruns, job retries or ambiguous network failures can duplicate posts.
  Weekly selection remains on the last unlocked week after the final fixture
  week until an admin closes the season. CronJobs continue running after closure,
  but completed seasons do not produce messages.
- **Division standings bug:** Fixtures/results are division-scoped, but the
  summary currently builds standings from all season players. Other divisions'
  players can appear with zero statistics and can affect the displayed leader
  when tied. This is a known limitation, not intended division-only behavior.
- **Manual summary dates:** The summary date format contains a literal `Fri`.
  Running it on another weekday still produces a Friday label.

## Implementation references

- [Message composition and routing](../backend/internal/notifications/service.go)
- [Notification command and signup wiring](../backend/cmd/api/main.go)
- [Slack client and retries](../backend/internal/slack/client.go)
- [Backend configuration](../backend/internal/config/config.go)
- [CronJob templates](../deploy/helm/darts-league/templates/backend-notification-cronjobs.yaml)
- [Weekly link tests](../backend/internal/notifications/weekly_links_test.go)
