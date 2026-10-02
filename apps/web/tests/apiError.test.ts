import { describe, expect, it } from 'vitest'
import { apiErrorMessage } from '~/utils/apiError'

describe('apiErrorMessage', () => {
  it('returns message from gateway error body', () => {
    expect(apiErrorMessage({ data: { message: 'email already taken' } })).toBe('email already taken')
  })

  it('falls back for unknown errors', () => {
    expect(apiErrorMessage(new Error('boom'), 'fallback')).toBe('fallback')
    expect(apiErrorMessage(undefined)).toBe('Что-то пошло не так')
  })
})
