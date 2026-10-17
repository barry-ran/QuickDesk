import assert from 'node:assert/strict';
import test from 'node:test';

import { SkillBridgeClient } from '../js/protocol/skill-bridge-client.js';

test('request sends a correlated tool call and resolves its result', async () => {
    const sent = [];
    const client = new SkillBridgeClient((data) => sent.push(JSON.parse(data)));

    const pending = client.request('list_directory', { path: 'C:\\' });
    assert.equal(sent.length, 1);
    assert.equal(sent[0].type, 'toolCall');
    assert.equal(sent[0].tool, 'list_directory');
    assert.deepEqual(sent[0].args, { path: 'C:\\' });

    client.handleMessage({
        id: sent[0].id,
        type: 'toolResult',
        result: { path: 'C:\\', entries: [] },
    });

    assert.deepEqual(await pending, { path: 'C:\\', entries: [] });
});

test('request rejects toolError and ignores unrelated responses', async () => {
    const sent = [];
    const client = new SkillBridgeClient((data) => sent.push(JSON.parse(data)));
    const pending = client.request('read_file_chunk', { path: 'missing.txt' });

    client.handleMessage({ id: 'another-client', type: 'toolResult', result: {} });
    client.handleMessage({
        id: sent[0].id,
        type: 'toolError',
        error: 'path does not exist',
    });

    await assert.rejects(pending, /path does not exist/);
});

test('request times out and late responses do not resolve it', async () => {
    const sent = [];
    const client = new SkillBridgeClient(
        (data) => sent.push(JSON.parse(data)),
        { timeoutMs: 10 },
    );

    const pending = client.request('get_default_directory', {});
    await assert.rejects(pending, /timed out/);
    assert.equal(client.pendingCount, 0);
    assert.doesNotThrow(() => client.handleMessage({
        id: sent[0].id,
        type: 'toolResult',
        result: { path: '/' },
    }));
});

test('capabilities track the available file tools', () => {
    const client = new SkillBridgeClient(() => {});
    let eventCount = 0;
    client.addEventListener('capabilitiesChanged', () => eventCount++);

    client.handleMessage({
        type: 'capabilitiesReady',
        tools: [
            { name: 'list_directory' },
            { name: 'read_file_chunk' },
        ],
    });
    assert.equal(client.supportsTool('list_directory'), true);
    assert.equal(client.supportsTool('trash_path'), false);

    client.handleMessage({
        type: 'capabilitiesChanged',
        added: [{ name: 'trash_path' }],
        removed: [{ name: 'read_file_chunk' }],
    });
    assert.equal(client.supportsTool('trash_path'), true);
    assert.equal(client.supportsTool('read_file_chunk'), false);
    assert.equal(eventCount, 2);
});
