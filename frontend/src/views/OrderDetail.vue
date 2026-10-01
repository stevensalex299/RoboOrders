<script setup>
import { computed, ref } from 'vue'
import { RouterLink, useRoute } from 'vue-router'
import OrderEventsList from '../components/OrderEventsList.vue'
import { usePolling } from '../composables/usePolling'
import { listQueryFromRoute } from '../composables/useListQuery'
import { formatName, formatOrderMeta, formatTotal, statusClass } from '../lib/format'

const route = useRoute()
const backToList = computed(() => ({ path: '/', query: listQueryFromRoute(route.query) }))
const order = ref(null)
const events = ref([])
const loading = ref(true)
const error = ref('')
const dispatchError = ref('')
const dispatching = ref(false)

const orderId = computed(() => route.params.id)

const canDispatch = computed(
  () => order.value && (order.value.status === 'received' || order.value.status === 'scheduled'),
)

async function loadDetail() {
  try {
    error.value = ''
    const [orderRes, eventsRes] = await Promise.all([
      fetch(`/orders/${orderId.value}`),
      fetch(`/orders/${orderId.value}/events`),
    ])
    if (!orderRes.ok) {
      const body = await orderRes.json().catch(() => ({}))
      throw new Error(body.error || orderRes.statusText)
    }
    order.value = await orderRes.json()
    if (eventsRes.ok) {
      const data = await eventsRes.json()
      events.value = data.events ?? []
    } else {
      events.value = []
    }
  } catch (e) {
    error.value = e.message || 'Failed to load order'
    order.value = null
  } finally {
    loading.value = false
  }
}

async function dispatch() {
  dispatchError.value = ''
  dispatching.value = true
  try {
    const res = await fetch(`/orders/${orderId.value}/dispatch`, { method: 'POST' })
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      throw new Error(body.error || res.statusText)
    }
    order.value = await res.json()
    await loadDetail()
  } catch (e) {
    dispatchError.value = e.message || 'Dispatch failed'
  } finally {
    dispatching.value = false
  }
}

usePolling(loadDetail)
</script>

<template>
  <div>
    <p class="mb-4">
      <RouterLink :to="backToList" class="text-sm text-sky-700 hover:underline">← Back to orders</RouterLink>
    </p>

    <p v-if="loading" class="rounded-lg border border-gray-200 bg-white px-4 py-8 text-center text-sm text-gray-500 shadow-sm">
      Loading order…
    </p>
    <div
      v-else-if="error"
      class="rounded-lg border border-red-200 bg-red-50 px-4 py-6 text-center text-sm text-red-700 shadow-sm"
      role="alert"
    >
      {{ error }}
    </div>

    <template v-else-if="order">
      <div class="mb-6 rounded-lg border border-gray-200 bg-white p-4 shadow-sm">
        <div class="flex flex-wrap items-start justify-between gap-4">
          <div>
            <h2 class="text-lg font-semibold">{{ formatName(order) }}</h2>
            <p class="mt-1 text-sm text-gray-600">
              {{ formatOrderMeta(order) }}
            </p>
          </div>
          <span :class="statusClass(order.status)">{{ order.status }}</span>
        </div>

        <dl class="mt-4 grid gap-3 text-sm sm:grid-cols-2">
          <div>
            <dt class="text-gray-500">Total</dt>
            <dd class="font-medium">{{ formatTotal(order) }}</dd>
          </div>
          <div v-if="order.scheduledFor">
            <dt class="text-gray-500">Scheduled for</dt>
            <dd class="font-medium">{{ order.scheduledFor }}</dd>
          </div>
          <div v-if="order.notes" class="sm:col-span-2">
            <dt class="text-gray-500">Notes</dt>
            <dd>{{ order.notes }}</dd>
          </div>
        </dl>

        <div v-if="order.lineItems?.length" class="mt-4">
          <h3 class="text-xs font-semibold uppercase tracking-wide text-gray-500">Line items</h3>
          <ul class="mt-2 list-inside list-disc text-sm text-gray-800">
            <li v-for="li in order.lineItems" :key="li.id">
              {{ li.itemName }}<span v-if="li.status" class="text-gray-500"> · {{ li.status.replaceAll('_', ' ') }}</span>
            </li>
          </ul>
        </div>

        <div v-if="canDispatch" class="mt-4 border-t border-gray-100 pt-4">
          <button
            type="button"
            class="rounded-md bg-violet-600 px-3 py-1.5 text-sm font-medium text-white hover:bg-violet-700 disabled:opacity-50"
            :disabled="dispatching"
            @click="dispatch"
          >
            {{ dispatching ? 'Dispatching…' : 'Dispatch to robot' }}
          </button>
          <p v-if="dispatchError" class="mt-2 text-sm text-red-600">{{ dispatchError }}</p>
        </div>
      </div>

      <OrderEventsList :events="events" />
      <p class="mt-3 text-xs text-gray-400">Refreshes every 5s</p>
    </template>
  </div>
</template>
