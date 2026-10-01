<script setup>
import { computed, ref } from 'vue'
import { RouterLink } from 'vue-router'
import { usePolling } from '../composables/usePolling'
import { formatName, formatTotal, statusClass, ORDER_STATUSES } from '../lib/format'

const orders = ref([])
const statusFilter = ref('')
const loading = ref(true)
const error = ref('')

const statusLabel = computed(() =>
  statusFilter.value === '' ? 'All statuses' : statusFilter.value,
)

async function loadOrders() {
  try {
    error.value = ''
    const q = statusFilter.value ? `?status=${encodeURIComponent(statusFilter.value)}` : ''
    const res = await fetch(`/orders${q}`)
    if (!res.ok) {
      const body = await res.json().catch(() => ({}))
      throw new Error(body.error || res.statusText)
    }
    const data = await res.json()
    orders.value = data.orders ?? []
  } catch (e) {
    error.value = e.message || 'Failed to load orders'
  } finally {
    loading.value = false
  }
}

usePolling(loadOrders)
</script>

<template>
  <div>
    <div class="mb-4 flex justify-end">
      <label class="flex items-center gap-2 text-sm text-gray-600">
        Status
        <select
          v-model="statusFilter"
          class="rounded-md border border-gray-300 bg-white px-2 py-1 text-gray-900"
          @change="loadOrders"
        >
          <option v-for="s in ORDER_STATUSES" :key="s || 'all'" :value="s">
            {{ s === '' ? 'All' : s }}
          </option>
        </select>
      </label>
    </div>

    <p v-if="loading" class="text-sm text-gray-500">Loading orders…</p>
    <p v-else-if="error" class="text-sm text-red-600">{{ error }}</p>
    <p v-else-if="orders.length === 0 && statusFilter === ''" class="text-sm text-gray-500">
      No orders yet. Ingest webhook fixtures with the backend CLI or POST JSON to
      <code class="text-xs">/webhooks/orders</code>. See
      <code class="text-xs">docs/runbook.md</code>.
    </p>
    <p v-else-if="orders.length === 0" class="text-sm text-gray-500">
      No {{ statusFilter }} orders. Try another filter or ingest more data (CLI or
      <code class="text-xs">/webhooks/orders</code>; see
      <code class="text-xs">docs/runbook.md</code>).
    </p>

    <div v-else class="overflow-hidden rounded-lg border border-gray-200 bg-white shadow-sm">
      <table class="min-w-full divide-y divide-gray-200 text-left text-sm">
        <thead class="bg-gray-50 text-xs uppercase tracking-wide text-gray-500">
          <tr>
            <th class="px-4 py-3">Customer</th>
            <th class="px-4 py-3">Source</th>
            <th class="px-4 py-3">Status</th>
            <th class="px-4 py-3">Items</th>
            <th class="px-4 py-3">Total</th>
            <th class="px-4 py-3">Restaurant</th>
          </tr>
        </thead>
        <tbody class="divide-y divide-gray-100">
          <tr v-for="o in orders" :key="o.id" class="hover:bg-gray-50">
            <td class="px-4 py-3 font-medium">
              <RouterLink :to="`/orders/${o.id}`" class="text-sky-700 hover:underline">
                {{ formatName(o) }}
              </RouterLink>
            </td>
            <td class="px-4 py-3 capitalize text-gray-600">{{ o.source }}</td>
            <td class="px-4 py-3">
              <span :class="statusClass(o.status)">{{ o.status }}</span>
            </td>
            <td class="px-4 py-3 text-gray-600">{{ o.lineItems?.length ?? 0 }}</td>
            <td class="px-4 py-3">{{ formatTotal(o) }}</td>
            <td class="px-4 py-3 text-gray-600">{{ o.restaurant || '—' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <p class="mt-3 text-xs text-gray-400">Showing {{ statusLabel }} · refreshes every 5s</p>
  </div>
</template>
