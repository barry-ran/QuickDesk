// /v1/admin/devices/* — admin device surface, plus secret rotation
// and forced unbind (§2.2 admin device routes).

import { authJson } from './auth.js'

const BASE = '/v1/admin'

// ----- List / detail -----------------------------------------------------

export function getDevices(params = {}) {
  const q = buildQuery(params)
  return authJson(`${BASE}/devices${q}`)
}

export function getDeviceDetail(deviceId) {
  return authJson(`${BASE}/devices/${encodeURIComponent(deviceId)}`)
}

// ----- Mutations (§2.2 admin device actions) -----------------------------

// §2.17: hard delete — pushes device.unbound to the owner.
export function deleteDevice(deviceId) {
  return authJson(`${BASE}/devices/${encodeURIComponent(deviceId)}`, {
    method: 'DELETE',
  })
}

// Force-unbind: keep the device row but clear user_id + logged_in_intent.
export function forceUnbindDevice(deviceId) {
  return authJson(`${BASE}/devices/${encodeURIComponent(deviceId)}/unbind`, {
    method: 'POST',
  })
}

// Rotate device_secret — host will 401 on next API call and re-provision.
export function rotateDeviceSecret(deviceId) {
  return authJson(`${BASE}/devices/${encodeURIComponent(deviceId)}/secret:rotate`, {
    method: 'POST',
  })
}

// §2.2: POST /v1/admin/devices:batch body = { ids:[...], op, group_id? }.
// Server contract (admin_devices_handler.go):
//   op ∈ {delete, assign_group, remove_group}
//   group_id required when op=assign_group|remove_group
export function batchDevices(op, ids, groupId) {
  const body = { ids, op }
  if (groupId !== undefined && groupId !== null) body.group_id = groupId
  return authJson(`${BASE}/devices:batch`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

// ----- Bindings ----------------------------------------------------------

export function getDeviceBindings(params = {}) {
  return authJson(`${BASE}/device-bindings${buildQuery(params)}`)
}

// ------------------------------------------------------------------------

function buildQuery(params) {
  const q = new URLSearchParams()
  // §3.1 cursor pagination uses ?cursor= and ?limit=. We accept both
  // (legacy `page/size` from existing views are silently dropped — the
  // server returns the first page when no cursor is supplied, which
  // covers the typical admin workflow on small/medium deployments).
  if (params.cursor) q.set('cursor', params.cursor)
  if (params.limit)  q.set('limit',  params.limit)
  if (params.size && !params.limit) q.set('limit', params.size)
  if (params.sort)  q.set('sort',  params.sort)
  if (params.order) q.set('order', params.order)
  if (params.search) q.set('search', params.search)
  if (params.os)     q.set('os',     params.os)
  if (params.online !== undefined && params.online !== '') q.set('online', params.online)
  const qs = q.toString()
  return qs ? `?${qs}` : ''
}
