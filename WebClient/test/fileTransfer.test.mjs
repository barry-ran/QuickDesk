import assert from 'node:assert/strict';
import test from 'node:test';
import { DataChannelHandler } from '../js/protocol/datachannel-handler.js';
import { filesFromDataTransfer } from '../js/files/browser-files.js';

const flush = () => new Promise(resolve => setImmediate(resolve));
const deferred = () => {
    let resolve, reject;
    const promise = new Promise((yes, no) => { resolve = yes; reject = no; });
    return { promise, resolve, reject };
};
function file(name = 'sample.bin', bytes = [1, 2], modified = 1) {
    return {
        name, size: bytes.length, lastModified: modified,
        slice(start, end) {
            return { arrayBuffer: async () => Uint8Array.from(bytes.slice(start, end)).buffer };
        },
    };
}
class Channel {
    constructor(label) { this.label = label; this.readyState = 'connecting'; this.bufferedAmount = 0; this.sent = []; }
    send(data) {
        if (this.readyState !== 'open') throw new Error('channel not open');
        this.sent.push(data);
    }
    open() { this.readyState = 'open'; return this.onopen?.(); }
    close() { this.readyState = 'closed'; this.onclose?.(); }
}
function setup(t, options) {
    const channels = [];
    const handler = new DataChannelHandler(options);
    handler.supportsFileTransfer = true;
    handler.setPeerConnection({ connectionState: 'connected',
        createDataChannel(label) { const channel = new Channel(label); channels.push(channel); return channel; },
    });
    const events = [];
    for (const name of ['fileTransferStarted', 'fileTransferProgress', 'fileTransferComplete',
        'fileTransferError', 'fileDownloadStarted', 'fileDownloadComplete', 'fileDownloadError']) {
        handler.addEventListener(name, event => events.push({ name, ...event.detail }));
    }
    t.after(() => handler.destroy());
    return { handler, channels, events };
}
function receive(handler, channel, message) {
    channel.onmessage({ data: handler._encodeFileTransfer(message) });
}
const count = (events, name) => events.filter(event => event.name === name).length;

test('same-name uploads have distinct immediate IDs and dispatch serially', async t => {
    const { handler, channels, events } = setup(t);
    const first = handler.startFileUpload(file('same', [1], 1));
    const second = handler.startFileUpload(file('same', [2], 1));
    assert.notEqual(first, second);
    assert.equal(channels.length, 1);
    assert.equal(count(events, 'fileTransferStarted'), 2);
    channels[0].open();
    await flush();
    receive(handler, channels[0], { success: {} });
    await flush();
    assert.equal(channels.length, 2);
    assert.equal(events.find(event => event.name === 'fileTransferComplete').transferId, first);
    handler.cancelFileUpload(second);
    assert.equal(channels[1].readyState, 'closed');
});

test('repeat paste deduplicates pending files and queued cancel never creates a channel', async t => {
    const { handler, channels, events } = setup(t);
    const firstFile = file();
    const first = handler.startFileUpload(firstFile);
    assert.equal(handler.startFileUpload(firstFile), first);
    const queued = handler.startFileUpload(file('queued'));
    handler.cancelFileUpload(queued);
    channels[0].open();
    await flush();
    receive(handler, channels[0], { success: {} });
    await flush();
    assert.equal(channels.length, 1);
    assert.equal(count(events, 'fileTransferStarted'), 2);
});

test('cancel-before-open closes channel and ignores late open/success', async t => {
    const { handler, channels, events } = setup(t);
    const id = handler.startFileUpload(file());
    const lateOpen = channels[0].onopen;
    const lateMessage = channels[0].onmessage;
    handler.cancelFileUpload(id);
    assert.equal(channels[0].readyState, 'closed');
    lateOpen();
    lateMessage({ data: handler._encodeFileTransfer({ success: {} }) });
    await flush();
    assert.equal(channels[0].sent.length, 0);
    assert.equal(count(events, 'fileTransferError'), 1);
    assert.equal(count(events, 'fileTransferComplete'), 0);
});

test('file read error and channel close each terminate exactly once', async t => {
    const { handler, channels, events } = setup(t);
    const broken = file();
    broken.slice = () => ({ arrayBuffer: async () => { throw new Error('read denied'); } });
    handler.startFileUpload(broken);
    channels[0].open();
    await flush();
    assert.match(events.find(event => event.name === 'fileTransferError').errorMessage, /read denied/);
    channels[0].close();
    assert.equal(count(events, 'fileTransferError'), 1);
    handler.startFileUpload(file('second'));
    channels[1].close();
    assert.equal(count(events, 'fileTransferError'), 2);
});

