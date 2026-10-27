/** A session-scoped bridge: committed IME text uses TextEvent, not US keycodes. */
export class MobileKeyboard {
    constructor(input, dcHandler) {
        this.input = input;
        this.dcHandler = dcHandler;
        this.composing = false;
        this.suspended = false;
        this.timer = null;
        this.specialTimer = null;
        this.specialHandled = false;
        this.listeners = [];
        this.listen('compositionstart', () => {
            clearTimeout(this.timer);
            this.composing = true;
        });
        this.listen('compositionend', (event) => {
            this.composing = false;
            if (this.suspended || event.data === '') {
                clearTimeout(this.timer);
                this.input.value = '';
                return;
            }
            // Browsers disagree whether the final input comes before or after
            // compositionend. Read the final value once, after either order.
            this.scheduleFlush();
        });
        this.listen('input', (event) => {
            if (this.suspended) { this.input.value = ''; return; }
            if (!this.composing && !event.isComposing) this.scheduleFlush();
        });
        this.listen('beforeinput', (event) => {
            if (this.composing || event.isComposing) return;
            const code = event.inputType === 'deleteContentBackward' ? 0x07002A
                : ['insertLineBreak', 'insertParagraph'].includes(event.inputType) ? 0x070028 : 0;
            if (code) {
                event.preventDefault();
                if (!this.specialHandled) { this.flush(); this.sendKey(code); }
            }
        });
        this.listen('keydown', (event) => {
            if (this.composing || event.isComposing || event.keyCode === 229) return;
            const code = { Backspace: 0x07002A, Enter: 0x070028, Tab: 0x07002B,
                Escape: 0x070029, ArrowLeft: 0x070050, ArrowRight: 0x07004F,
                ArrowUp: 0x070052, ArrowDown: 0x070051 }[event.key];
            if (!code) return;
            event.preventDefault();
            this.flush();
            this.sendKey(code);
            this.specialHandled = true;
            clearTimeout(this.specialTimer);
            this.specialTimer = setTimeout(() => { this.specialHandled = false; }, 0);
        });
        this.listen('focus', () => { this.suspended = false; });
        this.listen('blur', () => {
            if (!this.composing) this.flush();
            clearTimeout(this.timer);
            this.composing = false;
            this.suspended = true;
            this.input.value = '';
        });
    }
    listen(type, fn) {
        this.input.addEventListener(type, fn);
        this.listeners.push([type, fn]);
    }
    scheduleFlush() {
        clearTimeout(this.timer);
        this.timer = setTimeout(() => this.flush(), 0);
    }
    flush() {
        clearTimeout(this.timer);
        if (this.composing) return;
        const text = this.input.value;
        this.input.value = '';
        if (text) this.dcHandler.sendTextEvent(text);
    }
    sendKey(usbKeycode) {
        this.dcHandler.sendKeyEvent({ pressed: true, usbKeycode });
        this.dcHandler.sendKeyEvent({ pressed: false, usbKeycode });
    }
    destroy() {
        clearTimeout(this.timer);
        clearTimeout(this.specialTimer);
        for (const [type, fn] of this.listeners) this.input.removeEventListener(type, fn);
        this.listeners = [];
        this.input.value = '';
        this.input.blur();
    }
}
