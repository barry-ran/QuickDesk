/**
 * datachannel-handler.js - DataChannel 处理器
 * 
 * 参照 src/remoting/protocol/webrtc_data_stream_adapter.cc
 * 和 src/remoting/proto/internal.proto
 * 
 * 管理 Host 创建的 DataChannel (control, event)
 */

import {
    encodeEventMessage, encodeControlMessage,
    decodeControlMessage, decodeEventMessage
} from './protobuf-messages.js';

let nextFileTransferId = 1; // Unique across handler/session generations in this page.

export class DataChannelHandler extends EventTarget {
    constructor() {
        super();
        
        this.controlChannel = null;
        this.eventChannel = null;
        this.actionsChannel = null;
        this._controlReady = false;
        this._eventReady = false;
        this._actionsReady = false;

        this._hostCapabilities = '';
        this.supportsSendAttentionSequence = false;
        this.supportsLockWorkstation = false;
        this.supportsFileTransfer = false;
        this.supportsPrivacyScreen = false;
        this.supportsVirtualDisplay = false;

        this._fileTransfersClosed = false;
        this._activeUploads = new Map();
        this._activeDownloads = new Map();
    }

    /**
     * 处理新的 DataChannel
     * @param {RTCDataChannel} channel 
     */
    handleDataChannel(channel) {
        console.log(`[DataChannel] New channel: ${channel.label}, id=${channel.id}`);
        
        channel.binaryType = 'arraybuffer';

        switch (channel.label) {
            case 'control':
                this._setupControlChannel(channel);
                break;
            case 'event':
                this._setupEventChannel(channel);
                break;
            default:
                console.log(`[DataChannel] Unknown channel: ${channel.label}`);
                this._setupGenericChannel(channel);
        }
    }

    /**
     * 设置 control DataChannel
     * @private
     */
    _setupControlChannel(channel) {
        this.controlChannel = channel;
        
        channel.onopen = () => {
            console.log('[DataChannel] Control channel opened');
            this._controlReady = true;
            this.dispatchEvent(new CustomEvent('controlReady'));
        };

        channel.onclose = () => {
            console.log('[DataChannel] Control channel closed');
            this._controlReady = false;
        };

        channel.onmessage = (event) => {
            try {
                const data = new Uint8Array(event.data);
                const message = decodeControlMessage(data);
                this._handleControlMessage(message);
            } catch (e) {
                console.error('[DataChannel] Failed to decode control message:', e);
            }
        };

        channel.onerror = (event) => {
            console.error('[DataChannel] Control channel error:', event);
        };
    }

    /**
     * 设置 event DataChannel
     * @private
     */
    _setupEventChannel(channel) {
        this.eventChannel = channel;
        
        channel.onopen = () => {
            console.log('[DataChannel] Event channel opened');
            this._eventReady = true;
            this.dispatchEvent(new CustomEvent('eventReady'));
        };

        channel.onclose = () => {
            console.log('[DataChannel] Event channel closed');
            this._eventReady = false;
        };

        channel.onmessage = (event) => {
            try {
                const data = new Uint8Array(event.data);
                const message = decodeEventMessage(data);
                this._handleEventMessage(message);
            } catch (e) {
                console.error('[DataChannel] Failed to decode event message:', e);
            }
        };
    }

    /**
     * 设置通用 DataChannel (用于未知通道)
     * @private
     */
    _setupGenericChannel(channel) {
        channel.onopen = () => {
            console.log(`[DataChannel] ${channel.label} opened`);
            this.dispatchEvent(new CustomEvent('channelOpen', { 
                detail: { label: channel.label, channel } 
            }));
        };
        channel.onmessage = (event) => {
            this.dispatchEvent(new CustomEvent('channelMessage', { 
                detail: { label: channel.label, data: event.data } 
            }));
        };
    }

