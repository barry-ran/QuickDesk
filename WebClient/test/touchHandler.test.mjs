import assert from 'node:assert/strict';
import { afterEach, test } from 'node:test';
import { JSDOM } from 'jsdom';
import { readFileSync } from 'node:fs';
import { TouchHandler } from '../js/input/touch-handler.js';
import { MouseButton } from '../js/protocol/protobuf-messages.js';

let handler;
let dom;
afterEach(() => { handler?.destroy(); dom?.window.close(); });

function fixture() {
    dom = new JSDOM('<div id="viewport"><video></video></div>', { pretendToBeVisual: true });
    globalThis.window = dom.window;
    globalThis.document = dom.window.document;
    const target = document.querySelector('#viewport');
    const video = document.querySelector('video');
    const rect = { left: 0, top: 0, width: 400, height: 600 };
    target.getBoundingClientRect = () => ({ ...rect });
    const messages = [];
    handler = new TouchHandler(target, video, { sendMouseEvent: event => messages.push(event) });
    video.getBoundingClientRect = () => ({
        left: rect.left + rect.width / 2 * (1 - handler._scale) + handler._translateX,
        top: rect.top + rect.height / 2 * (1 - handler._scale) + handler._translateY,
        width: rect.width * handler._scale, height: rect.height * handler._scale,
    });
    handler.setRemoteResolution(1600, 900);
    handler.enable();
    const send = (type, points = []) => {
        const event = new dom.window.Event(type, { cancelable: true });
        Object.defineProperty(event, 'touches', { value: points.map(([clientX, clientY]) => ({ clientX, clientY })) });
        target.dispatchEvent(event);
        assert.equal(event.defaultPrevented, true);
    };
    return { handler, messages, rect, send };
}
function tap(send, position = [100, 100]) { send('touchstart', [position]); send('touchend'); }
function beginDrag(send) { tap(send); send('touchstart', [[100, 100]]); send('touchmove', [[120, 100]]); }
const buttons = messages => messages.filter(event => event.button !== undefined);

