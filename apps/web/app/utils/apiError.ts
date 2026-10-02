/** Достаёт сообщение из ошибки gateway ({ message }) или возвращает запасной текст. */
export function apiErrorMessage(error: unknown, fallback = 'Что-то пошло не так'): string {
  if (error && typeof error === 'object' && 'data' in error) {
    const data = (error as { data?: { message?: unknown } }).data
    if (typeof data?.message === 'string') return data.message
  }
  return fallback
}
