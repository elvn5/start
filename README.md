# Go microservices + Nuxt — шаблон монорепозитория

Шаблон fullstack-приложения: **микросервисы на Go (Gin, Connect-RPC, Postgres)** и **frontend на Nuxt 4**, настроенный для разработки с **Claude Code**.

Обоснование всех решений — в [docs/architecture.md](docs/architecture.md).

## Создание проекта из шаблона

1. На GitHub: **Use this template → Create a new repository** (или `gh repo create myshop --template <owner>/<this-repo> --clone`).
2. Переименуйте проект — имя используется для Docker Compose, томов и образов:

   ```bash
   task init NAME=myshop MODULE=github.com/me/myshop/backend
   ```

   `MODULE` необязателен: путь Go-модуля по умолчанию `app/backend` — это валидно для приложения.
3. `task setup && task check`, затем закоммитьте результат.
4. Обновите этот README под свой проект и при необходимости добавьте `LICENSE`.

## Быстрый старт

Нужны Go 1.26+, Node 24, pnpm 9, Docker и [Task](https://taskfile.dev). Проще всего — `mise install` (версии в [mise.toml](mise.toml)). Остальные инструменты (buf, sqlc, oapi-codegen, goose, air) закреплены в `backend/tools/go.mod` и запускаются через `go tool`.

```bash
task setup    # .env, зависимости, кодогенерация
task dev      # Postgres/NATS/Grafana в Docker + сервисы и Nuxt с hot reload
task check    # кодогенерация, линтеры, типы, тесты — то же, что в CI
lefthook install   # git-хуки (gofmt, go vet, eslint)
```

| Что | Адрес |
|---|---|
| Nuxt | http://localhost:3000 |
| gateway (REST) | http://localhost:8080/api/users |
| identity (Connect) | http://localhost:8081 |
| Grafana (трейсы, логи) | http://localhost:3001 |
| Postgres | `localhost:5442`, пользователь и пароль `app` |

Все команды: `task --list`.

## Что внутри

```
api/openapi.yaml        публичный контракт (Nuxt ⇄ gateway, REST)
backend/
  proto/                внутренние контракты (сервис ⇄ сервис, Connect-RPC)
  pkg/                  общая инфраструктура: httpx, rpcx, dbx, observability, config
  services/gateway/     публичный BFF без БД (:8080)
  services/identity/    пользователи, своя БД Postgres (:8081)
  gen/                  сгенерированный код (не редактировать)
apps/web/               Nuxt 4 + Nuxt UI, API только через composables
deploy/                 Caddyfile, init.sql (база на каждый сервис)
docs/architecture.md    исследование и обоснование стека
```

Стек: Gin · Connect-RPC + buf · oapi-codegen · sqlc + pgx · goose · OpenTelemetry → Grafana LGTM · Nuxt 4 · nuxt-open-fetch · Pinia · Zod · Vitest.

## Как добавлять функциональность

- **Новый публичный эндпоинт:** `api/openapi.yaml` → `task gen` → RPC в сервисе → хендлер в gateway → composable в Nuxt. Подробно — в [CLAUDE.md](CLAUDE.md).
- **Новая миграция:** `task new:migration SVC=identity NAME=add_roles`.
- **Новый сервис:** скопируйте `backend/services/identity`, добавьте proto в `backend/proto/<svc>/v1`, базу в `deploy/postgres/init.sql`, сервис в `compose*.yaml`, `Taskfile.yml` (`dev:<svc>`, `SQLC_SERVICES`) и матрицу образов в `.github/workflows/ci.yml`.

Правила для агента и людей: [CLAUDE.md](CLAUDE.md), [backend/CLAUDE.md](backend/CLAUDE.md), [apps/web/CLAUDE.md](apps/web/CLAUDE.md).

## CI и деплой

- [.github/workflows/ci.yml](.github/workflows/ci.yml): `task check` + проверка, что сгенерированный код закоммичен; на `main` — сборка образов в `ghcr.io/<owner>/<repo>/<service>`.
- `buf breaking` сравнивает proto с веткой `main` и пропускается, пока её нет.
- Prod: `docker compose -f compose.prod.yaml up --build -d`, секреты в `.env.prod` (`POSTGRES_USER`, `POSTGRES_PASSWORD`, `IDENTITY_DATABASE_URL`, `DOMAIN`).
- Dependabot обновляет Go, npm, Actions и Docker-образы сгруппированными PR.
