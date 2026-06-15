// User API — wraps all /v1/* endpoints used by the Vue WebClient.
//
// Matches the v1 contract defined in docs/dev/信令服务器API重构方案.md:
//   • RFC 7807 problem details for every non-2xx response
//   • access_token (Bearer) + refresh_token (POST /v1/auth/tokens:refresh)
//   • 401 → single silent refresh → retry; second 401 = session end (T10)
//   • list envelope {items, next_cursor}
//   • PATCH for partial updates (T5)
//   • X-API-Key is NOT attached by the browser — server must allow the
//     WebClient origin via allowed_origins (§2.2 H1 / T16).

const ACCESS_TOKEN_KEY  = 'quickdesk_user_access_token'
const REFRESH_TOKEN_KEY = 'quickdesk_user_refresh_token'
const USER_INFO_KEY     = 'quickdesk_user_info'
const SERVER_URL_KEY    = 'quickdesk_signaling_url'
export const DEFAULT_SERVER = 'ws://qdsignaling.quickcoder.cc:8060'

// Legacy key cleaned up on load so old sessions are not retained.
const LEGACY_TOKEN_KEY = 'quickdesk_user_token'

class UserApi {
  constructor() {
    this._baseUrl = ''
    this._refreshInFlight = null   // Promise<bool> — collapses concurrent refresh
    this._onSessionEndedHandler = null
    if (localStorage.getItem(LEGACY_TOKEN_KEY)) {
      localStorage.removeItem(LEGACY_TOKEN_KEY)
    }
  }

