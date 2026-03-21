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
