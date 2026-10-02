# Один Dockerfile на все Go-сервисы. Контекст сборки — корень репозитория.
#   docker build -f backend/Dockerfile --target prod --build-arg SERVICE=identity .

# ---------- base ----------
FROM golang:1.26-alpine AS base
WORKDIR /src/backend
COPY backend/go.mod backend/go.sum ./
COPY backend/tools/go.mod backend/tools/go.sum ./tools/
RUN --mount=type=cache,target=/go/pkg/mod go mod download

# ---------- dev: hot reload сервиса $SERVICE (код монтируется volume) ----------
FROM base AS dev
RUN --mount=type=cache,target=/go/pkg/mod go build -modfile=tools/go.mod -o /usr/local/bin/air github.com/air-verse/air
CMD air --build.cmd "go build -o ./tmp/$SERVICE/app ./services/$SERVICE/cmd/$SERVICE" \
        --build.entrypoint "./tmp/$SERVICE/app" --tmp_dir "tmp/$SERVICE" \
        --build.include_ext go,sql --build.exclude_dir "tmp,tools,proto"

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
USER nonroot:nonroot
ENTRYPOINT ["/app"]
