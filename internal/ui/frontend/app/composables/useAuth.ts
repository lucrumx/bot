const AUTH_TOKEN_KEY = 'auth_token'

const authToken = ref('')
const authReady = ref(false)

export const useAuth = () => {
  const config = useRuntimeConfig()

  const isAuthenticated = computed(() => Boolean(authToken.value))

  const ensureAuthLoaded = (): void => {
    if (authReady.value || !import.meta.client) {
      return
    }

    authToken.value = localStorage.getItem(AUTH_TOKEN_KEY) ?? ''
    authReady.value = true
  }

  const setToken = (token: string): void => {
    authToken.value = token

    if (!import.meta.client) {
      return
    }

    if (token) {
      localStorage.setItem(AUTH_TOKEN_KEY, token)
      return
    }

    localStorage.removeItem(AUTH_TOKEN_KEY)
  }

  const login = async (email: string, password: string): Promise<void> => {
    const response = await $fetch<{ token: string }>(`${config.public.apiBase}/auth`, {
      method: 'POST',
      body: { email, password },
    })

    setToken(response.token)
  }

  const register = async (email: string, password: string): Promise<void> => {
    await $fetch(`${config.public.apiBase}/users`, {
      method: 'POST',
      body: { email, password },
    })
  }

  const logout = (): void => {
    setToken('')
  }

  return {
    authToken,
    isAuthenticated,
    ensureAuthLoaded,
    login,
    logout,
    register,
    setToken,
  }
}
