# Fullstack монорепозиторий: Go-микросервисы + Nuxt

Исследование и рекомендации по структуре монорепозитория для fullstack-приложения: **микросервисный backend на Go (Gin)** и **frontend на Nuxt**, со стеком, оптимизированным для разработки с **Claude Code**.

> Актуально на октябрь 2026: Go 1.26, Nuxt 4.4, Vue 3.5+, Tailwind CSS 4.

---

## Быстрый старт

Нужны Go 1.26+, Node 24, pnpm 9, Docker и [Task](https://taskfile.dev) (`go install github.com/go-task/task/v3/cmd/task@latest`). Остальные инструменты (buf, sqlc, oapi-codegen, goose, air) закреплены в `backend/tools/go.mod` и запускаются через `go tool`.

```bash
task setup   # .env, зависимости, кодогенерация
task dev     # Postgres/NATS/Grafana в Docker + identity, gateway, Nuxt с hot reload
task check   # кодогенерация, линтеры, типы, тесты
```

| Что | Адрес |
|---|---|
| Nuxt | http://localhost:3000 |
| gateway (REST) | http://localhost:8080/api/users |
| identity (Connect) | http://localhost:8081 |
| Grafana | http://localhost:3001 |
| Postgres | `localhost:5442`, пользователь и пароль `app` |

Сейчас реализованы сервисы **gateway** и **identity** (раздел 8, шаги 1–7). order и notification с NATS и outbox — следующий этап.

---

## Принятые решения

| # | Решение | Статус |
|---|---|---|
| 1 | Backend — фреймворк **Gin** (не голый `net/http`) | ✅ Согласовано |
| 2 | Docker для dev и prod | ✅ Согласовано |
| 3 | Frontend — стандартная структура Nuxt + фичи в подпапках, API только через composables | ✅ Согласовано |
| 4 | Backend — **микросервисная архитектура** (gateway + сервисы, database-per-service) | ✅ Согласовано |
| 5 | Протоколы: **REST (OpenAPI) снаружи** (Nuxt → gateway), **gRPC/Connect внутри** (сервис ↔ сервис), события — NATS | ✅ Согласовано |

---

## TL;DR — рекомендуемый стек

### Общий

| Слой | Выбор | Почему |
|---|---|---|
| Монорепо (JS) | **pnpm workspaces** | Быстрый, строгий, стандарт для Nuxt-экосистемы |
| Монорепо (Go) | **Один Go-модуль `backend/`** для всех сервисов | Одна версия зависимостей, атомарные изменения контрактов, `go test ./...` проверяет всё. Изоляция сервисов — через `internal/` |
| Оркестрация задач | **Taskfile (go-task)** | Одна точка входа `task <cmd>` для Go и JS; YAML понятен и людям, и агенту |
| Версии инструментов | **mise** | Фиксирует версии Go, Node, pnpm, buf в `mise.toml` — одинаково у всех и в CI |
| Линт/тесты Go | **golangci-lint v2**, **`testing` + testcontainers-go** | |
| Hot reload Go | **air** | |
| Git-хуки | **lefthook** | Один бинарь для Go и JS |
| Контейнеризация | **Docker + Docker Compose** | Одинаковое окружение у всех, в CI и в проде |
| CI | **GitHub Actions** | Те же `task` команды, что и локально |

### Backend (микросервисы)

| Слой | Выбор | Почему |
|---|---|---|
| Фреймворк сервисов | **Gin** — во всех сервисах | Самый популярный Go-фреймворк (~48%), лучше всего знаком моделям; единые middleware (recovery, логи, трейсинг, health) |
| Точка входа | **API Gateway / BFF на Gin** | Единственный публичный сервис: REST для Nuxt, auth, агрегация данных |
| Публичный контракт | **OpenAPI 3.0** → oapi-codegen (`gin-server` + strict) | Типобезопасные хендлеры gateway и TS-клиент для Nuxt из одной спеки |
| Сервис ↔ сервис (sync) | **Connect-RPC** (protobuf, совместим с gRPC) + **buf** | Строгие контракты, кодогенерация, работает поверх `net/http` → монтируется прямо в Gin |
| Сервис ↔ сервис (async) | **NATS JetStream** + паттерн **Transactional Outbox** | Простой, лёгкий, написан на Go; at-least-once доставка, дедупликация |
| Формат событий | **Protobuf** (в том же `proto/`) | Версионируемые схемы, проверка breaking changes через `buf breaking` |
| Данные | **PostgreSQL 17, database-per-service** · **sqlc + pgx v5** · **goose** | У каждого сервиса своя БД; SQL → типизированный Go-код |
| Наблюдаемость | **OpenTelemetry** (otelgin, otelconnect) → **Grafana LGTM** | Без распределённого трейсинга микросервисы не отлаживаются |
| Логи / конфиг | **`log/slog`** · **caarlos0/env** | |
| Устойчивость | context-таймауты, ретраи (Connect interceptors), **sony/gobreaker** | |
| Reverse proxy | **Caddy** | HTTPS, `/api/*` → gateway, остальное → Nuxt |

### Frontend

| Слой | Выбор |
|---|---|
| Фреймворк | **Nuxt 4** (TypeScript strict) |
| API-клиент | **nuxt-open-fetch** (openapi-typescript) — из OpenAPI-спеки gateway |
| UI | **Nuxt UI 4** + Tailwind CSS 4 |
| Состояние | Кеш `useFetch`/`useAsyncData` для серверных данных, **Pinia** для клиентских |
| Формы | **Zod** + `UForm` |
| Линт / тесты | **@nuxt/eslint** + Prettier · **Vitest + @nuxt/test-utils**, **Playwright** |

---

## 1. Анализ

### 1.1 Две экосистемы в одном репозитории

Go и Node имеют свои менеджеры зависимостей, сборщики и тест-раннеры. JS-ориентированные инструменты (Nx, Turborepo) можно натянуть на Go, но это лишний слой: Go и так кеширует сборку и тесты.

**Вывод:** каждая экосистема использует нативные инструменты (Go-модуль и `pnpm-workspace.yaml`), а поверх — тонкий единый task runner (Taskfile).

### 1.2 Сравнение оркестраторов

| Инструмент | Плюсы | Минусы | Вердикт |
|---|---|---|---|
| **Taskfile** | Язык-независимый, один бинарь, зависимости между задачами, `sources/generates` для инкрементальности | Нет удалённого кеша | ✅ **Рекомендуется** |
| Makefile | Есть везде | Синтаксис табов, плохо на Windows, сложно читать | Допустимо |
| Turborepo | Отличный кеш для JS | Go-пакеты нужно оборачивать в `package.json` | Избыточно для 1 фронта |
| Nx | Мощный граф, плагины для Go | Тяжёлый, много конфигурации и магии | Только для больших команд |
| Bazel | Герметичные сборки | Огромный порог входа | ❌ |

### 1.3 Один Go-модуль или модуль на сервис

| Вариант | Плюсы | Минусы | Вердикт |
|---|---|---|---|
| **Один модуль `backend/go.mod`** | Одна версия каждой зависимости, изменение proto + всех сервисов — один коммит, один `go test ./...`, простые Dockerfile | Все сервисы обновляют зависимости вместе | ✅ Для одной команды и одного репо |
| Модуль на сервис + `go.work` | Независимые версии зависимостей | `replace`-директивы, сложнее Docker-сборка, дрейф версий | Когда сервисы ведут разные команды |
| Polyrepo | Полная независимость | Изменение контракта = PR в N репозиториев | ❌ Противоречит идее монорепо |

Изоляцию сервисов в одном модуле обеспечивает правило Go: пакеты из `services/order/internal/...` **нельзя импортировать** из других сервисов. Компилятор сам не даст «срастить» сервисы.

### 1.4 Контракты: три вида взаимодействия

| Взаимодействие | Контракт | Инструмент | Почему |
|---|---|---|---|
| **Nuxt → Gateway** | OpenAPI 3.0 (`api/openapi.yaml`) | oapi-codegen + nuxt-open-fetch | REST удобен браузеру и SSR, типы на обеих сторонах |
| **Сервис → сервис** (запрос/ответ) | Protobuf (`backend/proto/`) | buf + Connect-RPC | Строгие типы, бинарный формат, `buf breaking` ловит несовместимые изменения |
| **Сервис → сервисы** (события) | Protobuf (`backend/proto/events/`) | NATS JetStream | Слабая связанность, асинхронность, переживает падение получателя |

Сравнение вариантов для внутренней sync-связи:

| Вариант | Вердикт |
|---|---|
| **Connect-RPC** | ✅ Совместим с gRPC и gRPC-Web, но работает на обычном `net/http`: хендлер монтируется в Gin, вызов можно сделать даже `curl` с JSON. Меньше магии, чем у grpc-go |
| grpc-go | Свой сервер (не `net/http`), не монтируется в Gin, сложнее отладка |
| REST + OpenAPI между сервисами | Возможно, но много ручной работы с версиями, нет `buf breaking` |
| GraphQL Federation | Избыточно |

> Для OpenAPI используем **3.0.x**: у oapi-codegen поддержка 3.1 до сих пор неполная.

### 1.5 Фреймворк: Gin

| Фреймворк | Сильные стороны | Слабые стороны | Для Claude Code |
|---|---|---|---|
| **Gin** | Лидер по популярности (~48%), зрелый, экосистема `gin-contrib`, поддержка в oapi-codegen и otelgin | Хендлеры не возвращают `error` | ⭐⭐⭐ Больше всего примеров, модель знает API наизусть |
| **Echo** | Хендлеры возвращают `error`, чистый дизайн | v5 вышел в 2026 с breaking changes, примеры в сети ещё под v4 | ⭐⭐ |
| **Fiber** | Самый быстрый в бенчмарках | fasthttp вместо `net/http` → не работает с Connect, otel-инструментацией, `httptest` | ⭐ |
| **Kratos / go-zero** | Готовые «микросервисные фреймворки» | Свои конвенции, генераторы и DSL; меньше примеров | ⭐ Много специфичной магии |
| **go-kit / go-micro** | Исторически популярны | Малоактивны, многословны | ❌ |
| **Encore / Dapr** | Инфраструктура «из коробки» | Привязка к платформе / sidecar-сложность | ⭐ |

**Выбор: Gin во всех сервисах.** Публичный gateway использует Gin + oapi-codegen, внутренние сервисы — Gin как HTTP-сервер, в который монтируются Connect-хендлеры (плюс `/healthz`, `/readyz`, `/metrics`). Так у всех сервисов одинаковый bootstrap, middleware и наблюдаемость, а агенту нужно знать один фреймворк.

### 1.6 Доступ к данным

| Вариант | Вердикт |
|---|---|
| **sqlc + pgx** | ✅ SQL в `.sql` файлах, типизированный Go-код генерируется. Агенту легко: SQL явный, ошибки ловит компилятор |
| GORM | Много неявного поведения, ошибки всплывают в рантайме |
| ent | Мощный, но много генерируемого кода и своя DSL |
| sqlx / голый pgx | Без типобезопасности запросов |

### 1.7 Frontend: Nuxt 4

- Структура `app/` отделяет клиентский код от `server/`, конфигов и `shared/`.
- **Nuxt — только SSR-слой**, бизнес-логика — в Go-сервисах. Nuxt общается **только с gateway** и ничего не знает о внутренних сервисах.
- В dev Nuxt проксирует `/api/**` на gateway через `routeRules`, в проде это делает Caddy.

---

## 2. Микросервисная архитектура backend

### 2.1 Общая схема

```
                    ┌──────────────┐
  Браузер ────────▶ │    Caddy     │  HTTPS
                    └──────┬───────┘
              /*           │          /api/*
        ┌──────────────────┴──────────────────┐
        ▼                                     ▼
 ┌─────────────┐   SSR: /api/*         ┌──────────────┐
 │  Nuxt (web) │ ────────────────────▶ │   gateway    │  Gin + OpenAPI
 └─────────────┘                       │  (BFF, auth) │  единственный публичный API
                                       └──────┬───────┘
                         Connect-RPC (protobuf)│
               ┌───────────────────────────────┼──────────────────┐
               ▼                               ▼                  ▼
        ┌────────────┐                  ┌────────────┐     ┌──────────────┐
        │  identity  │ ◀── Connect ──── │   order    │     │ notification │
        │ Gin+Connect│                  │ Gin+Connect│     │  (consumer)  │
        └─────┬──────┘                  └─────┬──────┘     └──────┬───────┘
              │                               │ outbox            ▲
              ▼                               ▼                   │
         [identity DB]                   [orders DB]              │
                                              │                   │
                                              └──▶ NATS JetStream ┘
                                                  order.created

  Все сервисы ──OTLP──▶ OpenTelemetry ──▶ Grafana (трейсы, логи, метрики)
```

### 2.2 Сервисы и их границы

Сервис = **бизнес-возможность (bounded context)**, а не таблица и не технический слой.

| Сервис | Ответственность | Владеет данными | Наружу |
|---|---|---|---|
| **gateway** | Публичный REST API, аутентификация (проверка JWT), агрегация ответов, rate limiting | — (без БД) | OpenAPI для Nuxt |
| **identity** | Пользователи, регистрация, логин, выпуск токенов, роли | `identity` DB | Connect: `IdentityService` |
| **order** | Заказы, их статусы и правила | `orders` DB | Connect: `OrderService`; события `order.*` |
| **notification** | Email/push-уведомления по событиям | `notification` DB (история, идемпотентность) | Только подписки на события |

Новые сервисы добавляются по тому же шаблону (`task new:service -- <name>`, см. раздел 6.4).

> **Совет по старту:** начните с 3–4 сервисов, как в таблице. Дробить сервис позже проще, чем склеивать слишком мелкие.

### 2.3 Синхронное взаимодействие: Connect-RPC

**Контракт** — `backend/proto/identity/v1/identity.proto`:
```proto
syntax = "proto3";
package identity.v1;

service IdentityService {
  rpc GetUser(GetUserRequest) returns (GetUserResponse);
  rpc Authenticate(AuthenticateRequest) returns (AuthenticateResponse);
}

message GetUserRequest  { int64 id = 1; }
message GetUserResponse { User user = 1; }

message User {
  int64  id    = 1;
  string email = 2;
  string name  = 3;
}
```

**Сервер** — Connect-хендлер монтируется в Gin:
```go
// services/identity/internal/rpc/server.go
type Server struct{ svc *service.Users }

func (s *Server) GetUser(ctx context.Context, req *connect.Request[identityv1.GetUserRequest]) (*connect.Response[identityv1.GetUserResponse], error) {
	u, err := s.svc.Get(ctx, req.Msg.Id)
	if errors.Is(err, model.ErrNotFound) {
		return nil, connect.NewError(connect.CodeNotFound, err)
	}
	if err != nil {
		return nil, fmt.Errorf("get user: %w", err)
	}
	return connect.NewResponse(&identityv1.GetUserResponse{User: toProto(u)}), nil
}

// services/identity/cmd/identity/main.go
r := httpx.NewRouter(logger) // общий bootstrap Gin из backend/pkg/httpx: recovery, slog, otelgin, /healthz
path, h := identityv1connect.NewIdentityServiceHandler(rpcServer, connect.WithInterceptors(otelInterceptor))
r.Any(path+"*method", gin.WrapH(h))
```

**Клиент** (в gateway или order):
```go
client := identityv1connect.NewIdentityServiceClient(httpClient, cfg.IdentityURL,
	connect.WithInterceptors(otelInterceptor, retryInterceptor))

res, err := client.GetUser(ctx, connect.NewRequest(&identityv1.GetUserRequest{Id: id}))
if connect.CodeOf(err) == connect.CodeNotFound { /* ... */ }
```

### 2.4 Асинхронное взаимодействие: NATS JetStream + Outbox

**Проблема:** «записать в БД и опубликовать событие» — две операции. Если сервис упадёт между ними, данные и события разойдутся.

**Решение — Transactional Outbox:**
1. Сервис в **одной транзакции** пишет бизнес-данные и строку в таблицу `outbox`.
2. Фоновый relay (горутина в том же сервисе, из `backend/pkg/outbox`) читает `outbox` и публикует в JetStream с заголовком `Nats-Msg-Id = outbox.id` — JetStream отбрасывает дубликаты.
3. Потребитель обрабатывает сообщение **идемпотентно** (таблица `processed_messages`) и только потом делает `Ack`.

```go
// services/order/internal/service/orders.go
func (s *Orders) Create(ctx context.Context, in model.NewOrder) (model.Order, error) {
	var out model.Order
	err := s.tx.WithTx(ctx, func(q *db.Queries) error {
		o, err := q.InsertOrder(ctx, toDBParams(in))
		if err != nil {
			return fmt.Errorf("insert order: %w", err)
		}
		out = fromDB(o)
		return outbox.Add(ctx, q, "order.created", &eventsv1.OrderCreated{
			OrderId: o.ID, UserId: o.UserID, Total: o.Total,
		})
	})
	return out, err
}
```

```go
// services/notification/internal/consumer/order_created.go
func (c *Consumer) HandleOrderCreated(ctx context.Context, msgID string, ev *eventsv1.OrderCreated) error {
	return c.tx.WithTx(ctx, func(q *db.Queries) error {
		if done, _ := q.MarkProcessed(ctx, msgID); !done { // уже обработано → просто Ack
			return nil
		}
		return c.mailer.SendOrderConfirmation(ctx, ev.UserId, ev.OrderId)
	})
}
```

**Именование subjects:** `<сервис>.<сущность>.<событие>` → `order.order.created`, `identity.user.registered`. Один stream на сервис-источник (`ORDER`, `IDENTITY`), durable pull-consumers на стороне получателей.

| Брокер | Вердикт |
|---|---|
| **NATS JetStream** | ✅ Один маленький бинарь, написан на Go, отличный Go-клиент, persistence, дедупликация, request-reply |
| Kafka | Для огромных объёмов и event sourcing; тяжёлый в эксплуатации |
| RabbitMQ | Хорош, но NATS проще и ближе к Go-экосистеме |
| Redis Streams | Подходит для простого, но слабее гарантии и тулинг |

### 2.5 Gateway (BFF)

- **Единственный публичный сервис.** Внутренние сервисы не доступны снаружи (только внутренняя docker/k8s-сеть).
- Реализует `StrictServerInterface` из `api/openapi.yaml` (oapi-codegen, `gin-server`). Хендлеры разбиты по фичам — каждая фича реализует свою часть интерфейса:
  ```go
  type Server struct {
      *users.Handler
      *orders.Handler
  }
  var _ gen.StrictServerInterface = (*Server)(nil) // компилятор проверит полноту
  ```
- **Не содержит бизнес-логики:** проверка токена → вызов одного или нескольких сервисов (параллельно через `errgroup`) → маппинг protobuf → OpenAPI-типы.
- Маппит ошибки Connect в HTTP: `NotFound` → 404, `InvalidArgument` → 422, `PermissionDenied` → 403, `Unavailable` → 503.
- Rate limiting, CORS, request-id, лимиты размера тела — здесь, а не в каждом сервисе.

### 2.6 Внутреннее устройство сервиса

Каждый сервис — небольшое приложение со слоями:

```
services/order/
├── cmd/order/main.go       # точка входа: конфиг → app.New() → запуск
├── internal/
│   ├── app/                # ручной DI: сборка зависимостей, graceful shutdown
│   ├── rpc/                # транспорт: Connect-хендлеры (protobuf ⇄ домен)
│   ├── consumer/           # транспорт: обработчики NATS-событий
│   ├── service/            # бизнес-логика, транзакции, публикация событий
│   ├── repository/         # доступ к данным через sqlc (БД ⇄ домен)
│   ├── client/             # клиенты других сервисов (обёртки над Connect)
│   ├── model/              # доменные модели и ошибки (ErrNotFound…)
│   └── gen/db/             # ⚙️ sqlc (не редактировать)
├── queries/                # *.sql для sqlc
├── migrations/             # goose, встраиваются в бинарь через embed
└── sqlc.yaml
```

Правила внутри сервиса:
1. **Зависимости внутрь:** `rpc`/`consumer` → `service` → `repository`. Сервисный слой ничего не знает о Gin, Connect и NATS.
2. **Сгенерированные типы не выходят за свой слой:** protobuf — только в `rpc`/`consumer`/`client`, sqlc — только в `repository`.
3. **Интерфейсы объявляет потребитель** (`service` объявляет нужный ему интерфейс репозитория и клиентов) — это позволяет тестировать с фейками.
4. **DI вручную** в `internal/app`, без wire/fx.
5. Если сервис вырастает до нескольких поддоменов — внутри `service/` группируем по фичам; если до многих — это сигнал разделить сервис.

### 2.7 Данные и согласованность

- **Database-per-service.** Сервис читает и пишет только свою БД. Никаких JOIN между БД разных сервисов, никаких общих таблиц. В dev — один контейнер Postgres с отдельной базой на сервис; в проде — отдельные инстансы или как минимум отдельные базы и пользователи.
- **Нужны чужие данные?** Либо синхронный запрос к сервису-владельцу, либо локальная read-копия, обновляемая по событиям (например, notification хранит email пользователя из `identity.user.registered`).
- **Распределённые операции — саги (хореография через события)**, без распределённых транзакций:
  ```
  order: создан (PENDING) ──order.created──▶ payment: списать
                                              │
            ◀──payment.succeeded / failed─────┘
  order: CONFIRMED / CANCELLED (компенсация)
  ```

### 2.8 Аутентификация

- **identity** выпускает короткоживущий access JWT (подписан ключом, публикует JWKS) и refresh-токен.
- **gateway** хранит токены в httpOnly-cookie, проверяет JWT по JWKS и передаёт в сервисы идентичность через Connect-заголовок (`X-User-Id`, `X-User-Roles`) — interceptor кладёт её в `context.Context`.
- Внутренние сервисы доверяют этим заголовкам только потому, что недоступны снаружи. При росте требований — mTLS между сервисами (например, через service mesh в Kubernetes).

### 2.9 Наблюдаемость

В микросервисах это **обязательно с первого дня**: без трейсинга невозможно понять, где упал запрос.

- **OpenTelemetry SDK** во всех сервисах, общий setup в `backend/pkg/observability`.
- Автоинструментация: `otelgin` (HTTP), `otelconnect` (RPC), propagation контекста трейса через заголовки NATS-сообщений, `otelpgx` (запросы к БД).
- **slog** с `trace_id` в каждой записи — логи связываются с трейсами.
- В dev — один контейнер **`grafana/otel-lgtm`** (Grafana + Loki + Tempo + Prometheus + OTel Collector). В проде — то же самое по отдельности или managed-решение.

### 2.10 Надёжность

| Механизм | Где |
|---|---|
| Таймауты через `context` на каждый исходящий вызов | клиенты Connect, БД |
| Ретраи только для идемпотентных операций (с backoff) | Connect interceptor в `pkg/rpcx` |
| Circuit breaker | `sony/gobreaker` в клиентах gateway |
| `/healthz` (жив) и `/readyz` (БД и NATS доступны) | все сервисы |
| Graceful shutdown (перестать принимать → дождаться запросов → закрыть БД/NATS) | `internal/app` |
| Идемпотентность потребителей | таблица `processed_messages` |

### 2.11 Правила против «распределённого монолита»

1. Сервисы общаются **только** через proto-контракты (RPC или события). Никакого доступа к чужой БД.
2. `backend/pkg/` — **только инфраструктура** (httpx, rpcx, outbox, observability, dbx, auth). Доменных моделей и бизнес-логики в `pkg/` нет.
3. Контракты эволюционируют обратно совместимо: только добавляем поля, `buf breaking` в CI проверяет это относительно `main`. Несовместимые изменения — через новую версию пакета (`order.v2`).
4. Сервис должен разворачиваться независимо: отдельный бинарь, отдельный образ, свои миграции.
5. Если два сервиса всегда меняются вместе и постоянно синхронно зовут друг друга — это один сервис.

### 2.12 Тестирование

| Уровень | Что проверяет | Инструменты |
|---|---|---|
| Юнит | бизнес-логика `service/` | `testing` + фейки |
| Интеграционные | репозитории, outbox, консьюмеры | testcontainers-go (Postgres, NATS) |
| Контрактные | совместимость proto / OpenAPI | `buf breaking`, `buf lint`; oapi-codegen + `vue-tsc` после `task gen` |
| Компонентные | сервис целиком по RPC | `httptest` + реальный Gin-роутер + testcontainers |
| E2E | сценарии через gateway | `task dev:docker` + Playwright / Go-тесты к gateway |

---

## 3. Структура репозитория

```
.
├── CLAUDE.md                     # Главные инструкции для Claude Code
├── README.md
├── Taskfile.yml                  # Единая точка входа: task dev / gen / check / test
├── mise.toml                     # Версии go, node, pnpm, task, buf, golangci-lint
├── lefthook.yml
├── compose.yaml                  # 🐳 dev: инфраструктура + профиль full (все сервисы)
├── compose.prod.yaml             # 🐳 prod
├── .dockerignore
├── pnpm-workspace.yaml
├── package.json
│
├── api/
│   └── openapi.yaml              # ⭐ Публичный контракт gateway (для Nuxt)
│
├── backend/
│   ├── CLAUDE.md                 # Правила для Go-части
│   ├── go.mod                    # один модуль на все сервисы
│   ├── Dockerfile                # 🐳 один Dockerfile, ARG SERVICE=<name>
│   ├── .golangci.yml
│   ├── proto/                    # ⭐ Внутренние контракты
│   │   ├── buf.yaml
│   │   ├── buf.gen.yaml
│   │   ├── identity/v1/identity.proto
│   │   ├── order/v1/order.proto
│   │   └── events/v1/order_events.proto
│   ├── gen/proto/                # ⚙️ buf generate (не редактировать)
│   ├── pkg/                      # общая ИНФРАСТРУКТУРА (без бизнес-логики)
│   │   ├── httpx/                #   bootstrap Gin: recovery, slog, otelgin, health
│   │   ├── rpcx/                 #   Connect interceptors: auth, retry, errors
│   │   ├── outbox/               #   transactional outbox + relay
│   │   ├── natsx/                #   подключение, consumers, propagation
│   │   ├── dbx/                  #   pgx pool, WithTx, goose runner
│   │   ├── observability/        #   OpenTelemetry setup
│   │   └── config/
│   └── services/
│       ├── gateway/              # Gin + oapi-codegen, без БД
│       │   ├── CLAUDE.md
│       │   ├── cmd/gateway/main.go
│       │   ├── internal/{app,httpapi,users,orders,gen/api}
│       │   └── oapi-codegen.yaml
│       ├── identity/             # структура как в разделе 2.6
│       ├── order/
│       └── notification/
│
├── apps/
│   └── web/                      # Nuxt 4 frontend (см. раздел 5)
│       ├── CLAUDE.md
│       ├── nuxt.config.ts
│       ├── Dockerfile
│       ├── app/
│       │   ├── pages/  layouts/  middleware/  plugins/  utils/
│       │   ├── components/{base,user,order}/
│       │   ├── composables/{user,order}/
│       │   ├── stores/
│       │   ├── error.vue
│       │   └── app.vue
│       ├── shared/  server/  tests/  e2e/
│
├── deploy/
│   ├── Caddyfile
│   └── postgres/init.sql         # создаёт базы identity, orders, notification
│
├── templates/service/            # шаблон нового сервиса для task new:service
│
├── .claude/
│   ├── settings.json
│   ├── commands/
│   └── skills/
│
└── .github/workflows/ci.yml
```

### Ключевые конфиги

**`mise.toml`**
```toml
[tools]
go = "1.26"
node = "24"
pnpm = "10"
task = "latest"
buf = "latest"
golangci-lint = "2"
"go:google.golang.org/protobuf/cmd/protoc-gen-go" = "latest"
"go:connectrpc.com/connect/cmd/protoc-gen-connect-go" = "latest"
```

**`backend/proto/buf.gen.yaml`**
```yaml
version: v2
managed:
  enabled: true
  override:
    - file_option: go_package_prefix
      value: example.com/app/backend/gen/proto
plugins:
  - local: protoc-gen-go
    out: ../gen/proto
    opt: paths=source_relative
  - local: protoc-gen-connect-go
    out: ../gen/proto
    opt: paths=source_relative
```

**`Taskfile.yml`** (сокращённо)
```yaml
version: '3'

vars:
  SERVICES: gateway identity order notification

tasks:
  dev:
    desc: Инфраструктура в Docker + все сервисы и Nuxt нативно с hot reload
    deps: [infra:up]
    cmds:
      - task --parallel {{range splitList " " .SERVICES}}dev:svc:{{.}} {{end}}dev:web

  dev:svc:*:
    desc: Запустить один сервис, напр. task dev:svc:order
    vars: { SVC: '{{index .MATCH 0}}' }
    dir: backend
    cmd: go tool air --build.cmd "go build -o ./tmp/{{.SVC}} ./services/{{.SVC}}/cmd/{{.SVC}}" --build.bin "./tmp/{{.SVC}}"

  dev:web:
    cmd: pnpm --filter web dev

  dev:docker:
    desc: Всё в контейнерах
    cmd: docker compose --profile full up --build

  infra:up:
    cmd: docker compose up -d --wait postgres nats otel-lgtm

  gen:
    desc: Вся кодогенерация
    cmds: [task: gen:proto, task: gen:api, task: gen:sql]

  gen:proto:
    dir: backend/proto
    sources: ['**/*.proto']
    cmds:
      - buf lint
      - buf generate

  gen:api:
    sources: [api/openapi.yaml]
    cmds:
      - cd backend/services/gateway && go tool oapi-codegen -config oapi-codegen.yaml ../../../api/openapi.yaml
      - pnpm --filter web exec nuxi prepare

  gen:sql:
    cmds:
      - for: { var: SERVICES }
        cmd: '[ -f backend/services/{{.ITEM}}/sqlc.yaml ] && (cd backend/services/{{.ITEM}} && go tool sqlc generate) || true'

  new:service:
    desc: Создать сервис из шаблона — task new:service -- billing
    cmd: ./scripts/new-service.sh {{.CLI_ARGS}}

  lint:
    cmds:
      - cd backend && golangci-lint run ./...
      - cd backend/proto && buf breaking --against '../../.git#branch=main,subdir=backend/proto'
      - pnpm --filter web lint

  typecheck:
    cmds:
      - cd backend && go build ./...
      - pnpm --filter web typecheck

  test:
    cmds:
      - cd backend && go test ./...
      - pnpm --filter web test

  check:
    desc: Всё, что прогоняет CI — запускать перед коммитом
    cmds: [task: gen, task: lint, task: typecheck, task: test]

  docker:prod:
    cmd: docker compose -f compose.prod.yaml up --build -d

  down:
    cmd: docker compose --profile full down
```

> Go-инструменты (oapi-codegen, sqlc, goose, air) подключаются через **`tool`-директиву в `backend/go.mod`** (Go 1.24+) и вызываются как `go tool <name>`. Плагины protoc ставит mise.

**`apps/web/nuxt.config.ts`** (фрагмент)
```ts
export default defineNuxtConfig({
  compatibilityDate: '2026-10-01',
  modules: ['@nuxt/ui', '@nuxt/eslint', '@pinia/nuxt', 'nuxt-open-fetch'],
  typescript: { strict: true, typeCheck: false },
  imports: { dirs: ['composables/**'] },
  openFetch: {
    clients: {
      api: { baseURL: '/api', schema: '../../api/openapi.yaml' },
    },
  },
  routeRules: {
    // Читается при сборке: локально — localhost, в Docker — http://gateway:8080
    '/api/**': { proxy: `${process.env.NUXT_API_PROXY_TARGET ?? 'http://localhost:8080'}/api/**` },
  },
})
```

---

## 4. Docker

**Принцип:** Docker даёт одинаковое окружение везде, но в dev не должен замедлять цикл «изменил → проверил».

| Режим | Команда | Что в Docker | Когда |
|---|---|---|---|
| **Гибридный (по умолчанию)** | `task dev` | Postgres, NATS, Grafana LGTM | Ежедневная разработка и работа Claude Code: сервисы и Nuxt нативно с hot reload, тесты без `docker exec` |
| **Один сервис** | `task dev:svc:order` | + запускать остальное по необходимости | Работа над конкретным сервисом |
| **Полный** | `task dev:docker` | Всё | Онбординг, проверка «как в проде», e2e |
| **Prod** | `task docker:prod` | Caddy + все сервисы + Web + инфраструктура | Локальная проверка prod-сборки, деплой на VPS |

> Почему не «всё в Docker» по умолчанию: на macOS bind-mount и file watching в контейнерах заметно медленнее, а Claude Code постоянно запускает `go test`, `buf generate`, `vue-tsc`.

**`compose.yaml`** (dev)
```yaml
name: app

x-go-service: &go-service
  profiles: [full]
  build: { context: ., dockerfile: backend/Dockerfile, target: dev }
  volumes: [./backend:/src/backend]
  depends_on:
    postgres: { condition: service_healthy }
    nats: { condition: service_started }

x-env: &common-env
  NATS_URL: nats://nats:4222
  OTEL_EXPORTER_OTLP_ENDPOINT: http://otel-lgtm:4317

services:
  # ---------- инфраструктура ----------
  postgres:
    image: postgres:17-alpine
    environment: { POSTGRES_USER: app, POSTGRES_PASSWORD: app }
    ports: ["5432:5432"]
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./deploy/postgres/init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U app"]
      interval: 2s
      retries: 20

  nats:
    image: nats:2-alpine
    command: ["-js", "-sd", "/data", "-m", "8222"]
    ports: ["4222:4222", "8222:8222"]
    volumes: [natsdata:/data]

  otel-lgtm:
    image: grafana/otel-lgtm
    ports: ["3001:3000", "4317:4317", "4318:4318"]   # Grafana на :3001 (3000 занят Nuxt)

  # ---------- сервисы (профиль full) ----------
  gateway:
    <<: *go-service
    environment:
      <<: *common-env
      SERVICE: gateway
      IDENTITY_URL: http://identity:8080
      ORDER_URL: http://order:8080
    ports: ["8080:8080"]

  identity:
    <<: *go-service
    environment:
      <<: *common-env
      SERVICE: identity
      DATABASE_URL: postgres://app:app@postgres:5432/identity?sslmode=disable

  order:
    <<: *go-service
    environment:
      <<: *common-env
      SERVICE: order
      DATABASE_URL: postgres://app:app@postgres:5432/orders?sslmode=disable
      IDENTITY_URL: http://identity:8080

  notification:
    <<: *go-service
    environment:
      <<: *common-env
      SERVICE: notification
      DATABASE_URL: postgres://app:app@postgres:5432/notification?sslmode=disable

  web:
    profiles: [full]
    build: { context: ., dockerfile: apps/web/Dockerfile, target: dev }
    environment: { NUXT_API_PROXY_TARGET: http://gateway:8080 }
    ports: ["3000:3000"]
    develop:
      watch:
        - { action: sync, path: ./apps/web, target: /src/apps/web, ignore: [node_modules/, .nuxt/] }
        - { action: rebuild, path: ./apps/web/package.json }
    depends_on: [gateway]

volumes:
  pgdata:
  natsdata:
```

> Внутри docker-сети все сервисы слушают `:8080` и различаются по DNS-имени. Наружу проброшен только gateway (и Grafana для разработчика).

**`deploy/postgres/init.sql`**
```sql
CREATE DATABASE identity;
CREATE DATABASE orders;
CREATE DATABASE notification;
```

**`backend/Dockerfile`** — один на все сервисы
```dockerfile
# ---------- base ----------
FROM golang:1.26-alpine AS base
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# ---------- dev: hot reload нужного сервиса ----------
FROM base AS dev
CMD go tool air --build.cmd "go build -o ./tmp/$SERVICE ./services/$SERVICE/cmd/$SERVICE" --build.bin "./tmp/$SERVICE"

# ---------- build ----------
FROM base AS build
ARG SERVICE
COPY backend/ ./
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/app ./services/${SERVICE}/cmd/${SERVICE}

# ---------- prod ----------
FROM gcr.io/distroless/static-debian12:nonroot AS prod
COPY --from=build /out/app /app
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/app"]
```

Сборка образа сервиса: `docker build -f backend/Dockerfile --target prod --build-arg SERVICE=order -t app-order .`

> Миграции goose встраиваются в бинарь сервиса через `embed.FS` и применяются при старте — каждый сервис мигрирует только свою БД.

**`apps/web/Dockerfile`**
```dockerfile
FROM node:24-alpine AS base
RUN corepack enable
WORKDIR /src

FROM base AS deps
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./
COPY apps/web/package.json apps/web/
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile --filter web...

FROM deps AS dev
COPY api/ api/
COPY apps/web/ apps/web/
WORKDIR /src/apps/web
CMD ["pnpm", "dev", "--host", "0.0.0.0"]

FROM deps AS build
COPY api/ api/
COPY apps/web/ apps/web/
ARG NUXT_API_PROXY_TARGET=http://gateway:8080
ENV NUXT_API_PROXY_TARGET=$NUXT_API_PROXY_TARGET
RUN pnpm --filter web build

FROM node:24-alpine AS prod
WORKDIR /app
COPY --from=build /src/apps/web/.output ./.output
ENV NODE_ENV=production PORT=3000
EXPOSE 3000
USER node
CMD ["node", ".output/server/index.mjs"]
```

**`compose.prod.yaml`** (сокращённо)
```yaml
name: app-prod

x-go-prod: &go-prod
  restart: unless-stopped
  env_file: .env.prod
  depends_on:
    postgres: { condition: service_healthy }
    nats: { condition: service_started }

services:
  caddy:
    image: caddy:2-alpine
    ports: ["80:80", "443:443"]
    volumes:
      - ./deploy/Caddyfile:/etc/caddy/Caddyfile:ro
      - caddy_data:/data
    depends_on: [gateway, web]

  gateway:
    <<: *go-prod
    build: { context: ., dockerfile: backend/Dockerfile, target: prod, args: { SERVICE: gateway } }
  identity:
    <<: *go-prod
    build: { context: ., dockerfile: backend/Dockerfile, target: prod, args: { SERVICE: identity } }
  order:
    <<: *go-prod
    build: { context: ., dockerfile: backend/Dockerfile, target: prod, args: { SERVICE: order } }
  notification:
    <<: *go-prod
    build: { context: ., dockerfile: backend/Dockerfile, target: prod, args: { SERVICE: notification } }

  web:
    build: { context: ., dockerfile: apps/web/Dockerfile, target: prod }
    restart: unless-stopped

  postgres:
    image: postgres:17-alpine
    env_file: .env.prod
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./deploy/postgres/init.sql:/docker-entrypoint-initdb.d/init.sql:ro
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U $${POSTGRES_USER}"]
      interval: 5s
    restart: unless-stopped

  nats:
    image: nats:2-alpine
    command: ["-js", "-sd", "/data"]
    volumes: [natsdata:/data]
    restart: unless-stopped

volumes:
  pgdata:
  natsdata:
  caddy_data:
```

**`deploy/Caddyfile`**
```
{$DOMAIN:localhost} {
    encode zstd gzip
    handle /api/* {
        reverse_proxy gateway:8080
    }
    handle {
        reverse_proxy web:3000
    }
}
```

**`.dockerignore`**
```
**/node_modules
**/.nuxt
**/.output
**/tmp
.git
.env*
!.env.example
```

**Масштабирование прода.** Docker Compose на одном VPS подходит для старта. Когда понадобятся несколько нод, автоскейлинг и независимые выкладки сервисов — Kubernetes (k3s или managed) + Helm/Kustomize; образы и `/healthz`/`/readyz` уже готовы для этого.

**CI (GitHub Actions):**
1. `task check` — генерация, линт, `buf breaking`, типы, тесты (testcontainers работает на `ubuntu-latest`).
2. Определение изменённых сервисов (`dorny/paths-filter`): изменения в `backend/pkg/` или `backend/proto/` → пересобрать все, иначе — только затронутые.
3. `docker/build-push-action` с `cache-from/cache-to: type=gha` → образы в GHCR, тег = commit SHA.
4. (опционально) `trivy` — сканирование образов.

---

## 5. Архитектура frontend (Nuxt 4)

**Рекомендация: стандартная структура Nuxt с группировкой по фичам внутри папок + слой composables как единственная точка доступа к API.** При росте (5+ крупных доменов или несколько приложений) — переход на **Nuxt Layers** (`layers/<feature>/`).

```
pages/          ← роутинг и компоновка, минимум логики
   │
components/     ← UI: props вниз, events вверх, API не вызывают
   │
composables/    ← данные и логика фичи: useUser(), useUpdateUser()
   │
useApi / $api   ← типизированный клиент (nuxt-open-fetch)
   │
gateway (Go)
```

| Подход | Вердикт |
|---|---|
| **Nuxt-конвенции + фичи в подпапках** | ✅ Работает с auto-import и file-based routing; `components/user/Card.vue` → `<UserCard>` |
| **Nuxt Layers по доменам** | ✅ Следующий шаг при росте |
| Feature-Sliced Design (FSD) | Конфликтует с auto-import и роутингом Nuxt, много церемонии |
| Логика в компонентах | Невозможно переиспользовать и тестировать |

**Ключевые правила**

1. **Страницы тонкие** — параметры роута, composables, компоненты. Без бизнес-логики.
2. **Компоненты не вызывают API** — данные через props или composable, наружу события.
3. **Доступ к API — только через composables фичи:**
   ```ts
   // app/composables/user/useUser.ts
   export function useUser(id: MaybeRefOrGetter<number>) {
     return useApi('/users/{id}', {
       path: { id: toValue(id) },
       key: () => `user:${toValue(id)}`,
     })
   }

   export function useUpdateUser() {
     const { $api } = useNuxtApp()
     return async (id: number, body: UpdateUser) => {
       await $api('/users/{id}', { method: 'PATCH', path: { id }, body })
       await refreshNuxtData(`user:${id}`) // инвалидация кеша
     }
   }
   ```
4. **Серверное состояние ≠ Pinia.** Данные бэкенда — в кеше `useFetch`/`useAsyncData`. **Pinia — только клиентское состояние** (сессия, UI, черновики). При сложных мутациях и инвалидации — **Pinia Colada** (`@pinia/colada`).
5. **Типы — только из OpenAPI** (`components['schemas']['User']`).
6. **Формы:** `UForm` + Zod; ошибки 422 от gateway маппятся на поля.
7. **Сквозная логика — в plugins и middleware:** плагин API-клиента обрабатывает 401, `middleware/auth.ts` защищает страницы.
8. **`server/` в Nuxt — минимально:** только прокси и cookie. Бизнес-логика — в Go.
9. `<script setup lang="ts">`, только Composition API, без `any`.

**Тестирование:** Vitest + `@nuxt/test-utils` (composables, `mountSuspended` для компонентов), Playwright для e2e.

> Фронтенд не знает о микросервисах: для него существует только gateway и `api/openapi.yaml`. Разделение бэкенда на сервисы можно менять, не трогая Nuxt.

---

## 6. Почему этот стек хорошо работает с Claude Code

Claude Code эффективнее всего, когда у него **быстрый и однозначный цикл обратной связи**: изменил → проверил → увидел точную ошибку → исправил. Микросервисы усложняют этот цикл, поэтому стек специально компенсирует их сложность:

| Риск микросервисов для агента | Как компенсируется |
|---|---|
| Изменение затрагивает несколько сервисов | Монорепо + один Go-модуль: proto, gateway, сервисы и фронт меняются в одном коммите, `task check` проверяет всё сразу |
| Несовместимые контракты | proto + OpenAPI с кодогенерацией: компилятор Go и `vue-tsc` показывают все места поломки; `buf breaking` ловит несовместимость |
| Сложно понять, где упал запрос | OpenTelemetry + Grafana; `trace_id` в логах; `docker compose logs <svc>` |
| Каждый сервис устроен по-своему | Единый шаблон (`templates/service`) + общий `pkg/` → все сервисы выглядят одинаково |
| Агент теряется в большом репо | `CLAUDE.md` в корне, в `backend/`, в каждом сервисе и в `apps/web` — подгружаются по мере работы в папке |
| Нужно поднимать много инфраструктуры | `task dev` — одна команда; testcontainers даёт чистые Postgres/NATS в тестах |

Общие принципы:

| Принцип | Как реализовано |
|---|---|
| **Одна команда проверки** | `task check` = gen + lint + buf breaking + typecheck + test |
| **Типы сквозь весь стек** | SQL → sqlc → Go → proto → Go → OpenAPI → TS |
| **Явное лучше неявного** | Gin, Connect, sqlc, ручной DI, SQL-миграции — без ORM и DI-магии |
| **Популярные инструменты** | Хорошо представлены в обучающих данных — меньше выдуманных API |
| **Контракты в двух местах** | `api/openapi.yaml` (внешний) и `backend/proto/` (внутренние) — агент читает их и понимает всю систему |
| **Генерируемый код помечен** | `gen/` — «не редактировать, перегенерировать» |
| **Фиксированные версии** | mise + `go tool` + lockfile |

### 6.1 CLAUDE.md (корневой) — пример

```markdown
# Project

Go microservices (backend/) + Nuxt 4 (apps/web).
Public contract: api/openapi.yaml (gateway ⇄ web).
Internal contracts: backend/proto/ (Connect-RPC + NATS events).
Services: gateway (public, no DB), identity, order, notification.

## Commands
- `task dev`            — infra in Docker + all services and web natively
- `task dev:svc:<name>` — run one service with hot reload
- `task dev:docker`     — everything in containers
- `task gen`            — regenerate after changing *.proto, openapi.yaml or *.sql
- `task check`          — gen + lint + buf breaking + typecheck + test. Run before declaring work done.
- `task new:service -- <name>` — scaffold a service from templates/service

## Workflow: new public endpoint
1. api/openapi.yaml → `task gen`
2. If a service needs a new RPC: backend/proto/<svc>/v1/*.proto → `task gen`
3. Implement RPC in services/<svc>/internal/rpc, logic in internal/service
4. Implement gateway handler in services/gateway/internal/<feature>, calling the Connect client
5. Composable in apps/web/app/composables/<feature>/
6. `task check`

## Rules
- NEVER edit generated code: backend/gen/**, backend/services/*/internal/gen/**
- Services talk ONLY via proto contracts (Connect or NATS events). Never access another service's DB or internal/ package
- backend/pkg/ is infrastructure only — no domain logic there
- Proto changes must be backward compatible (add fields, never renumber/remove); breaking → new version package
- Writing data + publishing an event → always via outbox in the same transaction
- Event consumers must be idempotent
- Go: Gin in every service; gateway handlers implement gen.StrictServerInterface
- Go: rpc/consumer → service → repository; proto types never below rpc layer, sqlc types never above repository
- Go: errors wrapped with fmt.Errorf("...: %w", err); slog; context timeouts on every outgoing call; manual DI
- Vue: components never call the API — only composables; Pinia only for client state
- Vue: <script setup lang="ts">, Composition API, Nuxt UI first; no `any`
```

Плюс `backend/CLAUDE.md`, `backend/services/<svc>/CLAUDE.md` (ответственность сервиса, его события и зависимости) и `apps/web/CLAUDE.md`.

### 6.2 `.claude/settings.json`: разрешения и hooks

```json
{
  "permissions": {
    "allow": [
      "Bash(task:*)",
      "Bash(go test:*)",
      "Bash(go build:*)",
      "Bash(go tool:*)",
      "Bash(buf lint:*)",
      "Bash(buf generate:*)",
      "Bash(buf breaking:*)",
      "Bash(pnpm --filter web:*)",
      "Bash(golangci-lint run:*)",
      "Bash(docker compose ps)",
      "Bash(docker compose logs:*)",
      "Bash(docker compose up:*)",
      "Bash(docker build:*)",
      "Bash(git status)",
      "Bash(git diff:*)"
    ],
    "deny": [
      "Read(./.env)",
      "Read(./**/.env)",
      "Read(./.env.prod)"
    ]
  },
  "hooks": {
    "PostToolUse": [
      {
        "matcher": "Edit|Write",
        "hooks": [
          { "type": "command", "command": "scripts/claude-format.sh" }
        ]
      }
    ]
  }
}
```

`scripts/claude-format.sh` форматирует изменённый файл: `gofmt`/`goimports` для `.go`, `buf format -w` для `.proto`, `prettier`/`eslint --fix` для `.ts/.vue`.

### 6.3 Полезные MCP-серверы

| MCP | Зачем |
|---|---|
| **Playwright MCP** (или встроенный браузер Claude Code) | Агент сам открывает страницу и проверяет UI |
| **Postgres MCP** (read-only, локальные БД) | Смотреть схемы и данные сервисов при отладке |
| **Grafana MCP** | Искать трейсы и логи по `trace_id` при отладке межсервисных запросов |
| **Context7** | Актуальная документация Nuxt, Connect, NATS, Gin |
| **GitHub** | PR, issues, CI |

### 6.4 Свои slash-команды и skills

- `/new-service <name>` — `task new:service`, регистрация в compose, Taskfile, CI и CLAUDE.md.
- `/new-rpc <service> <описание>` — proto → gen → rpc-хендлер → service → тесты.
- `/new-event <описание>` — proto события → outbox в источнике → идемпотентный consumer в получателе.
- `/new-endpoint <описание>` — OpenAPI → gateway → вызов сервиса → composable на фронте.
- `/trace <trace_id>` — найти трейс в Grafana и объяснить, где и почему упал запрос.

Храните их в `.claude/commands/` или `.claude/skills/` — они в репозитории и доступны всей команде.

---

## 7. Рассмотренные и отклонённые альтернативы

| Альтернатива | Причина отказа |
|---|---|
| Модульный монолит | Требование — микросервисная архитектура. (Внутренняя структура сервисов сохраняет его плюсы) |
| grpc-go вместо Connect | Свой сервер, не монтируется в Gin, сложнее отлаживать |
| REST между сервисами | Нет строгой эволюции контрактов и `buf breaking` |
| Kafka / RabbitMQ | Тяжелее в эксплуатации; NATS JetStream достаточно и проще |
| Kratos / go-zero / go-kit | Собственные конвенции и генераторы, меньше примеров для модели |
| Dapr / Encore | Sidecar или привязка к платформе — лишний слой |
| Service mesh (Istio, Linkerd) с первого дня | Избыточно для старта; рассмотреть в Kubernetes при росте |
| Модуль на сервис + `go.work` | Дрейф версий и сложнее сборка; оправдано при разных командах |
| Echo | Хорошая альтернатива, но переходный период v4 → v5 |
| Fiber | fasthttp вместо `net/http` → не работает с Connect и otel |
| GORM / ent | Неявное поведение, рантайм-ошибки |
| Общая БД на все сервисы | Скрытая связанность → распределённый монолит |
| Nx / Bazel | Избыточно, Go кеширует сборку сам |
| Логика в Nuxt `server/` | Ещё один бэкенд → дублирование |
| npm / yarn, Husky | pnpm быстрее и строже; lefthook не требует Node |

---

## 8. План запуска

1. Установить Docker Desktop (или OrbStack на macOS), затем `mise install`.
2. Создать каркас по структуре из раздела 3: `backend/go.mod`, `backend/pkg/` (httpx, observability, dbx), `pnpm-workspace.yaml`, Taskfile.
3. `compose.yaml` с Postgres, NATS и otel-lgtm → `task infra:up`.
4. **Первый сервис — identity:** proto `GetUser`, Connect-хендлер в Gin, sqlc, миграции, тесты.
5. **gateway:** `api/openapi.yaml` с `GET /users/{id}`, oapi-codegen strict, Connect-клиент к identity.
6. `apps/web`: `pnpm dlx nuxi@latest init`, модули, первая страница через `useUser`.
7. Проверить сквозной трейс Nuxt → gateway → identity в Grafana.
8. **order + notification:** первый RPC между сервисами (order → identity) и первое событие через outbox (`order.created` → notification).
9. Вынести получившийся сервис в `templates/service` и сделать `task new:service`.
10. Добавить `CLAUDE.md` (корень, backend, каждый сервис, web), `.claude/settings.json`, lefthook и CI.

---

## Источники

- [Go 1.26 Release Notes](https://go.dev/doc/go1.26)
- [Go Release History](https://go.dev/doc/devel/release)
- [Nuxt Roadmap](https://nuxt.com/docs/3.x/community/roadmap)
- [Nuxt releases](https://releases.sh/nuxt/nuxt.md)
- [nuxt-open-fetch (Nuxt module)](https://nuxt.com/modules/nuxt-open-fetch)
- [nuxt-open-fetch (GitHub)](https://github.com/enkot/nuxt-open-fetch)
- [oapi-codegen](https://linuxcommandlibrary.com/man/oapi-codegen)
- [Gin vs Echo vs Fiber (Encore)](https://encore.dev/articles/gin-vs-echo-vs-fiber)
- [Choosing a Go Web Framework in 2026](https://medium.com/@samayun_pathan/choosing-a-go-web-framework-in-2026-a-minimalists-guide-to-gin-fiber-chi-echo-and-beego-c79b31b8474d)
- [Debian security tracker: labstack/echo (v5)](https://security-tracker.debian.org/tracker/source-package/golang-github-labstack-echo)
