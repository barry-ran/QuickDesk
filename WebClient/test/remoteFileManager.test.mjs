import assert from 'node:assert/strict';
import test from 'node:test';

import { RemoteFileManager } from '../js/files/remote-file-manager.js';

function fakeFile(name, bytes) {
    return {
        name,
        size: bytes.length,
        slice(start, end) {
            return {
                async arrayBuffer() {
                    return Uint8Array.from(bytes.slice(start, end)).buffer;
                },
            };
        },
    };
}

test('upload writes bounded chunks and truncates only the first chunk', async () => {
    const requests = [];
    const progress = [];
    const bridge = {
        async request(tool, args) {
            requests.push({ tool, args });
            return { bytes_written: args.offset === 0 ? 3 : 2 };
        },
    };
    const manager = new RemoteFileManager(bridge, {
        chunkBytes: 3,
        refreshAfterUpload: false,
    });
    manager.currentPath = '/tmp';
    manager.addEventListener('uploadProgress', (event) => progress.push(event.detail));

    await manager.upload(fakeFile('sample.bin', [1, 2, 3, 4, 5]), '/incoming');

    assert.deepEqual(requests.map(item => item.tool), [
        'write_file_chunk',
        'write_file_chunk',
    ]);
    assert.equal(requests[0].args.path, '/incoming/sample.bin');
    assert.equal(requests[0].args.offset, 0);
    assert.equal(requests[0].args.truncate, true);
    assert.equal(requests[1].args.offset, 3);
    assert.equal(requests[1].args.truncate, false);
    assert.deepEqual(progress.map(item => item.written), [3, 5]);
});

test('download assembles base64 chunks in server order', async () => {
    const bridge = {
        async request(tool, args) {
            assert.equal(tool, 'read_file_chunk');
            if (args.offset === 0) {
                return {
                    offset: 0,
                    next_offset: 2,
                    size: 3,
                    eof: false,
                    data_base64: 'AQI=',
                };
            }
            return {
                offset: 2,
                next_offset: 3,
                size: 3,
                eof: true,
                data_base64: 'Aw==',
            };
        },
    };
    const manager = new RemoteFileManager(bridge);

    const result = await manager.download({ name: 'sample.bin', isDirectory: false });

    assert.equal(result.filename, 'sample.bin');
    assert.deepEqual([...result.bytes], [1, 2, 3]);
});

test('rename validates the target and moves within the same directory', async () => {
    const requests = [];
    const bridge = {
        async request(tool, args) {
            requests.push({ tool, args });
            return {};
        },
    };
    const manager = new RemoteFileManager(bridge, { refreshAfterMutation: false });
    manager.currentPath = 'C:\\Users\\tim';

    await manager.rename({ name: 'old.txt' }, 'new.txt');
    assert.deepEqual(requests[0], {
        tool: 'move_file',
        args: {
            source: 'C:\\Users\\tim\\old.txt',
            destination: 'C:\\Users\\tim\\new.txt',
        },
    });
    await assert.rejects(manager.rename({ name: 'old.txt' }, 'bad/name'), /separator/);
});
