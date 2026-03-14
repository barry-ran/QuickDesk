/**
 * connection-history.js - 历史连接设备记录
 *
 * 使用 localStorage 存储连接成功的设备记录（不保存密码）
 */

const STORAGE_KEY = 'quickdesk_connection_history';
const MAX_DEVICES = 50;

export class ConnectionHistory {
    /**
     * 获取所有历史设备（按最近连接时间降序排列）
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

    /**
     * 保存一次成功的连接（新增或更新）
     */
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

    /**
     * 移除一条记录
     */
    static remove(deviceId) {
        const list = ConnectionHistory.getAll().filter(d => d.deviceId !== deviceId);
        ConnectionHistory._save(list);
    }

    /**
     * 清空所有记录
     */
    static clear() {
        localStorage.removeItem(STORAGE_KEY);
    }

    /**
     * 格式化最近连接时间为可读字符串
     */
    static formatTime(timestamp) {
        if (!timestamp) return '';
        const date = new Date(timestamp);
        const now = new Date();
        const diffMs = now - date;
        const diffMin = Math.floor(diffMs / 60000);

        if (diffMin < 1) return '刚刚';
        if (diffMin < 60) return `${diffMin} 分钟前`;

        const diffHour = Math.floor(diffMin / 60);
        if (diffHour < 24) return `${diffHour} 小时前`;

        const diffDay = Math.floor(diffHour / 24);
        if (diffDay < 30) return `${diffDay} 天前`;

        return date.toLocaleDateString();
    }

    /** @private */
    static _save(list) {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(list));
    }
}
