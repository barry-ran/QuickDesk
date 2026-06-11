// /v1/admin/users/* — business user CRUD (§2.2).
//
// Notes vs. the legacy /api/v1/admin/user-list path used pre-refactor:
//   • Path moved from /admin/user-list → /admin/users (§2.2)
//   • PATCH replaces PUT for partial updates
//   • New: POST /admin/users/:id/sessions:revoke kicks every active
//     session for a user (§2.17)
//   • Batch endpoint moved to /admin/users:batch (colon-action style)

import { authJson } from './auth.js'

const BASE = '/v1/admin/users'

export function getUsers(params = {}) {
  return authJson(`${BASE}${buildQuery(params)}`)
}

export function getUser(id)         { return authJson(`${BASE}/${id}`) }
export function getUserDetail(id)   { return authJson(`${BASE}/${id}/details`) }

export function createUser(data) {
  return authJson(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export function updateUser(id, data) {
  return authJson(`${BASE}/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export function deleteUser(id) {
  return authJson(`${BASE}/${id}`, { method: 'DELETE' })
}

// §2.2: POST /v1/admin/users:batch body = { ids:[...], op, level? }.
// Server contract (admin_users_handler.go):
//   op ∈ {enable, disable, delete, set_level}
//   level required when op=set_level
export function batchUsers(op, ids, level) {
  const body = { ids, op }
  if (level !== undefined && level !== null && level !== '') body.level = level
  return authJson(`${BASE}:batch`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })
}

// §2.17: kick a user off every active session and emit session.revoked.
export function revokeUserSessions(id) {
  return authJson(`${BASE}/${id}/sessions:revoke`, { method: 'POST' })
}

// PATCH /v1/admin/users/:id/device-count body = { device_count: N }.
// Server's adminPatchUserReq uses snake_case `device_count`
// (admin_users_handler.go — unified snake_case per architect review).
export function updateUserDeviceCount(id, deviceCount) {
  return authJson(`${BASE}/${id}/device-count`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ device_count: deviceCount }),
  })
}

// ------------------------------------------------------------------------

function buildQuery(params) {
  const q = new URLSearchParams()
  if (params.cursor) q.set('cursor', params.cursor)
  if (params.limit)  q.set('limit',  params.limit)
  if (params.size && !params.limit) q.set('limit', params.size)
  if (params.sort)  q.set('sort',  params.sort)
  if (params.order) q.set('order', params.order)
  if (params.search) q.set('search', params.search)
  if (params.level) q.set('level', params.level)
  if (params.status !== undefined && params.status !== '') q.set('status', params.status)
  // Server reads `channel_type` (snake_case, unified per architect review).
  // Callers may pass either snake_case (preferred) or the legacy camelCase
  // key; forward whichever is non-empty.
  const channelType = params.channel_type ?? params.channelType
  if (channelType) q.set('channel_type', channelType)
  const qs = q.toString()
  return qs ? `?${qs}` : ''
}
