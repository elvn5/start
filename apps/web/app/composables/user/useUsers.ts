import type { ApiRequestBody, ApiResponse } from '#open-fetch'

export type User = ApiResponse<'getUser'>
export type CreateUserInput = ApiRequestBody<'createUser'>

const USERS_KEY = 'users:list'

/** Список пользователей (SSR-friendly, кешируется по ключу). */
export function useUserList() {
  return useApi('/users', {
    key: USERS_KEY,
    query: { limit: 50 },
  })
}

/** Создание пользователя с инвалидацией списка. */
export function useCreateUser() {
  const { $api } = useNuxtApp()
  const pending = ref(false)

  async function create(body: CreateUserInput): Promise<User> {
    pending.value = true
    try {
      const user = await $api('/users', { method: 'POST', body })
      await refreshNuxtData(USERS_KEY)
      return user
    }
    finally {
      pending.value = false
    }
  }

  return { create, pending }
}
