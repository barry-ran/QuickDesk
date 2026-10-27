import test from 'node:test';
import assert from 'node:assert/strict';
import { MobileKeyboard } from '../js/input/mobile-keyboard.js';

class Input extends EventTarget {
    value = '';
    blur() { this.dispatchEvent(new Event('blur')); }
    emit(type, fields = {}) {
        const event = new Event(type, { cancelable: true });
        Object.assign(event, fields);
        this.dispatchEvent(event);
        return event;
    }
}
const wait = () => new Promise(resolve => setTimeout(resolve, 10));
function setup() {
    const input = new Input(), text = [], keys = [];
    const dc = { sendTextEvent: value => text.push(value), sendKeyEvent: value => keys.push(value) };
    return { input, text, keys, dc, keyboard: new MobileKeyboard(input, dc) };
}

test('committed Unicode and punctuation use one TextEvent, not layout-dependent keys', async () => {
    const s = setup();
    s.input.value = 'Hello, 中文 👋!'; s.input.emit('input');
    await wait();
    assert.deepEqual(s.text, ['Hello, 中文 👋!']); assert.deepEqual(s.keys, []);
    assert.equal(s.input.value, ''); s.keyboard.destroy();
});
for (const finalInputFirst of [false, true]) {
    test(`IME emits only its final value, final input first=${finalInputFirst}`, async () => {
        const s = setup();
        s.input.emit('compositionstart'); s.input.value = 'zhong';
        s.input.emit('input', { isComposing: true });
        s.input.emit('keydown', { key: 'Enter', isComposing: true });
        await wait(); assert.deepEqual(s.text, []); assert.deepEqual(s.keys, []);
        s.input.value = '中';
        if (finalInputFirst) s.input.emit('input');
        s.input.emit('compositionend', { data: '中' });
        if (!finalInputFirst) s.input.emit('input');
        await wait(); assert.deepEqual(s.text, ['中']);
        s.input.value = '中'; s.input.emit('input'); await wait();
        assert.deepEqual(s.text, ['中', '中']); s.keyboard.destroy();
    });
}
test('soft-keyboard delete works without keydown; physical controls do not double-send', async () => {
    const s = setup();
    assert.equal(s.input.emit('beforeinput', { inputType: 'deleteContentBackward' }).defaultPrevented, true);
    assert.equal(s.keys.length, 2);
    s.input.value = 'a';
    s.input.emit('keydown', { key: 'Enter' });
    s.input.emit('beforeinput', { inputType: 'insertParagraph' });
    assert.deepEqual(s.text, ['a']); assert.equal(s.keys.length, 4);
    assert.deepEqual(s.keys.slice(2), [{ pressed: true, usbKeycode: 0x070028 }, { pressed: false, usbKeycode: 0x070028 }]);
    await wait(); s.keyboard.destroy();
});
test('disconnect cancels pending text and reconnect leaves exactly one listener set', async () => {
    const s = setup();
    s.input.value = 'old session'; s.input.emit('input'); s.keyboard.destroy();
    const next = new MobileKeyboard(s.input, s.dc);
    s.input.value = 'new session'; s.input.emit('input');
    await wait(); assert.deepEqual(s.text, ['new session']);
    next.destroy();
    s.input.value = 'ignored'; s.input.emit('input'); await wait();
    assert.deepEqual(s.text, ['new session']);
});
test('canceled or interrupted composition never leaks preedit text', async () => {
    const s = setup();
    s.input.emit('compositionstart'); s.input.value = 'preedit';
    s.input.emit('compositionend', { data: '' });
    await wait(); assert.deepEqual(s.text, []);
    s.input.emit('compositionstart'); s.input.value = 'preedit'; s.input.blur();
    s.input.value = 'late commit'; s.input.emit('compositionend', { data: 'late commit' });
    s.input.emit('input'); await wait(); assert.deepEqual(s.text, []);
    s.input.emit('focus'); s.input.value = 'new text'; s.input.emit('input');
    await wait(); assert.deepEqual(s.text, ['new text']); s.keyboard.destroy();
});
