/**
 * touch-handler.js - 手机触摸输入处理
 *
 * 参考 Chrome Remote Desktop / Microsoft RD Client 的交互模式：
 *
 *  1x 缩放（默认）：
 *    单指拖动  → 移动鼠标光标（触控板模式）
 *    单击      → 左键点击
 *    双击      → 双击
 *    长按(500ms) → 右键点击
 *    双指滑动  → 滚轮
 *    双指捏合  → 缩放画面
 *
 *  放大状态（scale > 1）：
 *    单指拖动  → 平移画面（浏览放大区域）
 *    单击/双击/长按 → 同上（在当前光标位置操作）
 *    双指捏合  → 继续缩放
 */

import { MouseButton } from '../protocol/protobuf-messages.js';

const TAP_TIMEOUT = 200;
const LONG_PRESS_TIMEOUT = 500;
const DOUBLE_TAP_GAP = 300;
const MOVE_THRESHOLD = 8;
const CURSOR_SPEED = 2.5;

export class TouchHandler {
    constructor(targetElement, videoElement, dcHandler) {
        this.target = targetElement;
        this.video = videoElement;
        this.dcHandler = dcHandler;
        this._enabled = false;
        this._handlers = {};

        this.remoteWidth = 0;
        this.remoteHeight = 0;

        this._cursorX = 0;
        this._cursorY = 0;

        this._touchStartTime = 0;
        this._touchStartPos = null;
        this._lastSinglePos = null;
        this._lastTapTime = 0;
        this._longPressTimer = null;
        this._moved = false;
        this._activeTouches = 0;

        this._lastTwoFingerX = 0;
        this._lastTwoFingerY = 0;

        this._scale = 1;
        this._translateX = 0;
        this._translateY = 0;
        this._pinchStartDist = 0;
        this._pinchStartScale = 1;
        this._pinchStartCenter = null;
        this._pinchStartTranslate = null;

        this._cursorEl = null;
    }

    setRemoteResolution(w, h) {
        this.remoteWidth = w;
        this.remoteHeight = h;
        if (this._cursorX === 0 && this._cursorY === 0 && w > 0 && h > 0) {
            this._cursorX = Math.round(w / 2);
            this._cursorY = Math.round(h / 2);
        }
        this._updateCursorElement();
    }

    get isZoomed() {
        return this._scale > 1.05;
    }

    enable() {
        if (this._enabled) return;
        this._enabled = true;

        this._createCursorElement();

        const h = this._handlers;
        h.touchstart = (e) => this._onTouchStart(e);
        h.touchmove = (e) => this._onTouchMove(e);
        h.touchend = (e) => this._onTouchEnd(e);
        h.touchcancel = (e) => this._onTouchEnd(e);

        const opts = { passive: false };
        this.target.addEventListener('touchstart', h.touchstart, opts);
        this.target.addEventListener('touchmove', h.touchmove, opts);
        this.target.addEventListener('touchend', h.touchend, opts);
        this.target.addEventListener('touchcancel', h.touchcancel, opts);
    }

    disable() {
        if (!this._enabled) return;
        this._enabled = false;
        const h = this._handlers;
        this.target.removeEventListener('touchstart', h.touchstart);
        this.target.removeEventListener('touchmove', h.touchmove);
        this.target.removeEventListener('touchend', h.touchend);
        this.target.removeEventListener('touchcancel', h.touchcancel);
        this._cancelLongPress();
        this._removeCursorElement();
    }

    // ==================== Touch events ====================

    _onTouchStart(e) {
        e.preventDefault();
        // Blur any focused input to dismiss virtual keyboard
        const active = document.activeElement;
        if (active && (active.tagName === 'INPUT' || active.tagName === 'TEXTAREA')) {
            active.blur();
        }
        this._activeTouches = e.touches.length;

        if (e.touches.length === 1) {
            const t = e.touches[0];
            this._touchStartTime = Date.now();
            this._touchStartPos = { x: t.clientX, y: t.clientY };
            this._lastSinglePos = { x: t.clientX, y: t.clientY };
            this._moved = false;
            this._startLongPress();
        } else if (e.touches.length === 2) {
            this._cancelLongPress();
            this._moved = true;
            const t0 = e.touches[0], t1 = e.touches[1];
            this._lastTwoFingerX = (t0.clientX + t1.clientX) / 2;
            this._lastTwoFingerY = (t0.clientY + t1.clientY) / 2;
            this._pinchStartDist = this._touchDist(t0, t1);
            this._pinchStartScale = this._scale;
            this._pinchStartCenter = { x: this._lastTwoFingerX, y: this._lastTwoFingerY };
            this._pinchStartTranslate = { x: this._translateX, y: this._translateY };
        }
    }

