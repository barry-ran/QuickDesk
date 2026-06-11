// /v1/admin/stats|system/status|connections|trends|activity (§2.2).
import { authJson } from './auth.js'

const BASE = '/v1/admin'

export function getStats()          { return authJson(`${BASE}/stats`) }
export function getSystemStatus()   { return authJson(`${BASE}/system/status`) }
export function getConnectionStatus(){ return authJson(`${BASE}/connections`) }

export function getTrends(range = '24h') {
  return authJson(`${BASE}/trends?range=${encodeURIComponent(range)}`)
}

export function getActivity(params = {}) {
  // §2.2: GET /v1/admin/activity only consumes cursor/limit today
  // (admin_stats_handler.go:95). Extra filter params (deviceId, status,
  // date_from, date_to) are silently ignored server-side — don't send them
  // here so we don't mislead callers about what actually filters.
  const q = new URLSearchParams()
  if (params.cursor) q.set('cursor', params.cursor)
  if (params.limit)  q.set('limit',  params.limit)
  if (params.size && !params.limit) q.set('limit', params.size)
  const qs = q.toString()
  return authJson(`${BASE}/activity${qs ? `?${qs}` : ''}`)
}
