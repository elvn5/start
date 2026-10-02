# Project

Go microservices (`backend/`) + Nuxt 4 (`apps/web`). Full design rationale: docs/architecture.md.

- Public contract: `api/openapi.yaml` (Nuxt ⇄ gateway, REST)
- Internal contracts: `backend/proto/` (service ⇄ service, Connect-RPC; later NATS events)
- Services: `gateway` (public, no DB, :8080), `identity` (users, Postgres DB `identity`, :8081)
- Web: Nuxt dev server :3000, proxies `/api/**` → gateway

## Commands
- `task setup`        — first-time setup (.env, deps, codegen)
- `task dev`          — infra in Docker + all services and web natively with hot reload
- `task dev:identity` / `task dev:gateway` / `task dev:web` — run one part
- `task dev:docker`   — everything in containers
- `task gen`          — regenerate after changing *.proto, openapi.yaml or *.sql
- `task check`        — gen + lint + typecheck + test. **Run before declaring work done.**
- `task test:short`   — fast tests without Docker
- `task new:migration SVC=identity NAME=add_roles`

Grafana (traces/logs): http://localhost:3001 · Postgres on host port 5442 (user/pass `app`).

## Workflow: new public endpoint
1. `api/openapi.yaml` → `task gen`
2. If a service needs a new RPC: `backend/proto/<svc>/v1/*.proto` → `task gen`
3. Implement RPC in `services/<svc>/internal/rpc`, logic in `internal/service`, data in `internal/repository`
4. Gateway: handler in `services/gateway/internal/<feature>/`, embed it into `httpapi.Server`
5. Web: composable in `apps/web/app/composables/<feature>/`, use it from pages/components
6. `task check`

## Rules
- NEVER edit generated code: `backend/gen/**`, `backend/services/*/internal/gen/**`, `apps/web/.nuxt/**`
- Services talk ONLY via proto contracts. Never import another service's `internal/` or touch its DB
- `backend/pkg/` is infrastructure only — no domain logic
- Proto changes must be backward compatible (add fields; never renumber/remove). Breaking → new `v2` package
- Go tools run via `go tool -modfile=backend/tools/go.mod <tool>` (or just use `task`) — do not install globally
- See `backend/CLAUDE.md` and `apps/web/CLAUDE.md` for language-specific rules
