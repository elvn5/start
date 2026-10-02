<script setup lang="ts">
import type { CreateUserInput } from '~/composables/user/useUsers'

useSeoMeta({ title: 'Пользователи' })

const toast = useToast()
const { data: users, error, refresh } = await useUserList()
const { create, pending } = useCreateUser()

async function onCreate(input: CreateUserInput) {
  try {
    const user = await create(input)
    toast.add({ title: `Пользователь ${user.name} создан`, color: 'success' })
  }
  catch (e) {
    toast.add({ title: apiErrorMessage(e, 'Не удалось создать пользователя'), color: 'error' })
  }
}
</script>

<template>
  <UContainer class="py-10 max-w-2xl">
    <header class="mb-6">
      <h1 class="text-2xl font-semibold">
        Пользователи
      </h1>
      <p class="text-muted text-sm">
        Nuxt → gateway (REST) → identity (Connect-RPC) → PostgreSQL
      </p>
    </header>

    <UCard class="mb-6">
      <UserForm :pending="pending" @submit="onCreate" />
    </UCard>

    <UCard>
      <UAlert
        v-if="error"
        color="error"
        variant="subtle"
        title="Не удалось загрузить пользователей"
        :description="apiErrorMessage(error, 'Gateway недоступен — запущен ли backend?')"
        :actions="[{ label: 'Повторить', onClick: () => refresh() }]"
      />
      <UserList v-else :users="users ?? []" />
    </UCard>
  </UContainer>
</template>
