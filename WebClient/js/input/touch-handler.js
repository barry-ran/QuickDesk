/**
 * touch-handler.js - 手机触摸输入处理
 *
 * 参考 Chrome Remote Desktop / Microsoft RD Client 的交互模式：
 *
 *  1x 缩放（默认）：
 *    单指拖动  → 移动鼠标光标（触控板模式）
 *    单击      → 左键点击
 *    双击      → 双击
 *    双击后拖动 → 按住左键拖拽
 *    长按(500ms) → 右键点击
 *    双指滑动  → 滚轮（所有缩放级别）
 *    双指捏合  → 缩放画面
 *
 *  放大状态（scale > 1）：
 *    单指拖动  → 移动光标，画面自动跟随
 *    单击/双击/长按 → 同上（在当前光标位置操作）
 *    双指滑动 → 滚轮；双指捏合 → 继续缩放/平移
 */

import { MouseButton } from '../protocol/protobuf-messages.js';

const LONG_PRESS_TIMEOUT = 500;
const DOUBLE_TAP_GAP = 300;
const DOUBLE_TAP_DISTANCE = 48;
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

        this._touchStartPos = null;
        this._lastSinglePos = null;
        this._lastTapTime = 0;
        this._lastTapPos = null;
        this._secondTap = false;
        this._dragging = false;
        this._dragArmed = false;
        this.onDragStateChanged = null;
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
        this._gestureBlocked = false;
        this._twoFingerMode = null;
    }

    setRemoteResolution(w, h) {
        this.cancelGesture();
        this.remoteWidth = w;
        this.remoteHeight = h;
        if (this._cursorX === 0 && this._cursorY === 0 && w > 0 && h > 0) {
            this._cursorX = Math.round(w / 2);
            this._cursorY = Math.round(h / 2);
        }
        this._cursorX = Math.max(0, Math.min(w - 1, this._cursorX));
        this._cursorY = Math.max(0, Math.min(h - 1, this._cursorY));
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
        h.touchcancel = (e) => this._onTouchCancel(e);

        const opts = { passive: false };
        this.target.addEventListener('touchstart', h.touchstart, opts);
        this.target.addEventListener('touchmove', h.touchmove, opts);
        this.target.addEventListener('touchend', h.touchend, opts);
        this.target.addEventListener('touchcancel', h.touchcancel, opts);
        h.blur = () => this.cancelGesture();
        h.visibility = () => { if (document.hidden) this.cancelGesture(); };
        h.resize = () => {
            this.cancelGesture();
            this._applyTransform();
            this._updateCursorElement();
            if (this.isZoomed) this._autoPanToFollow();
        };
        if (typeof ResizeObserver !== 'undefined') {
            let previous = this.video.parentElement.getBoundingClientRect();
            this._resizeObserver = new ResizeObserver(() => {
                const current = this.video.parentElement.getBoundingClientRect();
                if (current.width === previous.width && current.height === previous.height) return;
                previous = current;
                h.resize();
            });
            this._resizeObserver.observe(this.video.parentElement);
        }
        window.addEventListener('blur', h.blur);
        document.addEventListener('visibilitychange', h.visibility);
        window.addEventListener('resize', h.resize);
        window.visualViewport?.addEventListener('resize', h.resize);
        this._updateCursorElement();
    }

    disable() {
        if (!this._enabled) return;
        this._enabled = false;
        const h = this._handlers;
        this.target.removeEventListener('touchstart', h.touchstart);
        this.target.removeEventListener('touchmove', h.touchmove);
        this.target.removeEventListener('touchend', h.touchend);
        this.target.removeEventListener('touchcancel', h.touchcancel);
        this._resizeObserver?.disconnect();
        this._resizeObserver = null;
        window.removeEventListener('blur', h.blur);
        document.removeEventListener('visibilitychange', h.visibility);
        window.removeEventListener('resize', h.resize);
        window.visualViewport?.removeEventListener('resize', h.resize);
        this._cancelActiveGesture();
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
        if (this._activeTouches === 0) this._gestureBlocked = false;
        this._activeTouches = e.touches.length;
        if (e.touches.length > 2) {
            this.cancelGesture();
            return;
        }
        if (this._gestureBlocked) return;

        if (e.touches.length === 1) {
            const t = e.touches[0];
            this._touchStartPos = { x: t.clientX, y: t.clientY };
            this._lastSinglePos = { x: t.clientX, y: t.clientY };
            this._moved = false;
            this._secondTap = this._isSecondTap(t);
            if (!this._secondTap && !this._dragArmed) {
                this._lastTapTime = 0;
                this._lastTapPos = null;
                this._startLongPress();
            }
        } else if (e.touches.length === 2) {
            this._cancelLongPress();
            if (this._dragging) {
                this._sendButton(MouseButton.BUTTON_LEFT, false);
                this._dragging = false;
            }
            this._dragArmed = false;
            this.onDragStateChanged?.(false);
            this._secondTap = false;
            this._lastTapTime = 0;
            this._lastTapPos = null;
            this._moved = true;
            const t0 = e.touches[0], t1 = e.touches[1];
            this._lastTwoFingerX = (t0.clientX + t1.clientX) / 2;
            this._lastTwoFingerY = (t0.clientY + t1.clientY) / 2;
            this._twoFingerMode = null;
            this._pinchStartDist = this._touchDist(t0, t1);
            this._pinchStartScale = this._scale;
            this._pinchStartCenter = { x: this._lastTwoFingerX, y: this._lastTwoFingerY };
            this._pinchStartTranslate = { x: this._translateX, y: this._translateY };
        }
    }

    _onTouchMove(e) {
        e.preventDefault();
        if (this._gestureBlocked) return;

        if (e.touches.length === 1 && this._activeTouches === 1) {
            const t = e.touches[0];
            const dx = t.clientX - this._lastSinglePos.x;
            const dy = t.clientY - this._lastSinglePos.y;
            this._lastSinglePos = { x: t.clientX, y: t.clientY };

            if (Math.hypot(
                t.clientX - this._touchStartPos.x,
                t.clientY - this._touchStartPos.y,
            ) > MOVE_THRESHOLD) {
                this._moved = true;
                this._cancelLongPress();
            }

            if (this._moved) {
                // A second tap followed by movement is a left-button drag.
                // The first tap already produced the first click, so this
                // preserves the platform-standard double-click sequence.
                if ((this._secondTap || this._dragArmed) && !this._dragging) {
                    this._sendButton(MouseButton.BUTTON_LEFT, true);
                    this._dragging = true;
                    this.onDragStateChanged?.(true);
                }
                this._moveCursor(dx * CURSOR_SPEED / this._scale, dy * CURSOR_SPEED / this._scale);
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
            if (!this._twoFingerMode) {
                const distanceChange = Math.abs(dist - this._pinchStartDist);
                const centerChange = Math.hypot(cx - this._pinchStartCenter.x, cy - this._pinchStartCenter.y);
                // Lock intent until every finger lifts: never scroll while pinching.
                if (distanceChange >= Math.max(8, this._pinchStartDist * 0.05)) {
                    this._twoFingerMode = 'pinch';
                } else if (centerChange >= MOVE_THRESHOLD) {
                    this._twoFingerMode = 'scroll';
                }
            }
            if (this._twoFingerMode === 'pinch' && this._pinchStartDist > 0) {
                const newScale = Math.max(1, Math.min(5, this._pinchStartScale * dist / this._pinchStartDist));
                const rect = this.video.parentElement.getBoundingClientRect();
                const centerX = rect.left + rect.width / 2;
                const centerY = rect.top + rect.height / 2;
                const ratio = newScale / this._pinchStartScale;
                this._translateX = cx - centerX - (this._pinchStartCenter.x - centerX - this._pinchStartTranslate.x) * ratio;
                this._translateY = cy - centerY - (this._pinchStartCenter.y - centerY - this._pinchStartTranslate.y) * ratio;
                this._scale = newScale;
                this._applyTransform();
                this._updateCursorElement();
            } else if (this._twoFingerMode === 'scroll' && (scrollDx || scrollDy)) {
                this._sendScroll(scrollDx, scrollDy);
            }
        }
    }

    _onTouchEnd(e) {
        e.preventDefault();
        this._cancelLongPress();
        if (this._gestureBlocked) {
            this._activeTouches = e.touches.length;
            if (!e.touches.length) this._gestureBlocked = false;
            return;
        }

        if (this._activeTouches === 1 && e.touches.length === 0) {
            if (this._dragging) {
                this._sendButton(MouseButton.BUTTON_LEFT, false);
                this._dragging = false;
                this._secondTap = false;
                this._dragArmed = false;
                this.onDragStateChanged?.(false);
            } else if (this._dragArmed) {
                this._dragArmed = false;
                this.onDragStateChanged?.(false);
            } else if (!this._moved) {
                // Send each click immediately. The host receives two normal
                // left clicks for a double-click instead of a platform-
                // specific button latch sequence.
                this._sendClick(MouseButton.BUTTON_LEFT);
                if (this._secondTap) {
                    this._lastTapTime = 0;
                    this._lastTapPos = null;
                } else {
                    this._lastTapTime = Date.now();
                    this._lastTapPos = { ...this._touchStartPos };
                }
                this._secondTap = false;
            }
        } else if (this._activeTouches > 1 && e.touches.length === 1) {
            // Do not turn the final finger of a pinch/scroll into a click or move.
            this._gestureBlocked = true;
        }

        if (e.touches.length === 0) {
            this._activeTouches = 0;
        } else {
            this._activeTouches = e.touches.length;
        }
    }

    _onTouchCancel(e) {
        e.preventDefault();
        this.cancelGesture();
        this._activeTouches = e.touches.length;
    }

    _isSecondTap(touch) {
        if (!this._lastTapTime || !this._lastTapPos ||
            Date.now() - this._lastTapTime >= DOUBLE_TAP_GAP) {
            return false;
        }
        const dx = touch.clientX - this._lastTapPos.x;
        const dy = touch.clientY - this._lastTapPos.y;
        return Math.hypot(dx, dy) <= DOUBLE_TAP_DISTANCE;
    }

    get isDragging() { return this._dragging || this._dragArmed; }

    click(button = MouseButton.BUTTON_LEFT) {
        this.cancelGesture();
        this._sendClick(button);
    }

    // Accessible alternative to double-tap-and-drag: arm, swipe, lift to release.
    toggleDrag() {
        const arm = !this.isDragging;
        this.cancelGesture();
        this._dragArmed = arm;
        this.onDragStateChanged?.(arm);
    }

    cancelGesture() {
        const activeTouches = this._activeTouches;
        this._cancelActiveGesture();
        this._activeTouches = activeTouches;
        this._gestureBlocked = activeTouches > 0;
    }

    _cancelActiveGesture() {
        this._cancelLongPress();
        if (this._dragging) {
            this._sendButton(MouseButton.BUTTON_LEFT, false);
        }
        this._dragging = false;
        this._dragArmed = false;
        this.onDragStateChanged?.(false);
        this._secondTap = false;
        this._lastTapTime = 0;
        this._lastTapPos = null;
        this._moved = false;
        this._activeTouches = 0;
        this._twoFingerMode = null;
    }

    // ==================== Cursor & input ====================

    _moveCursor(dx, dy) {
        if (this.remoteWidth <= 0 || this.remoteHeight <= 0) return;
        this._cursorX = Math.max(0, Math.min(this.remoteWidth - 1, this._cursorX + dx));
        this._cursorY = Math.max(0, Math.min(this.remoteHeight - 1, this._cursorY + dy));

        this.dcHandler.sendMouseEvent({
            x: Math.round(this._cursorX),
            y: Math.round(this._cursorY),
        });
        this._updateCursorElement();
    }

    _sendClick(button) {
        if (this.remoteWidth <= 0 || this.remoteHeight <= 0) return;
        const x = Math.round(this._cursorX);
        const y = Math.round(this._cursorY);
        this.dcHandler.sendMouseEvent({ x, y, button, buttonDown: true });
        this.dcHandler.sendMouseEvent({ x, y, button, buttonDown: false });
    }

    _sendButton(button, buttonDown) {
        this.dcHandler.sendMouseEvent({
            x: Math.round(this._cursorX),
            y: Math.round(this._cursorY),
            button,
            buttonDown,
        });
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

        const edgeMargin = Math.min(56, containerRect.width / 4, containerRect.height / 4);
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
        const fit = Math.min(rect.width / this.remoteWidth, rect.height / this.remoteHeight);
        const renderWidth = this.remoteWidth > 0 ? this.remoteWidth * fit : rect.width;
        const renderHeight = this.remoteHeight > 0 ? this.remoteHeight * fit : rect.height;
        const maxX = Math.max(0, (renderWidth * this._scale - rect.width) / 2);
        const maxY = Math.max(0, (renderHeight * this._scale - rect.height) / 2);
        this._translateX = Math.max(-maxX, Math.min(maxX, this._translateX));
        this._translateY = Math.max(-maxY, Math.min(maxY, this._translateY));
    }

    resetZoom() {
        this.cancelGesture();
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
