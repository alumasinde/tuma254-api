# Tuma254 API

Tuma254 is being rebuilt on PostgreSQL from the proven behavioural contract of the `tuma254-v1` branch.

## Current rebuild principles

- PostgreSQL is the only application database.
- UUIDs are used for application identifiers.
- Feature-first modules own models, DTOs, repositories, services, handlers and routes.
- Database constraints protect invariants that must not depend only on application code.
- Services own business rules and transaction orchestration.
- Repositories own SQL and locking.
- Secrets are environment-only and are never committed.

## Local development

### One-command development engine

Copy `.env.example` to `.env` once, then start the complete local stack with:

```
go run ./cmd/dev
```

The development engine automatically:

1. starts PostgreSQL through Docker Compose;
2. waits for PostgreSQL readiness;
3. runs pending migrations;
4. starts the protected local SMS sink;
5. configures the API's local SMS webhook integration;
6. generates `api-tests/environments/local.bru` for Bruno;
7. starts the API.

You no longer need separate PowerShell windows for PostgreSQL, migrations, SMS sink, and API.

Press `Ctrl+C` to stop the API and local SMS sink. PostgreSQL remains running so database data is preserved.

### Manual development

The individual commands remain available for debugging and production-like troubleshooting.

Health: `GET /health`  
Readiness: `GET /ready`
