<script setup lang="ts">
import { useAuth } from '~/composables/useAuth'
import { extractApiErrorMessage, extractApiStatusCode } from '~/utils/apiError'

interface Spread {
  id: string
  created_at: string
  updated_at: string
  symbol: string
  buy_on_exchange: string
  sell_on_exchange: string
  buy_price: string
  sell_price: string
  spread_percent: string
  max_spread_percent: string
  status: string
}

const config = useRuntimeConfig()
const { ensureAuthLoaded, authToken, logout } = useAuth()

const spreads = ref<Spread[]>([])
const isLoading = ref(false)
const errorMessage = ref('')

const statusFilter = ref('all')
const sellExchangeFilter = ref('all')
const buyExchangeFilter = ref('all')

const statusOptions = computed(() => {
  return [...new Set(spreads.value.map((spread) => spread.status))].sort()
})

const sellExchangeOptions = computed(() => {
  return [...new Set(spreads.value.map((spread) => spread.sell_on_exchange))].sort()
})

const buyExchangeOptions = computed(() => {
  return [...new Set(spreads.value.map((spread) => spread.buy_on_exchange))].sort()
})

const filteredSpreads = computed(() => {
  return spreads.value.filter((spread) => {
    if (statusFilter.value !== 'all' && spread.status !== statusFilter.value) {
      return false
    }

    if (
      sellExchangeFilter.value !== 'all' &&
      spread.sell_on_exchange !== sellExchangeFilter.value
    ) {
      return false
    }

    if (
      buyExchangeFilter.value !== 'all' &&
      spread.buy_on_exchange !== buyExchangeFilter.value
    ) {
      return false
    }

    return true
  })
})

const formatDate = (dateValue: string): string => {
  const date = new Date(dateValue)
  if (Number.isNaN(date.getTime())) {
    return dateValue
  }

  return new Intl.DateTimeFormat('ru-RU', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  }).format(date)
}

const formatNumber = (value: string): string => {
  const parsed = Number(value)
  if (Number.isNaN(parsed)) {
    return value
  }

  return parsed.toFixed(4)
}

const loadSpreads = async (): Promise<void> => {
  isLoading.value = true
  errorMessage.value = ''

  try {
    const headers: Record<string, string> = {}

    if (authToken.value) {
      headers.Authorization = `Bearer ${authToken.value}`
    }

    const response = await $fetch<Spread[]>(`${config.public.apiBase}/arbitrage-spreads`, {
      headers,
    })

    spreads.value = response ?? []
  } catch (error: unknown) {
    spreads.value = []

    const statusCode = extractApiStatusCode(error)
    if (statusCode === 401) {
      logout()
      await navigateTo('/login')
      return
    }

    errorMessage.value = extractApiErrorMessage(error, 'Не удалось загрузить спреды')
  } finally {
    isLoading.value = false
  }
}

onMounted(() => {
  ensureAuthLoaded()
  loadSpreads()
})
</script>

<template>
  <section class="space-y-4">
    <div class="card bg-base-100 shadow-sm">
      <div class="card-body gap-4">
        <div class="flex flex-col gap-2 sm:flex-row sm:items-center sm:justify-between">
          <h2 class="card-title">Список спредов</h2>
          <button class="btn btn-outline btn-sm" :disabled="isLoading" @click="loadSpreads">
            Обновить
          </button>
        </div>

        <div class="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          <label class="form-control w-full">
            <span class="label-text mb-1">Статус</span>
            <select v-model="statusFilter" class="select select-bordered w-full">
              <option value="all">Все</option>
              <option v-for="status in statusOptions" :key="status" :value="status">
                {{ status }}
              </option>
            </select>
          </label>

          <label class="form-control w-full">
            <span class="label-text mb-1">Биржа продажи</span>
            <select v-model="sellExchangeFilter" class="select select-bordered w-full">
              <option value="all">Все</option>
              <option v-for="exchange in sellExchangeOptions" :key="exchange" :value="exchange">
                {{ exchange }}
              </option>
            </select>
          </label>

          <label class="form-control w-full">
            <span class="label-text mb-1">Биржа покупки</span>
            <select v-model="buyExchangeFilter" class="select select-bordered w-full">
              <option value="all">Все</option>
              <option v-for="exchange in buyExchangeOptions" :key="exchange" :value="exchange">
                {{ exchange }}
              </option>
            </select>
          </label>
        </div>
      </div>
    </div>

    <div class="card bg-base-100 shadow-sm">
      <div class="card-body">
        <div v-if="errorMessage" role="alert" class="alert alert-error mb-4 text-sm">
          <span>{{ errorMessage }}</span>
        </div>

        <div class="overflow-x-auto">
          <table class="table table-zebra table-sm md:table-md">
            <thead>
              <tr>
                <th>Время</th>
                <th>Пара</th>
                <th>Статус</th>
                <th>Купить</th>
                <th>Цена покупки</th>
                <th>Продать</th>
                <th>Цена продажи</th>
                <th>Спред, %</th>
                <th>Макс. спред, %</th>
              </tr>
            </thead>
            <tbody>
              <tr v-if="isLoading">
                <td colspan="9" class="text-center">Загрузка...</td>
              </tr>
              <tr v-else-if="filteredSpreads.length === 0">
                <td colspan="9" class="text-center">Данных пока нет</td>
              </tr>
              <tr v-for="spread in filteredSpreads" :key="spread.id">
                <td>{{ formatDate(spread.created_at) }}</td>
                <td>{{ spread.symbol }}</td>
                <td>
                  <span class="badge badge-outline">{{ spread.status }}</span>
                </td>
                <td>{{ spread.buy_on_exchange }}</td>
                <td>{{ formatNumber(spread.buy_price) }}</td>
                <td>{{ spread.sell_on_exchange }}</td>
                <td>{{ formatNumber(spread.sell_price) }}</td>
                <td>{{ formatNumber(spread.spread_percent) }}</td>
                <td>{{ formatNumber(spread.max_spread_percent) }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </div>
  </section>
</template>
