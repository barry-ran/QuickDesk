import {
    joinRemotePath,
    normalizeEntries,
    parentPath,
    validateEntryName,
} from './file-manager-model.js';

const DEFAULT_CHUNK_BYTES = 64 * 1024;

export function bytesToBase64(bytes) {
    let binary = '';
    const block = 0x8000;
    for (let index = 0; index < bytes.length; index += block) {
        binary += String.fromCharCode(...bytes.subarray(index, index + block));
    }
    return btoa(binary);
}

export function base64ToBytes(value) {
    const binary = atob(value || '');
    const bytes = new Uint8Array(binary.length);
    for (let index = 0; index < binary.length; index++) {
        bytes[index] = binary.charCodeAt(index);
    }
    return bytes;
}

export class RemoteFileManager extends EventTarget {
    constructor(bridge, {
        chunkBytes = DEFAULT_CHUNK_BYTES,
        refreshAfterMutation = true,
        refreshAfterUpload = true,
        encodeBase64 = bytesToBase64,
        decodeBase64 = base64ToBytes,
    } = {}) {
        super();
        if (!bridge || typeof bridge.request !== 'function') {
            throw new TypeError('bridge.request is required');
        }
        this.bridge = bridge;
        this.chunkBytes = Math.max(1, Number(chunkBytes) || DEFAULT_CHUNK_BYTES);
        this.refreshAfterMutation = refreshAfterMutation;
        this.refreshAfterUpload = refreshAfterUpload;
        this.encodeBase64 = encodeBase64;
        this.decodeBase64 = decodeBase64;
        this.currentPath = '';
        this.entries = [];
        this.busy = false;
    }

    async initialize() {
        const result = await this.bridge.request('get_default_directory', {});
        const path = result?.path || '';
        if (!path) throw new Error('The host did not return a default directory');
        await this.navigate(path);
        return path;
    }

    async navigate(path) {
        if (!path) return;
        this._setBusy(true);
        try {
            const result = await this.bridge.request('list_directory', { path });
            const resolvedPath = result?.path || path;
            this.currentPath = resolvedPath;
            this.entries = normalizeEntries(result?.entries);
            this.dispatchEvent(new CustomEvent('listingChanged', {
                detail: { path: resolvedPath, entries: this.entries },
            }));
            return this.entries;
        } finally {
            this._setBusy(false);
        }
    }

    async refresh() {
        if (this.currentPath) return this.navigate(this.currentPath);
        return [];
    }

    async up() {
        const parent = parentPath(this.currentPath);
        if (parent && parent !== this.currentPath) return this.navigate(parent);
        return this.entries;
    }

    async createDirectory(name) {
        const validName = validateEntryName(name);
        const path = joinRemotePath(this.currentPath, validName);
        await this.bridge.request('create_directory', { path });
        await this._refreshMutation();
        return path;
    }

    async rename(entry, newName) {
        const validName = validateEntryName(newName);
        const source = joinRemotePath(this.currentPath, entry.name);
        const destination = joinRemotePath(parentPath(source), validName);
        await this.bridge.request('move_file', { source, destination });
        await this._refreshMutation();
        return destination;
    }

    async trash(entry) {
        const path = joinRemotePath(this.currentPath, entry.name);
        await this.bridge.request('trash_path', { path });
        await this._refreshMutation();
        return path;
    }

    async readTextPreview(entry, length = DEFAULT_CHUNK_BYTES) {
        const path = typeof entry === 'string'
            ? entry
            : joinRemotePath(this.currentPath, entry.name);
        const result = await this.bridge.request('read_file_chunk', {
            path,
            offset: 0,
            length: Math.min(this.chunkBytes, Math.max(1, Number(length) || DEFAULT_CHUNK_BYTES)),
        });
        const bytes = this.decodeBase64(result?.data_base64 || '');
        return {
            path,
            text: new TextDecoder().decode(bytes),
            size: Number(result?.size) || bytes.length,
            truncated: result?.eof === false,
        };
    }

    async upload(file, directory = this.currentPath) {
        if (!file || !file.name) throw new Error('A file is required');
        const filename = validateEntryName(file.name);
        const path = joinRemotePath(directory, filename);
        const total = Number(file.size) || 0;
        let offset = 0;

        if (total === 0) {
            await this.bridge.request('write_file_chunk', {
                path,
                offset: 0,
                truncate: true,
                data_base64: '',
            });
        } else {
            while (offset < total) {
                const end = Math.min(offset + this.chunkBytes, total);
                const buffer = await file.slice(offset, end).arrayBuffer();
                const data = new Uint8Array(buffer);
                const result = await this.bridge.request('write_file_chunk', {
                    path,
                    offset,
                    truncate: offset === 0,
                    data_base64: this.encodeBase64(data),
                });
                const next = Number(result?.next_offset);
                offset = Number.isFinite(next) && next > offset ? next : end;
                this.dispatchEvent(new CustomEvent('uploadProgress', {
                    detail: { filename, path, written: offset, total },
                }));
            }
        }

        if (this.refreshAfterUpload) await this.refresh();
        return { path, filename, size: total };
    }

    async download(entry) {
        if (!entry || entry.isDirectory) throw new Error('Select a file to download');
        const path = joinRemotePath(this.currentPath, entry.name);
        const chunks = [];
        let offset = 0;

        while (true) {
            const result = await this.bridge.request('read_file_chunk', {
                path,
                offset,
                length: this.chunkBytes,
            });
            chunks.push(this.decodeBase64(result?.data_base64 || ''));
            const next = Number(result?.next_offset);
            if (result?.eof === true) break;
            if (!Number.isFinite(next) || next <= offset) {
                throw new Error('The host returned a non-advancing file offset');
            }
            offset = next;
        }

        const size = chunks.reduce((sum, chunk) => sum + chunk.length, 0);
        const bytes = new Uint8Array(size);
        let cursor = 0;
        for (const chunk of chunks) {
            bytes.set(chunk, cursor);
            cursor += chunk.length;
        }
        return { filename: entry.name, path, bytes };
    }

    async _refreshMutation() {
        if (this.refreshAfterMutation) await this.refresh();
    }

    _setBusy(value) {
        this.busy = value;
        this.dispatchEvent(new CustomEvent('busyChanged', {
            detail: { busy: value },
        }));
    }
}
