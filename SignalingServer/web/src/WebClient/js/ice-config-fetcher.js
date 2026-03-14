/**
 * ice-config-fetcher.js - ICE 配置动态拉取器
 *
 * 对应 Chromium 侧 quickdesk_ice_config_fetcher.cc
 * 从信令服务器 /api/v1/ice-config 拉取时限 TURN 凭据，合并内置 STUN 和用户配置
 */

const BUILTIN_STUN_URLS = [
    'stun:stun.hot-chilli.net',
    'stun:stun.internetcalls.com',
    'stun:stun.miwifi.com',
];

const ICE_CONFIG_PATH = '/api/v1/ice-config';
const FETCH_TIMEOUT_MS = 3000;

export class IceConfigFetcher {
    /**
     * @param {string} signalingWsUrl - WebSocket 信令服务器地址 (ws:// 或 wss://)
     */
    constructor(signalingWsUrl) {
        this._httpBaseUrl = wsUrlToHttpUrl(signalingWsUrl);
    }

    /**
     * 获取完整 ICE 配置
     * @param {Array} userIceServers - 用户在设置中配置的 ICE 服务器
     * @returns {Promise<Array>} RTCPeerConnection 可用的 iceServers 数组
     */
    async getIceServers(userIceServers = []) {
        const builtinStun = BUILTIN_STUN_URLS.map(url => ({ urls: url }));
        const hasUserTurn = userIceServers.some(s => {
            const urls = Array.isArray(s.urls) ? s.urls : [s.urls];
            return urls.some(u => u.startsWith('turn:') || u.startsWith('turns:'));
        });

        if (hasUserTurn) {
            console.log('[IceConfigFetcher] User has TURN configured, skipping server fetch');
            return [...builtinStun, ...userIceServers];
        }

        try {
            const fetched = await this._fetchFromServer();
            if (fetched && fetched.length > 0) {
                console.log(`[IceConfigFetcher] Fetched ${fetched.length} server(s) from signaling`);
                return [...builtinStun, ...userIceServers, ...fetched];
            }
        } catch (e) {
            console.warn('[IceConfigFetcher] Fetch failed, using builtin only:', e.message);
        }

        return [...builtinStun, ...userIceServers];
    }

    /**
     * @private
     */
    async _fetchFromServer() {
        const url = this._httpBaseUrl + ICE_CONFIG_PATH;
        console.log(`[IceConfigFetcher] Fetching from ${url}`);

        const controller = new AbortController();
        const timeoutId = setTimeout(() => controller.abort(), FETCH_TIMEOUT_MS);

        try {
            const resp = await fetch(url, {
                method: 'GET',
                signal: controller.signal,
                cache: 'no-store',
            });
            clearTimeout(timeoutId);

            if (!resp.ok) {
                throw new Error(`HTTP ${resp.status}`);
            }

            const json = await resp.json();
            return this._parseIceConfig(json);
        } catch (e) {
            clearTimeout(timeoutId);
            if (e.name === 'AbortError') {
                throw new Error('ICE config fetch timed out');
            }
            throw e;
        }
    }

    /**
     * 解析服务器返回的 ICE 配置为 RTCIceServer 数组
     * @private
     */
    _parseIceConfig(json) {
        if (!json || !Array.isArray(json.iceServers)) {
            return [];
        }

        return json.iceServers.map(server => {
            const entry = { urls: server.urls };
            if (server.username) entry.username = server.username;
            if (server.credential) entry.credential = server.credential;
            return entry;
        });
    }
}

/**
 * 将 WebSocket URL 转换为 HTTP URL
 */
function wsUrlToHttpUrl(wsUrl) {
    let httpUrl = wsUrl;
    if (httpUrl.startsWith('wss://')) {
        httpUrl = 'https://' + httpUrl.slice(6);
    } else if (httpUrl.startsWith('ws://')) {
        httpUrl = 'http://' + httpUrl.slice(5);
    }
    while (httpUrl.endsWith('/')) {
        httpUrl = httpUrl.slice(0, -1);
    }
    return httpUrl;
}

export { BUILTIN_STUN_URLS };
