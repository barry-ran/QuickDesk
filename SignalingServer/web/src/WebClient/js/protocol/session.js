/**
 * session.js - Jingle 会话状态机
 * 
 * 参照 src/remoting/protocol/jingle_session.cc
 * 集成 WebSocket 传输、Jingle XML 编解码、SPAKE2 认证、RTCPeerConnection
 */

import { JingleBuilder } from '../signaling/jingle-builder.js';
import { JingleParser } from '../signaling/jingle-parser.js';
import { WebSocketTransport } from '../signaling/websocket-transport.js';
import { Spake2Authenticator, AuthState } from '../auth/spake2.js';
import { getSharedSecretHash, hmacSha256, base64Encode, generateUUID } from '../auth/auth-util.js';

// ==================== 会话状态 ====================

export const SessionState = {
    IDLE: 'IDLE',
    CONNECTING: 'CONNECTING',         // WebSocket 连接中
    INITIATING: 'INITIATING',         // 发送 session-initiate
    ACCEPTING: 'ACCEPTING',           // 等待 session-accept
    AUTHENTICATING: 'AUTHENTICATING', // SPAKE2 认证中
    CONNECTED: 'CONNECTED',           // WebRTC 连接已建立
    CLOSED: 'CLOSED',
    FAILED: 'FAILED',
};

// ==================== 会话类 ====================

export class Session extends EventTarget {
    /**
     * @param {object} options
     * @param {string} options.signalingUrl - 信令服务器地址
     * @param {object} [options.iceServers] - ICE 服务器配置
     */
    constructor(options = {}) {
        super();
        
        this.signalingUrl = options.signalingUrl || 'ws://localhost:8000';
        this.iceServers = options.iceServers || [];
        this.preferredVideoCodec = options.preferredVideoCodec || '';

        this.state = SessionState.IDLE;
        this.deviceId = null;
        this.accessCode = null;
        
        // 组件
        this.transport = null;
        this.jingleBuilder = new JingleBuilder();
        this.jingleParser = new JingleParser();
        this.authenticator = null;
        this.pc = null;

        // 缓冲
        this._pendingIceCandidates = [];
        this._pendingOutgoingCandidates = [];
        this._remoteDescriptionSet = false;
        this._authenticated = false;
    }

    /**
     * 开始连接
     * @param {string} deviceId 
     * @param {string} accessCode 
     */
    async connect(deviceId, accessCode) {
        this.deviceId = deviceId;
        this.accessCode = accessCode;

        try {
            this._setState(SessionState.CONNECTING);

            // 1. 计算共享密钥哈希
            // 参照 C++: GetSharedSecretHash(host_id, host_secret)
            // host_secret = device_id + access_code
            const hostSecret = deviceId + accessCode;
            const sharedSecretHash = await getSharedSecretHash(deviceId, hostSecret);

            // 2. 构建正确的 JID 格式
            // 参照 C++ QuickDeskClient:
            //   client local_id = "client_{client_device_id}@quickdesk.local/chromoting_ftl_quickdesk_client"
            //   remote_id = "{host_id}@quickdesk.local/chromoting_ftl_quickdesk_host"
            const clientUUID = generateUUID().replace(/-/g, '').substring(0, 12);
            const localJid = `webclient_${clientUUID}@quickdesk.local/chromoting_ftl_quickdesk_client`;
            const remoteJid = `${deviceId}@quickdesk.local/chromoting_ftl_quickdesk_host`;
            
            // 设置 JingleBuilder 的 JID
            this.jingleBuilder.localJid = localJid;
            this.jingleBuilder.remoteJid = remoteJid;

            // 3. 创建认证器 (异步初始化)
            this.authenticator = new Spake2Authenticator(
                localJid,
                remoteJid,
                sharedSecretHash
            );
            await this.authenticator.initialize();

            // 3. 连接 WebSocket
            this.transport = new WebSocketTransport({
                signalingUrl: this.signalingUrl,
                onMessage: (msg) => this._onSignalingMessage(msg),
                onOpen: () => this._log('WebSocket connected'),
                onClose: (code, reason) => this._onWebSocketClose(code, reason),
                onError: (err) => this._onWebSocketError(err),
            });

            await this.transport.connect(deviceId, accessCode);

            // 4. 创建 RTCPeerConnection
            this._createPeerConnection();

            // 5. 创建 SDP Offer
            this._setState(SessionState.INITIATING);
            await this._createAndSendOffer();

        } catch (error) {
            this._log(`Connection failed: ${error.message}`, 'error');
            this._setState(SessionState.FAILED);
            throw error;
        }
    }

