# Chihuo Go Backend

Production-oriented foundation for the Chihuo demand-driven food pre-sale
platform. This directory is intentionally independent from the existing
TypeScript prototype.

## Stack

- Go 1.24+
- Standard library `net/http`
- HMAC-SHA256 JWT middleware
- Structured JSON logs with `log/slog`
- Request ID propagation through `X-Request-ID`
- Replaceable `store.Store` interface
- Complete in-memory store for local development and tests
- PostgreSQL migration and `database/sql` storage seam

The server does not contain a hard-coded JWT secret. `JWT_SECRET` is required
and must be supplied through the environment.

## Run Locally

```bash
cd backend-go
export JWT_SECRET="$(openssl rand -hex 32)"
export DEV_LOGIN_ENABLED=true
export CORS_ALLOWED_ORIGINS="http://localhost:5173,http://localhost:5174"
go run ./cmd/server
```

Default address: `http://localhost:4000`.

Health endpoints:

```bash
curl http://localhost:4000/health/live
curl http://localhost:4000/health/ready
```

Development login is deliberately isolated at:

```text
POST /v1/auth/dev/wechat-login
```

Set `DEV_LOGIN_ENABLED=false` in any non-development environment. The
`WeChatLoginProvider` interface is the replacement point for a real
`wx.login`/WeChat identity exchange; app secrets must remain outside this
repository and outside request payloads.

## API Contract

The shared OpenAPI contract is in [`api/openapi.yaml`](api/openapi.yaml).
All successful responses use:

```json
{
  "data": {}
}
```

Errors use:

```json
{
  "error": {
    "code": "INVALID_JSON",
    "message": "request body is invalid",
    "request_id": "..."
  }
}
```

Money fields are integer cents. Campaign platform fees use basis points:
`500` means 5%. Core write endpoints accept an optional `Idempotency-Key`
header. Reusing a key with the same request returns the original response;
reusing it with a different request returns `409 IDEMPOTENCY_KEY_REUSED`.

## Vertical Flow

```text
dev/wechat login
  -> consumer creates or joins a demand cluster
  -> admin reviews the demand
  -> approved merchant submits an offer
  -> merchant creates a pre-sale campaign
  -> admin reviews the campaign
  -> consumer creates an order
```

Demand aggregation is deterministic in the memory implementation:

- category, service area, serving date and serving time must match;
- budget and weight ranges must overlap;
- hard constraints must be the same set;
- preference differences remain per-member data;
- a full cluster is not selected for another member.

This makes the production boundary explicit. A later ranking or AI service can
propose candidate clusters, but it must not silently merge incompatible hard
constraints.

## PostgreSQL Persistence

The idempotent migration is at:

```text
internal/store/migrations/001_initial.sql
```

`store.PostgresStore` uses the pgx database/sql driver and implements the same
`store.Store` interface as `MemoryStore`. When `DATABASE_URL` is configured,
the server opens PostgreSQL, runs the idempotent migration, and uses the
PostgreSQL store. Without it, local development and tests use the in-memory
store.

Demand membership and order creation use transactions and row locks. The
migration includes an `idempotency_records` primary key on
`(actor_id, idempotency_key)` so replay protection is database-atomic.

## Test and Build

The sandbox may not allow Go to use the default user cache. The following
command uses a writable temporary cache:

```bash
cd backend-go
GOCACHE=/tmp/chihuo-go-build go test ./...
GOCACHE=/tmp/chihuo-go-build go test -race ./...
GOCACHE=/tmp/chihuo-go-build go build ./cmd/server
```

Integration tests use `httptest.NewRecorder` and exercise authentication,
CORS, request IDs, demand aggregation, admin review, merchant offer and
campaign creation, order totals, and idempotency replay.

## Configuration

| Variable | Default | Notes |
| --- | --- | --- |
| `HTTP_ADDR` | `:4000` | HTTP listen address |
| `JWT_SECRET` | required | At least 32 bytes |
| `JWT_ISSUER` | `chihuo-api` | JWT issuer |
| `JWT_TTL` | `168h` | Go duration |
| `DATABASE_URL` | empty | PostgreSQL connection string; empty uses memory mode |
| `CORS_ALLOWED_ORIGINS` | `*` | Use explicit origins in production |
| `DEV_LOGIN_ENABLED` | `false` | Development-only login switch |
