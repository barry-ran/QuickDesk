// Admin-web auth module — targets /v1/admin/auth (§2.2 / §3.2).
//
// Behavioural contract (mirrors Qt stage 2 / WebClient stage 3):
//   • access_token (1h) + refresh_token (7d) — stored in localStorage.
//   • Single-flight silent refresh on 401. Second 401 = session ended.
//   • RFC 7807 problem parsing on all non-2xx.
//   • 2FA flow: POST /v1/admin/auth/sessions may return 401 with
//     code=TOTP_REQUIRED + { pre_token } → caller should redo with
//     /v1/admin/auth/sessions:totp.

const ACCESS_TOKEN_KEY  = 'quickdesk_admin_access_token'
const REFRESH_TOKEN_KEY = 'quickdesk_admin_refresh_token'
const ADMIN_INFO_KEY    = 'quickdesk_admin_info'
const LEGACY_TOKEN_KEY  = 'quickdesk_admin_token'

// Clear any stale pre-refactor token on load.
if (localStorage.getItem(LEGACY_TOKEN_KEY)) {
  localStorage.removeItem(LEGACY_TOKEN_KEY)
}

let _refreshInFlight = null
let _onSessionEnded = null

// Allow the shell to react to session-ended events (redirect to login).
export function onSessionEnded(fn) { _onSessionEnded = fn }

export function getToken()        { return localStorage.getItem(ACCESS_TOKEN_KEY) }
export function getRefreshToken() { return localStorage.getItem(REFRESH_TOKEN_KEY) }

export function setToken(token)        { localStorage.setItem(ACCESS_TOKEN_KEY, token) }
export function setRefreshToken(token) { localStorage.setItem(REFRESH_TOKEN_KEY, token) }

/**
 * Current admin info cached from the last /v1/admin/auth/sessions[:totp]
 * response. Shape: {id, username, email, role, status, totp_enabled,
 * last_login}. Null before login or after logout.
 *
 * Used by AdminUserPage to show the super_admin "please enable 2FA"
 * banner (§2.16) and by refreshAdminInfo() to stay current after the
 * admin edits their own account.
 */
export function getAdminInfo() {
  try { return JSON.parse(localStorage.getItem(ADMIN_INFO_KEY)) } catch { return null }
}

export function setAdminInfo(info) {
  if (info) localStorage.setItem(ADMIN_INFO_KEY, JSON.stringify(info))
  else      localStorage.removeItem(ADMIN_INFO_KEY)
}

export function removeToken() {
  localStorage.removeItem(ACCESS_TOKEN_KEY)
  localStorage.removeItem(REFRESH_TOKEN_KEY)
  localStorage.removeItem(ADMIN_INFO_KEY)
}

function saveSession(payload) {
  if (!payload) return
  if (payload.access_token)  setToken(payload.access_token)
  if (payload.refresh_token) setRefreshToken(payload.refresh_token)
  // writeAdminSession (admin_auth_handler.go) returns {admin: {...}} on
  // both password and TOTP login. Cache it so AdminUserPage can show
  // the super_admin forced-2FA banner without a second request.
  if (payload.admin) setAdminInfo(payload.admin)
}

function sessionEnded() {
  removeToken()
  try { _onSessionEnded && _onSessionEnded() } catch { /* noop */ }
}

// --- RFC 7807 helpers -----------------------------------------------------

async function readProblem(res) {
  const ct = (res.headers && res.headers.get && res.headers.get('Content-Type')) || ''
  try {
    if (ct.includes('json') || ct === '') return await res.json()
    const text = await res.text()
    return text ? { detail: text } : null
  } catch { return null }
}

export function problemToError(res, problem) {
  const code   = (problem && problem.code) || null
  const detail = (problem && (problem.detail || problem.title)) || `HTTP ${res.status}`
  const err = new Error(detail)
  err.code = code
  err.status = res.status
  err.problem = problem
  return err
}

// --- Login / refresh / logout --------------------------------------------

/**
 * Sign in with username+password.
 * @throws Error with err.code='TOTP_REQUIRED' and err.preToken when 2FA is needed.
 */