    /**
     * 断开连接
     */
    disconnect() {
        this._log('Disconnecting...');

        // 发送 session-terminate
        if (this.transport && this.transport.isConnected() && this.jingleBuilder.sessionId) {
            try {
                const xml = this.jingleBuilder.buildSessionTerminate('success');
                this.transport.send(xml);
            } catch (e) {
                // ignore
            }
        }

        this._cleanup();
        this._setState(SessionState.CLOSED);
    }

    /**
     * 获取当前状态
     * @returns {string}
     */
    getState() {
        return this.state;
    }

    // ==================== 私有方法 ====================

    /**
     * 创建 RTCPeerConnection
     * @private
     */
    _createPeerConnection() {
        const config = {
            iceServers: this.iceServers,
            bundlePolicy: 'max-bundle',
        };

        this.pc = new RTCPeerConnection(config);
        this._log('RTCPeerConnection created');

        this._localCandidates = [];
        this._remoteCandidates = [];

        // ICE candidate 事件
        this.pc.onicecandidate = (event) => {
            if (event.candidate) {
                this._localCandidates.push(event.candidate);
                this._log(`Local ICE candidate: ${event.candidate.type || 'unknown'} ${event.candidate.protocol} ${event.candidate.address}:${event.candidate.port} (${event.candidate.candidate})`);
                this._sendIceCandidate(event.candidate);
            } else {
                this._log(`ICE gathering complete, total local candidates: ${this._localCandidates.length}`);
            }
        };

        // ICE gathering 状态
        this.pc.onicegatheringstatechange = () => {
            this._log(`ICE gathering state: ${this.pc.iceGatheringState}`);
        };

        // ICE 连接状态
        this.pc.oniceconnectionstatechange = () => {
            this._log(`ICE state: ${this.pc.iceConnectionState}`);
            
            if (this.pc.iceConnectionState === 'connected' || 
                this.pc.iceConnectionState === 'completed') {
                if (this.state !== SessionState.CONNECTED) {
                    this._setState(SessionState.CONNECTED);
                }
                this.pc.getStats().then(stats => {
                    stats.forEach(report => {
                        if (report.type === 'candidate-pair' && report.state === 'succeeded') {
                            this._log(`Active candidate pair: local=${report.localCandidateId} remote=${report.remoteCandidateId} nominated=${report.nominated}`);
                        }
                        if (report.type === 'local-candidate') {
                            this._log(`Local candidate stat: ${report.candidateType} ${report.protocol} ${report.address}:${report.port}`);
                        }
                        if (report.type === 'remote-candidate') {
                            this._log(`Remote candidate stat: ${report.candidateType} ${report.protocol} ${report.address}:${report.port}`);
                        }
                    });
                });
            } else if (this.pc.iceConnectionState === 'failed') {
                this._log(`ICE FAILED! Local candidates: ${this._localCandidates.length}, Remote candidates: ${this._remoteCandidates.length}`);
                this._log(`ICE FAILED! signalingState=${this.pc.signalingState}, iceGatheringState=${this.pc.iceGatheringState}`);
                this._localCandidates.forEach((c, i) => {
                    this._log(`  local[${i}]: ${c.type || 'unknown'} ${c.protocol} ${c.address}:${c.port}`);
                });
                this._remoteCandidates.forEach((c, i) => {
                    this._log(`  remote[${i}]: ${c.candidate}`);
                });
                this.pc.getStats().then(stats => {
                    stats.forEach(report => {
                        if (report.type === 'candidate-pair') {
                            this._log(`Candidate pair: state=${report.state} local=${report.localCandidateId} remote=${report.remoteCandidateId} nominated=${report.nominated} bytesSent=${report.bytesSent} bytesReceived=${report.bytesReceived}`);
                        }
                    });
                    this._setState(SessionState.FAILED);
                }).catch(() => {
                    this._setState(SessionState.FAILED);
                });
                return;
            } else if (this.pc.iceConnectionState === 'disconnected') {
                this._emitEvent('iceDisconnected');
            }
        };

        // 接收远程媒体流
        this.pc.ontrack = (event) => {
            this._log(`Received track: ${event.track.kind}`);
            this._emitEvent('track', { track: event.track, streams: event.streams });
        };

        // 接收 DataChannel
        this.pc.ondatachannel = (event) => {
            const channel = event.channel;
            this._log(`Received DataChannel: ${channel.label}`);
            this._emitEvent('datachannel', { channel });
        };

        // 连接状态变化
        this.pc.onconnectionstatechange = () => {
            this._log(`Connection state: ${this.pc.connectionState}, signalingState: ${this.pc.signalingState}, iceState: ${this.pc.iceConnectionState}`);
            if (this.pc.connectionState === 'failed') {
                this._log(`Connection FAILED! iceState=${this.pc.iceConnectionState}, local=${this._localCandidates.length}, remote=${this._remoteCandidates.length}`);
                this.pc.getStats().then(stats => {
                    stats.forEach(report => {
                        if (report.type === 'candidate-pair') {
                            this._log(`  pair: state=${report.state} local=${report.localCandidateId} remote=${report.remoteCandidateId} nominated=${report.nominated} bytesSent=${report.bytesSent} bytesReceived=${report.bytesReceived}`);
                        }
                    });
                    if (this.pc.iceConnectionState !== 'failed') {
                        this._setState(SessionState.FAILED);
                    }
                }).catch(() => {
                    if (this.pc?.iceConnectionState !== 'failed') {
                        this._setState(SessionState.FAILED);
                    }
                });
            }
        };

    }