  setBaseUrl(url) {
    if (!url) { this._baseUrl = ''; return }
    let u = url.replace(/\/$/, '')
    if (u.startsWith('wss://')) u = u.replace(/^wss:\/\//, 'https://')
    else if (u.startsWith('ws://')) u = u.replace(/^ws:\/\//, 'http://')
    this._baseUrl = u
  }

  ensureBaseUrl() {
    if (!this._baseUrl) {
      const url = localStorage.getItem(SERVER_URL_KEY) || DEFAULT_SERVER
      this.setBaseUrl(url)
    }
  }

  getServerUrl() {
    return localStorage.getItem(SERVER_URL_KEY) || DEFAULT_SERVER
  }

  // ----- session helpers --------------------------------------------------

  getToken() { return localStorage.getItem(ACCESS_TOKEN_KEY) }
  getRefreshToken() { return localStorage.getItem(REFRESH_TOKEN_KEY) }
  isLoggedIn() { return !!this.getToken() }

  getUserInfo() {
    try { return JSON.parse(localStorage.getItem(USER_INFO_KEY)) } catch { return null }
  }

  _saveSession(payload) {
    if (!payload) return
    if (payload.access_token)  localStorage.setItem(ACCESS_TOKEN_KEY, payload.access_token)
    if (payload.refresh_token) localStorage.setItem(REFRESH_TOKEN_KEY, payload.refresh_token)
    if (payload.user) {
      const u = payload.user
      localStorage.setItem(USER_INFO_KEY, JSON.stringify({
        id: u.id, username: u.username, phone: u.phone, email: u.email,
      }))
    }
  }

  clearSession() {
    localStorage.removeItem(ACCESS_TOKEN_KEY)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
    localStorage.removeItem(USER_INFO_KEY)
  }

  /**
   * Register a callback fired when the HTTP layer decides the session
   * has ended (second 401 after refresh, refresh token rejected, etc.).
   */
  onSessionEnded(fn) { this._onSessionEndedHandler = fn }

  _sessionEnded() {
    this.clearSession()
    try { this._onSessionEndedHandler && this._onSessionEndedHandler() } catch { /* noop */ }
  }

  /**
   * External trigger for the session-ended flow, used by userSync.js
   * when the server pushes `session.revoked` on the realtime WS.
   * Fires a best-effort DELETE /v1/me/sessions/current (T9 — we still
   * hit the endpoint so any lingering server-side state gets cleaned
   * up; a 401 response is expected and ignored) and then delegates to
   * _sessionEnded() so the shell pops to login.
   */
  handleServerRevoked() {
    // Fire-and-forget; don't trigger the refresh cascade since the
    // server-initiated revoke means the refresh token family is dead.
    this._req('DELETE', '/v1/me/sessions/current', undefined, { noRefresh: true })
      .catch(() => {})
    this._sessionEnded()
  }

  // ----- headers ----------------------------------------------------------

  _headers(extra) {
    const h = { 'Content-Type': 'application/json', ...(extra || {}) }
    const t = this.getToken()
    if (t) h['Authorization'] = `Bearer ${t}`
    return h
  }

  // ----- RFC 7807 parsing -------------------------------------------------

  async _parseProblemBody(resp) {
    const ct = (resp.headers && resp.headers.get && resp.headers.get('Content-Type')) || ''
    try {
      if (ct.includes('json') || ct === '') {
        return await resp.json()
      }
      const text = await resp.text()
      return text ? { detail: text } : null
    } catch {
      return null
    }
  }

  _problemToResult(resp, problem) {
    const code = (problem && problem.code) || null
    const detail = (problem && (problem.detail || problem.title)) || `HTTP ${resp.status}`
    // §2.10 / §2.15: verify hands back Retry-After (seconds) when the
    // per-(device,ip), per-device or per-ip rate limiter kicks in. Surface
    // it so the UI can show a countdown instead of a static "try again
    // later" message.
    let retryAfter = 0
    try {
      const raw = resp.headers && resp.headers.get && resp.headers.get('Retry-After')
      if (raw) {
        const n = parseInt(raw, 10)
        if (!isNaN(n) && n > 0) retryAfter = n
      }
    } catch { /* headers might not be iterable in older engines */ }
    return { ok: false, data: problem, code, error: detail, status: resp.status, retryAfter }
  }

  // ----- core request (with single-flight refresh) ------------------------

  async _req(method, path, body, opts = {}) {
    this.ensureBaseUrl()
    const url = `${this._baseUrl}${path}`
    const doOnce = async () => {
      const init = { method, headers: this._headers(opts.headers), credentials: 'omit' }
      if (body !== undefined) init.body = JSON.stringify(body)
      return fetch(url, init)
    }

    let resp
    try { resp = await doOnce() }
    catch (err) { return { ok: false, data: null, code: null, error: err.message || String(err), status: 0 } }

    // Happy path / non-401 / auth endpoint / no refresh token on hand.
    if (resp.status !== 401 || opts.noRefresh || !this.getRefreshToken()) {
      if (!resp.ok) {
        const problem = await this._parseProblemBody(resp)
        if (resp.status === 401 && !opts.noRefresh) this._sessionEnded()
        return this._problemToResult(resp, problem)
      }
      const data = await resp.json().catch(() => null)
      return { ok: true, data, code: null, error: null, status: resp.status }
    }

    // 401 with a refresh token on hand → attempt single silent refresh.
    const refreshed = await this._refreshSingleFlight()
    if (!refreshed) {
      return { ok: false, data: null, code: 'REFRESH_INVALID', error: 'session ended', status: 401 }
    }

    // Retry once.
    try { resp = await doOnce() }
    catch (err) { return { ok: false, data: null, code: null, error: err.message || String(err), status: 0 } }

    if (resp.status === 401) {
      // §2.15 T10: second 401 after refresh = session ended.
      this._sessionEnded()
      const problem = await this._parseProblemBody(resp)
      return this._problemToResult(resp, problem)
    }
    if (!resp.ok) {
      const problem = await this._parseProblemBody(resp)
      return this._problemToResult(resp, problem)
    }
    const data = await resp.json().catch(() => null)
    return { ok: true, data, code: null, error: null, status: resp.status }
  }

  async _refreshSingleFlight() {
    if (this._refreshInFlight) return this._refreshInFlight
    this._refreshInFlight = (async () => {
      try {
        const rt = this.getRefreshToken()
        if (!rt) return false
        const resp = await fetch(`${this._baseUrl}/v1/auth/tokens:refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: rt }),
        })
        if (!resp.ok) {
          // §2.15: refresh rejected → clear session.
          this._sessionEnded()
          return false
        }
        const data = await resp.json().catch(() => null)
        if (!data || !data.access_token) { this._sessionEnded(); return false }
        this._saveSession(data)
        return true
      } catch {
        // Network error during refresh — do not clear session (user may retry).
        return false
      } finally {
        this._refreshInFlight = null
      }
    })()
    return this._refreshInFlight
  }

  // ========================================================================
  // Public surface
  // ========================================================================

  fetchFeatures()        { return this._req('GET', '/v1/features',        undefined, { noRefresh: true }) }
  fetchPublicSettings()  { return this._req('GET', '/v1/settings/public', undefined, { noRefresh: true }) }

  // scene ∈ {login, register, reset_password, bind_phone}
  sendSmsCode(phone, scene) {
    return this._req('POST', '/v1/verification-codes', { phone, scene }, { noRefresh: true })
  }

  // ========================================================================
  // Auth
  // ========================================================================

  async login(identifier, password) {
    const r = await this._req('POST', '/v1/auth/sessions', { identifier, password }, { noRefresh: true })
    if (r.ok && r.data) this._saveSession(r.data)
    return r
  }

  async loginWithSms(phone, smsCode) {
    const r = await this._req('POST', '/v1/auth/sessions:sms', { phone, sms_code: smsCode }, { noRefresh: true })
    if (r.ok && r.data) this._saveSession(r.data)
    return r
  }

  // §2.2 T20: register returns a full session envelope — user is logged in.
  async register(username, password, phone, email, smsCode) {
    const body = { username, password }
    if (phone)   body.phone     = phone
    if (email)   body.email     = email
    if (smsCode) body.sms_code  = smsCode
    const r = await this._req('POST', '/v1/auth/register', body, { noRefresh: true })
    if (r.ok && r.data) this._saveSession(r.data)
    return r
  }

  // §2.11 two-step logout. WebClient has no host process so step 1
  // (DELETE /v1/me/devices/:id/session) is skipped — see T12.
  // noRefresh: a 401 here means "server already revoked us" — exactly
  // what logout wants. Avoid a needless refresh+retry cascade.
  async logout() {
    if (this.isLoggedIn()) {
      await this._req('DELETE', '/v1/me/sessions/current', undefined, { noRefresh: true })
        .catch(() => {})
    }
    this.clearSession()
    return { ok: true }
  }

  fetchMe() { return this._req('GET', '/v1/me') }

  // ========================================================================
  // Account
  // ========================================================================

  changePassword(oldPassword, newPassword) {
    return this._req('PUT', '/v1/me/password', { old_password: oldPassword, new_password: newPassword })
  }

  sendResetPasswordCode(phone) {
    return this._req('POST', '/v1/auth/password-resets', { phone }, { noRefresh: true })
  }

  resetPassword(phone, smsCode, newPassword) {
    return this._req(
      'POST',
      '/v1/auth/password-resets:confirm',
      { phone, sms_code: smsCode, new_password: newPassword },
      { noRefresh: true },
    )
  }

  changeUsername(newUsername)          { return this._req('PUT', '/v1/me/username', { username: newUsername }) }
  changePhone(newPhone, smsCode)       { return this._req('PUT', '/v1/me/phone', { phone: newPhone, sms_code: smsCode }) }
  changeEmail(newEmail)                { return this._req('PUT', '/v1/me/email', { email: newEmail }) }
  listSessions()                       { return this._req('GET', '/v1/me/sessions') }

  // ========================================================================
  // Devices & favorites — all list endpoints use {items, next_cursor} (T2)
  // ========================================================================

  fetchMyDevices() { return this._req('GET', '/v1/me/devices') }

  unbindDevice(deviceId) {
    return this._req('DELETE', `/v1/me/devices/${encodeURIComponent(deviceId)}`)
  }

  // §2.2 / T5: PATCH for partial updates.
  setDeviceRemark(deviceId, remark) {
    return this._req('PATCH', `/v1/me/devices/${encodeURIComponent(deviceId)}`, { remark })
  }

  fetchConnectionLogs(cursor) {
    const q = cursor ? `?cursor=${encodeURIComponent(cursor)}` : ''
    return this._req('GET', `/v1/me/connections${q}`)
  }

  recordConnection(deviceId, duration, status, errorMsg) {
    return this._req('POST', '/v1/me/connections', {
      device_id: deviceId,
      duration:  duration || 0,
      status:    status   || 'success',
      error_msg: errorMsg || '',
    })
  }

  fetchFavorites() { return this._req('GET', '/v1/me/favorites') }

  addFavorite(deviceId, name, password) {
    const body = { device_id: deviceId }
    if (name)     body.device_name     = name
    if (password) body.access_password = password
    return this._req('POST', '/v1/me/favorites', body)
  }

  updateFavorite(deviceId, name, password) {
    return this._req(
      'PATCH',
      `/v1/me/favorites/${encodeURIComponent(deviceId)}`,
      { device_name: name, access_password: password },
    )
  }

  removeFavorite(deviceId) {
    return this._req('DELETE', `/v1/me/favorites/${encodeURIComponent(deviceId)}`)
  }

  // ========================================================================
  // Access-code verification (§2.6 / §2.18)
  //
  // Returns { ok, data:{signal_token, expires_at}, code, error, status }.
  // Error `code` ∈ { DEVICE_NOT_FOUND, HOST_OFFLINE, INVALID_CODE,
  //                  TOO_MANY_ATTEMPTS }. The browser MUST send this over
  //                  an origin the server has on its allow-list (§2.2 H1).
  // ========================================================================

  verifyAccessCode(deviceId, code) {
    return this._req(
      'POST',
      `/v1/devices/${encodeURIComponent(deviceId)}/access-code:verify`,
      { code },
      { noRefresh: true },
    )
  }
}

export const userApi = new UserApi()
