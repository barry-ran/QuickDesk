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

        this._nextFileTransferId = 1;
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
        if (!this._pc) {
            console.warn('[DataChannel] No PeerConnection for file transfer');
            return null;
        }
        if (!this.supportsFileTransfer) {
            console.warn('[DataChannel] Host does not support file transfer');
            return null;
        }

        const transferId = String(this._nextFileTransferId++);
        const channelName = `filetransfer-${transferId}`;
        const channel = this._pc.createDataChannel(channelName, { ordered: true });
        channel.binaryType = 'arraybuffer';

        const CHUNK_SIZE = 8192;
        let offset = 0;

        const state = {
            transferId, filename: file.name, totalBytes: file.size,
            bytesSent: 0, cancelled: false, channel
        };
        this._activeUploads.set(transferId, state);

        this.dispatchEvent(new CustomEvent('fileTransferStarted', {
            detail: { transferId, filename: file.name, totalBytes: file.size }
        }));

        channel.onopen = () => {
            if (state.cancelled) return;
            console.log(`[DataChannel] File transfer channel opened: ${channelName}`);
            const metadata = this._encodeFileTransfer({
                metadata: { filename: file.name, size: file.size }
            });
            channel.send(metadata);
            this._sendNextChunk(channel, file, offset, CHUNK_SIZE, state);
        };

        channel.onmessage = (event) => {
            const msg = this._decodeFileTransfer(new Uint8Array(event.data));
            if (msg.success) {
                console.log(`[DataChannel] File saved on host: ${file.name}`);
                this.dispatchEvent(new CustomEvent('fileTransferComplete', {
                    detail: { transferId, filename: file.name }
                }));
                this._activeUploads.delete(transferId);
            } else if (msg.error) {
                const errMsg = `Host error type=${msg.error.type}`;
                console.error(`[DataChannel] File transfer error: ${errMsg}`);
                this.dispatchEvent(new CustomEvent('fileTransferError', {
                    detail: { transferId, errorMessage: errMsg }
                }));
                this._activeUploads.delete(transferId);
            }
        };

        channel.onclose = () => {
            console.log(`[DataChannel] File transfer channel closed: ${channelName}`);
        };

        channel.onerror = (event) => {
            console.error(`[DataChannel] File transfer channel error:`, event);
            this.dispatchEvent(new CustomEvent('fileTransferError', {
                detail: { transferId, errorMessage: 'Channel error' }
            }));
            this._activeUploads.delete(transferId);
        };

        return transferId;
    }

    /**
     * Cancel an in-progress file upload.
     * @param {string} transferId
     */
    cancelFileUpload(transferId) {
        const state = this._activeUploads.get(transferId);
        if (!state) {
            console.warn(`[DataChannel] cancelFileUpload: unknown transfer ${transferId}`);
            return;
        }
        state.cancelled = true;
        if (state.channel && state.channel.readyState === 'open') {
            try {
                const errMsg = this._encodeFileTransfer({
                    error: { type: 3 }  // CANCELED
                });
                state.channel.send(errMsg);
            } catch (e) { /* ignore */ }
            state.channel.close();
        }
        this._activeUploads.delete(transferId);
        this.dispatchEvent(new CustomEvent('fileTransferError', {
            detail: { transferId, errorMessage: 'Cancelled by user' }
        }));
    }

    /**
     * Request a file download from the remote host.
     *
     * @param {FileSystemFileHandle|null} fileHandle - If provided (from
     *   showSaveFilePicker), data is streamed directly to disk via the
     *   File System Access API.  Otherwise falls back to buffering all
     *   chunks in memory and producing a Blob.
     * @returns {string|null} transferId or null on failure
     */
    startFileDownload(fileHandle = null) {
        if (!this._pc) {
            console.warn('[DataChannel] No PeerConnection for file download');
            return null;
        }
        if (!this.supportsFileTransfer) {
            console.warn('[DataChannel] Host does not support file transfer');
            return null;
        }

        const transferId = String(this._nextFileTransferId++);
        const channelName = `filetransfer-${transferId}`;
        const channel = this._pc.createDataChannel(channelName, { ordered: true });
        channel.binaryType = 'arraybuffer';

        const streaming = !!fileHandle;
        const state = {
            transferId, filename: '', totalBytes: 0,
            bytesReceived: 0, cancelled: false, channel,
            chunks: streaming ? null : [],
            metadataReceived: false,
            streaming,
            fileHandle,
            writable: null,
            writeQueue: Promise.resolve(),
        };
        this._activeDownloads.set(transferId, state);

        channel.onopen = async () => {
            if (state.cancelled) return;
            console.log(`[DataChannel] Download channel opened: ${channelName}, sending RequestTransfer`);

            if (state.streaming) {
                try {
                    state.writable = await state.fileHandle.createWritable();
                } catch (err) {
                    console.error('[DataChannel] Failed to create writable stream:', err);
                    this._failDownload(state, 'Failed to open file for writing');
                    return;
                }
            }

            const reqMsg = this._encodeFileTransfer({ requestTransfer: {} });
            channel.send(reqMsg);
        };

        channel.onmessage = (event) => {
            if (state.cancelled) return;
            const msg = this._decodeFileTransfer(new Uint8Array(event.data));

            if (msg.metadata) {
                state.filename = msg.metadata.filename || 'download';
                state.totalBytes = msg.metadata.size || 0;
                state.metadataReceived = true;
                console.log(`[DataChannel] Download metadata: ${state.filename} (${state.totalBytes} bytes)`);
                this.dispatchEvent(new CustomEvent('fileDownloadStarted', {
                    detail: { transferId, filename: state.filename, totalBytes: state.totalBytes }
                }));
            } else if (msg.data) {
                state.bytesReceived += msg.data.byteLength;

                if (state.streaming && state.writable) {
                    state.writeQueue = state.writeQueue.then(async () => {
                        if (state.cancelled) return;
                        try {
                            await state.writable.write(msg.data);
                        } catch (err) {
                            console.error('[DataChannel] Stream write error:', err);
                            this._failDownload(state, 'Disk write error');
                        }
                    });
                } else if (state.chunks) {
                    state.chunks.push(msg.data);
                }

                this.dispatchEvent(new CustomEvent('fileDownloadProgress', {
                    detail: {
                        transferId, filename: state.filename,
                        bytesReceived: state.bytesReceived, totalBytes: state.totalBytes
                    }
                }));
            } else if (msg.end) {
                console.log(`[DataChannel] Download end received: ${state.filename}`);
                const successMsg = this._encodeFileTransfer({ success: {} });
                channel.send(successMsg);

                if (state.streaming && state.writable) {
                    state.writeQueue = state.writeQueue.then(async () => {
                        if (state.cancelled) return;
                        try {
                            await state.writable.close();
                        } catch (err) {
                            console.error('[DataChannel] Stream close error:', err);
                        }
                        this.dispatchEvent(new CustomEvent('fileDownloadComplete', {
                            detail: { transferId, filename: state.filename, blob: null, streaming: true }
                        }));
                        this._activeDownloads.delete(transferId);
                    });
                } else {
                    const blob = new Blob(state.chunks);
                    this.dispatchEvent(new CustomEvent('fileDownloadComplete', {
                        detail: { transferId, filename: state.filename, blob, streaming: false }
                    }));
                    this._activeDownloads.delete(transferId);
                }
            } else if (msg.error) {
                const errMsg = `Host error type=${msg.error.type}`;
                console.error(`[DataChannel] Download error: ${errMsg}`);
                this._failDownload(state, errMsg);
            }
        };

        channel.onclose = () => {
            console.log(`[DataChannel] Download channel closed: ${channelName}`);
        };

        channel.onerror = (event) => {
            console.error(`[DataChannel] Download channel error:`, event);
            this._failDownload(state, 'Channel error');
        };

        return transferId;
    }

    /** @private */
    async _failDownload(state, errorMessage) {
        if (state.cancelled) return;
        state.cancelled = true;
        if (state.writable) {
            try { await state.writable.abort(); } catch (_) {}
            state.writable = null;
        }
        this.dispatchEvent(new CustomEvent('fileDownloadError', {
            detail: { transferId: state.transferId, errorMessage }
        }));
        this._activeDownloads.delete(state.transferId);
    }

    /**
     * Cancel an in-progress file download.
     * @param {string} transferId
     */
    cancelFileDownload(transferId) {
        const state = this._activeDownloads.get(transferId);
        if (!state) {
            console.warn(`[DataChannel] cancelFileDownload: unknown transfer ${transferId}`);
            return;
        }
        state.cancelled = true;
        if (state.writable) {
            state.writable.abort().catch(() => {});
            state.writable = null;
        }
        if (state.channel && state.channel.readyState === 'open') {
            try {
                const errMsg = this._encodeFileTransfer({ error: { type: 1 } });  // CANCELED
                state.channel.send(errMsg);
            } catch (e) { /* ignore */ }
            state.channel.close();
        }
        this._activeDownloads.delete(transferId);
        this.dispatchEvent(new CustomEvent('fileDownloadError', {
            detail: { transferId, errorMessage: 'Cancelled by user' }
        }));
    }

    /** @private */
    _sendNextChunk(channel, file, offset, chunkSize, state) {
        if (state.cancelled) return;

        if (offset >= file.size) {
            const end = this._encodeFileTransfer({ end: {} });
            channel.send(end);
            console.log(`[DataChannel] File upload sent all bytes: ${file.name}`);
            return;
        }

        const slice = file.slice(offset, Math.min(offset + chunkSize, file.size));
        const reader = new FileReader();
        reader.onload = () => {
            if (state.cancelled) return;
            const chunk = new Uint8Array(reader.result);
            const data = this._encodeFileTransfer({ data: { data: chunk } });
            channel.send(data);
            offset += chunk.length;
            state.bytesSent = offset;

            this.dispatchEvent(new CustomEvent('fileTransferProgress', {
                detail: {
                    transferId: state.transferId,
                    filename: state.filename,
                    bytesSent: offset,
                    totalBytes: state.totalBytes
                }
            }));

            if (channel.bufferedAmount > chunkSize * 8) {
                setTimeout(() => this._sendNextChunk(channel, file, offset, chunkSize, state), 50);
            } else {
                this._sendNextChunk(channel, file, offset, chunkSize, state);
            }
        };
        reader.readAsArrayBuffer(slice);
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
                    val |= (b & 0x7F) << shift;
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
        const bytes = [];
        while (value > 0x7F) {
            bytes.push((value & 0x7F) | 0x80);
            value >>>= 7;
        }
        bytes.push(value & 0x7F);
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
    destroy() {
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