    /**
     * 处理 control 消息
     * @private
     */
    _handleControlMessage(message) {
        if (message.cursorShape) {
            this.dispatchEvent(new CustomEvent('cursorShape', { detail: message.cursorShape }));
        }
        if (message.clipboardEvent) {
            this.dispatchEvent(new CustomEvent('clipboardEvent', { detail: message.clipboardEvent }));
        }
        if (message.capabilities) {
            this._onHostCapabilities(message.capabilities.capabilities || '');
            this.dispatchEvent(new CustomEvent('capabilities', { detail: message.capabilities }));
        }
        if (message.videoLayout) {
            this.dispatchEvent(new CustomEvent('videoLayout', { detail: message.videoLayout }));
        }
        if (message.videoControl) {
            this.dispatchEvent(new CustomEvent('videoControl', { detail: message.videoControl }));
        }
        if (message.extensionMessage) {
            this.dispatchEvent(new CustomEvent('extensionMessage', { detail: message.extensionMessage }));
        }
        if (message.transportInfo) {
            this.dispatchEvent(new CustomEvent('transportInfo', { detail: message.transportInfo }));
        }
    }

    /**
     * 处理 event 消息 (来自 host 的事件，通常不多)
     * @private
     */
    _handleEventMessage(message) {
        this.dispatchEvent(new CustomEvent('eventMessage', { detail: message }));
    }

    // ==================== 发送方法 ====================

    /**
     * 发送鼠标事件
     * @param {object} mouseEvent - { x, y, button, buttonDown, wheelDeltaX, wheelDeltaY, wheelTicksX, wheelTicksY }
     */
    sendMouseEvent(mouseEvent) {
        if (!this._eventReady) return;
        const data = encodeEventMessage({
            timestamp: Date.now(),
            mouseEvent,
        });
        this.eventChannel.send(data.buffer);
    }

    /**
     * 发送键盘事件
     * @param {object} keyEvent - { pressed, usbKeycode }
     */
    sendKeyEvent(keyEvent) {
        if (!this._eventReady) return;
        const data = encodeEventMessage({
            timestamp: Date.now(),
            keyEvent,
        });
        this.eventChannel.send(data.buffer);
    }

    /**
     * 发送文本事件（用于非物理键盘输入：中文、日文等 IME 输入）
     * @param {string} text - UTF-8 文本
     */
    sendTextEvent(text) {
        if (!this._eventReady) return;
        const data = encodeEventMessage({
            timestamp: Date.now(),
            textEvent: { text },
        });
        this.eventChannel.send(data.buffer);
    }

    /**
     * 发送剪贴板事件
     * @param {object} clipboardEvent - { mimeType, data }
     */
    sendClipboardEvent(clipboardEvent) {
        if (!this._controlReady) return;
        const data = encodeControlMessage({ clipboardEvent });
        this.controlChannel.send(data.buffer);
    }

    /**
     * 发送客户端分辨率
     * @param {object} resolution - { widthPixels, heightPixels, xDpi, yDpi, screenId }
     */
    sendClientResolution(resolution) {
        if (!this._controlReady) return;
        const data = encodeControlMessage({ clientResolution: resolution });
        this.controlChannel.send(data.buffer);
    }

    /**
     * 发送视频控制
     * @param {object} videoControl - { enable, targetFramerate, framerateBoost }
     */
    sendVideoControl(videoControl) {
        if (!this._controlReady) return;
        const data = encodeControlMessage({ videoControl });
        this.controlChannel.send(data.buffer);
    }

    /**
     * 发送音频控制
     * @param {object} audioControl - { enable }
     */
    sendAudioControl(audioControl) {
        if (!this._controlReady) return;
        const data = encodeControlMessage({ audioControl });
        this.controlChannel.send(data.buffer);
    }

    /**
     * 发送能力信息
     * @param {string} capabilities - 空格分隔的能力列表
     */
    sendCapabilities(capabilities) {
        if (!this._controlReady) return;
        const data = encodeControlMessage({ capabilities: { capabilities } });
        this.controlChannel.send(data.buffer);
    }

    /**
     * 发送 PeerConnection 参数
     * @param {object} params - { preferredMinBitrateBps, preferredMaxBitrateBps }
     */
    sendPeerConnectionParameters(params) {
        if (!this._controlReady) return;
        const data = encodeControlMessage({ peerConnectionParameters: params });
        this.controlChannel.send(data.buffer);
    }

    /**
     * 发送扩展消息
     * @param {string} type 
     * @param {string} msgData 
     */
    sendExtensionMessage(type, msgData) {
        if (!this._controlReady) return;
        const data = encodeControlMessage({ 
            extensionMessage: { type, data: msgData } 
        });
        this.controlChannel.send(data.buffer);
    }

    /**
     * Set the RTCPeerConnection for creating outgoing data channels.
     * Must be called before capabilities negotiation.
     * @param {RTCPeerConnection} pc 
     */
    setPeerConnection(pc) {
        this._pc = pc;
    }

