import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';

const windowSource = readFileSync(new URL('../../QuickDesk/qml/views/RemoteWindow.qml', import.meta.url), 'utf8');
const viewSource = readFileSync(new URL('../../QuickDesk/qml/quickdeskcomponent/RemoteDesktopView.qml', import.meta.url), 'utf8');

// Execute the actual production JS bodies, not a duplicate of their logic.
// The scanner skips strings/comments so braces in a diagnostic cannot break extraction.
function body(source, marker) {
    const markerAt = source.indexOf(marker);
    assert.notEqual(markerAt, -1, `Missing QML handler: ${marker}`);
    const begin = source.indexOf('{', markerAt);
    let depth = 1;
    let quote = '';
    let lineComment = false;
    let blockComment = false;
    for (let i = begin + 1; i < source.length; ++i) {
        const c = source[i], next = source[i + 1];
        if (lineComment) { if (c === '\n') lineComment = false; continue; }
        if (blockComment) { if (c === '*' && next === '/') { blockComment = false; ++i; } continue; }
        if (quote) {
            if (c === '\\') { ++i; continue; }
            if (c === quote) quote = '';
            continue;
        }
        if (c === '/' && next === '/') { lineComment = true; ++i; continue; }
        if (c === '/' && next === '*') { blockComment = true; ++i; continue; }
        if (c === '"' || c === "'" || c === '`') { quote = c; continue; }
        if (c === '{') ++depth;
        if (c === '}' && --depth === 0) return source.slice(begin + 1, i);
    }
    throw new Error(`Unclosed QML handler: ${marker}`);
}

function handler(source, marker, scope, argument = 'event') {
    return new Function('scope', argument, `with (scope) { ${body(source, marker)} }`).bind(null, scope);
}

function modelFixture() {
    const rows = [];
    const model = {
        get count() { return rows.length; },
        get: index => rows[index],
        append: item => rows.push({ ...item }),
        set: (index, item) => { rows[index] = { ...item }; },
        remove: index => rows.splice(index, 1),
    };
    const scope = {
        transferViewReady: true,
        clientManager: {
            fileTransferTasks: [],
            clearCompletedFileTasks() { ++scope.clearRequests; },
        },
        clearRequests: 0,
        transferModel: model,
        clearedTransferTasks: {},
        deletedTransferTasks: {},
        qsTr: text => text,
        fileTransferDrawer: { open() { ++scope.opens; } },
        opens: 0,
    };
    scope.findTransferTask = handler(windowSource, 'function findTransferTask(', scope, 'taskId');
    scope.transferIsActive = handler(windowSource, 'function transferIsActive(', scope, 'status');
    scope.remoteWindow = scope;
    scope.syncTransferTasks = handler(windowSource, 'function syncTransferTasks(', scope);
    return { rows, scope, sync: scope.syncTransferTasks };
}

test('same-name tasks from two sessions keep independent state and stable local identity', () => {
    const { rows, scope, sync } = modelFixture();
    scope.clientManager.fileTransferTasks = [
        { taskId: 'session-a-1', deviceId: 'a', filename: 'same.txt', status: 'queued' },
        { taskId: 'session-b-1', deviceId: 'b', filename: 'same.txt', status: 'uploading' },
    ];
    sync();
    assert.equal(rows.length, 2);
    assert.equal(scope.opens, 1);
    scope.clientManager.fileTransferTasks[0].status = 'cancelled';
    sync();
    assert.equal(rows[0].status, 'cancelled');
    assert.equal(rows[1].status, 'uploading');
    assert.equal(scope.opens, 1, 'progress updates must not reopen the drawer');
    scope.clientManager.fileTransferTasks = [scope.clientManager.fileTransferTasks[1]];
    sync();
    assert.deepEqual(rows.map(row => row.taskId), ['session-b-1']);
});

test('clear completed does not resurrect on the next snapshot; uncertain work remains visible', () => {
    const { rows, scope, sync } = modelFixture();
    scope.clientManager.fileTransferTasks = [
        { taskId: 'done', filename: 'done.txt', status: 'complete' },
        { taskId: 'pending', filename: 'pending.txt', status: 'uncertain' },
    ];
    sync();
    const clearSection = windowSource.slice(windowSource.indexOf('// Clear completed button'));
    handler(clearSection, 'onClicked:', scope)();
    assert.equal(scope.clearRequests, 1, 'the authoritative history is cleared asynchronously');
    assert.deepEqual(rows.map(row => row.taskId), ['pending']);
    sync();
    assert.deepEqual(rows.map(row => row.taskId), ['pending']);
});

