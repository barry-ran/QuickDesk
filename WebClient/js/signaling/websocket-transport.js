/**
 * websocket-transport.js - 信令 WebSocket 传输层
 *
 * 协议（§2.13 / §2.6）：
 *   1. 建立 WS 到 /v1/realtime/signal（URL 不带 token/code）
 *   2. 客户端首帧发送 {type:"auth", signal_token, role:"client", device_id, client_id}
 *   3. 服务端回 {type:"auth_ok"} 后才允许发送 SDP/ICE jingle XML
 *   4. auth_ok 5s 超时则服务端会主动 close(4401)
 *   5. signal_token 一次性，WS auth_ok 后服务端立即 DEL
 *
 * 旧版 URL 里带 access_code 的做法已废弃（第 1.3 节缺陷 #1/#3）。
 */

const AUTH_TIMEOUT_MS = 6000; // 稍大于服务端 5s，给 RTT 留余量

export class WebSocketTransport {
    /**
     * @param {object} options
     * @param {string} options.signalingUrl  - 信令服务器基础 URL
     * @param {Function} options.onMessage   - 业务消息回调 (message: string)
     * @param {Function} options.onOpen      - WS 连接成功（尚未 auth_ok）
     * @param {Function} options.onAuthOk    - 收到 auth_ok，可以发 SDP
     * @param {Function} options.onClose     - (code, reason)
     * @param {Function} options.onError     - (error)
     */
    constructor(options) {
        this.signalingUrl = options.signalingUrl || 'ws://localhost:8000';
        this.onMessage = options.onMessage || (() => {});
        this.onOpen    = options.onOpen    || (() => {});
        this.onAuthOk  = options.onAuthOk  || (() => {});
        this.onClose   = options.onClose   || (() => {});
        this.onError   = options.onError   || (() => {});

        this.ws = null;
        this._authOk = false;
        this._closed = false;
        this._authTimer = null;

        // Auth params for the first-frame handshake.
        this._deviceId = null;
        this._signalToken = null;
        this._clientId = null;
        this._role = 'client';
    }

    /**
     * @param {object} params
     * @param {string} params.deviceId
     * @param {string} params.signalToken  一次性 signal_token
     * @param {string} params.clientId     client 角色必填
     * @param {string} [params.role]       默认 'client'
     */
    connect(params) {
        this._deviceId    = params.deviceId;
        this._signalToken = params.signalToken;
        this._clientId    = params.clientId || '';
        this._role        = params.role || 'client';
        this._closed = false;
        this._authOk = false;

        return this._doConnect();
    }

    _doConnect() {
        return new Promise((resolve, reject) => {
            let base = this.signalingUrl.replace(/\/+$/, '');
            // §2.13: URL carries NO token.
            const wsUrl = `${base}/v1/realtime/signal`;
            console.log(`[WebSocket] Connecting to: ${wsUrl}`);

            try {
                this.ws = new WebSocket(wsUrl);
            } catch (e) {
                reject(new Error(`Failed to create WebSocket: ${e.message}`));
                return;
            }

            this._resolve = resolve;
            this._reject  = reject;

            this.ws.onopen = () => {
                console.log('[WebSocket] Opened, sending auth frame');
                this.onOpen();
                this._sendAuthFrame();
                // Abandon if auth_ok doesn't arrive in time.
                this._authTimer = setTimeout(() => {
                    if (!this._authOk && !this._closed) {
                        console.error('[WebSocket] auth_ok timeout');
                        this._fail(new Error('Auth timeout'));
                    }
                }, AUTH_TIMEOUT_MS);
            };

            this.ws.onerror = (event) => {
                console.error('[WebSocket] Error:', event);
                this.onError(event);
                // onclose will follow and handle cleanup + rejection.
            };

            this.ws.onmessage = (event) => this._handleMessage(event.data);

            this.ws.onclose = (event) => {
                const code = event && event.code;
                const reason = event && event.reason;
                console.log(`[WebSocket] Closed: code=${code}, reason=${reason}`);
                clearTimeout(this._authTimer); this._authTimer = null;
                this.onClose(code, reason);
                if (!this._authOk && this._reject) {
                    // Closed before auth succeeded — treat as connect failure.
                    this._reject(new Error(`WS closed before auth (code=${code})`));
                    this._reject = this._resolve = null;
                }
                this.ws = null;
            };
        });
    }