    /**
     * Send a remote action.
     * @param {'sendAttentionSequence'|'lockWorkstation'|'enablePrivacyScreen'|'disablePrivacyScreen'} action 
     */
    sendAction(action) {
        if (!this._actionsReady) {
            console.warn('[DataChannel] Actions channel not ready');
            return;
        }
        const actionMap = {
            'sendAttentionSequence': 1,
            'lockWorkstation': 2,
            'enablePrivacyScreen': 3,
            'disablePrivacyScreen': 4,
        };
        const actionEnum = actionMap[action];
        if (actionEnum === undefined) {
            console.warn(`[DataChannel] Unknown action: ${action}`);
            return;
        }
        const requestId = this._nextActionRequestId++;
        const bytes = [0x08, actionEnum, 0x10, ...this._encodeVarint(requestId)];
        this.actionsChannel.send(new Uint8Array(bytes).buffer);
        console.log(`[DataChannel] Sent action: ${action} (request_id=${requestId})`);
    }

    /**
     * Start uploading a file to the remote host.
     * @param {File} file - Browser File object from <input type="file">
     * @returns {string|null} transferId or null on failure
     */
    startFileUpload(file) {
        if (this._fileTransfersClosed || !this._pc || !this.supportsFileTransfer || this._pc.connectionState === 'closed' ||
            this._pc.connectionState === 'failed') return null;
        if (!file || !file.name || file.webkitRelativePath || typeof file.slice !== 'function' ||
            !Number.isSafeInteger(file.size) || file.size < 0) return null;
        const key = [file.name, file.size, file.lastModified || 0].join('\0');
        for (const task of this._activeUploads.values())
            if (task.key === key) return task.transferId;
        if (this._activeUploads.size >= 256) return null;
        const transferId = String(nextFileTransferId++);
        const state = {
            transferId, filename: file.name, totalBytes: file.size, bytesSent: 0,
            key, file, channel: null, status: 'queued', cancelled: false,
        };
        this._activeUploads.set(transferId, state);
        this.dispatchEvent(new CustomEvent('fileTransferStarted', {
            detail: { transferId, filename: file.name, totalBytes: file.size, queued: true },
        }));
        this._pumpUploads();
        return transferId;
    }

    _pumpUploads() {
        if (this._fileTransfersClosed || !this._pc || [...this._activeUploads.values()].some(task => task.status === 'running'))
            return;
        const state = [...this._activeUploads.values()].find(task => task.status === 'queued');
        if (!state) return;
        state.status = 'running';
        this.dispatchEvent(new CustomEvent('fileTransferState', {
            detail: { transferId: state.transferId, queued: false },
        }));
        try {
            const channel = this._pc.createDataChannel('filetransfer-' + state.transferId, { ordered: true });
            state.channel = channel;
            channel.binaryType = 'arraybuffer';
            state.openTimer = setTimeout(() => this._finishUpload(state, 'File transfer channel did not open'), 30000);
            channel.onopen = () => {
                clearTimeout(state.openTimer);
                this._sendUpload(state).catch(error => this._finishUpload(state, error.message));
            };
            channel.onmessage = event => {
                if (!this._uploadActive(state)) return;
                try {
                    const message = this._decodeFileTransfer(new Uint8Array(event.data));
                    if (message.success) {
                        if (!state.sentEnd) throw new Error('Premature upload acknowledgement');
                        this._finishUpload(state);
                    } else if (message.error) {
                        this._finishUpload(state, 'Host error type=' + message.error.type);
                    }
                } catch (error) { this._finishUpload(state, error.message); }
            };
            channel.onclose = () => this._finishUpload(state, 'Channel closed before upload was confirmed');
            channel.onerror = () => this._finishUpload(state, 'File transfer channel error');
        } catch (error) { this._finishUpload(state, error.message); }
    }

    _uploadActive(state) {
        return this._activeUploads.get(state.transferId) === state && !state.cancelled;
    }

