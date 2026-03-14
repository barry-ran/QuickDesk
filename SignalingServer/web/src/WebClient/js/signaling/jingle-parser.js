/**
 * jingle-parser.js - Jingle XML 解析器
 * 
 * 参照 src/remoting/protocol/jingle_messages.cc
 * 和 src/remoting/quickdesk/signaling/webrtc_jingle_converter.cc
 * 
 * 解析 Host 返回的 Jingle XML 信令消息
 */

// ==================== 命名空间常量 ====================

const NS_JABBER_CLIENT = 'jabber:client';
const NS_JINGLE = 'urn:xmpp:jingle:1';
const NS_CHROMOTING = 'google:remoting';
const NS_WEBRTC_TRANSPORT = 'google:remoting:webrtc';

// ==================== 解析后的消息类型 ====================

/**
 * @typedef {Object} JingleMessage
 * @property {string} action - Jingle action (session-accept, transport-info, session-info, session-terminate)
 * @property {string} sid - Session ID
 * @property {string} [from] - 发送方
 * @property {string} [to] - 接收方
 * @property {string} [initiator] - 发起方
 * @property {SdpInfo} [sdp] - SDP 信息
 * @property {IceCandidateInfo} [iceCandidate] - ICE candidate 信息
 * @property {AuthMessage} [authMessage] - 认证消息
 * @property {TerminateInfo} [terminateInfo] - 终止信息
 */

/**
 * @typedef {Object} SdpInfo
 * @property {string} type - "offer" 或 "answer"
 * @property {string} sdp - SDP 内容
 */

/**
 * @typedef {Object} IceCandidateInfo
 * @property {string} candidate - ICE candidate 字符串
 * @property {string} sdpMid - SDP mid
 * @property {number} [sdpMLineIndex] - SDP m-line index
 */

/**
 * @typedef {Object} AuthMessage
 * @property {string} [supportedMethods] - 支持的认证方法列表
 * @property {string} [method] - 选定的认证方法
 * @property {string} [spakeMessage] - SPAKE2 消息 (base64)
 * @property {string} [verificationHash] - 验证哈希 (base64)
 * @property {string} [certificate] - 证书 (base64)
 */

/**
 * @typedef {Object} TerminateInfo
 * @property {string} [reason] - 终止原因
 * @property {string} [errorCode] - 错误码
 * @property {string} [errorDetails] - 错误详情
 */

// ==================== XML 辅助函数 ====================

/**
 * 在指定命名空间下查找子元素
 */
function findChildNS(parent, ns, localName) {
    if (!parent) return null;
    const children = parent.childNodes;
    for (let i = 0; i < children.length; i++) {
        const child = children[i];
        if (child.nodeType === Node.ELEMENT_NODE &&
            child.namespaceURI === ns &&
            child.localName === localName) {
            return child;
        }
    }
    return null;
}

/**
 * 查找子元素（忽略命名空间，按 localName 匹配）
 */
function findChildByLocalName(parent, localName) {
    if (!parent) return null;
    const children = parent.childNodes;
    for (let i = 0; i < children.length; i++) {
        const child = children[i];
        if (child.nodeType === Node.ELEMENT_NODE && child.localName === localName) {
            return child;
        }
    }
    return null;
}

/**
 * 获取元素的文本内容
 */
function getTextContent(element) {
    return element ? element.textContent : null;
}

// ==================== Jingle 解析器 ====================

