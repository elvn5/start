<script setup lang="ts">
import type { User } from '~/composables/user/useUsers'

defineProps<{ users: User[] }>()

const dateFormat = new Intl.DateTimeFormat('ru-RU', { dateStyle: 'medium', timeStyle: 'short' })
</script>

<template>
  <p v-if="users.length === 0" class="text-muted text-sm">
    Пользователей пока нет — добавьте первого.
  </p>
  <ul v-else class="divide-y divide-default">
    <li v-for="user in users" :key="user.id" class="flex items-center justify-between gap-4 py-3">
      <div class="min-w-0">
        <p class="font-medium truncate">
          {{ user.name }}
        </p>
        <p class="text-muted text-sm truncate">
          {{ user.email }}
        </p>
      </div>
      <time class="text-muted text-xs shrink-0" :datetime="user.createdAt">
        {{ dateFormat.format(new Date(user.createdAt)) }}
      </time>
    </li>
  </ul>
</template>
