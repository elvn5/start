import { fileURLToPath } from 'node:url'

// Адрес gateway для SSR-прокси. Читается при сборке: локально — localhost, в Docker — http://gateway:8080.
const gatewayURL = process.env.NUXT_API_PROXY_TARGET ?? 'http://localhost:8080'

export default defineNuxtConfig({
  compatibilityDate: '2026-10-01',
  modules: ['@nuxt/ui', '@nuxt/eslint', '@pinia/nuxt', 'nuxt-open-fetch', '@nuxt/test-utils/module'],
  devtools: { enabled: true },
  css: ['~/assets/css/main.css'],
  typescript: { strict: true },

  // Composables лежат в подпапках по фичам: composables/user/useUsers.ts
  imports: { dirs: ['composables/**'] },

  // Типизированный клиент useApi/$api из публичного контракта gateway
  openFetch: {
    clients: {
      api: {
        baseURL: '/api',
        schema: fileURLToPath(new URL('../../api/openapi.yaml', import.meta.url)),
      },
    },
  },

  // Nuxt Icon по умолчанию слушает /api/_nuxt_icon — уводим с /api, который целиком проксируется в gateway
  icon: { localApiEndpoint: '/_nuxt_icon' },

  routeRules: {
    '/api/**': { proxy: `${gatewayURL}/api/**` },
  },
})
