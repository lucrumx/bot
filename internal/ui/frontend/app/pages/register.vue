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

const { register, login } = useAuth()

const submit = async (): Promise<void> => {
  isSubmitting.value = true
  errorMessage.value = ''

  try {
    await register(email.value, password.value)
    await login(email.value, password.value)
    await navigateTo('/spreads')
  } catch (error) {
    errorMessage.value = extractApiErrorMessage(error, 'Не удалось зарегистрироваться')
  } finally {
    isSubmitting.value = false
  }
}
</script>

<template>
  <section class="w-full">
    <div class="card border border-base-300 bg-white shadow-xl">
      <div class="card-body">
        <h1 class="text-2xl font-bold">Регистрация</h1>
        <p class="text-sm text-base-content/70">Создайте аккаунт для доступа к спредам.</p>

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
              autocomplete="new-password"
              minlength="8"
              required
            />
          </label>

          <button type="submit" class="btn btn-neutral w-full" :disabled="isSubmitting">
            {{ isSubmitting ? 'Создаем...' : 'Создать аккаунт' }}
          </button>
        </form>

        <p class="text-center text-sm">
          Уже есть аккаунт?
          <NuxtLink to="/login" class="link link-primary">Войти</NuxtLink>
        </p>
      </div>
    </div>
  </section>
</template>
