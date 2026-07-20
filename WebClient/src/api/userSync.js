// Real-time sync over /v1/realtime/events.
//
// First-frame auth protocol (§2.13 / T6):
//   client → {type:"auth", access_token, since_rev?}
//   server → {type:"auth_ok", server_rev}
//   server → {type:"snapshot", server_rev, data:{devices, favorites}}  (or a
//            stream of replay events when resume succeeds)
//   server → {type:"bootstrap_done", server_rev}
//   server → increments of {type:"device.*"/"favorite.*"/"session.revoked", server_rev}
//
// The transport maintains:
//   • server_rev so reconnects can send {since_rev} (§2.8 resume)
//   • local arrays of devices/favorites (snapshot replaces, events patch — T7/T8)
//   • exponential backoff + jitter on reconnect (§2.15 / T14)
//   • proactive HTTP refresh kick on WS TOKEN_INVALID (T11)

import { userApi } from './userApi'

const BASE_MS   = 1000
const MAX_MS    = 30000
const JITTER_MS = 500

// Event types that carry access_code implicitly — we don't embed plaintext
// in the event data (§2.16), so the client is expected to already have it
// from snapshot/my-devices or to refetch.
const DEVICE_EVENT_TYPES = [
  'device.online.changed',
  'device.session.updated',
  'device.ownership.lost',
  'device.unbound',
  'device.bound',
  'device.added',
  'device.remark.changed',
  'device.device_name.changed',
  'device.access_code.changed',
  'device.secret.rotated',
]

const FAVORITE_EVENT_TYPES = [
  'favorite.added',
  'favorite.updated',
  'favorite.removed',
]

class UserSync extends EventTarget {
  constructor() {
    super()
    this._ws = null
    this._reconnectTimer = null
    this._stopped = true
    this._authOk = false
    this._attempt = 0
    this._serverRev = 0
    this._lastAppliedRev = 0

    // Local mirrors — consumers read via getDevices()/getFavorites() or
    // listen to 'snapshot' / 'patch' events.
    this._devices   = []
    this._favorites = []
    this._deviceRevs = new Map()
  }

  start() {
    if (!userApi.getToken()) return
    this._stopped = false
    this._attempt = 0
    this._open()
  }

  stop() {
    this._stopped = true
    clearTimeout(this._reconnectTimer); this._reconnectTimer = null
    if (this._ws) {
      this._ws.onclose = null
      try { this._ws.close() } catch { /* noop */ }
      this._ws = null
    }
    this._authOk = false
    // Reset server_rev so a fresh login gets a full snapshot (not a resume
    // against a stale rev from a previous session).
    this._serverRev = 0
    this._lastAppliedRev = 0
    this._devices = []
    this._favorites = []
    this._deviceRevs.clear()
  }

  // Read-only accessors the views bind to.
  getDevices()   { return this._devices.slice() }
  getFavorites() { return this._favorites.slice() }
  serverRev()    { return this._serverRev }
  isConnected()  { return !!(this._ws && this._authOk) }

  // ------------------------------------------------------------------------

