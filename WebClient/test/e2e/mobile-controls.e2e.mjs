import assert from 'node:assert/strict';
import { after, before, test } from 'node:test';
import { readFile } from 'node:fs/promises';
import http from 'node:http';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

// Browser input integration, not physical-device or live-host interoperability QA.
const root = fileURLToPath(new URL('../../', import.meta.url));
let server, browser, base;
before(async () => {
    server = http.createServer(async (req, res) => {
        if (req.url === '/') {
            res.setHeader('content-type', 'text/html');
            res.end(`<!doctype html><meta name="viewport" content="width=device-width,initial-scale=1"><style>
                html,body{margin:0;overflow:hidden}#surface{position:relative;width:100vw;height:100dvh;overflow:hidden;touch-action:none}
                video{width:100%;height:100%;object-fit:contain;background:#243244}
                </style><div id="surface"><video></video></div><script type="module">
                import { TouchHandler } from '/js/input/touch-handler.js';
                window.events=[]; window.handler=new TouchHandler(document.querySelector('#surface'),document.querySelector('video'),{sendMouseEvent:e=>events.push(e)});
                handler.setRemoteResolution(1600,900);handler.enable();window.ready=true;
                </script>`);
            return;
        }
        if (req.url === '/remote.html') {
            // Use the real document, app, bindings, and installed crypto modules;
            // only replace CDN locations to keep this regression test offline.
            let html = await readFile(path.join(root, 'remote.html'), 'utf8');
            const imports = {
                '@noble/curves/ed25519': '/node_modules/@noble/curves/esm/ed25519.js',
                '@noble/curves/abstract/utils': '/node_modules/@noble/curves/esm/abstract/utils.js',
                '@noble/hashes/sha256': '/node_modules/@noble/hashes/esm/sha256.js',
                '@noble/hashes/sha512': '/node_modules/@noble/hashes/esm/sha512.js',
                '@noble/hashes/hmac': '/node_modules/@noble/hashes/esm/hmac.js',
                '@noble/hashes/crypto': '/node_modules/@noble/hashes/esm/crypto.js',
                '@noble/hashes/': '/node_modules/@noble/hashes/esm/',
            };
            html = html.replace(/<script type="importmap">[\s\S]*?<\/script>/, `<script type="importmap">${JSON.stringify({ imports })}</script>`);
            res.setHeader('content-type', 'text/html');
            res.end(html);
            return;
        }
        const file = path.resolve(root, `.${new URL(req.url, 'http://localhost').pathname}`);
        if (!file.startsWith(root)) { res.writeHead(403).end(); return; }
        try { res.setHeader('content-type', 'text/javascript'); res.end(await readFile(file)); }
        catch { res.writeHead(404).end(); }
    });
    await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
    base = `http://127.0.0.1:${server.address().port}`;
    browser = await chromium.launch({
        ...(process.env.QD_BROWSER_EXECUTABLE || process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE
            ? { executablePath: process.env.QD_BROWSER_EXECUTABLE || process.env.PLAYWRIGHT_CHROMIUM_EXECUTABLE }
            : process.env.QD_BROWSER_CHANNEL ? { channel: process.env.QD_BROWSER_CHANNEL } : {}),
        headless: true,
    });
});
after(async () => { await browser?.close(); await new Promise(resolve => server?.close(resolve)); });

for (const viewport of [{ width: 390, height: 844 }, { width: 844, height: 390 }]) {
    test(`mobile Chromium touch integration ${viewport.width}×${viewport.height}`, async () => {
        const context = await browser.newContext({ viewport, isMobile: true, hasTouch: true, deviceScaleFactor: 3 });
        const page = await context.newPage();
        const errors = [];
        page.on('pageerror', e => errors.push(e.message));
        try {
            await page.goto(base);
            await page.waitForFunction(() => window.ready);
            const cdp = await context.newCDPSession(page);
            const touch = async (type, points = []) => {
                await cdp.send('Input.dispatchTouchEvent', { type, touchPoints: points.map(([x, y], id) => ({ x, y, id })) });
            };
            const tap = async () => { await touch('touchStart', [[100, 150]]); await touch('touchEnd'); };
            await tap();
            assert.deepEqual(await page.evaluate(() => events.map(e => e.buttonDown)), [true, false]);
            await page.evaluate(() => { events.length = 0; });
            // A fresh gesture, not within double-tap interval.
            await page.waitForTimeout(320);
            await touch('touchStart', [[100, 150], [200, 150]]);
            await touch('touchMove', [[100, 170], [200, 170]]);
            await touch('touchEnd');
            const scroll = await page.evaluate(() => ({ events, scale: handler._scale }));
            assert.equal(scroll.scale, 1);
            assert.ok(scroll.events.some(e => e.wheelDeltaY > 0));
            assert.ok(scroll.events.every(e => e.button === undefined));

            await page.evaluate(() => { events.length = 0; });
            await touch('touchStart', [[100, 150], [200, 150]]);
            await touch('touchMove', [[50, 150], [250, 150]]);
            await touch('touchEnd');
            const pinch = await page.evaluate(() => ({ events, scale: handler._scale }));
            assert.equal(pinch.scale, 2);
            assert.equal(pinch.events.length, 0);

            await touch('touchStart', [[100, 150], [200, 150]]);
            await touch('touchMove', [[120, 150], [220, 150]]);
            await touch('touchEnd');
            assert.ok(await page.evaluate(() => events.some(e => e.wheelDeltaX > 0)));
            assert.equal(await page.evaluate(() => handler._scale), 2);

            await page.evaluate(() => { handler.resetZoom(); events.length = 0; });
            await tap();
            await touch('touchStart', [[100, 150]]);
            await touch('touchMove', [[125, 150]]);
            assert.equal(await page.evaluate(() => handler._dragging), true);
            await touch('touchCancel');
            assert.deepEqual(await page.evaluate(() => events.filter(e => e.button !== undefined).map(e => e.buttonDown)), [true, false, true, false]);
            assert.equal(await page.evaluate(() => handler._dragging), false);
            assert.deepEqual(errors, []);
        } finally { await context.close(); }
    });
}

