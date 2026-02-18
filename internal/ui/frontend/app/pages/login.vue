<script setup lang="ts">
import { useAuth } from '~/composables/useAuth'
import { extractApiErrorMessage } from '~/utils/apiError'

definePageMeta({
  layout: 'auth',
})

const email = ref('')
const password = ref('')
const isSubmitting = ref(false)
const errorMessage = ref('')

const { login } = useAuth()

const submit = async (): Promise<void> => {
  isSubmitting.value = true
  errorMessage.value = ''

  try {
    await login(email.value, password.value)
    await navigateTo('/spreads')
  } catch (error) {
    errorMessage.value = extractApiErrorMessage(error, 'Не удалось войти')
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <section class="w-full">
    <div class="card border border-base-300 bg-white shadow-xl">
      <div class="card-body">
        <h1 class="text-2xl font-bold">Вход</h1>
        <p class="text-sm text-base-content/70">Войдите, чтобы открыть страницу со спредами.</p>

        <div v-if="errorMessage" class="alert alert-error text-sm">
          {{ errorMessage }}
        </div>

        <form class="space-y-3" @submit.prevent="submit">
          <label class="form-control w-full">
            <span class="label-text mb-1">Email</span>
            <input
              v-model="email"
              type="email"
              class="input input-bordered w-full"
              autocomplete="email"
              required
            />
          </label>

          <label class="form-control w-full">
            <span class="label-text mb-1">Пароль</span>
            <input
              v-model="password"
              type="password"
              class="input input-bordered w-full"
              autocomplete="current-password"
              minlength="8"
              required
            />
          </label>

          <button type="submit" class="btn btn-neutral w-full" :disabled="isSubmitting">
            {{ isSubmitting ? 'Входим...' : 'Войти' }}
          </button>
        </form>

      </div>
    </div>
  </section>
</template>