  _wsUrl() {
    let base = userApi.getServerUrl().replace(/\/+$/, '')
    if (base.startsWith('http://'))  base = base.replace(/^http:\/\//,  'ws://')
    if (base.startsWith('https://')) base = base.replace(/^https:\/\//, 'wss://')
    // §2.16: URL carries NO token/secret/code. Auth is first-frame only.
    return `${base}/v1/realtime/events`
  }

  _open() {
    this._authOk = false
    try {
      this._ws = new WebSocket(this._wsUrl())
    } catch {
      this._scheduleReconnect()
      return
    }

    this._ws.onopen = () => this._sendAuthFrame()
    this._ws.onmessage = (e) => this._onMessage(e)
    this._ws.onerror = () => { /* onclose follows */ }
    this._ws.onclose = () => {
      const wasAuthOk = this._authOk
      this._authOk = false
      this.dispatchEvent(new CustomEvent('disconnected', { detail: { wasAuthOk } }))
      this._scheduleReconnect()
    }
  }

  _sendAuthFrame() {
    const token = userApi.getToken()
    if (!token) { try { this._ws.close() } catch { /* noop */ } return }
    const frame = { type: 'auth', access_token: token }
    if (this._lastAppliedRev > 0) frame.since_rev = this._lastAppliedRev
    try { this._ws.send(JSON.stringify(frame)) }
    catch { /* next onclose handles reconnect */ }
  }

  _scheduleReconnect() {
    if (this._stopped) return
    clearTimeout(this._reconnectTimer)
    const attempt = ++this._attempt
    // Cap the exponent so we don't overflow.
    const expCap = Math.min(attempt - 1, 5)
    const delay = Math.min(BASE_MS * Math.pow(2, expCap), MAX_MS)
                + Math.floor(Math.random() * JITTER_MS)
    this._reconnectTimer = setTimeout(() => {
      if (this._stopped) return
      if (!userApi.getToken()) return
      this._open()
    }, delay)
  }

  _onMessage(event) {
    let msg
    try { msg = JSON.parse(event.data) } catch { return }
    const type = msg && msg.type
    if (!type) return

    // Track server_rev whenever the frame carries one (§2.8).
    if (typeof msg.server_rev === 'number' && msg.server_rev > this._serverRev) {
      this._serverRev = msg.server_rev
    }

    if (type === 'auth_ok') {
      this._authOk = true
      this._attempt = 0
      return
    }

    if (type === 'error') {
      const code = (msg.data && msg.data.code) || ''
      this.dispatchEvent(new CustomEvent('error', { detail: { code, data: msg.data || null } }))
      // T11: TOKEN_INVALID on WS means the access_token we used is stale.
      // Kick an HTTP request so the 401→refresh cascade runs once, then
      // the reconnect picks up the new token.
      if (code === 'TOKEN_INVALID' || code === 'AUTH_INVALID') {
        userApi.fetchMe().catch(() => {})
      }
      try { this._ws.close() } catch { /* onclose handles reconnect */ }
      return
    }

    if (type === 'snapshot_required') {
      // We cannot prove a clean replay. The server keeps the connection
      // open and sends a full snapshot next for backwards compatibility
      // with older clients that treat this frame as a hint.
      return
    }

    if (type === 'snapshot') {
      if (!this._shouldApplyFrame(type, msg.server_rev)) return
      this._applySnapshot(msg)
      return
    }

    if (type === 'bootstrap_done') {
      if (typeof msg.server_rev === 'number' && msg.server_rev > this._lastAppliedRev) {
        this._lastAppliedRev = msg.server_rev
      }
      this.dispatchEvent(new CustomEvent('connected', { detail: { server_rev: this._serverRev } }))
      return
    }
    if (type === 'session.revoked') {
      if (!this._shouldApplyFrame(type, msg.server_rev)) return
      // The server already revoked this session family. Clear local user
      // credentials only; do not issue a second logout mutation.
      this.stop()
      this.dispatchEvent(new CustomEvent('session-revoked'))
      userApi.handleServerRevoked()
      return
    }

    if (!this._shouldApplyFrame(type, msg.server_rev)) return

    // Domain events — patch local state and notify.
    if (DEVICE_EVENT_TYPES.includes(type)) {
      this._applyDeviceEvent(type, msg.data || {})
      return
    }
    if (FAVORITE_EVENT_TYPES.includes(type)) {
      this._applyFavoriteEvent(type, msg.data || {})
      return
    }
  }

  // ----- snapshot / patch helpers ----------------------------------------

  _shouldApplyFrame(type, rev) {
    if (typeof rev !== 'number' || rev <= 0) return false
    if (rev <= this._lastAppliedRev) return false
    this._lastAppliedRev = rev
    return true
  }

  _applySnapshot(msg) {
    // T7: snapshot fully REPLACES local arrays.
    const data = msg.data || {}
    const rev = msg.server_rev || this._lastAppliedRev
    this._devices   = Array.isArray(data.devices)   ? data.devices.slice()   : []
    this._favorites = Array.isArray(data.favorites) ? data.favorites.slice() : []
    this._deviceRevs.clear()
    this._devices.forEach(d => { if (d && d.device_id) this._deviceRevs.set(d.device_id, rev) })
    this.dispatchEvent(new CustomEvent('snapshot', {
      detail: { devices: this._devices, favorites: this._favorites, server_rev: this._serverRev },
    }))
  }

  _findDeviceIdx(deviceId) {
    return this._devices.findIndex(d => d && d.device_id === deviceId)
  }
  _findFavoriteIdx(deviceId) {
    return this._favorites.findIndex(f => f && f.device_id === deviceId)
  }

  // T8: patch local arrays, do NOT refetch on ordinary updates. The
  // server_rev gate above makes snapshot/replay/event ordering authoritative.
  _applyDeviceEvent(type, data) {
    const deviceId = data.device_id
    if (!deviceId) return
    const idx = this._findDeviceIdx(deviceId)
    const rev = this._lastAppliedRev
    const currentRev = this._deviceRevs.get(deviceId) || 0
    let dirty = false

    if (type === 'device.unbound' || type === 'device.ownership.lost') {
      if (idx >= 0) { this._devices.splice(idx, 1); dirty = true }
      this._deviceRevs.set(deviceId, rev)
    } else if (type === 'device.bound' || type === 'device.added') {
      if (idx < 0) {
        this._devices.push({ ...data })
        this._deviceRevs.set(deviceId, rev)
        dirty = true
      } else if (rev > currentRev) {
        this._devices[idx] = { ...this._devices[idx], ...data }
        this._deviceRevs.set(deviceId, rev)
        dirty = true
      }
    } else if (type === 'device.access_code.changed') {
      if (idx >= 0 && rev > currentRev) {
        this._deviceRevs.set(deviceId, rev)
        this._refreshDeviceAccessCode(deviceId, rev)
        dirty = true
      }
    } else {
      if (idx < 0) {
        this._devices.push({ ...data })
        this._deviceRevs.set(deviceId, rev)
        dirty = true
      } else if (rev > currentRev) {
        const row = { ...this._devices[idx] }
        const copyField = (k) => {
          if (Object.prototype.hasOwnProperty.call(data, k)) {
            row[k] = data[k]; dirty = true
          }
        }
        if (type === 'device.online.changed') {
          copyField('online')
          if (Object.prototype.hasOwnProperty.call(data, 'logged_in')) {
            copyField('logged_in')
          } else if (Object.prototype.hasOwnProperty.call(data, 'online')) {
            row.logged_in = !!(data.online && row.logged_in)
            dirty = true
          }
        } else if (type === 'device.session.updated') {
          copyField('logged_in'); copyField('online')
        } else if (type === 'device.remark.changed') {
          copyField('remark')
        } else if (type === 'device.device_name.changed') {
          copyField('device_name')
        } else if (type === 'device.secret.rotated') {
          dirty = true
        }
        if (dirty) {
          this._devices[idx] = row
          this._deviceRevs.set(deviceId, rev)
        }
      }
    }

    if (dirty) {
      this.dispatchEvent(new CustomEvent('devices-changed', {
        detail: { devices: this.getDevices(), type, data },
      }))
    }
  }

  async _refreshDeviceAccessCode(deviceId, eventRev) {
    const r = await userApi.fetchMyDevice(deviceId).catch((err) => ({
      ok: false,
      error: err && err.message ? err.message : String(err),
    }))
    if (!r || !r.ok || !r.data || !r.data.access_code) return

    const idx = this._findDeviceIdx(deviceId)
    if (idx < 0) {
      this._devices.push({ ...r.data })
      this._deviceRevs.set(deviceId, Math.max(eventRev, this._deviceRevs.get(deviceId) || 0))
    } else {
      this._devices[idx] = { ...this._devices[idx], access_code: r.data.access_code }
      this._deviceRevs.set(deviceId, Math.max(eventRev, this._deviceRevs.get(deviceId) || 0))
    }

    this.dispatchEvent(new CustomEvent('devices-changed', {
      detail: { devices: this.getDevices(), type: 'device.access_code.refreshed', data: { device_id: deviceId } },
    }))
  }

  _applyFavoriteEvent(type, data) {
    const deviceId = data.device_id
    if (!deviceId) return
    const idx = this._findFavoriteIdx(deviceId)
    let dirty = false

    if (type === 'favorite.removed') {
      if (idx >= 0) { this._favorites.splice(idx, 1); dirty = true }
    } else if (type === 'favorite.added') {
      if (idx < 0) { this._favorites.push({ ...data }); dirty = true }
      else         { this._favorites[idx] = { ...this._favorites[idx], ...data }; dirty = true }
    } else if (type === 'favorite.updated') {
      if (idx >= 0) { this._favorites[idx] = { ...this._favorites[idx], ...data }; dirty = true }
      else          { this._favorites.push({ ...data }); dirty = true }
    }

    if (dirty) {
      this.dispatchEvent(new CustomEvent('favorites-changed', {
        detail: { favorites: this.getFavorites(), type, data },
      }))
    }
  }
}

export const userSync = new UserSync()
