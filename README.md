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

1. Copy `.env.example` to `.env`.
2. Start PostgreSQL with `docker compose up -d`.
3. Run `go run ./cmd/migrate up`.
4. Run `go run ./cmd/api`.

Health: `GET /health`  
Readiness: `GET /ready`
