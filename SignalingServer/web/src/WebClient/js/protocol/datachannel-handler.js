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
        this._controlReady = false;
        this._eventReady = false;
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
        this._controlReady = false;
        this._eventReady = false;
    }
}