test('tap and double tap produce balanced left-button pairs at cursor, not finger', () => {
    const { send, messages } = fixture();
    tap(send); tap(send);
    assert.deepEqual(buttons(messages), Array.from({ length: 4 }, (_, i) => ({ x: 800, y: 450, button: MouseButton.BUTTON_LEFT, buttonDown: i % 2 === 0 })));
});
test('double tap then drag holds left until lift', () => {
    const { send, messages } = fixture();
    beginDrag(send);
    assert.equal(handler._dragging, true);
    assert.equal(buttons(messages).at(-1).buttonDown, true);
    send('touchend');
    assert.equal(buttons(messages).at(-1).buttonDown, false);
    assert.equal(buttons(messages).length, 4);
});
test('long press emits one right click, never a trailing left click', async () => {
    const { send, messages } = fixture();
    send('touchstart', [[100, 100]]);
    await new Promise(resolve => setTimeout(resolve, 530));
    send('touchend');
    assert.deepEqual(buttons(messages).map(e => [e.button, e.buttonDown]), [[MouseButton.BUTTON_RIGHT, true], [MouseButton.BUTTON_RIGHT, false]]);
});
test('single-finger cursor gain reduces with zoom and clamps to remote bounds', () => {
    const { send, messages } = fixture();
    handler._scale = 2;
    send('touchstart', [[100, 100]]); send('touchmove', [[120, 100]]);
    assert.equal(messages.at(-1).x, 825);
    send('touchmove', [[10000, -10000]]);
    assert.deepEqual(messages.at(-1), { x: 1599, y: 0 });
    send('touchend');
    assert.equal(buttons(messages).length, 0);
});
for (const scale of [1, 2, 5]) {
    test(`two-finger horizontal and vertical scroll works at ${scale}x without zoom`, () => {
        const { send, messages } = fixture();
        handler._scale = scale;
        send('touchstart', [[100, 200], [200, 200]]);
        send('touchmove', [[110, 200], [210, 200]]);
        send('touchmove', [[110, 215], [210, 215]]);
        assert.equal(handler._twoFingerMode, 'scroll');
        assert.equal(handler._scale, scale);
        assert.deepEqual(messages.map(e => [e.wheelDeltaX, e.wheelDeltaY]), [[30, 0], [0, 45]]);
        send('touchend', [[110, 215]]); send('touchend');
        assert.equal(buttons(messages).length, 0);
    });
}
test('pinch locks intent and anchors off-center focal point without scrolling', () => {
    const { send, messages } = fixture();
    send('touchstart', [[50, 200], [150, 200]]);
    send('touchmove', [[0, 200], [200, 200]]);
    assert.equal(handler._scale, 2);
    assert.equal(handler._translateX, 100);
    assert.equal(handler._translateY, 0); // Letterboxed content is shorter than the viewport.
    send('touchmove', [[20, 220], [220, 220]]);
    assert.equal(handler._twoFingerMode, 'pinch');
    assert.equal(handler._translateX, 120);
    assert.equal(handler._translateY, 0);
    assert.equal(messages.length, 0);
});
test('scroll intent does not unexpectedly become pinch when finger spacing drifts', () => {
    const { send, messages } = fixture();
    send('touchstart', [[100, 200], [200, 200]]);
    send('touchmove', [[100, 220], [200, 220]]);
    send('touchmove', [[80, 240], [220, 240]]);
    assert.equal(handler._twoFingerMode, 'scroll');
    assert.equal(handler._scale, 1);
    assert.equal(messages.length, 2);
});
test('multi-touch residual finger is inert until all fingers lift', () => {
    const { send, messages } = fixture();
    send('touchstart', [[100, 200], [200, 200]]);
    send('touchend', [[100, 200]]);
    send('touchmove', [[160, 200]]);
    send('touchend');
    assert.deepEqual(messages, []);
    tap(send);
    assert.equal(buttons(messages).length, 2);
});
test('third finger releases active drag and suppresses remainder', () => {
    const { send, messages } = fixture();
    beginDrag(send);
    send('touchstart', [[120, 100], [180, 100], [240, 100]]);
    assert.equal(buttons(messages).at(-1).buttonDown, false);
    send('touchend', [[120, 100], [180, 100]]);
    send('touchend', [[120, 100]]); send('touchend');
    assert.equal(buttons(messages).length, 4);
});
for (const interruption of ['touchcancel', 'blur', 'hidden', 'resize', 'disable', 'destroy']) {
    test(`${interruption} releases drag exactly once and cannot create a trailing click`, () => {
        const { send, messages } = fixture();
        beginDrag(send);
        if (interruption === 'touchcancel') send('touchcancel');
        else if (interruption === 'hidden') {
            Object.defineProperty(document, 'hidden', { configurable: true, value: true });
            document.dispatchEvent(new window.Event('visibilitychange'));
        } else if (interruption === 'disable' || interruption === 'destroy') handler[interruption]();
        else window.dispatchEvent(new window.Event(interruption));
        assert.equal(buttons(messages).at(-1).buttonDown, false);
        if (interruption !== 'disable' && interruption !== 'destroy') send('touchend');
        assert.equal(buttons(messages).length, 4);
        handler.destroy();
        assert.equal(buttons(messages).length, 4);
    });
}
test('letterbox cursor maps correctly after portrait-to-landscape resize', () => {
    const { rect } = fixture();
    assert.equal(parseFloat(handler._cursorEl.style.left), 200);
    assert.equal(parseFloat(handler._cursorEl.style.top), 300);
    rect.width = 800; rect.height = 400;
    window.dispatchEvent(new window.Event('resize'));
    assert.equal(parseFloat(handler._cursorEl.style.left), 400);
    assert.equal(parseFloat(handler._cursorEl.style.top), 200);
});
test('resolution shrink clamps cursor and releases held button', () => {
    const { send, messages } = fixture();
    beginDrag(send);
    handler.setRemoteResolution(320, 240);
    assert.equal(buttons(messages).at(-1).buttonDown, false);
    assert.equal(handler._cursorX, 319);
    assert.equal(handler._cursorY, 239);
});

const shared = JSON.parse(readFileSync(new URL('../../test/fixtures/mobile-gestures.json', import.meta.url), 'utf8'));
for (const scenario of shared.cases) {
    test(`shared mobile contract: ${scenario.name}`, () => {
        const { send, messages, rect } = fixture();
        [rect.width, rect.height] = shared.viewport;
        handler.setRemoteResolution(...shared.remote);
        handler._scale = scenario.initialScale;
        send('touchstart', scenario.start);
        for (const points of scenario.moves) send('touchmove', points);
        assert.equal(handler._twoFingerMode, scenario.mode);
        assert.equal(handler._scale, scenario.scale);
        const wheels = messages.filter(e => e.wheelDeltaX !== undefined);
        if (scenario.wheel) {
            assert.deepEqual(wheels.reduce(([x, y], e) => [x + e.wheelDeltaX, y + e.wheelDeltaY], [0, 0]), scenario.wheel);
        } else assert.equal(wheels.length, 0);
        send('touchend');
        assert.equal(buttons(messages).length, 0);
    });
}