test('disconnect while local read awaits prevents late send and new admission', async t => {
    const { handler, channels, events } = setup(t);
    const reading = deferred();
    const slow = file();
    slow.slice = () => ({ arrayBuffer: () => reading.promise });
    handler.startFileUpload(slow);
    channels[0].open();
    handler.closeFileTransfers();
    reading.resolve(new Uint8Array([1, 2]).buffer);
    await flush();
    assert.equal(channels[0].sent.length, 1); // metadata only
    assert.equal(count(events, 'fileTransferError'), 1);
    assert.equal(handler.startFileUpload(file()), null);
});

test('handler generations never reuse transfer IDs', t => {
    const one = setup(t), two = setup(t);
    assert.notEqual(one.handler.startFileUpload(file()), two.handler.startFileUpload(file()));
});

test('uploads reject directories, invalid sizes, and unavailable channels', t => {
    const { handler, channels } = setup(t);
    assert.equal(handler.startFileUpload({ ...file(), webkitRelativePath: 'folder/a' }), null);
    assert.equal(handler.startFileUpload({ ...file(), size: Number.MAX_SAFE_INTEGER + 1 }), null);
    handler.supportsFileTransfer = false;
    assert.equal(handler.startFileUpload(file()), null);
    assert.equal(channels.length, 0);
});

test('empty upload sends End and waits for host acknowledgement', async t => {
    const { handler, channels, events } = setup(t);
    handler.startFileUpload(file('empty', []));
    channels[0].open();
    await flush();
    assert.equal(channels[0].sent.length, 2);
    assert.equal(count(events, 'fileTransferComplete'), 0);
    receive(handler, channels[0], { success: {} });
    assert.equal(count(events, 'fileTransferComplete'), 1);
});

test('64-bit file metadata does not truncate at 32 bits', t => {
    const { handler } = setup(t);
    for (const size of [0, 2 ** 32 + 7, 2 ** 40 + 3, Number.MAX_SAFE_INTEGER]) {
        const decoded = handler._decodeFileTransfer(new Uint8Array(handler._encodeFileTransfer({
            metadata: { filename: 'large', size },
        })));
        assert.equal(decoded.metadata.size, size);
    }
    assert.throws(() => handler._encodeVarint(Number.MAX_SAFE_INTEGER + 1), /Invalid/);
});

test('late writable creation after cancellation aborts without requesting remote data', async t => {
    const { handler, channels, events } = setup(t);
    const creating = deferred();
    let aborted = 0;
    const id = handler.startFileDownload({ createWritable: () => creating.promise });
    const opening = channels[0].open();
    handler.cancelFileDownload(id);
    creating.resolve({ abort: async () => { aborted++; } });
    await opening;
    assert.equal(aborted, 1);
    assert.equal(channels[0].sent.length, 1); // explicit cancellation only
    assert.equal(count(events, 'fileDownloadError'), 1);
});

test('download close failure never sends Success or emits Complete', async t => {
    const { handler, channels, events } = setup(t);
    let aborted = 0;
    handler.startFileDownload({ createWritable: async () => ({
        write: async () => {},
        close: async () => { throw new Error('disk full on close'); },
        abort: async () => { aborted++; },
    }) });
    await channels[0].open();
    receive(handler, channels[0], { metadata: { filename: 'safe', size: 2 } });
    receive(handler, channels[0], { data: { data: new Uint8Array([1, 2]) } });
    receive(handler, channels[0], { end: {} });
    await flush();
    assert.equal(count(events, 'fileDownloadComplete'), 0);
    assert.equal(count(events, 'fileDownloadError'), 1);
    assert.equal(aborted, 1);
    assert.equal(channels[0].sent.some(data =>
        handler._decodeFileTransfer(new Uint8Array(data)).success), false);
});

test('End is authoritative and metadata size is advisory per existing protocol', async t => {
    const { handler, channels, events } = setup(t);
    handler.startFileDownload();
    await channels[0].open();
    receive(handler, channels[0], { metadata: { filename: 'file', size: 1 } });
    receive(handler, channels[0], { data: { data: new Uint8Array([1, 2]) } });
    receive(handler, channels[0], { end: {} });
    await flush();
    const completed = events.find(event => event.name === 'fileDownloadComplete');
    assert.equal(completed.blob.size, 2);
    assert.equal(count(events, 'fileDownloadError'), 0);
});

