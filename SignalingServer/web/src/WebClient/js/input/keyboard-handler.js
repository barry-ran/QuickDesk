/**
 * keyboard-handler.js - 键盘输入处理
 * 
 * 与 Qt native client (RemoteDesktopView.qml) 保持一致的策略：
 * 每个 keydown（含浏览器 auto-repeat）直接发 pressed:true，
 * keyup 发 pressed:false。不做任何客户端过滤或自驱动定时器。
 */

export class KeyboardHandler {
    /**
     * @param {HTMLElement} targetElement - 键盘事件目标元素
     * @param {DataChannelHandler} dataChannelHandler - DataChannel 处理器
     */
    constructor(targetElement, dataChannelHandler) {
        this.targetElement = targetElement;
        this.dcHandler = dataChannelHandler;
        this._enabled = false;
        this._handlers = {};
        this._pressedKeys = new Set();
    }

    enable() {
        if (this._enabled) return;
        this._enabled = true;

        this._handlers.keydown = (e) => this._onKeyDown(e);
        this._handlers.keyup = (e) => this._onKeyUp(e);
        this._handlers.blur = () => this._releaseAllKeys();

        this.targetElement.addEventListener('keydown', this._handlers.keydown);
        this.targetElement.addEventListener('keyup', this._handlers.keyup);
        this.targetElement.addEventListener('blur', this._handlers.blur);

        if (!this.targetElement.getAttribute('tabindex')) {
            this.targetElement.setAttribute('tabindex', '0');
        }
    }

    disable() {
        if (!this._enabled) return;
        this._enabled = false;

        this.targetElement.removeEventListener('keydown', this._handlers.keydown);
        this.targetElement.removeEventListener('keyup', this._handlers.keyup);
        this.targetElement.removeEventListener('blur', this._handlers.blur);

        this._releaseAllKeys();
    }

    /** @private */
    _onKeyDown(event) {
        const code = event.code;
        const usbKeycode = this._mapKeyCode(code);
        if (usbKeycode === 0) return;

        event.preventDefault();
        event.stopPropagation();

        this._pressedKeys.add(code);
        this.dcHandler.sendKeyEvent({
            pressed: true,
            usbKeycode: usbKeycode,
            ...this._getLockStates(event),
        });
    }

    /** @private */
    _onKeyUp(event) {
        const code = event.code;
        const usbKeycode = this._mapKeyCode(code);
        if (usbKeycode === 0) return;

        event.preventDefault();
        event.stopPropagation();

        this._pressedKeys.delete(code);

        this.dcHandler.sendKeyEvent({
            pressed: false,
            usbKeycode: usbKeycode,
            ...this._getLockStates(event),
        });
    }

    /** @private */
    _releaseAllKeys() {
        for (const code of this._pressedKeys) {
            const usbKeycode = this._mapKeyCode(code);
            if (usbKeycode !== 0) {
                this.dcHandler.sendKeyEvent({
                    pressed: false,
                    usbKeycode: usbKeycode,
                });
            }
        }
        this._pressedKeys.clear();
    }

    /**
     * Query keyboard lock states from the browser KeyboardEvent.
     * Uses the standard Web API getModifierState().
     * Returns fields matching event.proto: capsLockState (field 5),
     * numLockState (field 6).
     * @param {KeyboardEvent} event
     * @returns {Object}
     * @private
     */
    _getLockStates(event) {
        const result = {};
        if (typeof event.getModifierState === 'function') {
            result.capsLockState = event.getModifierState('CapsLock') ? 1 : 0;
            result.numLockState = event.getModifierState('NumLock') ? 1 : 0;
        }
        return result;
    }

    /**
     * 浏览器 KeyboardEvent.code -> USB HID keycode 映射
     * 
     * USB HID Usage Table: Page 0x07 (Keyboard/Keypad)
     * USB keycode = 0x070000 | usage_id
     * 
     * @param {string} code - KeyboardEvent.code
     * @returns {number} USB HID keycode (0 = unknown)
     */
    _mapKeyCode(code) {
        return KEY_CODE_MAP[code] || 0;
    }

    /**
     * 清理
     */
    destroy() {
        this.disable();
    }
}

// ==================== USB HID Keycode 映射表 ====================
// USB Page 0x07 (Keyboard)
// keycode = 0x070000 | usage_id

const USB_PAGE = 0x070000;