    async _sendUpload(state) {
        if (!this._uploadActive(state)) return;
        const channel = state.channel;
        const file = state.file;
        channel.send(this._encodeFileTransfer({ metadata: { filename: file.name, size: file.size } }));
        const chunkSize = 8192;
        let offset = 0;
        while (offset < file.size) {
            while (this._uploadActive(state) && channel.bufferedAmount > chunkSize * 8)
                await new Promise(resolve => setTimeout(resolve, 25));
            if (!this._uploadActive(state)) return;
            if (channel.readyState !== 'open') throw new Error('Upload channel is not open');
            const end = Math.min(offset + chunkSize, file.size);
            const buffer = await file.slice(offset, end).arrayBuffer();
            if (!this._uploadActive(state)) return;
            if (buffer.byteLength !== end - offset) throw new Error('The local file changed during upload');
            channel.send(this._encodeFileTransfer({ data: { data: new Uint8Array(buffer) } }));
            offset = end;
            state.bytesSent = offset;
            this.dispatchEvent(new CustomEvent('fileTransferProgress', {
                detail: { transferId: state.transferId, filename: file.name,
                    bytesSent: offset, totalBytes: file.size },
            }));
        }
        if (!this._uploadActive(state)) return;
        state.sentEnd = true;
        // Do not report success merely because all bytes were queued locally.
        state.finishTimer = setTimeout(() =>
            this._finishUpload(state, 'Upload acknowledgement timed out; remote result is unconfirmed'), 30000);
        channel.send(this._encodeFileTransfer({ end: {} }));
    }

    _finishUpload(state, errorMessage = '') {
        if (this._activeUploads.get(state.transferId) !== state) return;
        this._activeUploads.delete(state.transferId);
        state.cancelled = true;
        state.file = null;
        clearTimeout(state.openTimer);
        clearTimeout(state.finishTimer);
        try { state.channel?.close(); } catch { /* already closed */ }
        this.dispatchEvent(new CustomEvent(errorMessage ? 'fileTransferError' : 'fileTransferComplete', {
            detail: { transferId: state.transferId, filename: state.filename, errorMessage },
        }));
        queueMicrotask(() => this._pumpUploads());
    }

    cancelFileUpload(transferId) {
        const state = this._activeUploads.get(transferId);
        if (!state) return;
        if (state.channel?.readyState === 'open') {
            try { state.channel.send(this._encodeFileTransfer({ error: { type: 1 } })); } catch {}
        }
        this._finishUpload(state, state.status === 'queued' ? 'Cancelled before upload started'
            : 'Stopped locally; a partial remote file may remain');
    }