    _onTouchMove(e) {
        e.preventDefault();

        if (e.touches.length === 1 && this._activeTouches === 1) {
            const t = e.touches[0];
            const dx = t.clientX - this._lastSinglePos.x;
            const dy = t.clientY - this._lastSinglePos.y;
            this._lastSinglePos = { x: t.clientX, y: t.clientY };

            if (Math.abs(t.clientX - this._touchStartPos.x) > MOVE_THRESHOLD ||
                Math.abs(t.clientY - this._touchStartPos.y) > MOVE_THRESHOLD) {
                this._moved = true;
                this._cancelLongPress();
            }

            if (this._moved) {
                this._moveCursor(dx * CURSOR_SPEED, dy * CURSOR_SPEED);
                if (this.isZoomed) {
                    this._autoPanToFollow();
                }
            }
        } else if (e.touches.length === 2) {
            const t0 = e.touches[0], t1 = e.touches[1];
            const cx = (t0.clientX + t1.clientX) / 2;
            const cy = (t0.clientY + t1.clientY) / 2;

            const scrollDx = cx - this._lastTwoFingerX;
            const scrollDy = cy - this._lastTwoFingerY;
            this._lastTwoFingerX = cx;
            this._lastTwoFingerY = cy;

            const dist = this._touchDist(t0, t1);
            if (this._pinchStartDist > 0) {
                const ratio = dist / this._pinchStartDist;
                const newScale = Math.max(1, Math.min(5, this._pinchStartScale * ratio));

                if (newScale > 1) {
                    const dcx = cx - this._pinchStartCenter.x;
                    const dcy = cy - this._pinchStartCenter.y;
                    this._translateX = this._pinchStartTranslate.x + dcx;
                    this._translateY = this._pinchStartTranslate.y + dcy;
                }

                this._scale = newScale;
                if (this._scale <= 1) {
                    this._translateX = 0;
                    this._translateY = 0;
                }
                this._applyTransform();
                this._updateCursorElement();
            }

            if (Math.abs(scrollDy) > 2 && !this.isZoomed) {
                this._sendScroll(scrollDx, scrollDy);
            }
        }
    }

    _onTouchEnd(e) {
        e.preventDefault();
        this._cancelLongPress();

        if (this._activeTouches === 1 && e.touches.length === 0) {
            const elapsed = Date.now() - this._touchStartTime;

            if (!this._moved && elapsed < TAP_TIMEOUT) {
                const now = Date.now();
                if (now - this._lastTapTime < DOUBLE_TAP_GAP) {
                    this._sendDoubleClick();
                    this._lastTapTime = 0;
                } else {
                    this._lastTapTime = now;
                    this._sendClick(MouseButton.BUTTON_LEFT);
                }
            }
        }

        if (e.touches.length === 0) {
            this._activeTouches = 0;
        } else {
            this._activeTouches = e.touches.length;
        }
    }

    // ==================== Cursor & input ====================

    _moveCursor(dx, dy) {
        this._cursorX = Math.max(0, Math.min(this.remoteWidth - 1, this._cursorX + dx));
        this._cursorY = Math.max(0, Math.min(this.remoteHeight - 1, this._cursorY + dy));

        this.dcHandler.sendMouseEvent({
            x: Math.round(this._cursorX),
            y: Math.round(this._cursorY),
        });
        this._updateCursorElement();
    }

    _sendClick(button) {
        const x = Math.round(this._cursorX);
        const y = Math.round(this._cursorY);
        this.dcHandler.sendMouseEvent({ x, y, button, buttonDown: true });
        this.dcHandler.sendMouseEvent({ x, y, button, buttonDown: false });
    }

    _sendDoubleClick() {
        this._sendClick(MouseButton.BUTTON_LEFT);
        this._sendClick(MouseButton.BUTTON_LEFT);
    }

    _sendScroll(dx, dy) {
        this.dcHandler.sendMouseEvent({
            x: Math.round(this._cursorX),
            y: Math.round(this._cursorY),
            wheelDeltaX: dx * 3,
            wheelDeltaY: dy * 3,
            wheelTicksX: dx / 40,
            wheelTicksY: dy / 40,
        });
    }

    // ==================== Long press (right click) ====================

    _startLongPress() {
        this._cancelLongPress();
        this._longPressTimer = setTimeout(() => {
            this._longPressTimer = null;
            if (!this._moved) {
                this._sendClick(MouseButton.BUTTON_RIGHT);
                this._moved = true;
            }
        }, LONG_PRESS_TIMEOUT);
    }

    _cancelLongPress() {
        if (this._longPressTimer !== null) {
            clearTimeout(this._longPressTimer);
            this._longPressTimer = null;
        }
    }

    // ==================== Auto-pan to follow cursor (when zoomed) ====================

