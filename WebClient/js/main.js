/**
 * main.js - QuickDesk Web Client entry
 *
 * Navigation, connection history, settings.
 * Remote desktop sessions open in new tabs via window.open.
 */

import { ConnectionHistory } from './storage/connection-history.js';
import { t, applyI18n, getLocale, setLocale, getSupportedLocales } from './i18n.js';

class QuickDeskApp {
    constructor() {
        this._currentPage = 'remote';
    }

    init() {
        this._initLangSelector();
        applyI18n();

        this._initNavigation();
        this._initConnectForm();
        this._initSettings();
        this._renderHistory();

        this._loadSavedServerUrl();
        this._applyUrlParams();

        window.addEventListener('message', (e) => this._onMessage(e));
    }

    // ==================== Language ====================

    _initLangSelector() {
        const select = document.getElementById('langSelect');
        if (!select) return;
        select.value = getLocale();
        select.addEventListener('change', () => {
            setLocale(select.value);
            location.reload();
        });
    }

    // ==================== Navigation ====================

    _initNavigation() {
        document.querySelectorAll('.nav-item').forEach(item => {
            item.addEventListener('click', () => {
                const page = item.dataset.page;
                if (page) this._switchPage(page);
            });
        });
    }

    _switchPage(page) {
        this._currentPage = page;

        document.querySelectorAll('.nav-item').forEach(el => {
            el.classList.toggle('active', el.dataset.page === page);
        });

        document.querySelectorAll('.page').forEach(el => {
            el.classList.toggle('active', el.id === `page-${page}`);
        });
    }

    // ==================== Connect Form ====================