    startFileDownload(fileHandle = null) {
        if (this._fileTransfersClosed || !this._pc || !this.supportsFileTransfer || this._pc.connectionState === 'closed' ||
            this._pc.connectionState === 'failed') return null;
        const transferId = String(nextFileTransferId++);
        let channel;
        try { channel = this._pc.createDataChannel('filetransfer-' + transferId, { ordered: true }); }
        catch { return null; }
        channel.binaryType = 'arraybuffer';
        const state = {
            transferId, filename: 'download', totalBytes: 0, bytesReceived: 0,
            cancelled: false, channel, chunks: fileHandle ? null : [],
            metadataReceived: false, streaming: !!fileHandle, fileHandle,
            writable: null, writeQueue: Promise.resolve(), pendingBytes: 0, ending: false,
        };
        this._activeDownloads.set(transferId, state);
        this.dispatchEvent(new CustomEvent('fileDownloadStarted', {
            detail: { transferId, filename: state.filename, totalBytes: 0, waiting: true },
        }));
        state.openTimer = setTimeout(() => this._failDownload(state, 'Download channel did not open'), 30000);
        channel.onopen = async () => {
            clearTimeout(state.openTimer);
            try {
                if (state.streaming) {
                    const writable = await state.fileHandle.createWritable();
                    if (!this._downloadActive(state)) { await writable.abort(); return; }
                    state.writable = writable;
                }
                if (!this._downloadActive(state)) return;
                channel.send(this._encodeFileTransfer({ requestTransfer: {} }));
            } catch (error) { this._failDownload(state, error.message || 'Failed to open destination'); }
        };
        channel.onmessage = event => {
            if (!this._downloadActive(state) || state.ending) return;
            try {
                const message = this._decodeFileTransfer(new Uint8Array(event.data));
                if (message.metadata) {
                    if (state.metadataReceived) throw new Error('Duplicate download metadata');
                    const size = message.metadata.size;
                    if (!Number.isSafeInteger(size) || size < 0) throw new Error('Invalid download size');
                    if (!state.streaming && size > 64 * 1024 * 1024)
                        throw new Error('This browser limits buffered downloads to 64 MiB; use Save File support for larger files');
                    state.filename = (message.metadata.filename || 'download').split(/[\\/]/).pop()
                        .replace(/[\u0000-\u001f]/g, '') || 'download';
                    state.totalBytes = size;
                    state.metadataReceived = true;
                    this.dispatchEvent(new CustomEvent('fileDownloadStarted', {
                        detail: { transferId, filename: state.filename, totalBytes: size },
                    }));
                } else if (message.data) {
                    if (!state.metadataReceived) throw new Error('Download data arrived before metadata');
                    const length = message.data.byteLength;
                    if (!Number.isSafeInteger(state.bytesReceived + length))
                        throw new Error('Download byte count exceeds the safe integer range');
                    state.bytesReceived += length;
                    if (state.streaming) {
                        state.pendingBytes += length;
                        if (state.pendingBytes > 8 * 1024 * 1024)
                            throw new Error('Disk writes cannot keep up; download stopped to limit memory');
                        state.writeQueue = state.writeQueue.then(async () => {
                            if (!this._downloadActive(state)) return;
                            await state.writable.write(message.data);
                            state.pendingBytes -= length;
                        }).catch(error => this._failDownload(state, error.message || 'Disk write error'));
                    } else {
                        if (state.bytesReceived > 64 * 1024 * 1024)
                            throw new Error('Buffered download limit exceeded');
                        state.chunks.push(message.data);
                    }
                    this.dispatchEvent(new CustomEvent('fileDownloadProgress', {
                        detail: { transferId, filename: state.filename,
                            bytesReceived: state.bytesReceived, totalBytes: state.totalBytes },
                    }));
                } else if (message.end) {
                    // Metadata.size is advisory in the existing protocol; End is authoritative.
                    if (!state.metadataReceived) throw new Error('Download ended before metadata');
                    state.ending = true;
                    this._completeDownload(state);
                } else if (message.error) {
                    this._failDownload(state, 'Host error type=' + message.error.type);
                }
            } catch (error) { this._failDownload(state, error.message); }
        };
        channel.onclose = () => {
            if (!state.ending) this._failDownload(state, 'Channel closed before download completed');
        };
        channel.onerror = () => this._failDownload(state, 'Download channel error');
        return transferId;
    }

    _downloadActive(state) {
        return this._activeDownloads.get(state.transferId) === state && !state.cancelled;
    }

    async _completeDownload(state) {
        try {
            await state.writeQueue;
            if (!this._downloadActive(state)) return;
            if (state.writable) await state.writable.close();
            if (!this._downloadActive(state)) return;
            const blob = state.streaming ? null : new Blob(state.chunks);
            state.channel.send(this._encodeFileTransfer({ success: {} }));
            this._activeDownloads.delete(state.transferId);
            state.chunks = null;
            state.writable = null;
            try { state.channel.close(); } catch {}
            this.dispatchEvent(new CustomEvent('fileDownloadComplete', {
                detail: { transferId: state.transferId, filename: state.filename,
                    blob, streaming: state.streaming },
            }));
        } catch (error) { this._failDownload(state, error.message || 'Download could not be saved'); }
    }

    async _failDownload(state, errorMessage) {
        if (!this._downloadActive(state)) return;
        state.cancelled = true;
        this._activeDownloads.delete(state.transferId);
        clearTimeout(state.openTimer);
        state.chunks = null;
        try { state.channel.close(); } catch {}
        const writable = state.writable;
        state.writable = null;
        this.dispatchEvent(new CustomEvent('fileDownloadError', {
            detail: { transferId: state.transferId, errorMessage },
        }));
        if (writable) {
            try { await writable.abort(); } catch {}
        }
    }

    cancelFileDownload(transferId) {
        const state = this._activeDownloads.get(transferId);
        if (!state) return;
        if (state.channel.readyState === 'open') {
            try { state.channel.send(this._encodeFileTransfer({ error: { type: 1 } })); } catch {}
        }
        this._failDownload(state, 'Stopped locally; no completed download was saved');
    }

