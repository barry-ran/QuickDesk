/**
 * ice-server-storage.js - 用户自定义 ICE 服务器持久化
 *
 * 使用 localStorage 存储用户配置的 TURN/STUN 服务器列表
 */

const STORAGE_KEY = 'quickdesk_user_ice_servers';

export class IceServerStorage {
    /**
     * 获取用户配置的所有 ICE 服务器
     * @returns {Array<{urls: string, username?: string, credential?: string}>}
     */
    static getAll() {
        try {
            const raw = localStorage.getItem(STORAGE_KEY);
            return raw ? JSON.parse(raw) : [];
        } catch {
            return [];
        }
    }

    /**
     * 添加 TURN 服务器
     * @returns {boolean}
     */
    static addTurnServer(url, username, credential) {
        if (!url || !username || !credential) return false;
        if (!url.startsWith('turn:') && !url.startsWith('turns:')) return false;

        const servers = IceServerStorage.getAll();
        if (servers.some(s => s.urls === url)) return false;

        servers.push({ urls: url, username, credential });
        IceServerStorage._save(servers);
        return true;
    }

    /**
     * 添加 STUN 服务器
     * @returns {boolean}
     */
    static addStunServer(url) {
        if (!url) return false;
        if (!url.startsWith('stun:') && !url.startsWith('stuns:')) return false;

        const servers = IceServerStorage.getAll();
        if (servers.some(s => s.urls === url)) return false;

        servers.push({ urls: url });
        IceServerStorage._save(servers);
        return true;
    }

    /**
     * 移除服务器（按索引）
     */
    static removeAt(index) {
        const servers = IceServerStorage.getAll();
        if (index < 0 || index >= servers.length) return;
        servers.splice(index, 1);
        IceServerStorage._save(servers);
    }

    /**
     * 清空所有用户配置
     */
    static clear() {
        localStorage.removeItem(STORAGE_KEY);
    }

    /**
     * 判断是否有 TURN 服务器
     */
    static hasTurn() {
        return IceServerStorage.getAll().some(s => {
            const url = Array.isArray(s.urls) ? s.urls[0] : s.urls;
            return url && (url.startsWith('turn:') || url.startsWith('turns:'));
        });
    }

    /** @private */
    static _save(servers) {
        localStorage.setItem(STORAGE_KEY, JSON.stringify(servers));
    }
}
