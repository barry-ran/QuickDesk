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
