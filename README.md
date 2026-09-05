# Tuma254 API

Backend foundation for Customer, Rider, Business, Admin, and Public Web applications.

## Stack
- Go
- PostgreSQL + PostGIS
- Redis
- Standard library HTTP server
- Structured logging with log/slog
- Custom migration runner
- Bruno collections for API testing

## Start

Copy `.env.example` to your local environment and set:

```text
DATABASE_URL=postgres://...
REDIS_URL=redis://...
```

Run migrations:

```bash
go run ./cmd/migrate up
```

Run API:

```bash
go run ./cmd/api
```

Health:

```
GET /health
```

## Safety
Production credentials and Bruno production environments are ignored by Git. Commit only safe examples.
