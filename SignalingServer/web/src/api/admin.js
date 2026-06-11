// /v1/admin/admins/* — admin account CRUD plus 2FA self-service.
//
// 2FA endpoints use sub-resource style (not colon-action) because
// gin/httprouter treats `2fa:setup` and `2fa:verify` as competing
// wildcards on the same path segment — see §6 W1/W4 in design doc:
//   POST   /v1/admin/admins/me/2fa/setup
//   POST   /v1/admin/admins/me/2fa/verify
//   DELETE /v1/admin/admins/me/2fa
//
// List/details follow the standard cursor envelope {items, next_cursor}.

import { authJson } from './auth.js'

const BASE = '/v1/admin'

// ----- Admin accounts (super_admin only writes; reads OK for any admin) --

export function getAdminUsers() {
  return authJson(`${BASE}/admins`)
}

export function getAdminUser(id) {
  return authJson(`${BASE}/admins/${id}`)
}

export function createAdminUser(payload) {
  return authJson(`${BASE}/admins`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

// PATCH per §2.2 (replaces the legacy PUT-with-full-body).
export function updateAdminUser(id, payload) {
  return authJson(`${BASE}/admins/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

export function deleteAdminUser(id) {
  return authJson(`${BASE}/admins/${id}`, { method: 'DELETE' })
}

// ----- Self-service 2FA --------------------------------------------------

export function setup2FA() {
  return authJson(`${BASE}/admins/me/2fa/setup`, { method: 'POST' })
}

export function verify2FA(code) {
  return authJson(`${BASE}/admins/me/2fa/verify`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ code }),
  })
}

export function disable2FA(code) {
  return authJson(`${BASE}/admins/me/2fa`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ code }),
  })
}
