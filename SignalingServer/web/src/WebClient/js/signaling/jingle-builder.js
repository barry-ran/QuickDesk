/**
 * jingle-builder.js - Jingle XML 生成器
 * 
 * 参照 src/remoting/quickdesk/signaling/webrtc_jingle_converter.cc
 * 和 src/remoting/protocol/jingle_messages.cc
 * 和 src/remoting/protocol/content_description.cc
 * 
 * 生成 Chromium Remoting 使用的 Jingle XML 信令消息
 */

import { generateUUID } from '../auth/auth-util.js';

// ==================== 命名空间常量 ====================

const NS_JABBER_CLIENT = 'jabber:client';
const NS_JINGLE = 'urn:xmpp:jingle:1';
const NS_CHROMOTING = 'google:remoting';
const NS_WEBRTC_TRANSPORT = 'google:remoting:webrtc';

// ==================== XML 辅助函数 ====================

/**
 * 创建带命名空间的 XML 元素
 */
function createElement(ns, localName) {
    return document.implementation.createDocument(ns, localName, null).documentElement;
}

/**
 * 创建 XML 子元素
 */
function createChildElement(parent, ns, localName) {
    const elem = parent.ownerDocument.createElementNS(ns, localName);
    parent.appendChild(elem);
    return elem;
}

/**
 * 将 XML Document 序列化为字符串
 */
function serializeXml(doc) {
    return new XMLSerializer().serializeToString(doc);
}

// ==================== Jingle 消息构建器 ====================

export class JingleBuilder {
    constructor() {
        this.sessionId = null;
        this.localJid = '';  // 由 session.js 设置正确的 JID
        this.remoteJid = ''; // 由 session.js 设置正确的 JID
    }

    /**
     * 生成新的会话ID
     */
    generateSessionId() {
        // 使用类似 Chromium 的大整数格式
        const arr = new Uint8Array(8);
        crypto.getRandomValues(arr);
        let num = 0n;
        for (let i = 0; i < 8; i++) {
            num = (num << 8n) | BigInt(arr[i]);
        }
        this.sessionId = num.toString();
        return this.sessionId;
    }

    /**
     * 创建基础 IQ stanza
     * @returns {Document}
     */
    _createIqStanza() {
        const doc = document.implementation.createDocument(NS_JABBER_CLIENT, 'cli:iq', null);
        const iq = doc.documentElement;
        iq.setAttribute('to', this.remoteJid);
        iq.setAttribute('from', this.localJid);
        iq.setAttribute('type', 'set');
        iq.setAttribute('id', generateUUID());
        return doc;
    }

    /**
     * 构建 session-initiate 消息
     * 
     * @param {string} sdpOffer - SDP Offer 字符串
     * @param {object} authMessage - 认证消息数据
     *   { supportedMethods?: string, method?: string, spakeMessage?: string, verificationHash?: string }
     * @returns {string} Jingle XML 字符串
     */
    buildSessionInitiate(sdpOffer, authMessage) {
        if (!this.sessionId) {
            this.generateSessionId();
        }

        const doc = this._createIqStanza();
        const iq = doc.documentElement;

        // <jingle>
        const jingle = doc.createElementNS(NS_JINGLE, 'jingle');
        jingle.setAttribute('action', 'session-initiate');
        jingle.setAttribute('sid', this.sessionId);
        jingle.setAttribute('initiator', this.localJid);
        iq.appendChild(jingle);

        // <content>
        const content = doc.createElementNS(NS_JINGLE, 'content');
        content.setAttribute('name', 'chromoting');
        content.setAttribute('creator', 'initiator');
        jingle.appendChild(content);

        // <description> (namespace: google:remoting)
        const description = doc.createElementNS(NS_CHROMOTING, 'description');
        content.appendChild(description);

        // <authentication> 内嵌到 description 中
        if (authMessage) {
            const auth = this._buildAuthElement(doc, authMessage);
            description.appendChild(auth);
        }

        // <transport> (namespace: google:remoting:webrtc)
        const transport = doc.createElementNS(NS_WEBRTC_TRANSPORT, 'transport');
        content.appendChild(transport);

        // <session-description type="offer">SDP</session-description>
        const sessionDesc = doc.createElementNS(NS_WEBRTC_TRANSPORT, 'session-description');
        sessionDesc.setAttribute('type', 'offer');
        sessionDesc.textContent = sdpOffer;
        transport.appendChild(sessionDesc);

        return serializeXml(doc);
    }

