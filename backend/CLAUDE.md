# Backend (Go microservices)

One Go module (`app/backend`) for all services. Framework: **Gin** in every service.

## Layout
- `proto/`          — internal contracts (buf). Generated → `gen/proto/`
- `pkg/`            — shared infrastructure: `httpx` (Gin bootstrap, health, graceful shutdown), `rpcx` (Connect interceptors, mounting), `dbx` (pgx pool, goose, WithTx), `observability` (OTel + slog), `config`
- `services/<svc>/` — one service = `cmd/<svc>/main.go` + `internal/` + `queries/` + `migrations/`
- `tools/go.mod`    — pinned dev tools (buf, sqlc, oapi-codegen, goose, air)

## Service layers (dependencies point inward)
`rpc` (Connect handlers) / `consumer` → `service` (business logic) → `repository` (sqlc)
- protobuf types: only in `rpc` (and gateway handlers); sqlc types: only in `repository`; `model` in between
- `service` declares the small interfaces it needs (consumer-side interfaces) → unit-test with fakes
- domain errors in `model` (`ErrNotFound`, `ErrInvalid`…); `rpc` maps them to Connect codes; gateway maps Connect codes to HTTP
- manual DI in `internal/app/app.go`; no global state
- config from env with service prefix (`IDENTITY_`, `GATEWAY_`) via `pkg/config`

## Gateway
- Implements `api.StrictServerInterface` generated from `api/openapi.yaml`; never register routes by hand
- One package per feature (`internal/users`), embedded into `httpapi.Server`
- No business logic — call services, map proto ⇄ API types, map errors

## Conventions
- Wrap errors: `fmt.Errorf("doing x: %w", err)`; log with `slog` using the request context (adds trace_id)
- Every outgoing call has a deadline (`rpcx` adds a 5s default)
- Migrations: goose SQL files, embedded and applied on service start
- Tests: unit (`service`, fakes), integration (`repository`, testcontainers), gateway (`httptest` + fake Connect server)
