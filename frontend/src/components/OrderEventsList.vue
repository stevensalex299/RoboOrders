<script setup>
import EmptyState from './EmptyState.vue'

defineProps({
  events: {
    type: Array,
    default: () => [],
  },
})

function formatPayload(payload) {
  if (!payload) return ''
  try {
    return JSON.stringify(payload, null, 2)
  } catch {
    return String(payload)
  }
}
</script>

<template>
  <section class="rounded-lg border border-gray-200 bg-white shadow-sm">
    <h2 class="border-b border-gray-100 px-4 py-3 text-sm font-semibold text-gray-800">Event history</h2>
    <EmptyState v-if="events.length === 0" inset title="No events yet">
      Events appear here when orders are created, updated, dispatched, or cancelled.
    </EmptyState>
    <ul v-else class="max-h-96 divide-y divide-gray-100 overflow-y-auto">
      <li v-for="e in events" :key="e.id" class="px-4 py-3 text-sm">
        <div class="flex flex-wrap items-baseline justify-between gap-2">
          <span class="font-medium capitalize text-gray-900">{{ e.type.replaceAll('_', ' ') }}</span>
          <time class="text-xs text-gray-500">{{ e.occurredAt }}</time>
        </div>
        <pre
          v-if="e.payload"
          class="mt-2 max-h-40 overflow-auto rounded bg-gray-50 p-2 text-xs text-gray-700"
          >{{ formatPayload(e.payload) }}</pre
        >
      </li>
    </ul>
  </section>
</template>
