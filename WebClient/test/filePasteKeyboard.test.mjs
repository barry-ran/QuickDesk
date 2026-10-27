import assert from 'node:assert/strict';
import test from 'node:test';
import { KeyboardHandler } from '../js/input/keyboard-handler.js';
import { ClipboardHandler } from '../js/input/clipboard-handler.js';
function event(code, extras = {}) {
    return { code, ...extras, prevented: false,
        preventDefault() { this.prevented = true; }, stopPropagation() {} };
}
function setup() {
    const calls = [];
    const transport = {
        sendKeyEvent(value) { calls.push({ kind: 'key', ...value }); },
        sendClipboardEvent(value) { calls.push({ kind: 'clipboard', ...value }); },
    };
    return { keyboard: new KeyboardHandler({}, transport), transport, calls };
}
test('Ctrl+V permits native paste and consumed file paste sends no remote V or unmatched release', () => {
    const { keyboard, calls } = setup();
    keyboard._onKeyDown(event('ControlLeft'));
    const down = event('KeyV', { ctrlKey: true }); keyboard._onKeyDown(down);
    assert.equal(down.prevented, false); keyboard.consumeBrowserPaste();
    const repeat = event('KeyV', { ctrlKey: true, repeat: true }); keyboard._onKeyDown(repeat);
    assert.equal(repeat.prevented, true);
    keyboard._onKeyUp(event('KeyV', { ctrlKey: true })); keyboard._onKeyUp(event('ControlLeft'));
    assert.deepEqual(calls.map(value => value.usbKeycode), [0x0700e0, 0x0700e0]);
});
test('text paste sync happens before remote V pair and V-up does not duplicate it', () => {
    const { keyboard, transport, calls } = setup();
    const clipboard = new ClipboardHandler(transport); clipboard._enabled = true;
    keyboard._onKeyDown(event('ControlLeft')); keyboard._onKeyDown(event('KeyV', { ctrlKey: true }));
    clipboard.syncText('fresh text'); keyboard.forwardBrowserPaste();
    keyboard._onKeyUp(event('KeyV', { ctrlKey: true })); keyboard._onKeyUp(event('ControlLeft'));
    assert.equal(calls[1].kind, 'clipboard');
    assert.equal(new TextDecoder().decode(calls[1].data), 'fresh text');
    assert.deepEqual(calls.filter(value => value.usbKeycode === 0x070019).map(value => value.pressed), [true, false]);
});
test('fallback after Control is released synthesizes a complete Ctrl+V pair', () => {
    const { keyboard, calls } = setup();
    keyboard._onKeyDown(event('ControlLeft')); keyboard._onKeyDown(event('KeyV', { ctrlKey: true }));
    keyboard._onKeyUp(event('ControlLeft')); keyboard._onKeyUp(event('KeyV'));
    assert.deepEqual(calls.slice(2).map(value => [value.usbKeycode, value.pressed]), [
        [0x0700e0, true], [0x070019, true], [0x070019, false], [0x0700e0, false],
    ]);
});
test('blur clears pending paste and later keyup does not type or release a stray V', () => {
    const { keyboard, calls } = setup();
    keyboard._onKeyDown(event('ControlLeft')); keyboard._onKeyDown(event('KeyV', { ctrlKey: true }));
    keyboard._releaseAllKeys(); keyboard._onKeyUp(event('KeyV'));
    assert.equal(calls.some(value => value.usbKeycode === 0x070019), false);
});

test('disabled clipboard ignores old remote events and pending local polls across re-enable', async () => {
    const oldNavigator = Object.getOwnPropertyDescriptor(globalThis, 'navigator');
    const oldDocument = Object.getOwnPropertyDescriptor(globalThis, 'document');
    let resolveRead;
    const reading = new Promise(resolve => { resolveRead = resolve; });
    const writes = [], sends = [];
    Object.defineProperty(globalThis, 'navigator', { configurable: true, value: {
        clipboard: { readText: () => reading, writeText: async text => writes.push(text) },
    } });
    Object.defineProperty(globalThis, 'document', { configurable: true, value: { hasFocus: () => true } });
    const transport = new EventTarget();
    transport.sendClipboardEvent = value => sends.push(value);
    const clipboard = new ClipboardHandler(transport);
    try {
        clipboard.enable();
        const pending = clipboard._pollClipboard();
        clipboard.disable();
        transport.dispatchEvent(new CustomEvent('clipboardEvent', { detail: {
            mimeType: 'text/plain', data: new TextEncoder().encode('old session'),
        } }));
        await clipboard._onRemoteClipboard({ detail: {
            mimeType: 'text/plain', data: new TextEncoder().encode('direct stale callback'),
        } });
        clipboard.enable();
        resolveRead('old poll result');
        await pending;
        assert.deepEqual(writes, []); assert.deepEqual(sends, []);
    } finally {
        clipboard.destroy();
        if (oldNavigator) Object.defineProperty(globalThis, 'navigator', oldNavigator);
        else delete globalThis.navigator;
        if (oldDocument) Object.defineProperty(globalThis, 'document', oldDocument);
        else delete globalThis.document;
    }
});

test('Cmd paste retains Meta rather than converting it to Control', () => {
    const { keyboard, calls } = setup();
    keyboard._onKeyDown(event('MetaLeft'));
    keyboard._onKeyDown(event('KeyV', { metaKey: true }));
    keyboard.forwardBrowserPaste();
    keyboard._onKeyUp(event('KeyV', { metaKey: true })); keyboard._onKeyUp(event('MetaLeft'));
    assert.deepEqual(calls.map(value => [value.usbKeycode, value.pressed]), [
        [0x0700e3, true], [0x070019, true], [0x070019, false], [0x0700e3, false],
    ]);
});
test('Cmd released before fallback is synthesized as the original Meta chord', () => {
    const { keyboard, calls } = setup();
    keyboard._onKeyDown(event('MetaLeft'));
    keyboard._onKeyDown(event('KeyV', { metaKey: true }));
    keyboard._onKeyUp(event('MetaLeft')); keyboard._onKeyUp(event('KeyV'));
    assert.deepEqual(calls.slice(2).map(value => [value.usbKeycode, value.pressed]), [
        [0x0700e3, true], [0x070019, true], [0x070019, false], [0x0700e3, false],
    ]);
    assert.equal(calls.some(value => value.usbKeycode === 0x0700e0), false);
});
test('mixed Ctrl+Meta chord is preserved and unrelated context paste does not invent keys', () => {
    const { keyboard, calls } = setup();
    assert.equal(keyboard.forwardBrowserPaste(), false);
    keyboard._onKeyDown(event('ControlLeft')); keyboard._onKeyDown(event('MetaRight'));
    keyboard._onKeyDown(event('KeyV', { ctrlKey: true, metaKey: true }));
    keyboard._onKeyUp(event('ControlLeft')); keyboard._onKeyUp(event('MetaRight'));
    keyboard._onKeyUp(event('KeyV'));
    assert.deepEqual(calls.slice(4).map(value => [value.usbKeycode, value.pressed]), [
        [0x0700e0, true], [0x0700e7, true], [0x070019, true], [0x070019, false],
        [0x0700e7, false], [0x0700e0, false],
    ]);
});

test('local editable controls do not forward a paste shortcut or fallback to the host', () => {
    const { keyboard, calls } = setup();
    const target = { isContentEditable: true };
    const down = event('KeyV', { ctrlKey: true, target });
    keyboard._onKeyDown(down); keyboard._onKeyUp(event('KeyV', { target }));
    assert.equal(down.prevented, false); assert.deepEqual(calls, []);
});
