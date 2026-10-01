const LIST_QUERY_KEYS = ['source', 'status', 'page']

export function listQueryFromRoute(query) {
  const out = {}
  if (query.source) out.source = String(query.source)
  if (query.status) out.status = String(query.status)
  if (query.page) out.page = String(query.page)
  return out
}

export function readListStateFromQuery(query) {
  return {
    sourceFilter: query.source ? String(query.source) : '',
    statusFilter: query.status ? String(query.status) : '',
    page: Math.max(1, parseInt(String(query.page || '1'), 10) || 1),
  }
}

export function writeListStateToQuery({ sourceFilter, statusFilter, page }) {
  const q = {}
  if (sourceFilter) q.source = sourceFilter
  if (statusFilter) q.status = statusFilter
  if (page > 1) q.page = String(page)
  return q
}

export { LIST_QUERY_KEYS }
