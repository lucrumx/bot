<script setup lang="ts">
import { useAuth } from '~/composables/useAuth'

const route = useRoute()
const { ensureAuthLoaded, isAuthenticated, logout } = useAuth()

onMounted(() => {
  ensureAuthLoaded()
})

const handleLogout = async (): Promise<void> => {
  logout()
  await navigateTo('/login')
}
</script>

<template>
  <header class="navbar border-b border-base-300 bg-base-100 px-4 sm:px-6 lg:px-8">
    <div class="mx-auto flex w-full max-w-7xl items-center justify-between">
      <h1 class="text-xl font-bold">Arbitrage Spreads</h1>
      <button
        v-if="isAuthenticated && route.path !== '/login' && route.path !== '/register'"
        class="btn btn-sm btn-outline"
        @click="handleLogout"
      >
        Выйти
      </button>
    </div>
  </header>
</template>
