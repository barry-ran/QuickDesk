/// jingle.dart - Jingle XML 信令编解码
///
/// 对照 WebClient/js/signaling/jingle-builder.js + jingle-parser.js
/// （其参照 src/remoting/protocol/jingle_messages.cc 与
///  webrtc_jingle_converter.cc），用 package:xml 替代浏览器 DOM。
library;

import 'dart:math';

import 'package:xml/xml.dart';

import '../auth/spake2_authenticator.dart' show AuthMessage;

// ==================== 命名空间常量 ====================

const nsJabberClient = 'jabber:client';
const nsJingle = 'urn:xmpp:jingle:1';
const nsChromoting = 'google:remoting';
const nsWebrtcTransport = 'google:remoting:webrtc';

// ==================== 数据模型 ====================

class SdpInfo {
  final String type; // 'offer' | 'answer'
  final String sdp;
  final String? signature;

  SdpInfo({required this.type, required this.sdp, this.signature});
}

class IceCandidateInfo {
  final String candidate;
  final String sdpMid;
  final int sdpMLineIndex;

  IceCandidateInfo({required this.candidate, required this.sdpMid, required this.sdpMLineIndex});
}

class TerminateInfo {
  String? reason;
  String? errorCode;
  String? errorDetails;
}

/// 解析后的 Jingle 消息
class JingleMessage {
  final String action; // session-initiate/accept/info/terminate/transport-info/_iq_response
  final String sid;
  final String from;
  final String to;
  final String iqId;
  final String iqType;
  final String initiator;
  SdpInfo? sdp;
  final List<IceCandidateInfo> iceCandidates = [];
  AuthMessage? authMessage;
  TerminateInfo? terminateInfo;

  JingleMessage({
    required this.action,
    this.sid = '',
    this.from = '',
    this.to = '',
    this.iqId = '',
    this.iqType = '',
    this.initiator = '',
  });
}

// ==================== 构建器 ====================

String _generateUuid() {
  final r = Random.secure();
  final b = List<int>.generate(16, (_) => r.nextInt(256));
  b[6] = (b[6] & 0x0f) | 0x40;
  b[8] = (b[8] & 0x3f) | 0x80;
  final h = b.map((x) => x.toRadixString(16).padLeft(2, '0')).join();
  return '${h.substring(0, 8)}-${h.substring(8, 12)}-${h.substring(12, 16)}-${h.substring(16, 20)}-${h.substring(20)}';
}

class JingleBuilder {
  String? sessionId;
  String localJid = '';
  String remoteJid = '';

  /// 生成新的会话 ID（Chromium 风格的大整数字符串）
  String generateSessionId() {
    final r = Random.secure();
    var num = BigInt.zero;
    for (var i = 0; i < 8; i++) {
      num = (num << 8) | BigInt.from(r.nextInt(256));
    }
    sessionId = num.toString();
    return sessionId!;
  }

  XmlElement _createIq({String type = 'set', String? id, String? to}) {
    final iq = XmlElement(XmlName('cli:iq'));
    iq.setAttribute('xmlns', nsJabberClient);
    iq.setAttribute('to', to ?? remoteJid);
    iq.setAttribute('from', localJid);
    iq.setAttribute('type', type);
    iq.setAttribute('id', id ?? _generateUuid());
    return iq;
  }

  XmlElement _createJingle(String action, {bool withInitiator = false}) {
    final jingle = XmlElement(XmlName('jingle'));
    jingle.setAttribute('xmlns', nsJingle);
    jingle.setAttribute('action', action);
    jingle.setAttribute('sid', sessionId!);
    if (withInitiator) {
      jingle.setAttribute('initiator', localJid);
    }
    return jingle;
  }

  XmlElement _createContent() {
    final content = XmlElement(XmlName('content'));
    content.setAttribute('name', 'chromoting');
    content.setAttribute('creator', 'initiator');
    return content;
  }

  XmlElement _buildAuthElement(AuthMessage auth) {
    final elem = XmlElement(XmlName('authentication'));
    elem.setAttribute('xmlns', nsChromoting);
    if (auth.supportedMethods != null) {
      elem.setAttribute('supported-methods', auth.supportedMethods!);
    }
    if (auth.method != null) {
      elem.setAttribute('method', auth.method!);
    }
    if (auth.spakeMessage != null) {
      final spake = XmlElement(XmlName('spake-message'));
      spake.innerText = auth.spakeMessage!;
      elem.children.add(spake);
    }
    if (auth.verificationHash != null) {
      final hash = XmlElement(XmlName('verification-hash'));
      hash.innerText = auth.verificationHash!;
      elem.children.add(hash);
    }
    if (auth.certificate != null) {
      final cert = XmlElement(XmlName('certificate'));
      cert.innerText = auth.certificate!;
      elem.children.add(cert);
    }
    return elem;
  }

