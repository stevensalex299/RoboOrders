export function formatName(o) {
  const parts = [o.firstName, o.lastName].filter(Boolean)
  return parts.length ? parts.join(' ') : '—'
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