    /**
     * 构建 transport-info 消息 (ICE Candidate)
     * 
     * @param {RTCIceCandidate} candidate - ICE candidate
     * @returns {string} Jingle XML 字符串
     */
    buildTransportInfo(candidate) {
        if (!this.sessionId) {
            throw new Error('Session ID not set');
        }

        const doc = this._createIqStanza();
        const iq = doc.documentElement;

        // <jingle>
        const jingle = doc.createElementNS(NS_JINGLE, 'jingle');
        jingle.setAttribute('action', 'transport-info');
        jingle.setAttribute('sid', this.sessionId);
        iq.appendChild(jingle);

        // <content>
        const content = doc.createElementNS(NS_JINGLE, 'content');
        content.setAttribute('name', 'chromoting');
        content.setAttribute('creator', 'initiator');
        jingle.appendChild(content);

        // <transport>
        const transport = doc.createElementNS(NS_WEBRTC_TRANSPORT, 'transport');
        content.appendChild(transport);

        // <candidate sdpMid="0" sdpMLineIndex="0">candidate:xxx ...</candidate>
        // 参照 webrtc_transport.cc:
        //   candidate_element->SetBodyText(candidate_str);  // 文本内容
        //   candidate_element->SetAttr("sdpMid", ...);
        //   candidate_element->SetAttr("sdpMLineIndex", ...);
        const candidateElem = doc.createElementNS(NS_WEBRTC_TRANSPORT, 'candidate');
        candidateElem.textContent = candidate.candidate;  // 文本内容，不是属性
        candidateElem.setAttribute('sdpMid', candidate.sdpMid || '');
        if (candidate.sdpMLineIndex !== undefined) {
            candidateElem.setAttribute('sdpMLineIndex', String(candidate.sdpMLineIndex));
        }
        transport.appendChild(candidateElem);

        return serializeXml(doc);
    }

    /**
     * 构建 transport-info 消息 (SDP answer/offer)
     * 参照 webrtc_transport.cc: session-description 通过 transport-info 发送
     * 
     * @param {string} sdp - SDP 字符串
     * @param {string} type - SDP 类型 ('answer' 或 'offer')
     * @returns {string} Jingle XML 字符串
     */
    buildTransportInfoSdp(sdp, type, signature) {
        if (!this.sessionId) {
            throw new Error('Session ID not set');
        }

        const doc = this._createIqStanza();
        const iq = doc.documentElement;

        const jingle = doc.createElementNS(NS_JINGLE, 'jingle');
        jingle.setAttribute('action', 'transport-info');
        jingle.setAttribute('sid', this.sessionId);
        iq.appendChild(jingle);

        const content = doc.createElementNS(NS_JINGLE, 'content');
        content.setAttribute('name', 'chromoting');
        content.setAttribute('creator', 'initiator');
        jingle.appendChild(content);

        const transport = doc.createElementNS(NS_WEBRTC_TRANSPORT, 'transport');
        content.appendChild(transport);

        const sessionDesc = doc.createElementNS(NS_WEBRTC_TRANSPORT, 'session-description');
        sessionDesc.setAttribute('type', type);
        if (signature) {
            sessionDesc.setAttribute('signature', signature);
        }
        sessionDesc.textContent = sdp;
        transport.appendChild(sessionDesc);

        return serializeXml(doc);
    }