test('display switch cancels two-finger input until every finger lifts', () => {
    const { send, messages } = fixture();
    send('touchstart', [[100, 200], [200, 200]]);
    handler.setRemoteResolution(1920, 1080);
    send('touchmove', [[100, 240], [200, 240]]);
    send('touchend', [[100, 240]]); send('touchend');
    assert.equal(messages.length, 0);
    tap(send);
    assert.equal(buttons(messages).length, 2);
});

test('zoom bounds use actual letterboxed content, not empty video-element bars', () => {
    const { handler } = fixture();
    handler._scale = 2;
    handler._translateX = 10000;
    handler._translateY = 10000;
    handler._applyTransform();
    assert.equal(handler._translateX, 200);
    assert.equal(handler._translateY, 0); // 225px video ×2 still fits within 600px viewport.
    handler.resetZoom();
    assert.equal(handler._scale, 1);
    assert.equal(handler._translateX, 0);
    assert.equal(handler._translateY, 0);
    assert.equal(handler.video.style.transform, '');
});

test('accessible drag action arms without sending input and releases on lift', () => {
    const { send, messages } = fixture();
    const states = [];
    handler.onDragStateChanged = state => states.push(state);
    handler.toggleDrag();
    assert.equal(handler.isDragging, true);
    assert.equal(messages.length, 0);
    send('touchstart', [[100, 100]]); send('touchmove', [[120, 100]]);
    assert.deepEqual(buttons(messages).map(e => e.buttonDown), [true]);
    send('touchend');
    assert.deepEqual(buttons(messages).map(e => e.buttonDown), [true, false]);
    assert.equal(handler.isDragging, false);
    assert.equal(states.at(-1), false);
});
test('armed drag can be toggled off or tapped to cancel without remote click', () => {
    const { send, messages } = fixture();
    handler.toggleDrag(); handler.toggleDrag();
    assert.equal(handler.isDragging, false);
    assert.equal(messages.length, 0);
    handler.toggleDrag(); tap(send);
    assert.equal(handler.isDragging, false);
    assert.equal(messages.length, 0);
});
test('explicit click first releases held drag then emits a balanced requested click', () => {
    const { send, messages } = fixture();
    handler.toggleDrag();
    send('touchstart', [[100, 100]]); send('touchmove', [[120, 100]]);
    handler.click(MouseButton.BUTTON_RIGHT);
    send('touchend');
    assert.deepEqual(buttons(messages).map(e => [e.button, e.buttonDown]), [
        [MouseButton.BUTTON_LEFT, true], [MouseButton.BUTTON_LEFT, false],
        [MouseButton.BUTTON_RIGHT, true], [MouseButton.BUTTON_RIGHT, false],
    ]);
});
test('a fresh simultaneous two-finger gesture works after touchcancel', () => {
    const { send, messages } = fixture();
    send('touchstart', [[100, 100]]); send('touchcancel');
    send('touchstart', [[100, 200], [200, 200]]);
    send('touchmove', [[100, 220], [200, 220]]);
    send('touchend');
    assert.equal(messages.length, 1);
    assert.equal(messages[0].wheelDeltaY, 60);
});

test('container-only resize remaps cursor, cancels drag, and disconnects observer', () => {
    let notify;
    let disconnected = false;
    globalThis.ResizeObserver = class {
        constructor(callback) { notify = callback; }
        observe() {}
        disconnect() { disconnected = true; }
    };
    try {
        const { rect, send, messages } = fixture();
        notify(); // Initial observation must not cancel a fresh gesture.
        beginDrag(send);
        rect.width = 300;
        rect.height = 400;
        notify();
        assert.equal(handler.isDragging, false);
        assert.equal(buttons(messages).at(-1).buttonDown, false);
        assert.equal(parseFloat(handler._cursorEl.style.top), 200);
        handler.disable();
        assert.equal(disconnected, true);
    } finally { delete globalThis.ResizeObserver; }
});