    /**
     * Encode a FileTransfer protobuf message (simplified hand-encoding).
     * Follows remoting/proto/file_transfer.proto
     * @private
     */
    _encodeFileTransfer(msg) {
        const parts = [];
        if (msg.metadata) {
            // field 1 = Metadata (sub-message)
            const inner = [];
            // filename: field 1, type string
            if (msg.metadata.filename) {
                const encoded = new TextEncoder().encode(msg.metadata.filename);
                inner.push(0x0A, ...this._encodeVarint(encoded.length), ...encoded);
            }
            // size: field 2, type int64 (varint)
            if (msg.metadata.size !== undefined) {
                inner.push(0x10, ...this._encodeVarint(msg.metadata.size));
            }
            parts.push(0x0A, ...this._encodeVarint(inner.length), ...inner);
        }
        if (msg.data) {
            // field 2 = Data (sub-message), data: field 1, type bytes
            const bytes = msg.data.data;
            const inner = [0x0A, ...this._encodeVarint(bytes.length), ...bytes];
            parts.push(0x12, ...this._encodeVarint(inner.length), ...inner);
        }
        if (msg.end) {
            // field 3 = End (sub-message, empty)
            parts.push(0x1A, 0x00);
        }
        if (msg.success) {
            // field 4 = Success (sub-message, empty)
            parts.push(0x22, 0x00);
        }
        if (msg.requestTransfer) {
            // field 5 = RequestTransfer (sub-message, empty)
            parts.push(0x2A, 0x00);
        }
        if (msg.error) {
            // field 6 = Error (sub-message), type: field 1 varint
            const inner = [0x08, ...this._encodeVarint(msg.error.type || 0)];
            parts.push(0x32, ...this._encodeVarint(inner.length), ...inner);
        }
        return new Uint8Array(parts).buffer;
    }

    /**
     * Decode a FileTransfer protobuf message.
     * Parses Metadata (1), Data (2), End (3), Success (4), Error (6).
     * @private
     */
    _decodeFileTransfer(data) {
        const result = {};
        let pos = 0;
        while (pos < data.length) {
            const tag = data[pos++];
            const fieldNumber = tag >> 3;
            const wireType = tag & 0x07;

            if (wireType === 2) {
                let len = 0;
                let shift = 0;
                while (pos < data.length) {
                    const b = data[pos++];
                    len |= (b & 0x7F) << shift;
                    if (!(b & 0x80)) break;
                    shift += 7;
                }
                const subdata = data.subarray(pos, pos + len);
                pos += len;

                if (fieldNumber === 1) {
                    // Metadata: filename (field 1 string), size (field 2 varint)
                    result.metadata = this._decodeMetadata(subdata);
                } else if (fieldNumber === 2) {
                    // Data: data bytes (field 1 bytes)
                    result.data = this._decodeDataField(subdata);
                } else if (fieldNumber === 3) {
                    result.end = true;
                } else if (fieldNumber === 4) {
                    result.success = true;
                } else if (fieldNumber === 6) {
                    let errType = 0;
                    if (subdata.length > 1 && subdata[0] === 0x08) {
                        errType = subdata[1];
                    }
                    result.error = { type: errType };
                }
            } else if (wireType === 0) {
                while (pos < data.length && data[pos] & 0x80) pos++;
                pos++;
            }
        }
        return result;
    }

    /** @private */
    _decodeMetadata(data) {
        const meta = { filename: '', size: 0 };
        let pos = 0;
        while (pos < data.length) {
            const tag = data[pos++];
            const fn = tag >> 3;
            const wt = tag & 0x07;
            if (wt === 2) {
                let len = 0, shift = 0;
                while (pos < data.length) {
                    const b = data[pos++];
                    len |= (b & 0x7F) << shift;
                    if (!(b & 0x80)) break;
                    shift += 7;
                }
                if (fn === 1) {
                    meta.filename = new TextDecoder().decode(data.subarray(pos, pos + len));
                }
                pos += len;
            } else if (wt === 0) {
                let val = 0, shift = 0;
                while (pos < data.length) {
                    const b = data[pos++];
                    val += (b & 0x7F) * 2 ** shift;
                    if (!Number.isSafeInteger(val) || shift > 49) throw new Error('Invalid file size varint');
                    if (!(b & 0x80)) break;
                    shift += 7;
                }
                if (fn === 2) meta.size = val;
            }
        }
        return meta;
    }