    /**
     * 创建并发送 SDP Offer
     * @private
     */
    async _createAndSendOffer() {
        const videoTransceiver = this.pc.addTransceiver('video', { direction: 'recvonly' });
        this.pc.addTransceiver('audio', { direction: 'recvonly' });

        if (this.preferredVideoCodec) {
            this._applyCodecPreference(videoTransceiver, this.preferredVideoCodec);
        }

        const offer = await this.pc.createOffer();
        await this.pc.setLocalDescription(offer);

        // 获取首条认证消息 (仅 supported-methods)
        const authMessage = this.authenticator.getFirstNegotiationMessage();

        // 构建 session-initiate Jingle XML
        const xml = this.jingleBuilder.buildSessionInitiate(offer.sdp, authMessage);
        
        this._log('Sending session-initiate');
        this.transport.send(xml);
        this._setState(SessionState.ACCEPTING);
    }

    /**
     * 设置视频编码器偏好顺序
     * @private
     * @param {RTCRtpTransceiver} transceiver
     * @param {string} codecName - "H264", "VP8", "VP9", "AV1"
     */
    _applyCodecPreference(transceiver, codecName) {
        if (!transceiver.setCodecPreferences) {
            this._log('浏览器不支持 setCodecPreferences', 'warning');
            return;
        }

        const capabilities = RTCRtpReceiver.getCapabilities('video');
        if (!capabilities) return;

        const preferred = [];
        const others = [];
        const target = codecName.toUpperCase();

        for (const codec of capabilities.codecs) {
            const mime = codec.mimeType.toUpperCase();
            if (mime === `VIDEO/${target}`) {
                preferred.push(codec);
            } else {
                others.push(codec);
            }
        }

        if (preferred.length > 0) {
            transceiver.setCodecPreferences([...preferred, ...others]);
            this._log(`视频编码器偏好已设置: ${codecName}`);
        } else {
            this._log(`浏览器不支持编码器: ${codecName}`, 'warning');
        }
    }

    /**
     * 发送 IQ result 响应
     * 参照 jingle_messages.cc JingleMessageReply::ToXml
     * @private
     */
    _sendIqResult(iqId, toJid) {
        if (!this.transport || !this.transport.isConnected()) return;
        const xml = this.jingleBuilder.buildIqResult(iqId, toJid);
        this._log(`IQ result XML: ${xml.substring(0, 300)}`);
        this.transport.send(xml);
    }

    /**
     * 发送 ICE candidate
     * @private
     */
    _sendIceCandidate(candidate) {
        if (!this.transport || !this.transport.isConnected()) {
            this._log(`Skipped sending ICE candidate: transport not connected`);
            return;
        }

        if (!this._authenticated) {
            this._pendingOutgoingCandidates.push(candidate);
            this._log(`Buffered outgoing ICE candidate (not authenticated yet), total=${this._pendingOutgoingCandidates.length}`);
            return;
        }

        const xml = this.jingleBuilder.buildTransportInfo(candidate);
        this.transport.send(xml);
        this._log(`Sent ICE candidate to host`);
    }

