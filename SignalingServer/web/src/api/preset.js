// /v1/admin/preset (§2.2). Returns a JSON object — not a list.
import { authJson } from './auth.js'

const BASE = '/v1/admin/preset'

export function getPreset() { return authJson(BASE) }

export function updatePreset(data) {
  return authJson(BASE, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}
