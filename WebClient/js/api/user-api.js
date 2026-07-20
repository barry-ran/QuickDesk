// Copyright 2026 AnyControl. All rights reserved.
//
// Slim user-API used by remote.html (legacy entry point).
//
// remote.html is opened by the Vue shell via window.open(same-origin),
// so it shares localStorage with the parent. We read the access_token
// the parent saved (key `quickdesk_user_access_token`) and use it for
// connection-history recording. We do NOT manage login flows here —
// the Vue shell owns that.
//
// All endpoints match the v1 contract from
// docs/dev/信令服务器API重构方案.md.

const ACCESS_TOKEN_KEY  = 'quickdesk_user_access_token';
const REFRESH_TOKEN_KEY = 'quickdesk_user_refresh_token';
const ACCESS_EXPIRES_KEY  = 'quickdesk_user_access_expires_at';
const REFRESH_EXPIRES_KEY = 'quickdesk_user_refresh_expires_at';
const USER_INFO_KEY     = 'quickdesk_user_info';

const PROACTIVE_REFRESH_MARGIN_MS = 5 * 60 * 1000;
const PROACTIVE_REFRESH_RETRY_MS = 60 * 1000;
const FALLBACK_REFRESH_INTERVAL_MS = 90 * 60 * 1000;

class UserApi {
  constructor(baseUrl) {
    this._baseUrl = '';
    this._refreshInFlight = null;
    this._refreshTimer = null;
    if (baseUrl) this.setBaseUrl(baseUrl);
  }

  setBaseUrl(url) {
    if (!url) { this._baseUrl = ''; return; }
    let httpUrl = url.replace(/\/$/, '');
    if (httpUrl.startsWith('wss://')) httpUrl = httpUrl.replace(/^wss:\/\//, 'https://');
    else if (httpUrl.startsWith('ws://')) httpUrl = httpUrl.replace(/^ws:\/\//, 'http://');
    this._baseUrl = httpUrl;
  }

  // -------- token / session --------

  getToken()        { return localStorage.getItem(ACCESS_TOKEN_KEY); }
  getRefreshToken() { return localStorage.getItem(REFRESH_TOKEN_KEY); }
  isLoggedIn()      { return !!this.getToken(); }

  getUserInfo() {
    try { const r = localStorage.getItem(USER_INFO_KEY); return r ? JSON.parse(r) : null; }
    catch { return null; }
  }

  _saveSessionPayload(p) {
    if (!p) return;
    if (p.access_token)  localStorage.setItem(ACCESS_TOKEN_KEY,  p.access_token);
    if (p.refresh_token) localStorage.setItem(REFRESH_TOKEN_KEY, p.refresh_token);
    if (p.access_expires_at) localStorage.setItem(ACCESS_EXPIRES_KEY, p.access_expires_at);
    if (p.refresh_expires_at) localStorage.setItem(REFRESH_EXPIRES_KEY, p.refresh_expires_at);
    this.scheduleProactiveRefresh();
  }

  _clearSession() {
    this.stopProactiveRefresh();
    localStorage.removeItem(ACCESS_TOKEN_KEY);
    localStorage.removeItem(REFRESH_TOKEN_KEY);
    localStorage.removeItem(ACCESS_EXPIRES_KEY);
    localStorage.removeItem(REFRESH_EXPIRES_KEY);
    localStorage.removeItem(USER_INFO_KEY);
  }

  scheduleProactiveRefresh() {
    this.stopProactiveRefresh();
    if (!this.getToken() || !this.getRefreshToken()) return;
    const expRaw = localStorage.getItem(ACCESS_EXPIRES_KEY);
    const expMs = expRaw ? Date.parse(expRaw) : NaN;
    let delay = Number.isFinite(expMs)
      ? expMs - Date.now() - PROACTIVE_REFRESH_MARGIN_MS
      : FALLBACK_REFRESH_INTERVAL_MS;
    if (delay < 0) delay = 0;
    this._refreshTimer = setTimeout(async () => {
      const ok = await this._refreshSingleFlight();
      if (ok) {
        this.scheduleProactiveRefresh();
      } else if (this.getToken() && this.getRefreshToken()) {
        this._refreshTimer = setTimeout(() => this.scheduleProactiveRefresh(), PROACTIVE_REFRESH_RETRY_MS);
      }
    }, delay);
  }

  stopProactiveRefresh() {
    if (this._refreshTimer) clearTimeout(this._refreshTimer);
    this._refreshTimer = null;
  }

  _authHeaders() {
    const headers = { 'Content-Type': 'application/json' };
    const t = this.getToken();
    if (t) headers['Authorization'] = `Bearer ${t}`;
    return headers;
  }

  // -------- request wrapper with single 401 retry --------

  async _request(method, path, body, opts = {}) {
    const url = `${this._baseUrl}${path}`;
    const doOnce = async () => {
      const init = { method, headers: this._authHeaders(), credentials: 'omit' };
      if (body !== undefined) init.body = JSON.stringify(body);
      return fetch(url, init);
    };

    let resp;
    try { resp = await doOnce(); }
    catch (err) { return { ok: false, data: null, error: err.message || String(err) }; }

    if (resp.status !== 401 || opts.noRefresh || !this.getRefreshToken()) {
      return this._finalize(resp);
    }

    const refreshed = await this._refreshSingleFlight();
    if (!refreshed) return { ok: false, data: null, error: 'session ended', code: 'REFRESH_INVALID' };

    try { resp = await doOnce(); }
    catch (err) { return { ok: false, data: null, error: err.message || String(err) }; }
    return this._finalize(resp);
  }

  async _finalize(resp) {
    if (!resp.ok) {
      const problem = await resp.json().catch(() => null);
      const code   = (problem && problem.code) || null;
      const detail = (problem && (problem.detail || problem.title)) || `HTTP ${resp.status}`;
      return { ok: false, data: problem, code, error: detail, status: resp.status };
    }
    const data = await resp.json().catch(() => null);
    return { ok: true, data, status: resp.status };
  }

  async _refreshSingleFlight() {
    if (this._refreshInFlight) return this._refreshInFlight;
    this._refreshInFlight = (async () => {
      try {
        const rt = this.getRefreshToken();
        if (!rt) return false;
        const resp = await fetch(`${this._baseUrl}/v1/auth/tokens:refresh`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ refresh_token: rt }),
        });
        if (!resp.ok) { this._clearSession(); return false; }
        const data = await resp.json().catch(() => null);
        if (!data || !data.access_token) { this._clearSession(); return false; }
        this._saveSessionPayload(data);
        return true;
      } catch {
        return false;
      } finally {
        this._refreshInFlight = null;
      }
    })();
    return this._refreshInFlight;
  }

  // -------- features / me --------

  async fetchFeatures() { return this._request('GET', '/v1/features', undefined, { noRefresh: true }); }
  async fetchMe()       { return this._request('GET', '/v1/me'); }

  // -------- connection history (only thing remote.html actually writes) --

  async recordConnection(deviceId, duration, status, errorMsg) {
    return this._request('POST', '/v1/me/connections', {
      device_id: deviceId,
      duration:  duration || 0,
      status:    status   || 'success',
      error_msg: errorMsg || '',
    });
  }

  // -------- access-code verification (§2.6 / §2.18) --------

  async verifyAccessCode(deviceId, code) {
    return this._request(
      'POST',
      `/v1/devices/${encodeURIComponent(deviceId)}/access-code:verify`,
      { code },
      { noRefresh: true },
    );
  }
}

export const userApi = new UserApi();
export { UserApi };
