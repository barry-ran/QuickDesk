// User API — wraps all /api/v1/user/* endpoints
// Returns {ok, data, code, error} where code is the server error code for i18n

const TOKEN_KEY = 'quickdesk_user_token'
const USER_INFO_KEY = 'quickdesk_user_info'
const SERVER_URL_KEY = 'quickdesk_signaling_url'
export const DEFAULT_SERVER = 'ws://qdsignaling.quickcoder.cc:8000'

class UserApi {
  constructor() {
    this._baseUrl = ''
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

  getToken() { return localStorage.getItem(TOKEN_KEY) }
  isLoggedIn() { return !!this.getToken() }

  getUserInfo() {
    try { return JSON.parse(localStorage.getItem(USER_INFO_KEY)) } catch { return null }
  }

  _saveSession(token, user) {
    localStorage.setItem(TOKEN_KEY, token)
    if (user) localStorage.setItem(USER_INFO_KEY, JSON.stringify({ id: user.id, username: user.username, phone: user.phone, email: user.email }))
  }

  clearSession() {
    localStorage.removeItem(TOKEN_KEY)
    localStorage.removeItem(USER_INFO_KEY)
  }

  _headers() {
    const h = { 'Content-Type': 'application/json' }
    const t = this.getToken()
    if (t) h['Authorization'] = `Bearer ${t}`
    return h
  }

  async _req(method, path, body) {
    this.ensureBaseUrl()
    try {
      const opts = { method, headers: this._headers() }
      if (body !== undefined) opts.body = JSON.stringify(body)
      const resp = await fetch(`${this._baseUrl}${path}`, opts)
      const data = await resp.json().catch(() => null)
      if (!resp.ok) {
        return { ok: false, data, code: data?.code || null, error: data?.error || `HTTP ${resp.status}` }
      }
      return { ok: true, data, code: null, error: null }
    } catch (err) {
      return { ok: false, data: null, code: null, error: err.message || String(err) }
    }
  }

  // Features
  fetchFeatures() { return this._req('GET', '/api/v1/features') }

  // SMS
  sendSmsCode(phone, scene) { return this._req('POST', '/api/v1/sms/send', { phone, scene }) }

  // Auth
  async login(username, password) {
    const r = await this._req('POST', '/api/v1/user/login', { username, password })
    if (r.ok && r.data) this._saveSession(r.data.token, r.data.user)
    return r
  }

  async loginWithSms(phone, smsCode) {
    const r = await this._req('POST', '/api/v1/user/login-sms', { phone, sms_code: smsCode })
    if (r.ok && r.data) this._saveSession(r.data.token, r.data.user)
    return r
  }

  register(username, password, phone, email, smsCode) {
    const body = { username, password, phone, email }
    if (smsCode) body.sms_code = smsCode
    return this._req('POST', '/api/v1/user/register', body)
  }

  async logout() {
    const r = await this._req('POST', '/api/v1/user/logout')
    this.clearSession()
    return r
  }

  fetchMe() { return this._req('GET', '/api/v1/user/me') }

  // Account management
  changePassword(oldPassword, newPassword) {
    return this._req('PUT', '/api/v1/user/password', { old_password: oldPassword, new_password: newPassword })
  }

  sendResetPasswordCode(phone) {
    return this._req('POST', '/api/v1/user/reset-password', { phone })
  }

  resetPassword(phone, smsCode, newPassword) {
    return this._req('PUT', '/api/v1/user/reset-password', { phone, sms_code: smsCode, new_password: newPassword })
  }

  changeUsername(newUsername) {
    return this._req('PUT', '/api/v1/user/username', { username: newUsername })
  }

  changePhone(newPhone, smsCode) {
    return this._req('PUT', '/api/v1/user/phone', { phone: newPhone, sms_code: smsCode })
  }

  changeEmail(newEmail) {
    return this._req('PUT', '/api/v1/user/email', { email: newEmail })
  }

  // Devices
  fetchMyDevices() { return this._req('GET', '/api/v1/user/devices') }
  setDeviceRemark(deviceId, remark) { return this._req('PUT', `/api/v1/user/devices/${encodeURIComponent(deviceId)}/remark`, { remark }) }
  fetchConnectionLogs() { return this._req('GET', '/api/v1/user/devices/logs') }

  // Favorites
  fetchFavorites() { return this._req('GET', '/api/v1/user/favorites') }
  addFavorite(deviceId, name, password) { return this._req('POST', '/api/v1/user/favorites', { device_id: deviceId, device_name: name, access_password: password }) }
  updateFavorite(deviceId, name, password) { return this._req('PUT', `/api/v1/user/favorites/${encodeURIComponent(deviceId)}`, { device_name: name, access_password: password }) }
  removeFavorite(deviceId) { return this._req('DELETE', `/api/v1/user/favorites/${encodeURIComponent(deviceId)}`) }
}

export const userApi = new UserApi()