const KEY_CODE_MAP = {
    // Letters
    'KeyA': USB_PAGE | 0x04,
    'KeyB': USB_PAGE | 0x05,
    'KeyC': USB_PAGE | 0x06,
    'KeyD': USB_PAGE | 0x07,
    'KeyE': USB_PAGE | 0x08,
    'KeyF': USB_PAGE | 0x09,
    'KeyG': USB_PAGE | 0x0A,
    'KeyH': USB_PAGE | 0x0B,
    'KeyI': USB_PAGE | 0x0C,
    'KeyJ': USB_PAGE | 0x0D,
    'KeyK': USB_PAGE | 0x0E,
    'KeyL': USB_PAGE | 0x0F,
    'KeyM': USB_PAGE | 0x10,
    'KeyN': USB_PAGE | 0x11,
    'KeyO': USB_PAGE | 0x12,
    'KeyP': USB_PAGE | 0x13,
    'KeyQ': USB_PAGE | 0x14,
    'KeyR': USB_PAGE | 0x15,
    'KeyS': USB_PAGE | 0x16,
    'KeyT': USB_PAGE | 0x17,
    'KeyU': USB_PAGE | 0x18,
    'KeyV': USB_PAGE | 0x19,
    'KeyW': USB_PAGE | 0x1A,
    'KeyX': USB_PAGE | 0x1B,
    'KeyY': USB_PAGE | 0x1C,
    'KeyZ': USB_PAGE | 0x1D,

    // Numbers
    'Digit1': USB_PAGE | 0x1E,
    'Digit2': USB_PAGE | 0x1F,
    'Digit3': USB_PAGE | 0x20,
    'Digit4': USB_PAGE | 0x21,
    'Digit5': USB_PAGE | 0x22,
    'Digit6': USB_PAGE | 0x23,
    'Digit7': USB_PAGE | 0x24,
    'Digit8': USB_PAGE | 0x25,
    'Digit9': USB_PAGE | 0x26,
    'Digit0': USB_PAGE | 0x27,

    // Function keys
    'Escape': USB_PAGE | 0x29,
    'Backspace': USB_PAGE | 0x2A,
    'Tab': USB_PAGE | 0x2B,
    'Space': USB_PAGE | 0x2C,
    'Enter': USB_PAGE | 0x28,

    // Punctuation
    'Minus': USB_PAGE | 0x2D,
    'Equal': USB_PAGE | 0x2E,
    'BracketLeft': USB_PAGE | 0x2F,
    'BracketRight': USB_PAGE | 0x30,
    'Backslash': USB_PAGE | 0x31,
    'Semicolon': USB_PAGE | 0x33,
    'Quote': USB_PAGE | 0x34,
    'Backquote': USB_PAGE | 0x35,
    'Comma': USB_PAGE | 0x36,
    'Period': USB_PAGE | 0x37,
    'Slash': USB_PAGE | 0x38,
    'CapsLock': USB_PAGE | 0x39,

    // F keys
    'F1': USB_PAGE | 0x3A,
    'F2': USB_PAGE | 0x3B,
    'F3': USB_PAGE | 0x3C,
    'F4': USB_PAGE | 0x3D,
    'F5': USB_PAGE | 0x3E,
    'F6': USB_PAGE | 0x3F,
    'F7': USB_PAGE | 0x40,
    'F8': USB_PAGE | 0x41,
    'F9': USB_PAGE | 0x42,
    'F10': USB_PAGE | 0x43,
    'F11': USB_PAGE | 0x44,
    'F12': USB_PAGE | 0x45,

    // Navigation
    'PrintScreen': USB_PAGE | 0x46,
    'ScrollLock': USB_PAGE | 0x47,
    'Pause': USB_PAGE | 0x48,
    'Insert': USB_PAGE | 0x49,
    'Home': USB_PAGE | 0x4A,
    'PageUp': USB_PAGE | 0x4B,
    'Delete': USB_PAGE | 0x4C,
    'End': USB_PAGE | 0x4D,
    'PageDown': USB_PAGE | 0x4E,
    'ArrowRight': USB_PAGE | 0x4F,
    'ArrowLeft': USB_PAGE | 0x50,
    'ArrowDown': USB_PAGE | 0x51,
    'ArrowUp': USB_PAGE | 0x52,

    // Numpad
    'NumLock': USB_PAGE | 0x53,
    'NumpadDivide': USB_PAGE | 0x54,
    'NumpadMultiply': USB_PAGE | 0x55,
    'NumpadSubtract': USB_PAGE | 0x56,
    'NumpadAdd': USB_PAGE | 0x57,
    'NumpadEnter': USB_PAGE | 0x58,
    'Numpad1': USB_PAGE | 0x59,
    'Numpad2': USB_PAGE | 0x5A,
    'Numpad3': USB_PAGE | 0x5B,
    'Numpad4': USB_PAGE | 0x5C,
    'Numpad5': USB_PAGE | 0x5D,
    'Numpad6': USB_PAGE | 0x5E,
    'Numpad7': USB_PAGE | 0x5F,
    'Numpad8': USB_PAGE | 0x60,
    'Numpad9': USB_PAGE | 0x61,
    'Numpad0': USB_PAGE | 0x62,
    'NumpadDecimal': USB_PAGE | 0x63,

    // Modifiers
    'ControlLeft': USB_PAGE | 0xE0,
    'ShiftLeft': USB_PAGE | 0xE1,
    'AltLeft': USB_PAGE | 0xE2,
    'MetaLeft': USB_PAGE | 0xE3,
    'ControlRight': USB_PAGE | 0xE4,
    'ShiftRight': USB_PAGE | 0xE5,
    'AltRight': USB_PAGE | 0xE6,
    'MetaRight': USB_PAGE | 0xE7,

    // Additional keys
    'IntlBackslash': USB_PAGE | 0x64,
    'ContextMenu': USB_PAGE | 0x65,
};
