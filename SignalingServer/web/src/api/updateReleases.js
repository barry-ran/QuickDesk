// Self-hosted update channel index (/v1/admin/update-releases).
//
// The server owns which version exists for each channel/platform/arch and what
// it hashes to. The artifact normally lives at the absolute download_url the
// operator hosts; UploadUpdateReleaseArtifact is the alternative that lets this
// server hold the bytes and serve them itself. Publishing the same tuple twice is
// a 409, so the UI shows the conflict instead of silently replacing an installed
// version's bytes.
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

// Uploads the installer for one release. The server hashes the bytes and
// refuses them unless they match the digest the release already advertises, so
// a wrong file fails loudly instead of becoming the artifact clients download.
export function uploadUpdateReleaseArtifact(id, file) {
  const form = new FormData()
  form.append('file', file, file.name)
  return authJson(`${BASE}/${id}/artifact`, { method: 'POST', body: form })
}
