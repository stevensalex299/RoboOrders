<script setup>
import { computed, ref, watch } from 'vue'
import { RouterLink, useRoute, useRouter } from 'vue-router'
import { usePolling } from '../composables/usePolling'
import {
  listQueryFromRoute,
  readListStateFromQuery,
  writeListStateToQuery,
} from '../composables/useListQuery'
import EmptyState from '../components/EmptyState.vue'
import { formatName, formatTotal, statusClass, ORDER_SOURCES, ORDER_STATUSES } from '../lib/format'

const route = useRoute()
const router = useRouter()

const orders = ref([])
const statusFilter = ref('')
const sourceFilter = ref('')
const loading = ref(true)
const error = ref('')
const page = ref(1)
const pageSize = 10

function applyRouteQuery() {
  const s = readListStateFromQuery(route.query)
  sourceFilter.value = s.sourceFilter
  statusFilter.value = s.statusFilter
  page.value = s.page
}

function pushRouteQuery() {
  const q = writeListStateToQuery({
    sourceFilter: sourceFilter.value,
    statusFilter: statusFilter.value,
    page: page.value,
  })
  if (JSON.stringify(q) !== JSON.stringify(listQueryFromRoute(route.query))) {
    router.replace({ path: '/', query: q })
  }
}

applyRouteQuery()

const statusLabel = computed(() =>
  statusFilter.value === '' ? 'All statuses' : statusFilter.value,
)

const sourceLabel = computed(() => {
  const match = ORDER_SOURCES.find((s) => s.value === sourceFilter.value)
  return match?.label ?? 'All sources'
})

const totalPages = computed(() => Math.max(1, Math.ceil(orders.value.length / pageSize)))

const pagedOrders = computed(() => {
  const start = (page.value - 1) * pageSize
  return orders.value.slice(start, start + pageSize)
})

watch(totalPages, (n) => {
  if (page.value > n) page.value = n
})

function onFiltersChange() {
  page.value = 1
  pushRouteQuery()
  loadOrders()
}

watch(page, () => {
  pushRouteQuery()
})

watch(
  () => route.query,
  () => {
    if (route.path !== '/') return
    applyRouteQuery()
  },
)

async function loadOrders() {
  try {
    error.value = ''
    const params = new URLSearchParams()
    if (statusFilter.value) params.set('status', statusFilter.value)
    if (sourceFilter.value) params.set('source', sourceFilter.value)
    const qs = params.toString()
    const res = await fetch(`/orders${qs ? `?${qs}` : ''}`)
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
    <div class="mb-4 flex flex-wrap justify-end gap-4">
      <label class="flex items-center gap-2 text-sm text-gray-600">
        Source
        <select
          v-model="sourceFilter"
          class="rounded-md border border-gray-300 bg-white px-2 py-1 text-gray-900"
          @change="onFiltersChange"
        >
          <option v-for="s in ORDER_SOURCES" :key="s.value || 'all'" :value="s.value">
            {{ s.label }}
          </option>
        </select>
      </label>
      <label class="flex items-center gap-2 text-sm text-gray-600">
        Status
        <select
          v-model="statusFilter"
          class="rounded-md border border-gray-300 bg-white px-2 py-1 text-gray-900"
          @change="onFiltersChange"
        >
          <option v-for="s in ORDER_STATUSES" :key="s || 'all'" :value="s">
            {{ s === '' ? 'All' : s }}
          </option>
        </select>
      </label>
    </div>

    <p v-if="loading" class="rounded-lg border border-gray-200 bg-white px-4 py-8 text-center text-sm text-gray-500 shadow-sm">
      Loading orders…
    </p>
    <div
      v-else-if="error"
      class="rounded-lg border border-red-200 bg-red-50 px-4 py-6 text-center text-sm text-red-700 shadow-sm"
      role="alert"
    >
      {{ error }}
    </div>
    <EmptyState v-else-if="orders.length === 0 && !statusFilter && !sourceFilter" title="No orders yet">
      From
      <code class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs text-gray-800">backend/</code>, run
      <code class="mt-1 block rounded bg-gray-100 px-2 py-1.5 font-mono text-xs text-gray-800">
        go run ./cmd/ingest webhook|poll|csv --file …
      </code>
      See
      <code class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs text-gray-800">docs/runbook.md</code>.
      Webhooks can also POST to
      <code class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs text-gray-800">/webhooks/orders</code>.
    </EmptyState>
    <EmptyState v-else-if="orders.length === 0" title="No matching orders">
      Nothing matches
      <span class="font-medium text-gray-800">{{ sourceLabel }} · {{ statusLabel }}</span>.
      Try clearing filters or ingest more fixture data via the CLI —
      <code class="rounded bg-gray-100 px-1.5 py-0.5 font-mono text-xs text-gray-800">docs/runbook.md</code>.
    </EmptyState>

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
          <tr v-for="o in pagedOrders" :key="o.id" class="hover:bg-gray-50">
            <td class="px-4 py-3 font-medium">
              <RouterLink
                :to="{ path: `/orders/${o.id}`, query: route.query }"
                class="text-sky-700 hover:underline"
              >
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
      <div
        v-if="totalPages > 1"
        class="flex flex-wrap items-center justify-between gap-3 border-t border-gray-100 px-4 py-3 text-sm text-gray-600"
      >
        <span>Page {{ page }} of {{ totalPages }} · {{ orders.length }} orders</span>
        <div class="flex gap-2">
          <button
            type="button"
            class="rounded-md border border-gray-300 bg-white px-3 py-1 text-gray-800 hover:bg-gray-50 disabled:opacity-40"
            :disabled="page <= 1"
            @click="page--"
          >
            Previous
          </button>
          <button
            type="button"
            class="rounded-md border border-gray-300 bg-white px-3 py-1 text-gray-800 hover:bg-gray-50 disabled:opacity-40"
            :disabled="page >= totalPages"
            @click="page++"
          >
            Next
          </button>
        </div>
      </div>
    </div>
    <p class="mt-3 text-xs text-gray-400">
      {{ sourceLabel }} · {{ statusLabel }} · {{ pageSize }} per page · refreshes every 5s
    </p>
  </div>
</template>