  XmlElement _createSessionDescription(String sdp, String type, {String? signature}) {
    final sessionDesc = XmlElement(XmlName('session-description'));
    sessionDesc.setAttribute('type', type);
    if (signature != null && signature.isNotEmpty) {
      sessionDesc.setAttribute('signature', signature);
    }
    sessionDesc.innerText = sdp;
    return sessionDesc;
  }

  /// session-initiate（Client 角色发起）
  String buildSessionInitiate(String sdpOffer, AuthMessage authMessage) {
    sessionId ??= generateSessionId();

    final iq = _createIq();
    final jingle = _createJingle('session-initiate', withInitiator: true);
    iq.children.add(jingle);

    final content = _createContent();
    jingle.children.add(content);

    final description = XmlElement(XmlName('description'));
    description.setAttribute('xmlns', nsChromoting);
    content.children.add(description);
    description.children.add(_buildAuthElement(authMessage));

    final transport = XmlElement(XmlName('transport'));
    transport.setAttribute('xmlns', nsWebrtcTransport);
    content.children.add(transport);
    transport.children.add(_createSessionDescription(sdpOffer, 'offer'));

    return iq.toXmlString();
  }

  /// session-accept（Host 角色应答，M3 被控端用）
  /// 对照 jingle_messages.cc: accept 消息结构与 initiate 一致，action 不同
  String buildSessionAccept(String sdpAnswer, AuthMessage authMessage) {
    if (sessionId == null) {
      throw StateError('Session ID not set (must reuse initiator sid)');
    }

    final iq = _createIq();
    final jingle = _createJingle('session-accept');
    iq.children.add(jingle);

    final content = _createContent();
    jingle.children.add(content);

    final description = XmlElement(XmlName('description'));
    description.setAttribute('xmlns', nsChromoting);
    content.children.add(description);
    description.children.add(_buildAuthElement(authMessage));

    final transport = XmlElement(XmlName('transport'));
    transport.setAttribute('xmlns', nsWebrtcTransport);
    content.children.add(transport);
    transport.children.add(_createSessionDescription(sdpAnswer, 'answer'));

    return iq.toXmlString();
  }

  /// transport-info：单个 ICE candidate
  String buildTransportInfo(IceCandidateInfo candidate) {
    if (sessionId == null) throw StateError('Session ID not set');

    final iq = _createIq();
    final jingle = _createJingle('transport-info');
    iq.children.add(jingle);

    final content = _createContent();
    jingle.children.add(content);

    final transport = XmlElement(XmlName('transport'));
    transport.setAttribute('xmlns', nsWebrtcTransport);
    content.children.add(transport);

    // candidate 字符串在文本内容中（BodyText），不在属性里
    final candidateElem = XmlElement(XmlName('candidate'));
    candidateElem.innerText = candidate.candidate;
    candidateElem.setAttribute('sdpMid', candidate.sdpMid);
    candidateElem.setAttribute('sdpMLineIndex', candidate.sdpMLineIndex.toString());
    transport.children.add(candidateElem);

    return iq.toXmlString();
  }

  /// transport-info：SDP（重协商 offer/answer）
  String buildTransportInfoSdp(String sdp, String type, {String? signature}) {
    if (sessionId == null) throw StateError('Session ID not set');

    final iq = _createIq();
    final jingle = _createJingle('transport-info');
    iq.children.add(jingle);

    final content = _createContent();
    jingle.children.add(content);

    final transport = XmlElement(XmlName('transport'));
    transport.setAttribute('xmlns', nsWebrtcTransport);
    content.children.add(transport);
    transport.children.add(_createSessionDescription(sdp, type, signature: signature));

    return iq.toXmlString();
  }

  /// session-info（认证消息交换）
  String buildSessionInfo(AuthMessage authMessage) {
    if (sessionId == null) throw StateError('Session ID not set');

    final iq = _createIq();
    final jingle = _createJingle('session-info');
    iq.children.add(jingle);
    jingle.children.add(_buildAuthElement(authMessage));

    return iq.toXmlString();
  }

  /// session-terminate
  String buildSessionTerminate([String reason = 'success']) {
    if (sessionId == null) throw StateError('Session ID not set');

    final iq = _createIq();
    final jingle = _createJingle('session-terminate');
    iq.children.add(jingle);

    final reasonElem = XmlElement(XmlName('reason'));
    reasonElem.children.add(XmlElement(XmlName(reason)));
    jingle.children.add(reasonElem);

    return iq.toXmlString();
  }

  /// IQ result 响应（XMPP 协议要求对 type=set 回 ack）
  String buildIqResult(String iqId, String toJid) {
    final iq = _createIq(type: 'result', id: iqId, to: toJid);
    iq.children.add(XmlElement(XmlName('jingle'))..setAttribute('xmlns', nsJingle));
    return iq.toXmlString();
  }
}

// ==================== 解析器 ====================

