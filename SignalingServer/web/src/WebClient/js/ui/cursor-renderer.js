/**
 * cursor-renderer.js - 远程光标渲染
 * 
 * 参照 RemoteDesktopView.qml 第 84-103 行
 * 解析 CursorShapeInfo protobuf 并渲染自定义光标
 */

export class CursorRenderer {
    /**
     * @param {HTMLElement} containerElement - 光标容器元素
     */
    constructor(containerElement) {
        this.container = containerElement;
        this._cursorCanvas = null;
        this._currentUrl = null;
    }

    /**
     * 更新光标形状
     * @param {object} cursorShape - { width, height, hotspotX, hotspotY, data }
     */
    updateCursor(cursorShape) {
        if (!cursorShape || !cursorShape.data || !cursorShape.width || !cursorShape.height) {
            return;
        }

        const { width, height, hotspotX, hotspotY, data } = cursorShape;

        // 创建 canvas 渲染光标
        if (!this._cursorCanvas) {
            this._cursorCanvas = document.createElement('canvas');
        }

        this._cursorCanvas.width = width;
        this._cursorCanvas.height = height;
        const ctx = this._cursorCanvas.getContext('2d');

        // 数据格式: BGRA (32-bit per pixel)
        const imageData = ctx.createImageData(width, height);
        const pixels = imageData.data;

        for (let i = 0; i < width * height; i++) {
            const srcOffset = i * 4;
            const dstOffset = i * 4;

            if (srcOffset + 3 < data.length) {
                // BGRA -> RGBA
                pixels[dstOffset] = data[srcOffset + 2];     // R <- B
                pixels[dstOffset + 1] = data[srcOffset + 1]; // G
                pixels[dstOffset + 2] = data[srcOffset];     // B <- R
                pixels[dstOffset + 3] = data[srcOffset + 3]; // A
            }
        }

        ctx.putImageData(imageData, 0, 0);

        // 生成 CSS cursor URL
        const dataUrl = this._cursorCanvas.toDataURL('image/png');
        
        // 释放旧的 URL
        if (this._currentUrl) {
            // Data URLs 不需要 revoke
        }
        this._currentUrl = dataUrl;

        // 设置 CSS cursor
        const hx = Math.max(0, Math.min(hotspotX || 0, width - 1));
        const hy = Math.max(0, Math.min(hotspotY || 0, height - 1));
        
        this.container.style.cursor = `url(${dataUrl}) ${hx} ${hy}, auto`;
    }

    /**
     * 隐藏本地光标
     */
    hideLocalCursor() {
        this.container.style.cursor = 'none';
    }

    /**
     * 恢复默认光标
     */
    restoreDefaultCursor() {
        this.container.style.cursor = '';
        this._currentUrl = null;
    }

    /**
     * 清理
     */
    destroy() {
        this.restoreDefaultCursor();
        this._cursorCanvas = null;
    }
}