    /**
     * 构建 session-info 消息 (用于认证消息交换)
     * 
     * @param {object} authMessage - 认证消息数据
     * @returns {string} Jingle XML 字符串
     */
    buildSessionInfo(authMessage) {
        if (!this.sessionId) {
            throw new Error('Session ID not set');
        }

        const doc = this._createIqStanza();
        const iq = doc.documentElement;

        // <jingle>
        const jingle = doc.createElementNS(NS_JINGLE, 'jingle');
        jingle.setAttribute('action', 'session-info');
        jingle.setAttribute('sid', this.sessionId);
        iq.appendChild(jingle);

        // <authentication> 直接在 jingle 下
        if (authMessage) {
            const auth = this._buildAuthElement(doc, authMessage);
            jingle.appendChild(auth);
        }

        return serializeXml(doc);
    }

    /**
     * 构建 session-terminate 消息
     * 
     * @param {string} reason - 终止原因 (如 "success", "general-error")
     * @returns {string} Jingle XML 字符串
     */
    buildSessionTerminate(reason = 'success') {
        if (!this.sessionId) {
            throw new Error('Session ID not set');
        }

        const doc = this._createIqStanza();
        const iq = doc.documentElement;

        // <jingle>
        const jingle = doc.createElementNS(NS_JINGLE, 'jingle');
        jingle.setAttribute('action', 'session-terminate');
        jingle.setAttribute('sid', this.sessionId);
        iq.appendChild(jingle);

        // <reason>
        const reasonElem = doc.createElementNS(NS_JINGLE, 'reason');
        const reasonChild = doc.createElementNS(NS_JINGLE, reason);
        reasonElem.appendChild(reasonChild);
        jingle.appendChild(reasonElem);

        return serializeXml(doc);
    }

    /**
     * 构建 IQ result 响应
     * 参照 jingle_messages.cc JingleMessageReply::ToXml
     * 
     * @param {string} iqId - 收到的 IQ 的 id 属性
     * @param {string} toJid - 收到的 IQ 的 from 属性（回复方向反转）
     * @returns {string} IQ result XML 字符串
     */
    buildIqResult(iqId, toJid) {
        const doc = document.implementation.createDocument(NS_JABBER_CLIENT, 'cli:iq', null);
        const iq = doc.documentElement;
        iq.setAttribute('type', 'result');
        iq.setAttribute('id', iqId);
        iq.setAttribute('to', toJid);
        iq.setAttribute('from', this.localJid);

        const jingle = doc.createElementNS(NS_JINGLE, 'jingle');
        iq.appendChild(jingle);

        return serializeXml(doc);
    }

    /**
     * 构建认证 XML 元素
     * @private
     */
    _buildAuthElement(doc, authMessage) {
        const auth = doc.createElementNS(NS_CHROMOTING, 'authentication');

        // supported-methods (客户端首次消息)
        if (authMessage.supportedMethods) {
            auth.setAttribute('supported-methods', authMessage.supportedMethods);
        }

        // method (后续消息)
        if (authMessage.method) {
            auth.setAttribute('method', authMessage.method);
        }

        // <spake-message>
        if (authMessage.spakeMessage) {
            const spakeElem = doc.createElementNS(NS_CHROMOTING, 'spake-message');
            spakeElem.textContent = authMessage.spakeMessage;
            auth.appendChild(spakeElem);
        }

        // <verification-hash>
        if (authMessage.verificationHash) {
            const hashElem = doc.createElementNS(NS_CHROMOTING, 'verification-hash');
            hashElem.textContent = authMessage.verificationHash;
            auth.appendChild(hashElem);
        }

        // <certificate>
        if (authMessage.certificate) {
            const certElem = doc.createElementNS(NS_CHROMOTING, 'certificate');
            certElem.textContent = authMessage.certificate;
            auth.appendChild(certElem);
        }

        return auth;
    }
}