class JingleParser {
  /// 解析 Jingle XML，失败返回 null
  JingleMessage? parse(String xmlString) {
    XmlDocument doc;
    try {
      doc = XmlDocument.parse(xmlString);
    } catch (e) {
      return null;
    }

    final root = doc.rootElement;

    XmlElement? jingleElem = _findChildByLocalName(root, 'jingle');
    if (jingleElem == null && root.name.local == 'jingle') {
      jingleElem = root;
    }

    if (jingleElem == null) {
      final iqType = root.getAttribute('type') ?? '';
      if (iqType == 'result' || iqType == 'error') {
        return JingleMessage(action: '_iq_response', iqType: iqType);
      }
      return null;
    }

    final result = JingleMessage(
      action: jingleElem.getAttribute('action') ?? '',
      sid: jingleElem.getAttribute('sid') ?? '',
      from: root.getAttribute('from') ?? '',
      to: root.getAttribute('to') ?? '',
      iqId: root.getAttribute('id') ?? '',
      iqType: root.getAttribute('type') ?? '',
      initiator: jingleElem.getAttribute('initiator') ?? '',
    );

    switch (result.action) {
      case 'session-accept':
      case 'session-initiate':
        _parseContentMessage(jingleElem, result);
        break;
      case 'transport-info':
        _parseTransportInfo(jingleElem, result);
        break;
      case 'session-info':
        result.authMessage = _parseAuthElement(jingleElem);
        break;
      case 'session-terminate':
        _parseSessionTerminate(jingleElem, result);
        break;
      default:
        break;
    }

    return result;
  }

  XmlElement? _findChildByLocalName(XmlElement parent, String localName) {
    for (final child in parent.childElements) {
      if (child.name.local == localName) return child;
    }
    return null;
  }

  void _parseContentMessage(XmlElement jingleElem, JingleMessage result) {
    final content = _findChildByLocalName(jingleElem, 'content');
    if (content == null) return;

    final transport = _findChildByLocalName(content, 'transport');
    if (transport != null) {
      final sessionDesc = _findChildByLocalName(transport, 'session-description');
      if (sessionDesc != null) {
        result.sdp = SdpInfo(
          type: sessionDesc.getAttribute('type') ??
              (result.action == 'session-initiate' ? 'offer' : 'answer'),
          sdp: sessionDesc.innerText,
          signature: sessionDesc.getAttribute('signature'),
        );
      }
    }

    final description = _findChildByLocalName(content, 'description');
    if (description != null) {
      result.authMessage = _parseAuthElement(description);
    }
  }

  void _parseTransportInfo(XmlElement jingleElem, JingleMessage result) {
    final content = _findChildByLocalName(jingleElem, 'content');
    if (content == null) return;

    final transport = _findChildByLocalName(content, 'transport');
    if (transport == null) return;

    final sessionDesc = _findChildByLocalName(transport, 'session-description');
    if (sessionDesc != null) {
      result.sdp = SdpInfo(
        type: sessionDesc.getAttribute('type') ?? 'answer',
        sdp: sessionDesc.innerText,
        signature: sessionDesc.getAttribute('signature'),
      );
    }

    for (final child in transport.childElements) {
      if (child.name.local != 'candidate') continue;
      final candidateStr = child.innerText;
      if (candidateStr.isEmpty) continue;
      result.iceCandidates.add(IceCandidateInfo(
        candidate: candidateStr,
        sdpMid: child.getAttribute('sdpMid') ?? '',
        sdpMLineIndex: int.tryParse(child.getAttribute('sdpMLineIndex') ?? '0') ?? 0,
      ));
    }
  }

  void _parseSessionTerminate(XmlElement jingleElem, JingleMessage result) {
    final info = TerminateInfo();

    final reason = _findChildByLocalName(jingleElem, 'reason');
    if (reason != null) {
      for (final child in reason.childElements) {
        info.reason = child.name.local;
        break;
      }
    }

    final errorCode = _findChildByLocalName(jingleElem, 'error-code');
    if (errorCode != null) info.errorCode = errorCode.innerText;

    final errorDetails = _findChildByLocalName(jingleElem, 'error-details');
    if (errorDetails != null) info.errorDetails = errorDetails.innerText;

    result.terminateInfo = info;
  }

  AuthMessage? _parseAuthElement(XmlElement parent) {
    final auth = _findChildByLocalName(parent, 'authentication');
    if (auth == null) return null;

    final result = AuthMessage();
    result.supportedMethods = auth.getAttribute('supported-methods');
    result.method = auth.getAttribute('method');

    final spakeMsg = _findChildByLocalName(auth, 'spake-message');
    if (spakeMsg != null) result.spakeMessage = spakeMsg.innerText;

    final verHash = _findChildByLocalName(auth, 'verification-hash');
    if (verHash != null) result.verificationHash = verHash.innerText;

    final cert = _findChildByLocalName(auth, 'certificate');
    if (cert != null) result.certificate = cert.innerText;

    return result.isEmpty ? null : result;
  }
}
