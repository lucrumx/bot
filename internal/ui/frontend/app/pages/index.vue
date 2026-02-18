<script setup lang="ts">
import { useAuth } from '~/composables/useAuth'

const { ensureAuthLoaded, isAuthenticated } = useAuth()

if (import.meta.client) {
  ensureAuthLoaded()
  await navigateTo(isAuthenticated.value ? '/spreads' : '/login', { replace: true })
}
</script>

<template>
  <div class="p-4">Redirecting...</div>
</template>
