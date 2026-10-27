import assert from 'node:assert/strict';
import test from 'node:test';
import http from 'node:http';
import { readFile } from 'node:fs/promises';
import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { chromium } from 'playwright';

test('actual remote-view handlers queue browser files, reject folders and render names as text', async () => {
    const root = fileURLToPath(new URL('../../', import.meta.url));
    const server = http.createServer(async (req, res) => {
        if (req.url === '/') {
            res.setHeader('Content-Type', 'text/html');
            res.end('<div id="remoteContainer"><div id="target" tabindex="0"></div></div><div id="logContent"></div>');
            return;
        }
        // Session/auth is intentionally stubbed: this test exercises the real
        // transfer UI + DataChannelHandler with a mock transport, never a host.
        if (req.url === '/js/protocol/session.js') {
            res.setHeader('Content-Type', 'text/javascript');
            res.end('export class Session {} export const SessionState = {};');
            return;
        }
        const file = path.resolve(root, '.' + new URL(req.url, 'http://localhost').pathname);
        if (!file.startsWith(root)) { res.writeHead(403).end(); return; }
        try {
            res.setHeader('Content-Type', file.endsWith('.js') ? 'text/javascript' : 'application/octet-stream');
            res.end(await readFile(file));
        } catch { res.writeHead(404).end(); }
    });
    await new Promise(resolve => server.listen(0, '127.0.0.1', resolve));
    const browser = await chromium.launch({ headless: true });
    const page = await browser.newPage();
    const errors = [];
    page.on('pageerror', error => errors.push(error.message));
    try {
        await page.goto('http://127.0.0.1:' + server.address().port, { waitUntil: 'load' });
        const result = await page.evaluate(async () => {
            // Import after DOMContentLoaded so the production entrypoint does not
            // open a real session. All transfer UI/clipboard handlers are real.
            const { RemoteDesktopApp } = await import('/js/remote-main.js');
            const { DataChannelHandler } = await import('/js/protocol/datachannel-handler.js');
            const { KeyboardHandler } = await import('/js/input/keyboard-handler.js');
            const { ClipboardHandler } = await import('/js/input/clipboard-handler.js');
            const app = Object.create(RemoteDesktopApp.prototype);
            const handler = new DataChannelHandler();
            const channels = [];
            handler.supportsFileTransfer = true;
            handler.setPeerConnection({ connectionState: 'connected', createDataChannel(label) {
                const channel = {
                    label, readyState: 'connecting', bufferedAmount: 0, sent: [],
                    send(data) { this.sent.push(data); },
                    close() { this.readyState = 'closed'; this.onclose?.(); },
                };
                channels.push(channel); return channel;
            } });
            app.dcHandler = handler;
            app._log = () => {};
            app.floatingToolbar = { updateTransferCount() {} };
            handler.addEventListener('fileTransferStarted', event => {
                app._addTransferItem(event.detail.transferId, event.detail.filename, event.detail.totalBytes);
                app._setTransferQueued(event.detail.transferId, event.detail.queued);
            });
            handler.addEventListener('fileTransferError', event =>
                app._updateTransferStatus(event.detail.transferId, 'error', event.detail.errorMessage));
            const target = document.getElementById('target');
            const inputEvents = [];
            handler.sendKeyEvent = value => inputEvents.push({ kind: 'key', ...value });
            handler.sendClipboardEvent = value => inputEvents.push({ kind: 'clipboard', ...value });
            app.keyboardHandler = new KeyboardHandler(target, handler);
            app.keyboardHandler.enable();
            app.clipboardHandler = new ClipboardHandler(handler);
            app.clipboardHandler._enabled = true;
            app._setupPasteUpload(target);
            target.dispatchEvent(new KeyboardEvent('keydown', { code: 'ControlLeft', bubbles: true, cancelable: true }));
            const vdown = new KeyboardEvent('keydown', { code: 'KeyV', ctrlKey: true, bubbles: true, cancelable: true });
            target.dispatchEvent(vdown);
            const data = new DataTransfer();
            data.items.add(new File(['hello'], '<img src=x onerror=alert(1)>.txt', { lastModified: 1 }));
            data.items.add(new File(['world'], 'second.txt', { lastModified: 2 }));
            const pasted = new ClipboardEvent('paste', { clipboardData: data, bubbles: true, cancelable: true });
            target.dispatchEvent(pasted);
            target.dispatchEvent(new KeyboardEvent('keyup', { code: 'KeyV', ctrlKey: true, bubbles: true }));
            target.dispatchEvent(new KeyboardEvent('keyup', { code: 'ControlLeft', bubbles: true }));
            const filePasteVEvents = inputEvents.filter(value => value.usbKeycode === 0x070019).length;
            const firstIds = [...handler._activeUploads.keys()];
            // Re-submission of the exact File objects is deduplicated. A new
            // browser File with equal metadata must not be silently discarded.
            app._uploadFiles([...handler._activeUploads.values()].map(task => task.file));
            const rows = document.querySelectorAll('.transfer-item');
            const queuedId = firstIds[1];
            document.querySelector('[data-transfer-id="' + queuedId + '"] .transfer-item-cancel').click();
            const plain = new DataTransfer();
            plain.setData('text/plain', 'leave ordinary clipboard paste alone');
            target.dispatchEvent(new KeyboardEvent('keydown', { code: 'ControlLeft', bubbles: true, cancelable: true }));
            target.dispatchEvent(new KeyboardEvent('keydown', { code: 'KeyV', ctrlKey: true, bubbles: true, cancelable: true }));
            const textPaste = new ClipboardEvent('paste', { clipboardData: plain, bubbles: true, cancelable: true });
            target.dispatchEvent(textPaste);
            target.dispatchEvent(new KeyboardEvent('keyup', { code: 'KeyV', ctrlKey: true, bubbles: true }));
            target.dispatchEvent(new KeyboardEvent('keyup', { code: 'ControlLeft', bubbles: true }));
            const localInput = document.createElement('input');
            target.appendChild(localInput);
            const beforeLocal = inputEvents.length;
            const localDown = new KeyboardEvent('keydown', { code: 'KeyV', ctrlKey: true, bubbles: true, cancelable: true });
            localInput.dispatchEvent(localDown);
            const localPaste = new ClipboardEvent('paste', { clipboardData: data, bubbles: true, cancelable: true });
            localInput.dispatchEvent(localPaste);
            localInput.dispatchEvent(new KeyboardEvent('keyup', { code: 'KeyV', bubbles: true }));
            const snapshot = {
                editableIgnored: !localDown.defaultPrevented && !localPaste.defaultPrevented
                    && inputEvents.length === beforeLocal,
                handled: pasted.defaultPrevented,
                keydownPrevented: vdown.defaultPrevented,
                filePasteVEvents,
                textPasteVEvents: inputEvents.filter(value => value.usbKeycode === 0x070019).length,
                clipboardBeforeV: inputEvents.findIndex(value => value.kind === 'clipboard')
                    < inputEvents.findIndex(value => value.usbKeycode === 0x070019),
                textHandled: textPaste.defaultPrevented,
                rows: rows.length,
                name: rows[0].querySelector('.transfer-item-name').textContent,
                injectedImages: document.querySelectorAll('.transfer-item img').length,
                channels: channels.length,
                pending: handler._activeUploads.size,
            };
            app._cleanupInputHandlers();
            let submitted = 0;
            app._uploadFiles = () => { submitted++; };
            app._setupPasteUpload(target);
            target.dispatchEvent(new ClipboardEvent('paste', { clipboardData: data, bubbles: true, cancelable: true }));
            snapshot.afterReconnectSubmissions = submitted;
            app._cleanupInputHandlers();
            handler.destroy();
            return snapshot;
        });
        assert.equal(result.handled, true);
        assert.equal(result.editableIgnored, true);
        assert.equal(result.textHandled, true);
        assert.equal(result.keydownPrevented, false);
        assert.equal(result.filePasteVEvents, 0);
        assert.equal(result.textPasteVEvents, 2);
        assert.equal(result.clipboardBeforeV, true);
        assert.equal(result.afterReconnectSubmissions, 1);
        assert.equal(result.rows, 2);
        assert.equal(result.injectedImages, 0);
        assert.equal(result.name, '<img src=x onerror=alert(1)>.txt');
        assert.equal(result.channels, 1);
        assert.equal(result.pending, 1);
        assert.deepEqual(errors, []);
    } finally {
        await browser.close();
        await new Promise(resolve => server.close(resolve));
    }
});
