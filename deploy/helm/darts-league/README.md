# darts-league Helm chart

This chart deploys the `darts-league` frontend, backend, and an optional
PostgreSQL database.

## Production posture

- `frontend` and `backend` are enabled by default.
- `postgres.enabled=false` by default.
- Production should prefer `externalDatabase.*` or `externalDatabase.existingSecret`.

## Required values before install

Set image coordinates for the application containers:

```yaml
frontend:
  image:
    repository: your-registry/darts-league-frontend
    tag: "2026-03-20"

backend:
  image:
    repository: your-registry/darts-league-backend
    tag: "2026-03-20"
```

The chart defaults to a standard Kubernetes `Ingress` resource. Override the
default host before production use:

```yaml
ingress:
  enabled: true
  hosts:
    - host: darts.example.com
      paths:
        - path: /
          pathType: Prefix
```

For Istio-based ingress, disable the standard `Ingress` and enable the Istio
resources instead:

```yaml
ingress:
  enabled: false

istio:
  enabled: true
  virtualService:
    hosts:
      - darts.k8s.sharpe.wales
```

## Database options

Recommended production option:

```yaml
externalDatabase:
  host: postgres.example.internal
  port: 5432
  name: darts_league
  user: darts_league
  password: replace-me
  sslmode: require
```

Or reference an existing secret containing a `DATABASE_URL` key:

```yaml
externalDatabase:
  existingSecret: darts-league-db
  existingSecretKey: DATABASE_URL
```

Development option:

```yaml
postgres:
  enabled: true
```

## Autodarts automatic approval

`backend.env.autodartsAutoSubmit` defaults to `false` and renders the backend
environment variable `AUTODARTS_AUTO_SUBMIT` as `"false"`. No environment override
enables it by default. To opt in, set `backend.env.autodartsAutoSubmit=true` in
your deployment values and deploy a backend image supporting this setting.
It is deployment configuration, not an admin UI toggle.

Before imports arrive, open `/admin`, use **Autodarts mappings** to select each
league player, enter their exact Autodarts name, and save. Edit or remove mappings
there at any time, including after roster lock or current-season completion;
completed-season scoring remains closed. Mappings apply only to the current
season and are not carried into the next. Source names are assumed unique and
matched case-sensitively after trimming surrounding whitespace, preserving
internal whitespace and punctuation; no account lookup or nickname fallback occurs.

Only new imports at first successful ingestion can be automatically approved,
and both names must already map to distinct players in the same current-season
division. The season must be started and writable, with exactly one compatible,
unscored fixture. Supported source-reported format (`501`, first to `2` or `3`,
double out), a valid matching score, and a source `played_at` between season
start and server time inclusive are required. Unchanged source data and no review
conflict are required; manual format attestation or missing-date explanations
cannot qualify. Approval records audit actor `autodarts:auto-submit` and never
automatically replaces a result. Unmapped, ambiguous, or otherwise ineligible
imports remain pending for manual admin review. Public release rules still apply.

Previously pending results stay manual after mapping edits, flag changes,
duplicate deliveries, refetches, or restarts; mappings only improve their prefill.
Manual result entry and editing remain admin-only. To disable, set
`backend.env.autodartsAutoSubmit=false` and redeploy the backend so its environment
is refreshed. Disabling affects future ingestions only and does not undo scores
or audits. See the [admin operating guide](../../../README.md#autodarts-player-mappings-and-automatic-approval).

## Slack notifications

See the [notifications guide](../../../docs/notifications.md) for message examples,
division channel routing, manual delivery precautions, and troubleshooting.
Weekly CronJobs are disabled by default; enabling them does not configure Slack
channel membership or division channel IDs.

To enable Slack app delivery, provide the public and admin channel IDs through
backend environment values and the bot token through the Slack secret values:

```yaml
backend:
  env:
    slackPublicChannelId: C0123456789
    slackAdminChannelId: C0987654321
    publicBaseUrl: https://darts.example.com
  slack:
    botToken: xoxb-replace-me
  notifications:
    enabled: true
    timeZone: Europe/London
    weeklyFixturesSchedule: "0 9 * * 1"
    weeklySummarySchedule: "0 9 * * 5"
```

Or reference an existing secret containing the bot token key:

```yaml
backend:
  slack:
    existingSecret: darts-league-slack
    existingSecretKeys:
      botToken: SLACK_BOT_TOKEN
```

When enabled, the chart creates two backend CronJobs:

- Monday `09:00 Europe/London` for weekly fixtures, one message per eligible division
- Friday `09:00 Europe/London` for the weekly summary and cumulative standings, one message per eligible division

Messages use the division's channel ID when set, otherwise
`backend.env.slackPublicChannelId`. Set division channel IDs through `/admin`
before the first weekly release. Both jobs use the same backend ConfigMap and
database/Slack Secrets. Disabling the CronJobs does not disable signup messages.

Set `backend.env.publicBaseUrl` to your public frontend URL to include a link to
the division's standings in both messages. Only "here" in "View the standings
here." is clickable. If left empty, the link is omitted.

The bundled PostgreSQL chart stores its cluster data under a dedicated `PGDATA`
subdirectory so mounted volumes do not fail on `lost+found`, and it applies an
`fsGroup` compatible with the upstream `postgres` image.

## Notes

- The frontend proxies `/api` traffic to the backend service through nginx.
- The backend uses `/healthz` for Kubernetes probes.
- The backend application can fall back to an in-memory store if database
  connectivity is missing; treat this as unsafe for production and validate DB
  configuration carefully.
- The chart can expose the app either through a Kubernetes `Ingress` or through
  Istio `Gateway` and `VirtualService` resources.
- The Istio path is HTTP-only by default and routes traffic to the frontend
  service, which then proxies `/api` traffic to the backend.
- To enable TLS on the Istio Gateway, add an HTTPS server entry with a
  `credentialName` pointing to the Kubernetes Secret (in `istio-system`) that
  holds your certificate:

```yaml
istio:
  enabled: true
  gateway:
    servers:
      - port:
          number: 443
          name: https
          protocol: HTTPS
        tls:
          mode: SIMPLE
          credentialName: my-tls-secret
      - port:
          number: 80
          name: http-redirect
          protocol: HTTP
        tls:
          httpsRedirect: true
  virtualService:
    hosts:
      - darts.example.com
```

## Makefile workflow

From `deploy/`, you can manage the chart with the local Makefile:

```bash
make helm-lint
make helm-template
make helm-diff
make helm-deploy
```

You can override the default release, namespace, or values file:

```bash
make helm-deploy RELEASE_NAME=darts-league-dev NAMESPACE=darts-dev VALUES_FILE=helm/darts-league/values-dev.yaml
```
