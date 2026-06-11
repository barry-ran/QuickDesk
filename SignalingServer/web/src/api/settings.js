// /v1/settings/public (no auth) and /v1/admin/settings (admin auth).
//
// §2.2: write uses PUT (was POST in the pre-refactor admin web).

import { authJson, authFetch } from './auth.js'

// Public — read-only, anyone can hit it.
export async function getSettings() {
  const res = await fetch('/v1/settings/public')
  if (!res.ok) throw new Error(`HTTP ${res.status}`)
  return res.json()
}

// Admin read.
export function getAdminSettings() {
  return authJson('/v1/admin/settings')
}

export function updateSettings(data) {
  return authJson('/v1/admin/settings', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}