    /**
     * 发送缓冲的出站 ICE candidates
     * @private
     */
    _flushOutgoingCandidates() {
        for (const candidate of this._pendingOutgoingCandidates) {
            const xml = this.jingleBuilder.buildTransportInfo(candidate);
            this.transport.send(xml);
        }
        this._log(`Flushed ${this._pendingOutgoingCandidates.length} buffered outgoing ICE candidates`);
        this._pendingOutgoingCandidates = [];
    }

    /**
     * 处理信令消息（排队串行处理，防止 async 操作交叉）
     * @private
     */
    _onSignalingMessage(message) {
        if (!this._messageQueue) {
            this._messageQueue = [];
            this._processingMessage = false;
        }
        this._messageQueue.push(message);
        if (!this._processingMessage) {
            this._processNextMessage();
        }
    }

    /**
     * @private
     */
    async _processNextMessage() {
        if (!this._messageQueue || this._messageQueue.length === 0) {
            this._processingMessage = false;
            return;
        }
        this._processingMessage = true;
        const message = this._messageQueue.shift();
        try {
            await this._processSignalingMessage(message);
        } catch (e) {
            this._log(`Error processing signaling message: ${e.message}`, 'error');
        }
        this._processNextMessage();
    }

    /**
     * @private
     */
    async _processSignalingMessage(message) {
        const trimmed = message.trim();

        if (trimmed.startsWith('{')) {
            try {
                const json = JSON.parse(trimmed);
                this._handleJsonMessage(json);
                return;
            } catch (e) {
                // 不是JSON，继续作为 XML 处理
            }
        }

        if (trimmed.startsWith('<')) {
            const parsed = this.jingleParser.parse(trimmed);
            if (parsed) {
                await this._handleJingleMessage(parsed);
            }
        }
    }

    /**
     * 处理 JSON 控制消息
     * @private
     */
    _handleJsonMessage(json) {
        this._log(`JSON message: ${json.type || 'unknown'}`);
        this._emitEvent('jsonMessage', json);
    }

    /**
     * 处理 Jingle 消息
     * @private
     */
    async _handleJingleMessage(message) {
        this._log(`Jingle action: ${message.action}`);

        switch (message.action) {
            case 'session-accept':
                await this._handleSessionAccept(message);
                break;
            case 'transport-info':
                await this._handleTransportInfo(message);
                break;
            case 'session-info':
                await this._handleSessionInfo(message);
                break;
            case 'session-terminate':
                this._handleSessionTerminate(message);
                break;
            case '_iq_response':
                break;
            default:
                this._log(`Unhandled Jingle action: ${message.action}`, 'warning');
        }

        // 对所有 IQ type="set" 回复 IQ result（XMPP 协议要求）
        if (message.iqType === 'set' && message.iqId && message.from) {
            this._log(`Sending IQ result for id=${message.iqId} to=${message.from}`);
            this._sendIqResult(message.iqId, message.from);
        } else if (message.action !== '_iq_response') {
            this._log(`No IQ result sent: iqType=${message.iqType}, iqId=${message.iqId}, from=${message.from}`);
        }
    }

    /**
     * 处理 session-accept
     * @private
     */
    async _handleSessionAccept(message) {
        this._log('Processing session-accept');

        // 1. 处理认证消息
        if (message.authMessage) {
            this._setState(SessionState.AUTHENTICATING);
            this._log(`Auth message received: method=${message.authMessage.method}, ` +
                      `hasSpake=${!!message.authMessage.spakeMessage}, ` +
                      `hasCert=${!!message.authMessage.certificate}, ` +
                      `hasVerHash=${!!message.authMessage.verificationHash}`);
            
            await this.authenticator.processMessage(message.authMessage);

            const authState = this.authenticator.getState();
            this._log(`Auth state after processing: ${authState}`);
            
            if (authState === AuthState.REJECTED) {
                this._log(`Authentication rejected: ${this.authenticator.rejectionReason}`, 'error');
                this._setState(SessionState.FAILED);
                return;
            }

            // 如果需要发送更多认证消息
            if (authState === AuthState.MESSAGE_READY) {
                const nextAuth = this.authenticator.getNextMessage();
                this._log(`Sending auth: hasSpake=${!!nextAuth.spakeMessage}, hasVerHash=${!!nextAuth.verificationHash}`);
                const xml = this.jingleBuilder.buildSessionInfo(nextAuth);
                this._log('Sending authentication message (session-info)');
                this.transport.send(xml);
            }
        }

        // 2. 设置远程 SDP
        if (message.sdp) {
            try {
                const answer = new RTCSessionDescription({
                    type: message.sdp.type,
                    sdp: message.sdp.sdp,
                });
                await this.pc.setRemoteDescription(answer);
                this._remoteDescriptionSet = true;
                this._log('Remote SDP set successfully');

                // 处理缓冲的 ICE candidates
                await this._processPendingCandidates();
            } catch (e) {
                this._log(`Failed to set remote SDP: ${e.message}`, 'error');
                this._setState(SessionState.FAILED);
            }
        }
    }

