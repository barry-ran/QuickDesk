// /v1/admin/webhooks/* (§2.2). Test delivery uses sub-resource form
// `/:id/test` because gin/httprouter rejects `:param:verb` patterns
// (see W1 deviation in the refactor doc).
import { authJson } from './auth.js'

const BASE = '/v1/admin/webhooks'

export function getWebhooks() { return authJson(BASE) }

export function getWebhook(id) { return authJson(`${BASE}/${id}`) }

export function createWebhook(data) {
  return authJson(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

// PATCH per §2.2.
export function updateWebhook(id, data) {
  return authJson(`${BASE}/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export function deleteWebhook(id) {
  return authJson(`${BASE}/${id}`, { method: 'DELETE' })
}

// §2.2: webhook test delivery. Uses sub-resource form `/:id/test`
// (not the AIP-136 colon-action `:test`) because gin/httprouter cannot
// register a literal suffix on a wildcard segment — see comment in
// cmd/signaling/main.go where the route is registered.
export function testWebhook(id) {
  return authJson(`${BASE}/${id}/test`, { method: 'POST' })
}