export class JingleParser {
    /**
     * 解析 Jingle XML 字符串
     * 
     * @param {string} xmlString - Jingle XML 字符串
     * @returns {JingleMessage|null} 解析后的消息，解析失败返回 null
     */
    parse(xmlString) {
        let doc;
        try {
            const parser = new DOMParser();
            doc = parser.parseFromString(xmlString, 'text/xml');
            
            // 检查解析错误
            const parseError = doc.querySelector('parsererror');
            if (parseError) {
                console.error('XML parse error:', parseError.textContent);
                return null;
            }
        } catch (e) {
            console.error('Failed to parse XML:', e);
            return null;
        }

        const root = doc.documentElement;
        
        // 查找 <jingle> 元素
        let jingleElem = findChildNS(root, NS_JINGLE, 'jingle');
        
        // 如果根元素就是 <jingle>，直接使用
        if (!jingleElem && root.localName === 'jingle' && root.namespaceURI === NS_JINGLE) {
            jingleElem = root;
        }
        
        // 如果根元素是 <iq>，从中查找 <jingle>
        if (!jingleElem && (root.localName === 'iq' || root.localName === 'cli:iq')) {
            // 尝试不同的方式查找
            jingleElem = findChildByLocalName(root, 'jingle');
        }

        if (!jingleElem) {
            // IQ result/error stanzas don't have <jingle> element - this is normal
            const iqType = root.getAttribute('type');
            if (iqType === 'result' || iqType === 'error') {
                // 这是对我们发送的 IQ 的确认/错误回复，可以安全忽略
                return { action: '_iq_response', iqType };
            }
            console.warn('No <jingle> element found in message');
            return null;
        }

        const action = jingleElem.getAttribute('action');
        const sid = jingleElem.getAttribute('sid');
        const from = root.getAttribute('from') || '';
        const to = root.getAttribute('to') || '';
        const iqId = root.getAttribute('id') || '';
        const iqType = root.getAttribute('type') || '';
        const initiator = jingleElem.getAttribute('initiator') || '';

        const result = {
            action,
            sid,
            from,
            to,
            iqId,
            iqType,
            initiator,
        };

        switch (action) {
            case 'session-accept':
                this._parseSessionAccept(jingleElem, result);
                break;
            case 'session-initiate':
                this._parseSessionInitiate(jingleElem, result);
                break;
            case 'transport-info':
                this._parseTransportInfo(jingleElem, result);
                break;
            case 'session-info':
                this._parseSessionInfo(jingleElem, result);
                break;
            case 'session-terminate':
                this._parseSessionTerminate(jingleElem, result);
                break;
            default:
                console.warn('Unknown Jingle action:', action);
        }

        return result;
    }

    /**
     * 解析 session-accept
     * @private
     */
    _parseSessionAccept(jingleElem, result) {
        const content = findChildNS(jingleElem, NS_JINGLE, 'content')
                     || findChildByLocalName(jingleElem, 'content');
        
        if (content) {
            // 解析 SDP (从 transport 元素)
            const transport = findChildNS(content, NS_WEBRTC_TRANSPORT, 'transport')
                           || findChildNS(content, NS_CHROMOTING, 'transport')
                           || findChildByLocalName(content, 'transport');

            if (transport) {
                const sessionDesc = findChildNS(transport, NS_WEBRTC_TRANSPORT, 'session-description')
                                 || findChildNS(transport, NS_CHROMOTING, 'session-description')
                                 || findChildByLocalName(transport, 'session-description');
                if (sessionDesc) {
                    result.sdp = {
                        type: sessionDesc.getAttribute('type') || 'answer',
                        sdp: getTextContent(sessionDesc),
                    };
                }
            }

            // 解析认证消息 (从 description 元素)
            const description = findChildNS(content, NS_CHROMOTING, 'description')
                             || findChildByLocalName(content, 'description');
            if (description) {
                const auth = this._parseAuthElement(description);
                if (auth) {
                    result.authMessage = auth;
                }
            }
        }
    }

    /**
     * 解析 session-initiate (web端通常不需要，但为完整性实现)
     * @private
     */
    _parseSessionInitiate(jingleElem, result) {
        // 同 session-accept 逻辑
        this._parseSessionAccept(jingleElem, result);
        if (result.sdp) {
            result.sdp.type = 'offer';
        }
    }

