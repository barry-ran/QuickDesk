/**
 * connection-ui.js - 连接页面 UI 逻辑
 * 
 * 参照 RemoteControlPage.qml
 * 设备ID/访问码输入、连接状态显示、服务器地址配置
 */

export class ConnectionUI extends EventTarget {
    constructor() {
        super();
        this._elements = {};
    }

    /**
     * 初始化 UI 绑定
     */
    init() {
        this._elements = {
            serverUrl: document.getElementById('serverUrl'),
            deviceId: document.getElementById('deviceId'),
            accessCode: document.getElementById('accessCode'),
            connectBtn: document.getElementById('connectBtn'),
            disconnectBtn: document.getElementById('disconnectBtn'),
            status: document.getElementById('statusText'),
            statusDot: document.getElementById('statusDot'),
            connectPage: document.getElementById('connectPage'),
            remotePage: document.getElementById('remotePage'),
            logContainer: document.getElementById('logContainer'),
        };

        // 回车键支持
        this._elements.deviceId?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this._elements.accessCode?.focus();
        });

        this._elements.accessCode?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this._onConnect();
        });

        this._elements.connectBtn?.addEventListener('click', () => this._onConnect());
        this._elements.disconnectBtn?.addEventListener('click', () => this._onDisconnect());

        // 加载保存的服务器地址
        const savedUrl = localStorage.getItem('quickdesk_signaling_url');
        if (savedUrl && this._elements.serverUrl) {
            this._elements.serverUrl.value = savedUrl;
        }
    }

    /**
     * 连接按钮点击
     * @private
     */
    _onConnect() {
        const serverUrl = this._elements.serverUrl?.value?.trim() || 'ws://localhost:8000';
        const deviceId = this._elements.deviceId?.value?.trim();
        const accessCode = this._elements.accessCode?.value?.trim();

        if (!deviceId || !accessCode) {
            this.addLog('请输入设备 ID 和访问码', 'error');
            return;
        }

        // 保存服务器地址
        localStorage.setItem('quickdesk_signaling_url', serverUrl);

        this.setConnecting();
        this.dispatchEvent(new CustomEvent('connect', {
            detail: { serverUrl, deviceId, accessCode }
        }));
    }

    /**
     * 断开按钮点击
     * @private
     */
    _onDisconnect() {
        this.dispatchEvent(new CustomEvent('disconnect'));
    }

    /**
     * 设置连接中状态
     */
    setConnecting() {
        this._setStatus('connecting', '连接中...');
        if (this._elements.connectBtn) this._elements.connectBtn.disabled = true;
        if (this._elements.disconnectBtn) this._elements.disconnectBtn.disabled = false;
    }

    /**
     * 设置已连接状态
     */
    setConnected() {
        this._setStatus('connected', '已连接');
        if (this._elements.connectBtn) this._elements.connectBtn.disabled = true;
        if (this._elements.disconnectBtn) this._elements.disconnectBtn.disabled = false;
    }

    /**
     * 设置断开状态
     */
    setDisconnected(reason = '') {
        this._setStatus('disconnected', reason || '未连接');
        if (this._elements.connectBtn) this._elements.connectBtn.disabled = false;
        if (this._elements.disconnectBtn) this._elements.disconnectBtn.disabled = true;
    }

    /**
     * 设置失败状态
     */
    setFailed(reason = '') {
        this._setStatus('failed', reason || '连接失败');
        if (this._elements.connectBtn) this._elements.connectBtn.disabled = false;
        if (this._elements.disconnectBtn) this._elements.disconnectBtn.disabled = true;
    }

    /**
     * 设置状态
     * @private
     */
    _setStatus(status, text) {
        if (this._elements.statusDot) {
            this._elements.statusDot.className = `status-dot ${status}`;
        }
        if (this._elements.status) {
            this._elements.status.textContent = text;
        }
    }

    /**
     * 切换到远程桌面视图
     */
    showRemotePage() {
        if (this._elements.connectPage) this._elements.connectPage.style.display = 'none';
        if (this._elements.remotePage) this._elements.remotePage.style.display = 'flex';
    }

    /**
     * 切换到连接页面
     */
    showConnectPage() {
        if (this._elements.connectPage) this._elements.connectPage.style.display = 'flex';
        if (this._elements.remotePage) this._elements.remotePage.style.display = 'none';
    }

    /**
     * 添加日志
     * @param {string} message 
     * @param {string} [level='info'] - info, success, error, warning
     */
    addLog(message, level = 'info') {
        const container = this._elements.logContainer;
        if (!container) return;

        const time = new Date().toLocaleTimeString();
        const entry = document.createElement('div');
        entry.className = `log-entry log-${level}`;
        entry.innerHTML = `<span class="log-time">[${time}]</span> ${message}`;
        
        container.appendChild(entry);
        container.scrollTop = container.scrollHeight;

        // 限制日志条目数量
        while (container.children.length > 500) {
            container.removeChild(container.firstChild);
        }
    }

    /**
     * 清理
     */
    destroy() {
        // nothing to clean up
    }
}
