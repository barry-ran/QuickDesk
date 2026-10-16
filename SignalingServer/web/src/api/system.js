import { authFetch, problemToError } from './auth.js'

const BASE = '/v1/admin/system/logs'

export async function getSystemLogs() {
  const response = await authFetch(BASE)
  if (!response.ok) throw problemToError(response, await response.json().catch(() => null))
  return response.json()
}

// searchSystemLogs drives GET /v1/admin/system/logs:search.
//
// `scope` is the interesting parameter: `local` reads the files on the instance
// that answers the request, `all` reads the lines every instance has shipped
// into the shared store (which needs INSTANCE_LOG_SINK on each instance, and
// answers 400 with that setting named when it is off). Empty values are dropped
// rather than sent as `q=`, so "no filter" means no filter on both sides.
export async function searchSystemLogs(params = {}) {
  const query = new URLSearchParams()
  for (const [key, value] of Object.entries(params)) {
    if (value !== undefined && value !== null && value !== '') {
      query.set(key, String(value))
    }
  }
  const response = await authFetch(`${BASE}:search?${query.toString()}`)
  if (!response.ok) throw problemToError(response, await response.json().catch(() => null))
  return response.json()
}

export async function downloadSystemLog(name) {
  return downloadFile(`${BASE}/${encodeURIComponent(name)}`)
}

export async function downloadFile(url) {
  const response = await authFetch(url)
  if (!response.ok) throw problemToError(response, await response.json().catch(() => null))
  return {
    blob: await response.blob(),
    filename: filenameFromDisposition(response.headers.get('Content-Disposition'))
  }
}

function filenameFromDisposition(value) {
  const match = value?.match(/filename\*?=(?:UTF-8'')?("?)([^";]+)\1/i)
  return match ? decodeURIComponent(match[2]) : ''
}