test('buffered browser downloads reject large metadata without allocating contents', async t => {
    const { handler, channels, events } = setup(t);
    handler.startFileDownload();
    await channels[0].open();
    receive(handler, channels[0], { metadata: { filename: 'large', size: 2 ** 32 + 1 } });
    await flush();
    assert.match(events.find(event => event.name === 'fileDownloadError').errorMessage, /64 MiB/);
    assert.equal(handler._activeDownloads.size, 0);
    assert.equal(channels[0].readyState, 'closed');
});

test('plain text clipboard remains untouched and directory drops are explicit', () => {
    const regular = file();
    const selection = filesFromDataTransfer({ items: [
        { kind: 'string', getAsFile() { throw new Error('must not read text'); } },
        { kind: 'file', webkitGetAsEntry: () => ({ isDirectory: true }) },
        { kind: 'file', getAsFile: () => regular },
    ] });
    assert.equal(selection.directories, true);
    assert.deepEqual(selection.files, [regular]);
    assert.deepEqual(filesFromDataTransfer({ items: [{ kind: 'string' }] }).files, []);
});

test('backpressure does not read the next chunk while the channel remains full', async t => {
    const { handler, channels } = setup(t);
    let reads = 0;
    const source = file();
    source.slice = () => { reads++; return { arrayBuffer: async () => new Uint8Array([1, 2]).buffer }; };
    handler.startFileUpload(source);
    channels[0].bufferedAmount = 1024 * 1024;
    channels[0].open();
    await new Promise(resolve => setTimeout(resolve, 60));
    assert.equal(reads, 0);
    channels[0].bufferedAmount = 0;
    await new Promise(resolve => setTimeout(resolve, 40));
    assert.equal(reads, 1);
});

test('upload idle watchdog releases a permanently backpressured queue', async t => {
    const { handler, channels, events } = setup(t, { uploadIdleMs: 15 });
    handler.startFileUpload(file('stuck')); handler.startFileUpload(file('next'));
    channels[0].bufferedAmount = 1024 * 1024; channels[0].open();
    await new Promise(resolve => setTimeout(resolve, 50));
    assert.match(events.find(event => event.name === 'fileTransferError').errorMessage, /no progress/);
    assert.equal(channels.length, 2); assert.equal(channels[0].readyState, 'closed');
});
test('malformed protobuf lengths and unterminated varints fail without looping', t => {
    const { handler } = setup(t);
    for (const bytes of [
        [0x0a, 0x80, 0x80, 0x80, 0x80, 0x08], [0x0a, 0x05, 0x01], [0x0a, 0x80],
        [0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x80, 0x00],
        [0x0b], [0x00], [0x12, 0x03, 0x0a, 0x05, 0x01], [0x0a, 0x02, 0x10, 0x80],
    ]) assert.throws(() => handler._decodeFileTransfer(Uint8Array.from(bytes)), /file|File|Invalid|Truncated/);
});
test('malformed channel message ends only its own upload once', t => {
    const { handler, channels, events } = setup(t);
    handler.startFileUpload(file());
    channels[0].onmessage({ data: Uint8Array.from([0x0a, 0x80]).buffer });
    assert.equal(count(events, 'fileTransferError'), 1); assert.equal(channels[0].readyState, 'closed');
});
test('local committed download remains successful if host acknowledgement fails', async t => {
    const { handler, channels, events } = setup(t);
    let closed = 0;
    handler.startFileDownload({ createWritable: async () => ({
        write: async () => {}, close: async () => { closed++; },
        abort: async () => { throw new Error('must not abort committed file'); },
    }) });
    await channels[0].open();
    receive(handler, channels[0], { metadata: { filename: 'saved', size: 0 } });
    receive(handler, channels[0], { end: {} });
    channels[0].close(); await flush();
    assert.equal(closed, 1); assert.equal(count(events, 'fileDownloadError'), 0);
    const result = events.find(event => event.name === 'fileDownloadComplete');
    assert.equal(result.acknowledgementSent, false); assert.equal(result.streaming, true);
});
test('cancel during writable close cannot claim rollback of a committed file', async t => {
    const { handler, channels, events } = setup(t);
    const closing = deferred(); let aborts = 0;
    const id = handler.startFileDownload({ createWritable: async () => ({
        write: async () => {}, close: () => closing.promise, abort: async () => { aborts++; },
    }) });
    await channels[0].open();
    receive(handler, channels[0], { metadata: { filename: 'saved', size: 0 } });
    receive(handler, channels[0], { end: {} }); await flush();
    handler.cancelFileDownload(id); closing.resolve(); await flush();
    assert.equal(aborts, 0); assert.equal(count(events, 'fileDownloadError'), 0);
    assert.equal(count(events, 'fileDownloadComplete'), 1);
});