    /**
     * 解析 transport-info
     * @private
     */
    _parseTransportInfo(jingleElem, result) {
        const content = findChildNS(jingleElem, NS_JINGLE, 'content')
                     || findChildByLocalName(jingleElem, 'content');
        
        if (!content) return;

        const transport = findChildNS(content, NS_WEBRTC_TRANSPORT, 'transport')
                       || findChildNS(content, NS_CHROMOTING, 'transport')
                       || findChildByLocalName(content, 'transport');
        
        if (!transport) return;

        // 检查是否包含 SDP (transport-info 可能包含 SDP answer)
        const sessionDesc = findChildNS(transport, NS_WEBRTC_TRANSPORT, 'session-description')
                         || findChildNS(transport, NS_CHROMOTING, 'session-description')
                         || findChildByLocalName(transport, 'session-description');
        if (sessionDesc) {
            result.sdp = {
                type: sessionDesc.getAttribute('type') || 'answer',
                sdp: getTextContent(sessionDesc),
            };
        }

        // 检查 ICE candidates (可能有多个)
        // 参照 webrtc_transport.cc:
        //   candidate_str = candidate_element->BodyText();  (文本内容，不是属性!)
        //   sdpMid = candidate_element->Attr("sdpMid");
        //   sdpMLineIndex = candidate_element->Attr("sdpMLineIndex");
        const candidates = [];
        const children = transport.childNodes;
        for (let i = 0; i < children.length; i++) {
            const child = children[i];
            if (child.nodeType === Node.ELEMENT_NODE && child.localName === 'candidate') {
                // Candidate 字符串在文本内容中 (BodyText)，不在 attribute 中
                const candidateStr = getTextContent(child);
                const sdpMid = child.getAttribute('sdpMid') || '';
                const sdpMLineIndex = parseInt(child.getAttribute('sdpMLineIndex') || '0', 10);
                if (candidateStr) {
                    candidates.push({
                        candidate: candidateStr,
                        sdpMid,
                        sdpMLineIndex,
                    });
                }
            }
        }
        if (candidates.length === 1) {
            result.iceCandidate = candidates[0];
        } else if (candidates.length > 1) {
            result.iceCandidates = candidates;
        }
    }

    /**
     * 解析 session-info
     * @private
     */
    _parseSessionInfo(jingleElem, result) {
        // session-info 的认证消息直接在 jingle 下
        const auth = this._parseAuthElement(jingleElem);
        if (auth) {
            result.authMessage = auth;
        }
    }

    /**
     * 解析 session-terminate
     * @private
     */
    _parseSessionTerminate(jingleElem, result) {
        const terminateInfo = {};

        // <reason>
        const reason = findChildNS(jingleElem, NS_JINGLE, 'reason')
                    || findChildByLocalName(jingleElem, 'reason');
        if (reason) {
            // reason 的第一个子元素的标签名就是原因
            for (let i = 0; i < reason.childNodes.length; i++) {
                if (reason.childNodes[i].nodeType === Node.ELEMENT_NODE) {
                    terminateInfo.reason = reason.childNodes[i].localName;
                    break;
                }
            }
        }

        // <error-code>
        const errorCode = findChildNS(jingleElem, NS_CHROMOTING, 'error-code')
                       || findChildByLocalName(jingleElem, 'error-code');
        if (errorCode) {
            terminateInfo.errorCode = getTextContent(errorCode);
        }

        // <error-details>
        const errorDetails = findChildNS(jingleElem, NS_CHROMOTING, 'error-details')
                          || findChildByLocalName(jingleElem, 'error-details');
        if (errorDetails) {
            terminateInfo.errorDetails = getTextContent(errorDetails);
        }

        result.terminateInfo = terminateInfo;
    }

    /**
     * 解析认证元素
     * @private
     * @param {Element} parent - 父元素 (description 或 jingle)
     * @returns {AuthMessage|null}
     */
    _parseAuthElement(parent) {
        const auth = findChildNS(parent, NS_CHROMOTING, 'authentication')
                  || findChildByLocalName(parent, 'authentication');
        
        if (!auth) return null;

        const result = {};

        // 属性
        const supportedMethods = auth.getAttribute('supported-methods');
        if (supportedMethods) result.supportedMethods = supportedMethods;

        const method = auth.getAttribute('method');
        if (method) result.method = method;

        // <spake-message>
        const spakeMsg = findChildNS(auth, NS_CHROMOTING, 'spake-message')
                      || findChildByLocalName(auth, 'spake-message');
        if (spakeMsg) {
            result.spakeMessage = getTextContent(spakeMsg);
        }

        // <verification-hash>
        const verHash = findChildNS(auth, NS_CHROMOTING, 'verification-hash')
                     || findChildByLocalName(auth, 'verification-hash');
        if (verHash) {
            result.verificationHash = getTextContent(verHash);
        }

        // <certificate>
        const cert = findChildNS(auth, NS_CHROMOTING, 'certificate')
                  || findChildByLocalName(auth, 'certificate');
        if (cert) {
            result.certificate = getTextContent(cert);
        }

        return result;
    }
}