    _initConnectForm() {
        const connectBtn = document.getElementById('connectBtn');
        const deviceIdInput = document.getElementById('deviceId');
        const accessCodeInput = document.getElementById('accessCode');

        connectBtn?.addEventListener('click', () => this._onConnect());

        deviceIdInput?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') accessCodeInput?.focus();
        });

        accessCodeInput?.addEventListener('keypress', (e) => {
            if (e.key === 'Enter') this._onConnect();
        });
    }

    _loadSavedServerUrl() {
        const savedUrl = localStorage.getItem('quickdesk_signaling_url');
        const serverUrlInput = document.getElementById('serverUrl');
        if (savedUrl && serverUrlInput) {
            serverUrlInput.value = savedUrl;
        }
    }

    _applyUrlParams() {
        const params = new URLSearchParams(window.location.search);
        const server = params.get('server');
        const device = params.get('device');
        const code = params.get('code');

        if (server) {
            const serverUrlInput = document.getElementById('serverUrl');
            if (serverUrlInput) serverUrlInput.value = server;
        }
        if (device) {
            const deviceIdInput = document.getElementById('deviceId');
            if (deviceIdInput) deviceIdInput.value = device;
        }
        if (code) {
            const accessCodeInput = document.getElementById('accessCode');
            if (accessCodeInput) accessCodeInput.value = code;
        }
    }

    _onConnect() {
        const serverUrl = document.getElementById('serverUrl')?.value?.trim() || 'ws://localhost:8000';
        const deviceId = document.getElementById('deviceId')?.value?.trim();
        const accessCode = document.getElementById('accessCode')?.value?.trim();

        if (!deviceId || !accessCode) {
            this._showToast(t('connect.inputRequired'), 'error');
            return;
        }

        localStorage.setItem('quickdesk_signaling_url', serverUrl);

        const videoCodec = localStorage.getItem('quickdesk_video_codec') || 'H264';
        const params = new URLSearchParams({
            server: serverUrl,
            device: deviceId,
            code: accessCode,
            codec: videoCodec,
        });

        const remoteUrl = `remote.html?${params.toString()}`;
        const isMobile = 'ontouchstart' in window || navigator.maxTouchPoints > 0;
        if (isMobile) {
            window.location.href = remoteUrl;
        } else {
            window.open(remoteUrl, `quickdesk_${deviceId}`);
        }

        this._showToast(t('connect.connecting', { deviceId }), 'info');
    }

    // ==================== Connection History ====================

    _renderHistory() {
        const container = document.getElementById('historyList');
        if (!container) return;

        const devices = ConnectionHistory.getAll();
        if (devices.length === 0) {
            container.innerHTML = `
                <div class="empty-state">
                    <div class="empty-icon">📋</div>
                    <p>${t('history.empty')}</p>
                </div>`;
            return;
        }

        container.innerHTML = '';
        for (const device of devices) {
            const item = document.createElement('div');
            item.className = 'history-item';
            item.innerHTML = `
                <div class="history-icon">🖥️</div>
                <div class="history-info">
                    <div class="history-device">${this._escapeHtml(device.deviceId)}</div>
                    <div class="history-meta">
                        ${ConnectionHistory.formatTime(device.lastConnected)}
                        · ${t('history.connectCount', { count: device.connectCount || 1 })}
                    </div>
                </div>
                <div class="history-actions">
                    <button class="icon-btn" data-action="fill" title="${t('history.fill')}">↗</button>
                    <button class="icon-btn danger" data-action="delete" title="${t('history.delete')}">✕</button>
                </div>`;

            item.querySelector('[data-action="fill"]').addEventListener('click', (e) => {
                e.stopPropagation();
                this._fillFromHistory(device);
            });

            item.querySelector('[data-action="delete"]').addEventListener('click', (e) => {
                e.stopPropagation();
                ConnectionHistory.remove(device.deviceId);
                this._renderHistory();
                this._showToast(t('history.deleted'), 'info');
            });

            item.addEventListener('click', () => this._fillFromHistory(device));

            container.appendChild(item);
        }
    }

    _fillFromHistory(device) {
        const deviceIdInput = document.getElementById('deviceId');
        const serverUrlInput = document.getElementById('serverUrl');
        const accessCodeInput = document.getElementById('accessCode');

        if (deviceIdInput) deviceIdInput.value = device.deviceId;
        if (serverUrlInput && device.serverUrl) serverUrlInput.value = device.serverUrl;
        if (accessCodeInput) {
            accessCodeInput.value = '';
            accessCodeInput.focus();
        }
    }

    // ==================== Settings ====================

    _initSettings() {
        const settingsServerUrl = document.getElementById('settingsServerUrl');
        const connectServerUrl = document.getElementById('serverUrl');
        if (settingsServerUrl) {
            const saved = localStorage.getItem('quickdesk_signaling_url') || '';
            settingsServerUrl.value = saved || (connectServerUrl?.value || '');
            settingsServerUrl.addEventListener('change', () => {
                const url = settingsServerUrl.value.trim();
                if (url) {
                    localStorage.setItem('quickdesk_signaling_url', url);
                    if (connectServerUrl) connectServerUrl.value = url;
                    this._showToast(t('settings.signalingServerSaved'), 'info');
                }
            });
        }

        const codecSelect = document.getElementById('videoCodecSelect');
        if (codecSelect) {
            const saved = localStorage.getItem('quickdesk_video_codec') || 'H264';
            codecSelect.value = saved;
            codecSelect.addEventListener('change', () => {
                localStorage.setItem('quickdesk_video_codec', codecSelect.value);
                this._showToast(t('settings.codecChanged', { codec: codecSelect.value }), 'info');
            });
        }
    }

    // ==================== Messages from remote tabs ====================

    _onMessage(event) {
        if (!event.data || event.data.type !== 'quickdesk-connected') return;

        const { deviceId, serverUrl } = event.data;
        if (deviceId) {
            ConnectionHistory.save(deviceId, serverUrl);
            this._renderHistory();
        }
    }

    // ==================== Utils ====================

    _showToast(message, type = 'info') {
        const toast = document.getElementById('toast');
        if (!toast) return;

        toast.textContent = message;
        toast.className = `toast ${type} show`;

        clearTimeout(this._toastTimer);
        this._toastTimer = setTimeout(() => {
            toast.classList.remove('show');
        }, 3000);
    }

    _escapeHtml(str) {
        const div = document.createElement('div');
        div.textContent = str;
        return div.innerHTML;
    }
}

// ==================== App Startup ====================

document.addEventListener('DOMContentLoaded', () => {
    const app = new QuickDeskApp();
    app.init();
    window.quickdeskApp = app;
});
