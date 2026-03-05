/**
 * ice-config-fetcher.js - ICE configuration fetcher
 *
 * Fetches time-limited TURN/STUN credentials from the signaling server
 * at /api/v1/ice-config.
 */

const ICE_CONFIG_PATH = '/api/v1/ice-config';
const FETCH_TIMEOUT_MS = 3000;

export class IceConfigFetcher {
    /**
     * @param {string} signalingWsUrl - WebSocket signaling server URL (ws:// or wss://)
     */
    constructor(signalingWsUrl) {
        this._httpBaseUrl = wsUrlToHttpUrl(signalingWsUrl);
    }

    /**
     * Fetch ICE server configuration from the signaling server.
     * @returns {Promise<Array>} iceServers array for RTCPeerConnection
     */
    async getIceServers() {
        try {
            const fetched = await this._fetchFromServer();
            if (fetched && fetched.length > 0) {
                console.log(`[IceConfigFetcher] Fetched ${fetched.length} server(s) from signaling`);
                return fetched;
            }
        } catch (e) {
            console.warn('[IceConfigFetcher] Fetch failed:', e.message);
        }
        return [];
    }

    /** @private */
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

    /** @private */
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