export async function login(username, password, totpCode) {
  const body = { username, password }
  if (totpCode) body.totp_code = totpCode

  const res = await fetch('/v1/admin/auth/sessions', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(body),
  })

  if (!res.ok) {
    const problem = await readProblem(res)
    const err = problemToError(res, problem)
    // Server may emit a TOTP step-up with a short-lived pre_token (§2.16).
    if (err.code === 'TOTP_REQUIRED' && problem && problem.pre_token) {
      err.preToken = problem.pre_token
    }
    // Legacy shim: old backend returned {error:"2fa_required"} — map to
    // the new code so callers have one path.
    if (problem && problem.error === '2fa_required') {
      err.code = 'TOTP_REQUIRED'
    }
    throw err
  }

  const data = await res.json()
  if (data?.two_factor_needed) {
    const err = new Error('Two-factor authentication required')
    err.code = 'TOTP_REQUIRED'
    err.preToken = data.pre_token
    throw err
  }
  saveSession(data)
  return data
}

/** Complete 2FA step-up using the pre_token from the previous call. */
export async function loginWithTotp(preToken, totpCode) {
  const res = await fetch('/v1/admin/auth/sessions:totp', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ pre_token: preToken, totp_code: totpCode }),
  })
  if (!res.ok) throw problemToError(res, await readProblem(res))
  const data = await res.json()
  saveSession(data)
  return data
}

export async function logout() {
  // Best-effort remote revocation + local wipe.
  // _noRefresh: a 401 here doesn't mean "session expired, refresh and
  // retry" — it means the server already considers us logged out, which
  // is exactly what we want. Skipping the refresh-cascade also avoids
  // the onSessionEnded callback firing on a normal logout.
  try {
    await authFetch('/v1/admin/auth/sessions/current', { method: 'DELETE', _noRefresh: true })
  } catch { /* ignore */ }
  removeToken()
}

async function refreshSingleFlight() {
  if (_refreshInFlight) return _refreshInFlight
  _refreshInFlight = (async () => {
    try {
      const rt = getRefreshToken()
      if (!rt) return false
      const res = await fetch('/v1/admin/auth/tokens:refresh', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ refresh_token: rt }),
      })
      if (!res.ok) { sessionEnded(); return false }
      const data = await res.json().catch(() => null)
      if (!data || !data.access_token) { sessionEnded(); return false }
      saveSession(data)
      return true
    } catch { return false }
    finally { _refreshInFlight = null }
  })()
  return _refreshInFlight
}

// --- authFetch: used by every other src/api/*.js module ------------------

/**
 * Fetch with Bearer header + silent 401→refresh→retry.
 *
 * On a terminal 401 (refresh failed or second 401 after refresh) the
 * session is cleared and the registered onSessionEnded() callback fires.
 * Callers still receive a normal Response so they can decide what to do
 * with it (the shell will usually redirect to /login on 401).
 */
export async function authFetch(url, options = {}) {
  const { _noRefresh, ...fetchOptions } = options
  const doOnce = () => {
    const headers = { ...(fetchOptions.headers || {}) }
    const tok = getToken()
    if (tok) headers['Authorization'] = `Bearer ${tok}`
    return fetch(url, { ...fetchOptions, headers })
  }

  let res = await doOnce()
  if (res.status !== 401 || _noRefresh || !getRefreshToken()) {
    if (res.status === 401 && !_noRefresh) sessionEnded()
    return res
  }

  const refreshed = await refreshSingleFlight()
  if (!refreshed) return res  // session already ended; callers see 401

  res = await doOnce()
  if (res.status === 401) sessionEnded()
  return res
}

/**
 * Helper: perform an authFetch and return the parsed JSON body. Throws
 * on non-2xx with err.code / err.detail / err.status populated from the
 * RFC 7807 problem.
 */
export async function authJson(url, options = {}) {
  const res = await authFetch(url, options)
  if (!res.ok) throw problemToError(res, await readProblem(res))
  return res.json().catch(() => null)
}
