// /v1/admin/groups/* — device groups (§2.2).
import { authJson } from './auth.js'

const BASE = '/v1/admin/groups'

export function getGroups()       { return authJson(BASE) }

export function createGroup(data) {
  return authJson(BASE, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export function updateGroup(id, data) {
  return authJson(`${BASE}/${id}`, {
    method: 'PATCH',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(data),
  })
}

export function deleteGroup(id) {
  return authJson(`${BASE}/${id}`, { method: 'DELETE' })
}

export function addDevicesToGroup(groupId, deviceIds) {
  return authJson(`${BASE}/${groupId}/devices`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ device_ids: deviceIds }),
  })
}

export function removeDevicesFromGroup(groupId, deviceIds) {
  return authJson(`${BASE}/${groupId}/devices`, {
    method: 'DELETE',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ device_ids: deviceIds }),
  })
}

export function getGroupDevices(groupId) {
  return authJson(`${BASE}/${groupId}/devices`)
}