test('successful explicit deletion remains deleted in later snapshots', () => {
    const { rows, scope, sync } = modelFixture();
    scope.clientManager.fileTransferTasks = [
        { taskId: 'done', filename: 'done.txt', status: 'complete', direction: 'download', savePath: '/tmp/done.txt' },
    ];
    sync();
    let deletedPath;
    scope.clientManager.deleteDownloadedFile = path => { deletedPath = path; return true; };
    scope.model = rows[0];
    const deleteSection = windowSource.slice(windowSource.indexOf('ToolTip.text: qsTr("Delete File")'));
    handler(deleteSection, 'onClicked:', scope)();
    assert.equal(deletedPath, '/tmp/done.txt');
    sync();
    assert.equal(rows[0].status, 'deleted');
});

function keyboardFixture() {
    const calls = [];
    let fileClipboard = true;
    const scope = {
        root: {
            inputEnabled: true, deviceId: 'a', filePasteKeyHeld: false, forwardedKeys: {},
            clientManager: {
                pasteFilesFromClipboard() { calls.push('paste'); return fileClipboard; },
                sendKeyPress() { calls.push('down'); },
                sendKeyRelease() { calls.push('up'); },
                releaseAllInput() { calls.push('releaseAll'); },
            },
        },
        Qt: { Key_V: 86, ControlModifier: 1 },
        KeyboardStateTracker: { getLastNativeKeycode: () => 42, getLockStates: () => 0 },
    };
    return {
        scope, calls,
        setFileClipboard: value => { fileClipboard = value; },
        press: handler(viewSource, 'Keys.onPressed: function(event)', scope),
        release: handler(viewSource, 'Keys.onReleased: function(event)', scope),
        // releaseRemoteInput is a member function: its unqualified properties are root properties.
        reset: handler(viewSource, 'function releaseRemoteInput()', scope.root),
    };
}
const key = (repeat = false) => ({ key: 86, modifiers: 1, isAutoRepeat: repeat });

test('one file-paste press consumes repeats and matching release, even if clipboard changes', () => {
    const { calls, press, release, setFileClipboard } = keyboardFixture();
    press(key());
    setFileClipboard(false);
    press(key(true));
    release(key(true));
    release(key());
    assert.deepEqual(calls, ['paste']);
});

test('focus reset does not forward orphan repeats or key-up from intercepted paste', () => {
    const { calls, press, release, reset } = keyboardFixture();
    press(key());
    reset();
    press(key(true));
    release(key());
    assert.deepEqual(calls, ['paste', 'releaseAll']);
    assert.match(viewSource, /onActiveFocusChanged:[\s\S]*?releaseRemoteInput\(\)/);
});

test('plain-text paste keeps normal remote press/release and auto-repeat behavior', () => {
    const { calls, press, release, setFileClipboard } = keyboardFixture();
    setFileClipboard(false);
    press(key());
    release(key(true));
    press(key(true));
    release(key());
    release(key());
    assert.deepEqual(calls, ['paste', 'down', 'up', 'down', 'up']);
});

test('upload dialog captures destination before a tab switch and does not add ad-hoc tasks', () => {
    const calls = [];
    const scope = {
        currentTabIndex: 0,
        connectionModel: { count: 2, deviceIdAt: index => ['a', 'b'][index] },
        fileDialog: { targetDeviceId: '', open() { calls.push('open'); } },
        remoteWindow: { clientManager: { startFileUpload: (id, file) => calls.push([id, file]) } },
    };
    handler(windowSource, 'onUploadFileRequested:', scope)();
    scope.currentTabIndex = 1;
    const dialogScope = {
        ...scope,
        targetDeviceId: scope.fileDialog.targetDeviceId,
        selectedFiles: ['file:///first.txt', 'file:///second.txt'],
    };
    const dialogSection = windowSource.slice(windowSource.indexOf('id: fileDialog'));
    handler(dialogSection, 'onAccepted:', dialogScope)();
    assert.deepEqual(calls, ['open', ['a', 'file:///first.txt'], ['a', 'file:///second.txt']]);
    assert.equal(dialogScope.targetDeviceId, '');
    assert.doesNotMatch(body(dialogSection, 'onAccepted:'), /transferModel\.append/);
    assert.doesNotMatch(body(windowSource, 'onFilesDropped: function(urls)'), /transferModel\.append/);
});

test('copy downloaded file is an explicit task-ID action, never a snapshot side effect', () => {
    const { scope, sync } = modelFixture();
    const copied = [];
    scope.clientManager.copyDownloadedFile = id => { copied.push(id); return true; };
    scope.clientManager.fileTransferTasks = [{ taskId: 'download-1', status: 'complete', direction: 'download', savePath: '/tmp/a' }];
    sync();
    assert.deepEqual(copied, []);
    scope.model = { taskId: 'download-1' };
    scope.toast = { show() {} };
    scope.QDToast = { Type: { Success: 1 } };
    const copySection = windowSource.slice(windowSource.indexOf('ToolTip.text: qsTr("Copy Downloaded File")'));
    handler(copySection, 'onClicked:', scope)();
    assert.deepEqual(copied, ['download-1']);
    assert.doesNotMatch(windowSource, /function onFileTransferProgress|function findTransferByFilename/);
});
