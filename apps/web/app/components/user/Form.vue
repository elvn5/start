<script setup lang="ts">
import { z } from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

const schema = z.object({
  email: z.email('Некорректный email'),
  name: z.string().trim().min(1, 'Введите имя').max(100),
})
type Schema = z.output<typeof schema>

const emit = defineEmits<{ submit: [value: Schema] }>()
defineProps<{ pending?: boolean }>()

const state = reactive<Partial<Schema>>({ email: '', name: '' })

function onSubmit(event: FormSubmitEvent<Schema>) {
  emit('submit', event.data)
  state.email = ''
  state.name = ''
}
</script>

<template>
  <UForm :schema="schema" :state="state" class="flex flex-col gap-3 sm:flex-row sm:items-start" @submit="onSubmit">
    <UFormField name="email" class="flex-1">
      <UInput v-model="state.email" placeholder="email@example.com" class="w-full" />
    </UFormField>
    <UFormField name="name" class="flex-1">
      <UInput v-model="state.name" placeholder="Имя" class="w-full" />
    </UFormField>
    <UButton type="submit" :loading="pending" icon="i-lucide-user-plus">
      Добавить
    </UButton>
  </UForm>
</template>
