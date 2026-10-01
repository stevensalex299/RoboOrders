export function formatName(o) {
  const parts = [o.firstName, o.lastName].filter(Boolean)
  if (parts.length) return parts.join(' ')
  if (o.source === 'external_poll' && o.sourceId) return `Order ${o.sourceId}`
  return '—'
}

export function formatOrderMeta(o) {
  const sourceLabels = {
    webhook: 'Webhook',
    external_poll: 'External poll',
    csv: 'CSV survey',
  }
  const parts = [sourceLabels[o.source] || o.source || 'Order']
  if (o.source === 'external_poll' && o.sourceId) {
    parts.push(`Order #${o.sourceId}`)
  }
  if (o.orderPlatform) parts.push(o.orderPlatform)
  if (o.restaurant) parts.push(o.restaurant)
  return parts.join(' · ')
}

export function formatTotal(o) {
  if (o.total == null) return '—'
  return `$${Number(o.total).toFixed(2)}`
}

export function statusClass(status) {
  const base = 'inline-block rounded-full px-2 py-0.5 text-xs font-medium capitalize'
  const map = {
    received: 'bg-sky-100 text-sky-800',
    scheduled: 'bg-amber-100 text-amber-900',
    dispatched: 'bg-violet-100 text-violet-900',
    cancelled: 'bg-gray-200 text-gray-700',
  }
  return `${base} ${map[status] || 'bg-gray-100 text-gray-800'}`
}

export const ORDER_STATUSES = ['', 'received', 'scheduled', 'dispatched', 'cancelled']

export const ORDER_SOURCES = [
  { value: '', label: 'All sources' },
  { value: 'webhook', label: 'Webhook' },
  { value: 'external_poll', label: 'External poll' },
  { value: 'csv', label: 'CSV' },
]
