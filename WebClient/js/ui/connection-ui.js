/**
 * connection-ui.js - Connection page UI logic
 * 
 * Device ID/access code input, connection status display, server URL config.
 */

import { t } from '../i18n.js';

export class ConnectionUI extends EventTarget {
    constructor() {
        super();
        this._elements = {};
    }

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

        this._elements.deviceId?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this._elements.accessCode?.focus();
        });

        this._elements.accessCode?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this._onConnect();
        });

        this._elements.connectBtn?.addEventListener('click', () => this._onConnect());
        this._elements.disconnectBtn?.addEventListener('click', () => this._onDisconnect());

        const savedUrl = localStorage.getItem('quickdesk_signaling_url');
        if (savedUrl && this._elements.serverUrl) {
            this._elements.serverUrl.value = savedUrl;
        }
    }

    /** @private */
    _onConnect() {
        const serverUrl = this._elements.serverUrl?.value?.trim() || 'ws://localhost:8000';
        const deviceId = this._elements.deviceId?.value?.trim();
        const accessCode = this._elements.accessCode?.value?.trim();

        if (!deviceId || !accessCode) {
            this.addLog(t('connui.inputRequired'), 'error');
            return;
        }

        localStorage.setItem('quickdesk_signaling_url', serverUrl);

        this.setConnecting();
        this.dispatchEvent(new CustomEvent('connect', {
            detail: { serverUrl, deviceId, accessCode }
        }));
    }

    /** @private */
    _onDisconnect() {
        this.dispatchEvent(new CustomEvent('disconnect'));
    }

    setConnecting() {
        this._setStatus('connecting', t('connui.connecting'));
        if (this._elements.connectBtn) this._elements.connectBtn.disabled = true;
        if (this._elements.disconnectBtn) this._elements.disconnectBtn.disabled = false;
    }

    setConnected() {
        this._setStatus('connected', t('connui.connected'));
        if (this._elements.connectBtn) this._elements.connectBtn.disabled = true;
        if (this._elements.disconnectBtn) this._elements.disconnectBtn.disabled = false;
    }

    setDisconnected(reason = '') {
        this._setStatus('disconnected', reason || t('connui.disconnected'));
        if (this._elements.connectBtn) this._elements.connectBtn.disabled = false;
        if (this._elements.disconnectBtn) this._elements.disconnectBtn.disabled = true;
    }

    setFailed(reason = '') {
        this._setStatus('failed', reason || t('connui.failed'));
        if (this._elements.connectBtn) this._elements.connectBtn.disabled = false;
        if (this._elements.disconnectBtn) this._elements.disconnectBtn.disabled = true;
    }

    /** @private */
    _setStatus(status, text) {
        if (this._elements.statusDot) {
            this._elements.statusDot.className = `status-dot ${status}`;
        }
        if (this._elements.status) {
            this._elements.status.textContent = text;
        }
    }

    showRemotePage() {
        if (this._elements.connectPage) this._elements.connectPage.style.display = 'none';
        if (this._elements.remotePage) this._elements.remotePage.style.display = 'flex';
    }

    showConnectPage() {
        if (this._elements.connectPage) this._elements.connectPage.style.display = 'flex';
        if (this._elements.remotePage) this._elements.remotePage.style.display = 'none';
    }

    /**
     * @param {string} message 
     * @param {string} [level='info']
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

        while (container.children.length > 500) {
            container.removeChild(container.firstChild);
        }
    }

    destroy() {
        // nothing to clean up
    }
}
