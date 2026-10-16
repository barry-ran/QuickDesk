// Self-hosted update channel index (/v1/admin/update-releases).
//
// The server owns which version exists for each channel/platform/arch and what
// it hashes to. The artifact normally lives at the absolute download_url the
// operator hosts; UploadUpdateReleaseArtifact is the alternative that lets this
// server hold the bytes and serve them itself. Publishing the same tuple twice is
// a 409, so the UI shows the conflict instead of silently replacing an installed
// version's bytes.
import { authFetch, authJson, problemToError } from './auth.js'

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

// One chunk of a resumable upload. Returns either the finished release row or
// {resync}, which is how a 409 tells the caller where the server actually
// stands (the offset travels in a response header, so authJson - which throws
// the response away - is not usable here).
async function patchUpdateArtifactChunk(id, filename, chunk, { offset, length, complete }) {
  const headers = {
    'Content-Type': 'application/octet-stream',
    'Upload-Offset': String(offset),
    'Upload-Length': String(length),
  }
  if (complete) headers['Upload-Complete'] = '1'
  const res = await authFetch(
    `${BASE}/${id}/artifact?filename=${encodeURIComponent(filename)}`,
    { method: 'PATCH', headers, body: chunk },
  )
  const serverOffset = Number(res.headers.get('Upload-Offset'))
  if (res.status === 409) {
    return { resync: Number.isFinite(serverOffset) ? serverOffset : offset, res }
  }
  if (!res.ok) {
    const problem = await res.json().catch(() => null)
    return { error: problemToError(res, problem), res }
  }
  return { row: await res.json().catch(() => null), res }
}

const RESUME_CHUNK_BYTES = 8 * 1024 * 1024
const RESUME_MAX_RESYNCS = 4

// Uploads an installer in chunks so a dropped connection costs one chunk
// instead of the whole file. The server is the authority on how much it already
// holds: on a 409 this reads its Upload-Offset and continues from there, which
// also recovers the case where the response to a chunk was lost after the
// server had already stored it.
export async function uploadUpdateReleaseArtifactResumable(id, file, { onProgress } = {}) {
  let offset = 0
  let resyncs = 0
  for (;;) {
    const end = Math.min(offset + RESUME_CHUNK_BYTES, file.size)
    const complete = end >= file.size
    const outcome = await patchUpdateArtifactChunk(id, file.name, file.slice(offset, end), {
      offset,
      length: file.size,
      complete,
    })
    if (outcome.error) throw outcome.error
    if (outcome.resync !== undefined) {
      // Going backwards is fine and is the point: it re-sends what the server
      // is missing. Going nowhere twice in a row would spin, so give up.
      if (outcome.resync === offset || resyncs >= RESUME_MAX_RESYNCS) {
        throw new Error(`upload lost its place at ${offset} bytes; try again`)
      }
      resyncs += 1
      offset = outcome.resync
      continue
    }
    offset = end
    if (onProgress) onProgress(offset, file.size)
    if (complete) return outcome.row
  }
}
