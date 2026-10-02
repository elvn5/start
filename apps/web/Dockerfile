# Контекст сборки — корень репозитория (нужен api/openapi.yaml).

FROM node:26-alpine AS base
RUN corepack enable
WORKDIR /src

# ---------- deps ----------
FROM base AS deps
COPY pnpm-lock.yaml pnpm-workspace.yaml package.json ./
COPY apps/web/package.json apps/web/
RUN --mount=type=cache,target=/root/.local/share/pnpm/store \
    pnpm install --frozen-lockfile --filter web --ignore-scripts

# ---------- dev ----------
FROM deps AS dev
COPY api/ api/
COPY apps/web/ apps/web/
WORKDIR /src/apps/web
CMD ["sh", "-c", "pnpm exec nuxt prepare && pnpm exec nuxt dev --host 0.0.0.0"]

# ---------- build ----------
FROM deps AS build
COPY api/ api/
COPY apps/web/ apps/web/
ARG NUXT_API_PROXY_TARGET=http://gateway:8080
ENV NUXT_API_PROXY_TARGET=$NUXT_API_PROXY_TARGET
RUN pnpm --filter web build

# ---------- prod ----------
FROM node:26-alpine AS prod
WORKDIR /app
COPY --from=build /src/apps/web/.output ./.output
ENV NODE_ENV=production PORT=3000
EXPOSE 3000
USER node
CMD ["node", ".output/server/index.mjs"]
