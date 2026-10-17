import assert from 'node:assert/strict';
import test from 'node:test';

import {
    breadcrumbs,
    formatFileSize,
    joinRemotePath,
    normalizeEntries,
    parentPath,
    validateEntryName,
} from '../js/files/file-manager-model.js';

test('remote path helpers handle Unix and Windows hosts', () => {
    assert.equal(parentPath('/home/user/docs'), '/home/user');
    assert.equal(parentPath('/'), '/');
    assert.equal(joinRemotePath('/home/user', 'docs'), '/home/user/docs');

    assert.equal(parentPath('C:\\Users\\tim\\docs'), 'C:\\Users\\tim');
    assert.equal(parentPath('C:\\'), 'C:\\');
    assert.equal(joinRemotePath('C:\\Users\\tim', 'docs'), 'C:\\Users\\tim\\docs');
});

test('breadcrumbs expose every navigable parent', () => {
    assert.deepEqual(breadcrumbs('C:\\Users\\tim'), [
        { label: 'C:', path: 'C:\\' },
        { label: 'Users', path: 'C:\\Users' },
        { label: 'tim', path: 'C:\\Users\\tim' },
    ]);
    assert.deepEqual(breadcrumbs('/home/tim'), [
        { label: '/', path: '/' },
        { label: 'home', path: '/home' },
        { label: 'tim', path: '/home/tim' },
    ]);
});

test('entries are normalized and directories sort first', () => {
    assert.deepEqual(normalizeEntries([
        { name: 'b.txt', type: 'file', size: 10 },
        { name: 'A', type: 'directory', size: 0 },
        { name: 'a.txt', type: 'file', size: 2 },
    ]), [
        { name: 'A', type: 'directory', size: 0, isDirectory: true },
        { name: 'a.txt', type: 'file', size: 2, isDirectory: false },
        { name: 'b.txt', type: 'file', size: 10, isDirectory: false },
    ]);
});

test('entry names reject path traversal and separators', () => {
    assert.equal(validateEntryName('report.txt'), 'report.txt');
    assert.throws(() => validateEntryName(''), /required/);
    assert.throws(() => validateEntryName('.'), /invalid/);
    assert.throws(() => validateEntryName('..'), /invalid/);
    assert.throws(() => validateEntryName('a/b'), /separator/);
    assert.throws(() => validateEntryName('a\\b'), /separator/);
});

test('file sizes use compact binary units', () => {
    assert.equal(formatFileSize(0), '0 B');
    assert.equal(formatFileSize(1024), '1.0 KB');
    assert.equal(formatFileSize(1024 * 1024), '1.0 MB');
});