test('orientation change keeps the letterboxed cursor visible and centered', async () => {
    const context = await browser.newContext({ viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true });
    const page = await context.newPage();
    try {
        await page.goto(base); await page.waitForFunction(() => window.ready);
        await page.setViewportSize({ width: 844, height: 390 });
        await page.waitForFunction(() => Math.abs(parseFloat(document.querySelector('#virtualCursor').style.left) - 422) < 1);
        const pos = await page.evaluate(() => ({ x: parseFloat(handler._cursorEl.style.left), y: parseFloat(handler._cursorEl.style.top) }));
        assert.equal(pos.x, 422); assert.equal(pos.y, 195);
    } finally { await context.close(); }
});


test('real remote page: 320px toolbar, help, keyboard and accessible drag bindings', async () => {
    const context = await browser.newContext({ viewport: { width: 320, height: 740 }, isMobile: true, hasTouch: true });
    await context.addInitScript(() => localStorage.setItem('quickdesk_lang', 'en-US'));
    const page = await context.newPage();
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    try {
        await page.goto(`${base}/remote.html`);
        await page.waitForFunction(() => window.remoteDesktopApp);
        await page.evaluate(() => {
            // No credentials/session/network are needed: attach a recording host
            // at the real input boundary after normal init reports missing params.
            const app = window.remoteDesktopApp;
            window.inputEvents = [];
            app.dcHandler = Object.assign(new EventTarget(), {
                sendMouseEvent: value => inputEvents.push({ kind: 'mouse', ...value }),
                sendKeyEvent: value => inputEvents.push({ kind: 'key', ...value }),
                sendTextEvent: value => inputEvents.push({ kind: 'text', text: value }),
                sendClipboardEvent: () => {},
            });
            app._setupMobileToolbar();
            app._setupInputHandlers();
        });
        const row = page.locator('#mobileToolbar > .mobile-toolbar-row');
        assert.equal(await row.evaluate(el => el.scrollWidth > el.clientWidth), true);
        for (const id of ['btnKeyboard', 'btnLeftClick', 'btnDrag', 'btnRightClick', 'btnZoomReset', 'btnGestureHelp', 'btnLogs', 'btnFullscreen', 'btnDisconnect']) {
            const button = page.locator(`#${id}`);
            await button.scrollIntoViewIfNeeded();
            const bounds = await button.boundingBox();
            assert.ok(bounds.width >= 48 && bounds.height >= 48, `${id} minimum target`);
            assert.ok(bounds.x >= -1 && bounds.x + bounds.width <= 321, `${id} reachable by horizontal scroll`);
        }
        const help = page.locator('#btnGestureHelp');
        await help.tap();
        assert.equal(await help.getAttribute('aria-expanded'), 'true');
        assert.equal(await page.locator('#mobileGestureHelp').isVisible(), true);
        await help.tap();
        assert.equal(await help.getAttribute('aria-expanded'), 'false');
        assert.equal(await page.locator('#mobileGestureHelp').isVisible(), false);

        const drag = page.locator('#btnDrag');
        await drag.tap();
        assert.equal(await drag.getAttribute('aria-pressed'), 'true');
        assert.deepEqual(await page.evaluate(() => inputEvents), []);
        await drag.tap();
        assert.equal(await drag.getAttribute('aria-pressed'), 'false');
        await page.locator('#btnLeftClick').tap();
        assert.deepEqual(await page.evaluate(() => inputEvents.map(e => e.buttonDown)), [true, false]);

        const keyboard = page.locator('#btnKeyboard');
        await keyboard.tap();
        assert.equal(await keyboard.getAttribute('aria-pressed'), 'true');
        assert.equal(await page.locator('#mobileKeyRow').isVisible(), true);
        await page.locator('#mobileKeyRow button').first().tap();
        assert.deepEqual(await page.evaluate(() => inputEvents.filter(e => e.kind === 'key').map(e => [e.usbKeycode, e.pressed])), [[0x070029, true], [0x070029, false]]);
        assert.equal(await keyboard.getAttribute('aria-pressed'), 'true');
        // Synthetic browser composition sequence tests the real bridge, not a
        // physical Android/iOS IME, whose suggestion UI needs device QA.
        await page.evaluate(() => {
            const input = document.querySelector('#mobileKeyInput');
            input.dispatchEvent(new CompositionEvent('compositionstart'));
            input.value = '你好';
            input.dispatchEvent(new InputEvent('input', { data: '你好', isComposing: true }));
            input.dispatchEvent(new CompositionEvent('compositionend', { data: '你好' }));
            input.dispatchEvent(new InputEvent('input', { data: '你好' }));
        });
        await page.waitForFunction(() => inputEvents.some(e => e.kind === 'text'));
        assert.deepEqual(await page.evaluate(() => inputEvents.filter(e => e.kind === 'text').map(e => e.text)), ['你好']);
        await keyboard.tap();
        assert.equal(await keyboard.getAttribute('aria-pressed'), 'false');
        assert.equal(await page.locator('#mobileKeyRow').isVisible(), false);
        await page.evaluate(() => remoteDesktopApp._cleanupInputHandlers());
        assert.deepEqual(errors, []);
    } finally { await context.close(); }
});
