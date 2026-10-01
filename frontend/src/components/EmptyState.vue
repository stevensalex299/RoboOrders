<script setup>
defineProps({
  title: {
    type: String,
    required: true,
  },
  /** When true, omit outer card chrome (e.g. inside another bordered panel). */
  inset: {
    type: Boolean,
    default: false,
  },
})
</script>

<template>
  <div
    :class="
      inset
        ? 'px-4 py-10 text-center'
        : 'rounded-lg border border-dashed border-gray-200 bg-white px-6 py-12 text-center shadow-sm'
    "
  >
    <div
      class="mx-auto flex h-11 w-11 items-center justify-center rounded-full bg-gray-100 text-gray-400"
      aria-hidden="true"
    >
      <svg class="h-6 w-6" fill="none" viewBox="0 0 24 24" stroke-width="1.5" stroke="currentColor">
        <path
          stroke-linecap="round"
          stroke-linejoin="round"
          d="M20.25 7.5l-.625 10.632a2.25 2.25 0 01-2.247 2.118H6.622a2.25 2.25 0 01-2.247-2.118L3.75 7.5M10 11.25h4M3.375 7.5h17.25c.621 0 1.125-.504 1.125-1.125v-1.5c0-.621-.504-1.125-1.125-1.125H3.375c-.621 0-1.125.504-1.125 1.125v1.5c0 .621.504 1.125 1.125 1.125z"
        />
      </svg>
    </div>
    <h3 class="mt-4 text-sm font-semibold text-gray-900">{{ title }}</h3>
    <div class="mx-auto mt-2 max-w-md text-sm leading-relaxed text-gray-600">
      <slot />
    </div>
  </div>
</template>