    /**
     * 处理 transport-info
     * @private
     */
    async _handleTransportInfo(message) {
        if (message.sdp) {
            try {
                const sdpType = message.sdp.type || 'offer';
                this._log(`Received SDP type=${sdpType}`);

                if (sdpType === 'offer') {
                    // 支持重协商：host 可能多次发送 offer（如添加视频 track）
                    if (this.pc.signalingState === 'have-local-offer') {
                        await this.pc.setLocalDescription({ type: 'rollback' });
                    }
                    await this.pc.setRemoteDescription(new RTCSessionDescription({
                        type: 'offer',
                        sdp: message.sdp.sdp,
                    }));
                    this._remoteDescriptionSet = true;
                    this._log('Remote SDP (offer) set from transport-info');

                    if (this.preferredVideoCodec) {
                        for (const t of this.pc.getTransceivers()) {
                            if (t.receiver?.track?.kind === 'video') {
                                this._applyCodecPreference(t, this.preferredVideoCodec);
                            }
                        }
                    }

                    const answer = await this.pc.createAnswer();
                    await this.pc.setLocalDescription(answer);
                    this._log('Created and set local SDP answer');

                    const signature = await this._signSdp(answer.sdp, 'answer');
                    this._log(`SDP answer signature: ${signature}`);

                    const xml = this.jingleBuilder.buildTransportInfoSdp(answer.sdp, 'answer', signature);
                    this.transport.send(xml);
                    this._log('Sent SDP answer via transport-info');

                    // Host 要求 client 创建 "event" DataChannel (用于键鼠输入)
                    // 必须在 SCTP 传输协商完成后创建，通过已有 SCTP 传输建立
                    // 参照 webrtc_connection_to_host.cc: OnWebrtcTransportConnecting()
                    if (!this._eventChannel) {
                        this._eventChannel = this.pc.createDataChannel('event', {
                            ordered: true,
                        });
                        this._log('Created outgoing "event" DataChannel');
                        this._emitEvent('datachannel', { channel: this._eventChannel });
                    }
                } else {
                    await this.pc.setRemoteDescription(new RTCSessionDescription({
                        type: sdpType,
                        sdp: message.sdp.sdp,
                    }));
                    this._remoteDescriptionSet = true;
                    this._log(`Remote SDP (${sdpType}) set from transport-info`);
                }

                await this._processPendingCandidates();
            } catch (e) {
                this._log(`Failed to set SDP from transport-info: ${e.message}`, 'error');
            }
        }

        // 收集所有 candidates (单个或多个)
        const candidateInfos = [];
        if (message.iceCandidate) {
            candidateInfos.push(message.iceCandidate);
        }
        if (message.iceCandidates) {
            candidateInfos.push(...message.iceCandidates);
        }

        for (const info of candidateInfos) {
            const candidate = new RTCIceCandidate({
                candidate: info.candidate,
                sdpMid: info.sdpMid,
                sdpMLineIndex: info.sdpMLineIndex,
            });
            this._remoteCandidates.push(candidate);
            this._log(`Remote ICE candidate: ${info.candidate}`);

            if (this._remoteDescriptionSet) {
                try {
                    await this.pc.addIceCandidate(candidate);
                    this._log(`Added remote ICE candidate OK (mid=${info.sdpMid})`);
                } catch (e) {
                    this._log(`Failed to add ICE candidate: ${e.message}`, 'warning');
                }
            } else {
                this._pendingIceCandidates.push(candidate);
                this._log(`Buffered remote ICE candidate (remoteDesc not set yet)`);
            }
        }
    }

