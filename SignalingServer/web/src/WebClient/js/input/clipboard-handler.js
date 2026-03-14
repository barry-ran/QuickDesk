/**
 * clipboard-handler.js - 剪贴板同步
 * 
 * 使用 Clipboard API 监听本地剪贴板变化，
 * 通过 control DataChannel 与远程主机同步
 */

export class ClipboardHandler {
    /**
     * @param {DataChannelHandler} dataChannelHandler 
     */
    constructor(dataChannelHandler) {
        this.dcHandler = dataChannelHandler;
        this._enabled = false;
        this._lastClipboardText = '';
        this._pollInterval = null;
    }

    /**
     * 启用剪贴板同步
     */
    enable() {
        if (this._enabled) return;
        this._enabled = true;

        // 监听来自 Host 的剪贴板事件
        this.dcHandler.addEventListener('clipboardEvent', this._onRemoteClipboard.bind(this));

        // 定时轮询本地剪贴板 (Clipboard API 没有 change 事件)
        this._pollInterval = setInterval(() => this._pollClipboard(), 1000);
    }

    /**
     * 禁用剪贴板同步
     */
    disable() {
        if (!this._enabled) return;
        this._enabled = false;

        if (this._pollInterval) {
            clearInterval(this._pollInterval);
            this._pollInterval = null;
        }
    }

    /**
     * 轮询本地剪贴板
     * @private
     */
    async _pollClipboard() {
        try {
            // 需要文档有焦点才能读取剪贴板
            if (!document.hasFocus()) return;

            const text = await navigator.clipboard.readText();
            if (text && text !== this._lastClipboardText) {
                this._lastClipboardText = text;
                this._sendToRemote(text);
            }
        } catch (e) {
            // 剪贴板权限被拒绝，静默处理
        }
    }

    /**
     * 发送剪贴板内容到远程
     * @private
     */
    _sendToRemote(text) {
        const encoder = new TextEncoder();
        this.dcHandler.sendClipboardEvent({
            mimeType: 'text/plain',
            data: encoder.encode(text),
        });
    }

    /**
     * 处理来自远程的剪贴板事件
     * @private
     */
    async _onRemoteClipboard(event) {
        const { mimeType, data } = event.detail;
        
        if (mimeType === 'text/plain' && data) {
            const text = new TextDecoder().decode(data);
            this._lastClipboardText = text; // 防止回环
            
            try {
                await navigator.clipboard.writeText(text);
            } catch (e) {
                console.warn('[Clipboard] Failed to write to clipboard:', e);
            }
        }
    }

    /**
     * 清理
     */
    destroy() {
        this.disable();
    }
}
