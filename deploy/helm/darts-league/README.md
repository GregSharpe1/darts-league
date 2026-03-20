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

If enabling ingress, you must also provide hostnames:

```yaml
ingress:
  enabled: true
  hosts:
    - host: darts.example.com
      paths:
        - path: /
          pathType: Prefix
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

## Notes

- The frontend proxies `/api` traffic to the backend service through nginx.
- The backend uses `/healthz` for Kubernetes probes.
- The backend application can fall back to an in-memory store if database
  connectivity is missing; treat this as unsafe for production and validate DB
  configuration carefully.
