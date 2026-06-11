// /v1/admin/audit-logs (§2.2).
import { authJson } from './auth.js'

const BASE = '/v1/admin/audit-logs'

export function getAuditLogs(params = {}) {
  const q = new URLSearchParams()
  if (params.cursor) q.set('cursor', params.cursor)
  if (params.limit)  q.set('limit',  params.limit)
  if (params.size && !params.limit) q.set('limit', params.size)
  if (params.action)   q.set('action', params.action)
  if (params.admin)    q.set('admin',  params.admin)
  // Server reads snake_case date_from / date_to.
  if (params.dateFrom) q.set('date_from', params.dateFrom)
  if (params.dateTo)   q.set('date_to',   params.dateTo)
  const qs = q.toString()
  return authJson(`${BASE}${qs ? `?${qs}` : ''}`)
}