    /** @private */
    _decodeDataField(data) {
        let pos = 0;
        while (pos < data.length) {
            const tag = data[pos++];
            const fn = tag >> 3;
            const wt = tag & 0x07;
            if (wt === 2 && fn === 1) {
                let len = 0, shift = 0;
                while (pos < data.length) {
                    const b = data[pos++];
                    len |= (b & 0x7F) << shift;
                    if (!(b & 0x80)) break;
                    shift += 7;
                }
                return data.slice(pos, pos + len).buffer;
            } else if (wt === 0) {
                while (pos < data.length && data[pos] & 0x80) pos++;
                pos++;
            } else if (wt === 2) {
                let len = 0, shift = 0;
                while (pos < data.length) {
                    const b = data[pos++];
                    len |= (b & 0x7F) << shift;
                    if (!(b & 0x80)) break;
                    shift += 7;
                }
                pos += len;
            }
        }
        return new ArrayBuffer(0);
    }

    /**
     * Encode an unsigned integer as a protobuf varint.
     * @private
     */
    _encodeVarint(value) {
        if (!Number.isSafeInteger(value) || value < 0) throw new Error('Invalid file transfer integer');
        const bytes = [];
        while (value > 0x7F) {
            bytes.push((value % 128) | 0x80);
            value = Math.floor(value / 128);
        }
        bytes.push(value);
        return bytes;
    }

    /**
     * Handle host capabilities and create actions channel if supported.
     * @private
     */
    _onHostCapabilities(hostCaps) {
        this._hostCapabilities = hostCaps;
        const hostSet = new Set(hostCaps.split(' ').filter(Boolean));

        this.supportsSendAttentionSequence = hostSet.has('sendAttentionSequenceAction');
        this.supportsLockWorkstation = hostSet.has('lockWorkstationAction');
        this.supportsFileTransfer = hostSet.has('fileTransfer');
        this.supportsPrivacyScreen = hostSet.has('privacyScreen');
        this.supportsVirtualDisplay = hostSet.has('virtualDisplay');

        console.log(`[DataChannel] Host caps: SAS=${this.supportsSendAttentionSequence} Lock=${this.supportsLockWorkstation} FileTransfer=${this.supportsFileTransfer} PrivacyScreen=${this.supportsPrivacyScreen} VD=${this.supportsVirtualDisplay}`);

        // Create "actions" outgoing data channel if any action is supported.
        if ((this.supportsSendAttentionSequence || this.supportsLockWorkstation || this.supportsPrivacyScreen) && this._pc) {
            this.actionsChannel = this._pc.createDataChannel('actions', { ordered: true });
            this.actionsChannel.binaryType = 'arraybuffer';
            this.actionsChannel.onopen = () => {
                this._actionsReady = true;
                console.log('[DataChannel] Actions channel opened');
                this.dispatchEvent(new CustomEvent('actionsReady'));
            };
            this.actionsChannel.onclose = () => {
                this._actionsReady = false;
                console.log('[DataChannel] Actions channel closed');
            };
            this.actionsChannel.onmessage = (event) => {
                console.log('[DataChannel] Actions response received');
            };
            this._nextActionRequestId = 1;
        }

        this.dispatchEvent(new CustomEvent('hostCapabilities', {
            detail: {
                supportsSendAttentionSequence: this.supportsSendAttentionSequence,
                supportsLockWorkstation: this.supportsLockWorkstation,
                supportsFileTransfer: this.supportsFileTransfer,
                supportsPrivacyScreen: this.supportsPrivacyScreen,
                supportsVirtualDisplay: this.supportsVirtualDisplay,
            }
        }));
    }

    /**
     * 检查是否就绪
     */
    isReady() {
        return this._controlReady && this._eventReady;
    }

    /**
     * 清理
     */
    closeFileTransfers() {
        this._fileTransfersClosed = true;
        for (const state of [...this._activeUploads.values()])
            this._finishUpload(state, 'Disconnected; upload completion was not confirmed');
        for (const state of [...this._activeDownloads.values()])
            this._failDownload(state, 'Disconnected; download was not completed');
    }

    destroy() {
        this.closeFileTransfers();
        this._pc = null;
        if (this.controlChannel) {
            this.controlChannel.close();
            this.controlChannel = null;
        }
        if (this.eventChannel) {
            this.eventChannel.close();
            this.eventChannel = null;
        }
        if (this.actionsChannel) {
            this.actionsChannel.close();
            this.actionsChannel = null;
        }
        this._controlReady = false;
        this._eventReady = false;
        this._actionsReady = false;
        this._activeUploads.clear();
        this._activeDownloads.clear();
    }
}
