import { describe, expect, it } from 'vitest'
import { mountSuspended } from '@nuxt/test-utils/runtime'
import UserList from '~/components/user/List.vue'

describe('UserList', () => {
  it('renders users', async () => {
    const wrapper = await mountSuspended(UserList, {
      props: { users: [{ id: 1, email: 'ann@example.com', name: 'Ann', createdAt: '2026-10-01T10:00:00Z' }] },
    })
    expect(wrapper.text()).toContain('Ann')
    expect(wrapper.text()).toContain('ann@example.com')
  })

  it('renders empty state', async () => {
    const wrapper = await mountSuspended(UserList, { props: { users: [] } })
    expect(wrapper.text()).toContain('Пользователей пока нет')
  })
})
