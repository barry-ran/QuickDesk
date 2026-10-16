// Self-hosted update channel index (/v1/admin/update-releases).
//
// The server owns which version exists for each channel/platform/arch and what
// it hashes to; the artifact itself lives at the absolute download_url the
// operator hosts. Publishing the same tuple twice is a 409, so the UI shows the
// conflict instead of silently replacing an installed version's bytes.
import { authJson } from './auth.js'

const BASE = '/v1/admin/update-releases'

export function getUpdateReleases(params = {}) {
  const query = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') {
      query.set(key, String(value))
    }
  }
  const suffix = query.toString()
  return authJson(suffix ? `${BASE}?${suffix}` : BASE)
}

export function publishUpdateRelease(data) {
  return authJson(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export function deleteUpdateRelease(id) {
  return authJson(`${BASE}/${id}`, { method: 'DELETE' })
}