    /**
     * 处理 session-info (认证消息交换)
     * @private
     */
    async _handleSessionInfo(message) {
        if (message.authMessage) {
            await this.authenticator.processMessage(message.authMessage);

            const authState = this.authenticator.getState();
            if (authState === AuthState.REJECTED) {
                this._log(`Authentication rejected: ${this.authenticator.rejectionReason}`, 'error');
                this._setState(SessionState.FAILED);
                return;
            }

            if (authState === AuthState.MESSAGE_READY) {
                const nextAuth = this.authenticator.getNextMessage();
                const xml = this.jingleBuilder.buildSessionInfo(nextAuth);
                this._log('Sending authentication message');
                this.transport.send(xml);
            }

            if (authState === AuthState.ACCEPTED) {
                this._log('Authentication successful!');
                this._authenticated = true;
                this._emitEvent('authenticated');
                this._flushOutgoingCandidates();
            }
        }
    }

    /**
     * 处理 session-terminate
     * @private
     */
    _handleSessionTerminate(message) {
        const info = message.terminateInfo || {};
        this._log(`Session terminated: ${info.reason || 'unknown'} ${info.errorDetails || ''}`);
        this._cleanup();
        this._setState(SessionState.CLOSED);
    }

    /**
     * 处理缓冲的 ICE candidates
     * @private
     */
    async _processPendingCandidates() {
        this._log(`Processing ${this._pendingIceCandidates.length} buffered ICE candidates`);
        for (const candidate of this._pendingIceCandidates) {
            try {
                await this.pc.addIceCandidate(candidate);
                this._log(`Added buffered ICE candidate OK: ${candidate.candidate}`);
            } catch (e) {
                this._log(`Failed to add buffered ICE candidate: ${e.message} (${candidate.candidate})`, 'warning');
            }
        }
        this._pendingIceCandidates = [];
    }

    /**
     * WebSocket 关闭处理
     * @private
     */
    _onWebSocketClose(code, reason) {
        if (this.state !== SessionState.CLOSED && this.state !== SessionState.FAILED) {
            this._emitEvent('signalingDisconnected', { code, reason });
        }
    }

    /**
     * WebSocket 错误处理
     * @private
     */
    _onWebSocketError(error) {
        this._log('WebSocket error', 'error');
    }

    /**
     * 签名 SDP
     * 参照 webrtc_transport.cc: HMAC-SHA256(auth_key, type + " " + NormalizedForSignature(sdp))
     * @private
     * @param {string} sdp - SDP 字符串
     * @param {string} type - "offer" 或 "answer"
     * @returns {Promise<string>} Base64 编码的签名
     */
    async _signSdp(sdp, type) {
        const authKey = this.authenticator.getAuthKey();
        if (!authKey) {
            this._log('Warning: no auth key available for SDP signing', 'warning');
            return '';
        }

        const normalizedSdp = this._normalizeSdpForSignature(sdp);
        const message = type + ' ' + normalizedSdp;
        const messageBytes = new TextEncoder().encode(message);
        const signature = await hmacSha256(authKey, messageBytes);
        return base64Encode(signature);
    }

    /**
     * 标准化 SDP 用于签名
     * 参照 sdp_message.cc SdpMessage::NormalizedForSignature()
     * 按 "\n" 分割 → trim 每行 → 去除空行 → 用 "\n" 连接 + 末尾 "\n"
     * @private
     * @param {string} sdp
     * @returns {string}
     */
    _normalizeSdpForSignature(sdp) {
        const lines = sdp.split('\n')
            .map(line => line.trim())
            .filter(line => line.length > 0);
        return lines.join('\n') + '\n';
    }

    /**
     * 清理资源
     * @private
     */
    _cleanup() {
        this._eventChannel = null;
        this._messageQueue = [];
        this._processingMessage = false;
        if (this.pc) {
            this.pc.close();
            this.pc = null;
        }
        if (this.transport) {
            this.transport.disconnect();
            this.transport = null;
        }
        this._pendingIceCandidates = [];
        this._pendingOutgoingCandidates = [];
        this._remoteDescriptionSet = false;
        this._authenticated = false;
    }

    /**
     * 设置状态
     * @private
     */
    _setState(newState) {
        const oldState = this.state;
        this.state = newState;
        this._emitEvent('stateChange', { oldState, newState });
    }

    /**
     * 发送事件
     * @private
     */
    _emitEvent(type, detail = {}) {
        this.dispatchEvent(new CustomEvent(type, { detail }));
    }

    /**
     * 日志
     * @private
     */
    _log(message, level = 'info') {
        const prefix = '[Session]';
        switch (level) {
            case 'error': console.error(prefix, message); break;
            case 'warning': console.warn(prefix, message); break;
            default: console.log(prefix, message);
        }
        this._emitEvent('log', { message, level });
    }
}
