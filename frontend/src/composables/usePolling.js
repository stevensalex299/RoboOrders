import { onMounted, onUnmounted } from 'vue'

export function usePolling(loadFn, intervalMs = 5000) {
  let timer
  onMounted(() => {
    loadFn()
    timer = setInterval(loadFn, intervalMs)
  })
  onUnmounted(() => {
    if (timer) clearInterval(timer)
  })
}