    _sendAuthFrame() {
        const frame = {
            type: 'auth',
            signal_token: this._signalToken,
            role: this._role,
            device_id: this._deviceId,
        };
        if (this._clientId) frame.client_id = this._clientId;
        try { this.ws.send(JSON.stringify(frame)); }
        catch (e) { this._fail(e); }
    }

    _handleMessage(data) {
        const message = typeof data === 'string' ? data : data.toString();
        const trimmed = message.trim();

        // Prefer JSON control frames. Jingle XML is routed straight through.
        if (trimmed.startsWith('{')) {
            let json;
            try { json = JSON.parse(trimmed); }
            catch { /* not JSON; fall through to XML */ }
            if (json && typeof json === 'object') {
                if (json.type === 'auth_ok') {
                    this._authOk = true;
                    clearTimeout(this._authTimer); this._authTimer = null;
                    console.log('[WebSocket] auth_ok');
                    this.onAuthOk(json);
                    if (this._resolve) { this._resolve(); this._resolve = this._reject = null; }
                    return;
                }
                if (json.type === 'error') {
                    const code = json.code || (json.data && json.data.code) || '';
                    console.warn(`[WebSocket] server error code=${code}`);
                    // Surface to session as a message for state-machine logic
                    // (PEER_DISCONNECTED / HOST_OFFLINE etc. — §2.15).
                    this.onMessage(trimmed);
                    return;
                }
                // JSON envelope carrying Jingle XML: {payload:"<iq...>", client_id:"..."}
                // Unwrap and pass the XML payload to the session layer.
                if (json.payload && typeof json.payload === 'string') {
                    if (!this._authOk) {
                        console.warn('[WebSocket] Dropping pre-auth payload');
                        return;
                    }
                    this.onMessage(json.payload);
                    return;
                }
                // Other JSON control frames (session.revoked etc.).
                this.onMessage(trimmed);
                return;
            }
        }
        // XML (Jingle) — ignore if auth hasn't completed (server shouldn't
        // send these but guard anyway per §2.13).
        if (!this._authOk) {
            console.warn('[WebSocket] Dropping pre-auth payload');
            return;
        }
        this.onMessage(trimmed);
    }

    /**
     * Send a payload. Refuses to send SDP/ICE before auth_ok (§2.13).
     * Wraps XML (Jingle) messages in the JSON envelope {payload, client_id}
     * expected by the v1 signaling server (Chromium host parses this format).
     */
    send(message) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.error('[WebSocket] Not connected, cannot send message');
            return false;
        }
        if (!this._authOk) {
            console.error('[WebSocket] auth not complete, refusing to send');
            return false;
        }
        // Wrap in JSON envelope matching Chromium client's SendJingleEnvelope format.
        const envelope = JSON.stringify({
            client_id: this._clientId || '',
            payload: message
        });
        this.ws.send(envelope);
        return true;
    }

    isConnected() { return !!(this.ws && this.ws.readyState === WebSocket.OPEN && this._authOk); }

    _fail(err) {
        this._closed = true;
        clearTimeout(this._authTimer); this._authTimer = null;
        try { this.ws && this.ws.close(); } catch { /* noop */ }
        if (this._reject) { this._reject(err); this._reject = this._resolve = null; }
    }

    disconnect() {
        this._closed = true;
        clearTimeout(this._authTimer); this._authTimer = null;
        if (this.ws) {
            this.ws.onclose = null;
            try { this.ws.close(); } catch { /* noop */ }
            this.ws = null;
        }
    }
}