    _autoPanToFollow() {
        if (!this._cursorEl) return;
        const containerRect = this.video.parentElement.getBoundingClientRect();
        const cursorScreenX = parseFloat(this._cursorEl.style.left);
        const cursorScreenY = parseFloat(this._cursorEl.style.top);

        const edgeMargin = 40;
        let panX = 0, panY = 0;

        if (cursorScreenX < edgeMargin) {
            panX = edgeMargin - cursorScreenX;
        } else if (cursorScreenX > containerRect.width - edgeMargin) {
            panX = containerRect.width - edgeMargin - cursorScreenX;
        }
        if (cursorScreenY < edgeMargin) {
            panY = edgeMargin - cursorScreenY;
        } else if (cursorScreenY > containerRect.height - edgeMargin) {
            panY = containerRect.height - edgeMargin - cursorScreenY;
        }

        if (panX !== 0 || panY !== 0) {
            this._translateX += panX;
            this._translateY += panY;
            this._applyTransform();
            this._updateCursorElement();
        }
    }

    // ==================== Pinch zoom ====================

    _touchDist(t0, t1) {
        const dx = t0.clientX - t1.clientX;
        const dy = t0.clientY - t1.clientY;
        return Math.sqrt(dx * dx + dy * dy);
    }

    _applyTransform() {
        this._clampTranslate();
        this.video.style.transform = `translate(${this._translateX}px, ${this._translateY}px) scale(${this._scale})`;
        this.video.style.transformOrigin = 'center center';
    }

    _clampTranslate() {
        if (this._scale <= 1) {
            this._translateX = 0;
            this._translateY = 0;
            return;
        }
        const rect = this.video.parentElement.getBoundingClientRect();
        const maxX = (rect.width * (this._scale - 1)) / 2;
        const maxY = (rect.height * (this._scale - 1)) / 2;
        this._translateX = Math.max(-maxX, Math.min(maxX, this._translateX));
        this._translateY = Math.max(-maxY, Math.min(maxY, this._translateY));
    }

    resetZoom() {
        this._scale = 1;
        this._translateX = 0;
        this._translateY = 0;
        this.video.style.transform = '';
        this._updateCursorElement();
    }

    // ==================== Virtual cursor ====================

    _createCursorElement() {
        if (this._cursorEl) return;
        this._cursorEl = document.createElement('div');
        this._cursorEl.id = 'virtualCursor';
        Object.assign(this._cursorEl.style, {
            position: 'absolute',
            width: '20px',
            height: '20px',
            pointerEvents: 'none',
            zIndex: '90',
            transform: 'translate(-3px, -1px)',
            display: 'none',
        });
        // Standard arrow cursor SVG
        this._cursorEl.innerHTML = `<svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
            <path d="M3 2L3 17L7.5 12.5L11 19L13.5 18L10 11.5L16 11.5L3 2Z" fill="white" stroke="black" stroke-width="1.2" stroke-linejoin="round"/>
        </svg>`;

        const container = this.video.parentElement;
        if (container) container.appendChild(this._cursorEl);
    }

    _removeCursorElement() {
        if (this._cursorEl) {
            this._cursorEl.remove();
            this._cursorEl = null;
        }
    }

    /**
     * Map remote cursor coordinates to screen position within the video container,
     * accounting for object-fit:contain black bars and CSS zoom transform.
     */
    _updateCursorElement() {
        if (!this._cursorEl || this.remoteWidth <= 0 || this.remoteHeight <= 0) return;

        const rect = this.video.getBoundingClientRect();
        const containerRect = this.video.parentElement.getBoundingClientRect();
        const vw = this.video.videoWidth || this.remoteWidth;
        const vh = this.video.videoHeight || this.remoteHeight;
        if (vw <= 0 || vh <= 0) return;

        // Compute rendered video area within the (transformed) video element
        const containerAspect = rect.width / rect.height;
        const videoAspect = vw / vh;
        let renderW, renderH, offX, offY;
        if (containerAspect > videoAspect) {
            renderH = rect.height;
            renderW = renderH * videoAspect;
            offX = (rect.width - renderW) / 2;
            offY = 0;
        } else {
            renderW = rect.width;
            renderH = renderW / videoAspect;
            offX = 0;
            offY = (rect.height - renderH) / 2;
        }

        const screenX = rect.left + offX + (this._cursorX / this.remoteWidth) * renderW;
        const screenY = rect.top + offY + (this._cursorY / this.remoteHeight) * renderH;

        // Convert to container-relative coordinates
        const cx = screenX - containerRect.left;
        const cy = screenY - containerRect.top;

        this._cursorEl.style.left = `${cx}px`;
        this._cursorEl.style.top = `${cy}px`;
        this._cursorEl.style.display = 'block';
    }

    destroy() {
        this.disable();
        this.resetZoom();
    }
}
