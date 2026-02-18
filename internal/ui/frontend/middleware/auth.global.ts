import { useAuth } from '~/composables/useAuth'

export default defineNuxtRouteMiddleware((to) => {
  if (!import.meta.client) {
    return
  }

  const { ensureAuthLoaded, isAuthenticated } = useAuth()
  ensureAuthLoaded()

  if (to.path === '/login') {
    return
  }

  if (!isAuthenticated.value) {
    return navigateTo('/login')
  }
})
