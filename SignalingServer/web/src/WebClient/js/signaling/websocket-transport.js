/**
 * websocket-transport.js - WebSocket 传输层
 * 
 * 参照 src/remoting/quickdesk/signaling/quickdesk_signal_strategy.cc
 * 管理与信令服务器的 WebSocket 连接
 */

export class WebSocketTransport {
    /**
     * @param {object} options
     * @param {string} options.signalingUrl - 信令服务器基础 URL (如 ws://localhost:8000)
     * @param {Function} options.onMessage - 消息回调 (message: string)
     * @param {Function} options.onOpen - 连接成功回调
     * @param {Function} options.onClose - 连接关闭回调 (code, reason)
     * @param {Function} options.onError - 错误回调 (error)
     */
    constructor(options) {
        this.signalingUrl = options.signalingUrl || 'ws://localhost:8000';
        this.onMessage = options.onMessage || (() => {});
        this.onOpen = options.onOpen || (() => {});
        this.onClose = options.onClose || (() => {});
        this.onError = options.onError || (() => {});
        
        this.ws = null;
        this.reconnectAttempts = 0;
        this.maxReconnectAttempts = 5;
        this.reconnectDelay = 1000;
        this.reconnectTimer = null;
        this._closed = false;
        this._deviceId = null;
        this._accessCode = null;
    }

    /**
     * 连接到信令服务器
     * 
     * @param {string} deviceId - 设备 ID
     * @param {string} accessCode - 访问码
     * @returns {Promise<void>}
     */
    connect(deviceId, accessCode) {
        this._deviceId = deviceId;
        this._accessCode = accessCode;
        this._closed = false;
        this.reconnectAttempts = 0;

        return this._doConnect();
    }

    /**
     * 执行连接
     * @private
     */
    _doConnect() {
        return new Promise((resolve, reject) => {
            const wsUrl = `${this.signalingUrl}/client/${this._deviceId}/${this._accessCode}`;
            console.log(`[WebSocket] Connecting to: ${wsUrl}`);

            try {
                this.ws = new WebSocket(wsUrl);
            } catch (e) {
                reject(new Error(`Failed to create WebSocket: ${e.message}`));
                return;
            }

            this.ws.onopen = () => {
                console.log('[WebSocket] Connected');
                this.reconnectAttempts = 0;
                this.onOpen();
                resolve();
            };

            this.ws.onerror = (event) => {
                console.error('[WebSocket] Error:', event);
                this.onError(event);
                reject(new Error('WebSocket connection failed'));
            };

            this.ws.onmessage = (event) => {
                this._handleMessage(event.data);
            };

            this.ws.onclose = (event) => {
                console.log(`[WebSocket] Closed: code=${event.code}, reason=${event.reason}`);
                this.onClose(event.code, event.reason);
                
                // 尝试重连
                if (!this._closed && this.reconnectAttempts < this.maxReconnectAttempts) {
                    this._scheduleReconnect();
                }
            };
        });
    }

    /**
     * 处理收到的消息
     * @private
     */
    _handleMessage(data) {
        // 信令服务器转发的消息可能是：
        // 1. JSON 控制消息 (如 set_temp_password)
        // 2. Jingle XML 消息
        // 参照 quickdesk_signal_strategy.cc OnWebSocketMessage
        
        const message = typeof data === 'string' ? data : data.toString();
        
        // 尝试检测消息类型
        const trimmed = message.trim();
        if (trimmed.startsWith('{')) {
            // JSON 消息
            try {
                const json = JSON.parse(trimmed);
                console.log('[WebSocket] Received JSON message:', json.type || 'unknown');
                this.onMessage(trimmed);
            } catch (e) {
                console.warn('[WebSocket] Failed to parse JSON, treating as XML');
                this.onMessage(trimmed);
            }
        } else if (trimmed.startsWith('<')) {
            // XML (Jingle) 消息
            console.log('[WebSocket] Received XML message');
            this.onMessage(trimmed);
        } else {
            console.warn('[WebSocket] Unknown message format');
            this.onMessage(trimmed);
        }
    }

    /**
     * 发送消息
     * @param {string} message - XML 或 JSON 字符串
     */
    send(message) {
        if (!this.ws || this.ws.readyState !== WebSocket.OPEN) {
            console.error('[WebSocket] Not connected, cannot send message');
            return false;
        }

        this.ws.send(message);
        return true;
    }

    /**
     * 检查是否已连接
     * @returns {boolean}
     */
    isConnected() {
        return this.ws && this.ws.readyState === WebSocket.OPEN;
    }

    /**
     * 安排重连
     * @private
     */
    _scheduleReconnect() {
        if (this._closed) return;
        
        this.reconnectAttempts++;
        const delay = this.reconnectDelay * Math.pow(2, this.reconnectAttempts - 1);
        console.log(`[WebSocket] Reconnecting in ${delay}ms (attempt ${this.reconnectAttempts}/${this.maxReconnectAttempts})`);
        
        this.reconnectTimer = setTimeout(async () => {
            if (this._closed) return;
            try {
                await this._doConnect();
            } catch (e) {
                console.error('[WebSocket] Reconnection failed:', e);
            }
        }, delay);
    }

    /**
     * 断开连接
     */
    disconnect() {
        this._closed = true;
        
        if (this.reconnectTimer) {
            clearTimeout(this.reconnectTimer);
            this.reconnectTimer = null;
        }
        
        if (this.ws) {
            this.ws.onclose = null; // 防止触发重连
            this.ws.close();
            this.ws = null;
        }
    }
}
