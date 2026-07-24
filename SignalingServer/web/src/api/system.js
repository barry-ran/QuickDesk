import { authFetch, problemToError } from './auth.js'

const BASE = '/v1/admin/system/logs'

export async function getSystemLogs() {
  const response = await authFetch(BASE)
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
