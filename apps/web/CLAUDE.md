# Web (Nuxt 4)

Talks ONLY to the gateway via the typed client generated from `api/openapi.yaml` (nuxt-open-fetch: `useApi`, `$api`). It knows nothing about internal services.

## Structure
- `app/pages/`                 — thin: route params + composables + components
- `app/components/<feature>/`  — UI only, props down / events up, never call the API (`components/user/List.vue` → `<UserList>`)
- `app/composables/<feature>/` — the ONLY place that calls `useApi` / `$api`; owns cache keys and invalidation (`refreshNuxtData`)
- `app/stores/`                — Pinia, client-only state (session, UI prefs). Never copy API responses into stores
- `app/utils/`                 — pure helpers

## Rules
- Types come from the OpenAPI schema: `ApiResponse<'getUser'>`, `ApiRequestBody<'createUser'>` from `#open-fetch`. No hand-written API types
- `<script setup lang="ts">`, Composition API only, no `any`
- Nuxt UI components first (`UButton`, `UForm`, `UInput`…); forms validated with Zod
- After changing `api/openapi.yaml`: `task gen` (regenerates the client types)
- Checks: `pnpm --filter web lint`, `pnpm --filter web typecheck`, `pnpm --filter web test`
