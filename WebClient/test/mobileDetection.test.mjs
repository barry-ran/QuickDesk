import assert from 'node:assert/strict';
import test from 'node:test';
import { JSDOM } from 'jsdom';

// Import the real app without starting a session or fetching credentials.
globalThis.document = { addEventListener() {} };
const { RemoteDesktopApp } = await import('../js/remote-main.js');
delete globalThis.document;

function withGlobals(values, run) {
    const previous = new Map(Object.keys(values).map(key => [key, Object.getOwnPropertyDescriptor(globalThis, key)]));
    try {
        for (const [key, value] of Object.entries(values)) Object.defineProperty(globalThis, key, { configurable: true, value });
        return run();
    } finally {
        for (const [key, descriptor] of previous) {
            if (descriptor) Object.defineProperty(globalThis, key, descriptor);
            else delete globalThis[key];
        }
    }
}

for (const [name, width, height, screenWidth, screenHeight, touch, ua, platform, coarse, fine, expected] of [
    ['desktop-site phone portrait with mouse', 390, 844, 390, 844, 5, 'Linux x86_64', '', false, true, true],
    ['desktop-site phone landscape with mouse', 844, 390, 844, 390, 5, 'Linux x86_64', '', false, true, true],
    ['desktop-site wide layout viewport', 980, 450, 844, 390, 5, 'Linux x86_64', '', false, true, true],
    ['Android landscape', 844, 390, 844, 390, 5, 'Android', '', false, true, true],
    ['iPhone landscape', 844, 390, 844, 390, 5, 'iPhone', '', false, true, true],
    ['iPad desktop UA', 1366, 1024, 1366, 1024, 5, 'Macintosh', 'MacIntel', false, true, true],
    ['coarse-only tablet', 1366, 1024, 1366, 1024, 5, 'Linux', '', true, false, true],
    ['short touchscreen desktop window', 1280, 390, 1920, 1080, 10, 'Windows', '', false, true, false],
    ['desktop without touch', 844, 390, 844, 390, 0, 'Windows', '', false, true, false],
]) {
    test(`mobile classification: ${name}`, () => withGlobals({
        navigator: { userAgent: ua, platform, maxTouchPoints: touch },
        window: { innerWidth: width, innerHeight: height, screen: { width: screenWidth, height: screenHeight },
            matchMedia: query => ({ matches: query === '(pointer: coarse)' ? coarse : fine }) },
    }, () => assert.equal(new RemoteDesktopApp()._isMobile, expected)));
}

test('real toolbar keeps synchronous focus across viewport resize, toggles closed and cancels on disconnect', () => {
    const dom = new JSDOM('<body><div id="mobileToolbar"><button id="btnKeyboard"></button><div id="mobileKeyRow" hidden></div></div><input id="mobileKeyInput"></body>');
    const vv = new dom.window.EventTarget(); vv.height = 390;
    Object.defineProperty(dom.window, 'visualViewport', { value: vv });
    dom.window.scrollTo = () => {};
    withGlobals({ window: dom.window, document: dom.window.document,
        navigator: { userAgent: 'iPhone', maxTouchPoints: 5 },
        requestAnimationFrame: () => 1, cancelAnimationFrame: () => {},
    }, () => {
        const app = new RemoteDesktopApp();
        app.dcHandler = { sendTextEvent() {}, sendKeyEvent() {} };
        app._setupMobileToolbar(); app._enableMobileKeyboard();
        const input = document.getElementById('mobileKeyInput');
        const button = document.getElementById('btnKeyboard');
        button.click();
        assert.equal(document.activeElement, input); // no timer/await before focus
        assert.equal(button.getAttribute('aria-pressed'), 'true');
        vv.height = 180; vv.dispatchEvent(new dom.window.Event('resize'));
        assert.equal(document.body.style.height, '180px');
        assert.equal(document.activeElement, input);
        button.click(); assert.notEqual(document.activeElement, input);
        assert.equal(document.getElementById('mobileKeyRow').hidden, true);
        button.click(); assert.equal(document.activeElement, input);
        app._cleanupInputHandlers();
        assert.notEqual(document.activeElement, input);
        assert.equal(document.body.style.height, '');
        assert.equal(button.getAttribute('aria-pressed'), 'false');
    });
    dom.window.close();
});
