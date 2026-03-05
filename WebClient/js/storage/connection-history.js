/**
 * connection-history.js - Connection history storage
 *
 * Stores successful device connections in localStorage (no passwords).
 */

import { t } from '../i18n.js';

const STORAGE_KEY = 'quickdesk_connection_history';
const MAX_DEVICES = 50;

export class ConnectionHistory {
    /**
     * @returns {Array<{deviceId: string, serverUrl: string, lastConnected: number, connectCount: number}>}
     */
    static getAll() {
        try {
            const raw = localStorage.getItem(STORAGE_KEY);
            const list = raw ? JSON.parse(raw) : [];
            list.sort((a, b) => b.lastConnected - a.lastConnected);
            return list;
        } catch {
            return [];
        }
    }

    static save(deviceId, serverUrl) {
        if (!deviceId) return;

        const list = ConnectionHistory.getAll();
        const existing = list.find(d => d.deviceId === deviceId);

        if (existing) {
            existing.lastConnected = Date.now();
            existing.connectCount = (existing.connectCount || 0) + 1;
            if (serverUrl) existing.serverUrl = serverUrl;
        } else {
            list.push({
                deviceId,
                serverUrl: serverUrl || '',
                lastConnected: Date.now(),
                connectCount: 1,
            });
        }

        list.sort((a, b) => b.lastConnected - a.lastConnected);

        while (list.length > MAX_DEVICES) {
            list.pop();
        }

        ConnectionHistory._save(list);
    }

    static remove(deviceId) {
        const list = ConnectionHistory.getAll().filter(d => d.deviceId !== deviceId);
        ConnectionHistory._save(list);
    }

    static clear() {
        localStorage.removeItem(STORAGE_KEY);
    }

    static formatTime(timestamp) {
        if (!timestamp) return '';
        const date = new Date(timestamp);
        const now = new Date();
        const diffMs = now - date;
        const diffMin = Math.floor(diffMs / 60000);

        if (diffMin < 1) return t('time.justNow');
        if (diffMin < 60) return t('time.minutesAgo', { n: diffMin });

        const diffHour = Math.floor(diffMin / 60);
        if (diffHour < 24) return t('time.hoursAgo', { n: diffHour });

        const diffDay = Math.floor(diffHour / 24);
        if (diffDay < 30) return t('time.daysAgo', { n: diffDay });

        return date.toLocaleDateString();
    }

    /** @private */
    static _save(list) {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(list));
    }
}
