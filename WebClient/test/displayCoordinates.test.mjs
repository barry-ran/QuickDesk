import assert from 'node:assert/strict';
import { test } from 'node:test';
import { MouseHandler } from '../js/input/mouse-handler.js';

for (const [originX, originY] of [[0, 0], [1920, 100], [-1280, -1024]]) {
    test(`physical mouse uses virtual desktop coordinates at ${originX},${originY}`, () => {
        const events = [];
        const video = { videoWidth: 2560, videoHeight: 2048,
            getBoundingClientRect: () => ({ left: 0, top: 0, width: 1280, height: 1024 }) };
        const handler = new MouseHandler(video, { sendMouseEvent: e => events.push(e) });
        handler.setRemoteDisplay(1280, 1024, originX, originY);
        handler._onMouseMove({ clientX: 640, clientY: 512 });
        handler._onWheel({ clientX: 1280, clientY: 1024, deltaX: 0, deltaY: 120, deltaMode: 0, preventDefault() {} });
        assert.deepEqual(events[0], { x: originX + 640, y: originY + 512 });
        assert.equal(events[1].x, originX + 1279);
        assert.equal(events[1].y, originY + 1023);
        assert.equal(handler._mapCoordinates({ clientX: -1, clientY: 0 }), null);
    });
}

test('production display selection and layout refresh feed signed origin to touch and mouse', async () => {
    globalThis.document = { addEventListener() {}, getElementById() { return null; } };
    globalThis.localStorage = { getItem() { return null; } };
    const { RemoteDesktopApp } = await import('../js/remote-main.js');
    const { TouchHandler } = await import('../js/input/touch-handler.js');
    const { encodeEventMessage, decodeEventMessage } = await import('../js/protocol/protobuf-messages.js');
    const events = [];
    const channel = { sendMouseEvent(event) {
        // Real wire encoding/decoding must preserve negative int32 coordinates.
        events.push(decodeEventMessage(encodeEventMessage({ mouseEvent: event })).mouseEvent);
    } };
    const touch = new TouchHandler({}, {}, channel);
    const mouse = new MouseHandler({}, channel);
    const app = Object.create(RemoteDesktopApp.prototype);
    Object.assign(app, { _log() {}, touchHandler: touch, mouseHandler: mouse });
    const primary = { mediaStreamId: 'primary', screenId: 1, width: 1600, height: 900 };
    const secondary = { mediaStreamId: 'secondary', screenId: 2, width: 1600, height: 900, positionX: -1600, positionY: -900 };
    app._onVideoLayout({ videoTracks: [primary, secondary], primaryScreenId: 1 });
    touch.click(); assert.equal(events.at(-1).x, 800);
    app._selectDisplay(1);
    touch.click(); assert.equal(events.at(-1).x, -800); assert.equal(events.at(-1).y, -450);
    assert.equal(mouse.remoteOriginX, -1600);
    app._onVideoLayout({ videoTracks: [primary, { ...secondary, positionX: 1600, positionY: 100 }], primaryScreenId: 1 });
    touch.click(); assert.equal(events.at(-1).x, 2400); assert.equal(events.at(-1).y, 550);
    app._selectDisplay(0);
    touch.click(); assert.equal(events.at(-1).x, 800); assert.equal(events.at(-1).y, 450);
});
