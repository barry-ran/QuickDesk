/// host_session.dart - 被控端会话状态机（Host / responder 角色）
///
/// 与 client_session.dart 互为镜像，负责被桌面端/其它客户端控制时的协议处理。
/// 采用与 Chromium Host 一致的**两次协商**流程（对齐 Qt/Web 各端客户端）：
///
///   第一次协商（基础连接）：
///   1. client → session-initiate（offer #1 + auth supported-methods）
///   2. Host：setRemote(offer #1) → createAnswer #1（视频轨未加，媒体 inactive）
///      → session-accept（answer #1 + method + 自己的 SPAKE2 消息）
///      answer #1 未签名（auth_key 尚未就绪，client 不校验 session-accept 签名）
///   3. session-info 往返完成 SPAKE2（client 发 spake+hash，Host 回 hash）
///
///   第二次协商（视频，认证后，走已连好的同一传输）：
///   4. Host 加屏幕视频轨 + 建 'control' DataChannel → createOffer #2
///      → 用 auth_key 对 SDP 签名 → transport-info(offer)
///   5. client → transport-info(answer #2, 带签名)，Host 校验签名后 setRemote
///   6. 双向 transport-info 交换 ICE candidate（认证前缓冲，认证后 flush）
///   7. client 创建 'event' DataChannel，Host 收到后把输入事件交给注入层
///
/// 一个 HostSession 复用一条 host 信令 WS，可服务多个并发客户端
/// （按 signaling client_id 区分），共享同一路屏幕采集 MediaStream。
library;

import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:crypto/crypto.dart' as crypto;
import 'package:flutter_webrtc/flutter_webrtc.dart';

import '../api/signaling_api.dart' show IceServerEntry;
import 'auth/spake2_authenticator.dart';
import 'host_input.dart';
import 'proto/protobuf_messages.dart';
import 'signaling/jingle.dart';
import 'signaling/websocket_transport.dart';

const String _hostResource = 'chromoting_ftl_quickdesk_host';

enum HostSessionState {
  idle,
  listening, // 信令已连接，等待客户端接入
  failed,
  closed,
}

/// 单个客户端的连接状态
enum HostPeerState {
  negotiating,
  authenticating,
  connected,
  closed,
  failed,
}

class HostSession {
  final String signalingUrl;
  final List<IceServerEntry> iceServers;

  /// SPAKE2 共享密钥哈希：HMAC(device_id, device_id + access_code)
  final Uint8List sharedSecretHash;

  final String deviceId;

  /// 屏幕采集流（含视频轨）。认证完成后每个客户端 PC 都会加入其视频轨。
  final MediaStream Function() screenStreamProvider;

  /// 屏幕像素尺寸（用于下发 VideoLayout，client 据此做坐标映射）
  int screenWidth;
  int screenHeight;

  /// 收到的客户端输入事件（鼠标/键盘/文本）回调 → 交注入层
  final void Function(HostInputEvent event)? onInput;

  /// Host 声明的能力集（最小集：无文件传输/隐私屏）
  final String capabilities;

  HostSessionState state = HostSessionState.idle;
  String? failureReason;

  WebSocketTransport? _transport;
  final Map<String, _HostPeer> _peers = {}; // client_id -> peer

  final _stateCtrl = StreamController<HostSessionState>.broadcast();
  final _peerCtrl = StreamController<int>.broadcast(); // 当前连接的客户端数
  final _logCtrl = StreamController<String>.broadcast();

  Stream<HostSessionState> get onStateChange => _stateCtrl.stream;
  Stream<int> get onPeerCountChange => _peerCtrl.stream;
  Stream<String> get onLog => _logCtrl.stream;

  int get peerCount => _peers.values.where((p) => p.state == HostPeerState.connected).length;

  HostSession({
    required this.signalingUrl,
    required this.deviceId,
    required this.sharedSecretHash,
    required this.screenStreamProvider,
    required this.screenWidth,
    required this.screenHeight,
    this.iceServers = const [],
    this.onInput,
    this.capabilities = '',
  });

  String get _hostJid => '$deviceId@quickdesk.local/$_hostResource';

  /// 连接信令并进入监听状态。[signalToken] 由 HostApi.issueSignalToken 换取（一次性）。
  Future<void> listen(String signalToken) async {
    _transport = WebSocketTransport(
      signalingUrl: signalingUrl,
      onMessage: _onSignalingMessage,
      onAuthOk: () => _log('host signaling auth_ok'),
      onClose: (code, reason) {
        _log('host signaling closed: $code $reason');
        if (state != HostSessionState.closed) {
          _setState(HostSessionState.failed);
        }
      },
      onError: (e) => _log('host signaling error: $e'),
    );

    await _transport!.connect(
      deviceId: deviceId,
      signalToken: signalToken,
      role: 'host',
    );
    _setState(HostSessionState.listening);
  }

  Future<void> stop() async {
    for (final peer in _peers.values.toList()) {
      await peer.close(sendTerminate: true);
    }
    _peers.clear();
    _transport?.disconnect();
    _transport = null;
    _setState(HostSessionState.closed);
  }

  /// 屏幕尺寸变化（旋转/分辨率切换）时更新并向所有客户端重发 VideoLayout
  void updateScreenSize(int width, int height) {
    screenWidth = width;
    screenHeight = height;
    for (final peer in _peers.values) {
      if (peer.state == HostPeerState.connected) {
        peer.sendVideoLayout(width, height);
      }
    }
  }

  // ==================== 信令入口 ====================

  void _onSignalingMessage(String message, String? fromClientId) {
    final clientId = fromClientId ?? '';
    final trimmed = message.trim();

    if (trimmed.startsWith('{')) {
      // JSON 控制帧（错误等）；host 侧只记录
      try {
        final json = jsonDecode(trimmed) as Map<String, dynamic>;
        if (json['type'] == 'error') {
          _log('signaling error from server: ${json['code']}');
          final peer = _peers[clientId];
          if (peer != null) {
            peer.close(sendTerminate: false);
            _peers.remove(clientId);
            _peerCtrl.add(peerCount);
          }
        }
      } catch (_) {}
      return;
    }

    if (!trimmed.startsWith('<')) return;

    // 每个客户端的消息串行处理
    var peer = _peers[clientId];
    if (peer == null) {
      peer = _HostPeer(session: this, clientId: clientId);
      _peers[clientId] = peer;
    }
    peer.enqueue(trimmed);
  }

  // ==================== 供 _HostPeer 使用的工具 ====================

  void _sendToClient(String clientId, String xml) {
    if (_transport == null || !_transport!.isConnected) return;
    _transport!.send(xml, targetClientId: clientId);
  }

  void _onPeerClosed(String clientId) {
    _peers.remove(clientId);
    _peerCtrl.add(peerCount);
  }

  void _onPeerConnected() {
    _peerCtrl.add(peerCount);
  }

  void _setState(HostSessionState s) {
    state = s;
    _stateCtrl.add(s);
  }

  void _log(String message) {
    // ignore: avoid_print
    print('[HostSession] $message');
    _logCtrl.add(message);
  }

  void dispose() {
    stop();
    _stateCtrl.close();
    _peerCtrl.close();
    _logCtrl.close();
  }
}

/// 单客户端连接：一套 Jingle sid + RTCPeerConnection + SPAKE2 Bob。
class _HostPeer {
  final HostSession session;
  final String clientId;

  final JingleBuilder _jingle = JingleBuilder();
  final JingleParser _parser = JingleParser();
  Spake2HostAuthenticator? _auth;
  RTCPeerConnection? _pc;

  RTCDataChannel? _controlChannel;
  RTCDataChannel? _eventChannel;
  bool _controlReady = false;

  HostPeerState state = HostPeerState.negotiating;
  bool _remoteDescriptionSet = false;
  bool _mediaNegotiationStarted = false;
  bool _authenticated = false;

  final List<RTCIceCandidate> _pendingRemoteCandidates = [];
  final List<RTCIceCandidate> _pendingLocalCandidates = [];

  // 逐条串行处理信令
  final List<String> _queue = [];
  bool _processing = false;

  _HostPeer({required this.session, required this.clientId});

  void enqueue(String xml) {
    _queue.add(xml);
    if (!_processing) _processNext();
  }

  Future<void> _processNext() async {
    if (_queue.isEmpty) {
      _processing = false;
      return;
    }
    _processing = true;
    final xml = _queue.removeAt(0);
    try {
      final parsed = _parser.parse(xml);
      if (parsed != null) await _handle(parsed);
    } catch (e) {
      session._log('peer[$clientId] error: $e');
    }
    await _processNext();
  }

  Future<void> _handle(JingleMessage message) async {
    // 过滤不属于本会话的消息
    if (message.sid.isNotEmpty &&
        _jingle.sessionId != null &&
        message.sid != _jingle.sessionId) {
      if (message.iqType == 'set' && message.iqId.isNotEmpty && message.from.isNotEmpty) {
        session._sendToClient(clientId, _jingle.buildIqResult(message.iqId, message.from));
      }
      return;
    }

    switch (message.action) {
      case 'session-initiate':
        await _handleSessionInitiate(message);
        break;
      case 'session-info':
        await _handleSessionInfo(message);
        break;
      case 'transport-info':
        await _handleTransportInfo(message);
        break;
      case 'session-terminate':
        session._log('peer[$clientId] terminated by client: ${message.terminateInfo?.reason}');
        await close(sendTerminate: false);
        session._onPeerClosed(clientId);
        break;
      case '_iq_response':
        break;
      default:
        session._log('peer[$clientId] unhandled action: ${message.action}');
    }

    // 对 type=set 的 IQ 回 result（XMPP 要求）
    if (message.iqType == 'set' && message.iqId.isNotEmpty && message.from.isNotEmpty) {
      session._sendToClient(clientId, _jingle.buildIqResult(message.iqId, message.from));
    }
  }

  Future<void> _handleSessionInitiate(JingleMessage message) async {
    session._log('peer[$clientId] session-initiate (sid=${message.sid})');

    // 复用 client 的 sid，remote JID 取 initiator（client 的精确 FTL resource）
    _jingle.sessionId = message.sid;
    _jingle.localJid = session._hostJid;
    _jingle.remoteJid = message.initiator.isNotEmpty ? message.initiator : message.from;

    // SPAKE2 Bob：local=hostJid, remote=clientJid
    _auth = Spake2HostAuthenticator(
      _jingle.localJid,
      _jingle.remoteJid,
      session.sharedSecretHash,
    );

    // 校验 client 支持的方法
    final clientAuth = message.authMessage;
    if (clientAuth == null || clientAuth.supportedMethods == null) {
      session._log('peer[$clientId] session-initiate missing supported-methods');
      await _fail('no auth in session-initiate');
      return;
    }
    _auth!.processMessage(clientAuth);
    if (_auth!.state == AuthState.rejected) {
      await _fail('auth method rejected: ${_auth!.rejectionReason}');
      return;
    }

    // 需要 client 的初始 offer 来建立基础连接（第一次协商）
    if (message.sdp == null || message.sdp!.sdp.isEmpty) {
      await _fail('session-initiate missing offer SDP');
      return;
    }

    await _createPeerConnection();

    // 第一次协商（基础连接）：应答 client 的 offer。
    // 此时视频轨尚未加入（认证后才推），故 answer 里媒体为 inactive；
    // 该 answer 未签名（auth_key 尚未就绪），client 不校验 session-accept 的签名。
    // DataChannel 与视频留待认证后的第二次协商（host 作为 offerer，带签名）。
    try {
      await _pc!.setRemoteDescription(
          RTCSessionDescription(message.sdp!.sdp, 'offer'));
      _remoteDescriptionSet = true;
      final answer = await _pc!.createAnswer();
      await _pc!.setLocalDescription(answer);

      state = HostPeerState.authenticating;
      final authMsg = _auth!.getNextMessage();
      session._sendToClient(
          clientId, _jingle.buildSessionAccept(answer.sdp!, authMsg));
      session._log('peer[$clientId] sent session-accept (answer + auth)');

      await _flushRemoteCandidates();
    } catch (e) {
      await _fail('base negotiation failed: $e');
    }
  }

  Future<void> _handleSessionInfo(JingleMessage message) async {
    if (message.authMessage == null || _auth == null) return;

    _auth!.processMessage(message.authMessage!);
    final st = _auth!.state;

    if (st == AuthState.rejected) {
      await _fail('auth rejected: ${_auth!.rejectionReason}');
      return;
    }

    if (st == AuthState.messageReady) {
      final next = _auth!.getNextMessage();
      session._sendToClient(clientId, _jingle.buildSessionInfo(next));
    }

    if (_auth!.state == AuthState.accepted) {
      session._log('peer[$clientId] authenticated');
      _authenticated = true;
      // 认证完成 → host 作为 offerer 发起媒体协商
      await _startMediaNegotiation();
      _flushLocalCandidates();
    }
  }

  Future<void> _handleTransportInfo(JingleMessage message) async {
    // client 的 answer（对 host offer 的应答）
    if (message.sdp != null && message.sdp!.type == 'answer') {
      if (!_verifySignature(message.sdp!.sdp, 'answer', message.sdp!.signature)) {
        session._log('peer[$clientId] WARNING: answer signature mismatch');
      }
      try {
        await _pc!.setRemoteDescription(
            RTCSessionDescription(message.sdp!.sdp, 'answer'));
        _remoteDescriptionSet = true;
        session._log('peer[$clientId] remote answer set');
        await _flushRemoteCandidates();
      } catch (e) {
        session._log('peer[$clientId] setRemoteDescription failed: $e');
      }
    }

    for (final info in message.iceCandidates) {
      final candidate = RTCIceCandidate(info.candidate, info.sdpMid, info.sdpMLineIndex);
      if (_remoteDescriptionSet) {
        try {
          await _pc!.addCandidate(candidate);
        } catch (e) {
          session._log('peer[$clientId] addCandidate failed: $e');
        }
      } else {
        _pendingRemoteCandidates.add(candidate);
      }
    }
  }

  Future<void> _createPeerConnection() async {
    final config = <String, dynamic>{
      'iceServers': session.iceServers.map((e) => e.toRtcConfig()).toList(),
      'bundlePolicy': 'max-bundle',
      'sdpSemantics': 'unified-plan',
    };
    _pc = await createPeerConnection(config);

    _pc!.onIceCandidate = (candidate) {
      if (candidate.candidate == null || candidate.candidate!.isEmpty) return;
      if (!_authenticated) {
        _pendingLocalCandidates.add(candidate);
        return;
      }
      _sendLocalCandidate(candidate);
    };

    _pc!.onIceConnectionState = (s) {
      session._log('peer[$clientId] ICE: $s');
      if (s == RTCIceConnectionState.RTCIceConnectionStateConnected ||
          s == RTCIceConnectionState.RTCIceConnectionStateCompleted) {
        if (state != HostPeerState.connected) {
          state = HostPeerState.connected;
          session._onPeerConnected();
        }
      } else if (s == RTCIceConnectionState.RTCIceConnectionStateFailed) {
        _fail('ICE failed');
      }
    };

    // client 会创建 'event' DataChannel（输入事件）
    _pc!.onDataChannel = (channel) {
      session._log('peer[$clientId] datachannel: ${channel.label}');
      if (channel.label == 'event') {
        _setupEventChannel(channel);
      }
    };
  }

  /// 认证完成后：加视频轨 + 建 control 通道 + 生成签名 offer
  Future<void> _startMediaNegotiation() async {
    if (_mediaNegotiationStarted) return;
    _mediaNegotiationStarted = true;

    // 加入屏幕视频轨（sendonly）
    final stream = session.screenStreamProvider();
    for (final track in stream.getVideoTracks()) {
      await _pc!.addTrack(track, stream);
    }

    // host 创建 'control' 通道（下发 cursor/clipboard/capabilities/VideoLayout）
    _controlChannel = await _pc!.createDataChannel(
      'control',
      RTCDataChannelInit()..ordered = true,
    );
    _controlChannel!.onDataChannelState = (s) {
      if (s == RTCDataChannelState.RTCDataChannelOpen) {
        _controlReady = true;
        _onControlOpen();
      } else if (s == RTCDataChannelState.RTCDataChannelClosed) {
        _controlReady = false;
      }
    };
    _controlChannel!.onMessage = (msg) {
      if (msg.isBinary) _handleControlMessage(msg.binary);
    };

    final offer = await _pc!.createOffer();
    await _pc!.setLocalDescription(offer);

    final signature = _signSdp(offer.sdp!, 'offer');
    session._sendToClient(
      clientId,
      _jingle.buildTransportInfoSdp(offer.sdp!, 'offer', signature: signature),
    );
    session._log('peer[$clientId] sent signed offer (transport-info)');
  }

  void _onControlOpen() {
    // 首先协商能力，再下发屏幕布局
    if (session.capabilities.isNotEmpty) {
      _sendControl(encodeControlMessage(capabilities: session.capabilities));
    }
    sendVideoLayout(session.screenWidth, session.screenHeight);
  }

  void sendVideoLayout(int width, int height) {
    if (!_controlReady) return;
    // mediaStreamId 必须与 SDP msid 一致，client 才能把 VideoLayout 与收到的
    // WebRTC 流对应起来，故取采集流的真实 id 而非占位串。
    final streamId = session.screenStreamProvider().id;
    final layout = VideoLayoutMsg()
      ..supportsFullDesktopCapture = false
      ..primaryScreenId = 0;
    layout.videoTracks.add(VideoTrackLayout()
      ..mediaStreamId = streamId
      ..positionX = 0
      ..positionY = 0
      ..width = width
      ..height = height
      ..screenId = 0);
    _sendControl(encodeControlMessage(videoLayout: layout));
  }

  // ==================== event 通道（客户端输入） ====================

  void _setupEventChannel(RTCDataChannel channel) {
    _eventChannel = channel;
    channel.onMessage = (msg) {
      if (!msg.isBinary) return;
      try {
        final event = decodeEventMessage(msg.binary);
        _dispatchInput(event);
      } catch (e) {
        session._log('peer[$clientId] bad event message: $e');
      }
    };
  }

  void _dispatchInput(EventMessage event) {
    final sink = session.onInput;
    if (sink == null) return;

    if (event.mouseEvent != null) {
      final m = event.mouseEvent!;
      sink(HostInputEvent.mouse(
        x: m.x,
        y: m.y,
        button: m.button,
        buttonDown: m.buttonDown,
        wheelDeltaX: m.wheelDeltaX,
        wheelDeltaY: m.wheelDeltaY,
      ));
    }
    if (event.keyEvent != null) {
      final k = event.keyEvent!;
      sink(HostInputEvent.key(usbKeycode: k.usbKeycode, pressed: k.pressed));
    }
    if (event.textEventText != null) {
      sink(HostInputEvent.text(event.textEventText!));
    }
  }

  void _handleControlMessage(Uint8List data) {
    // client → host 的 control 消息（能力应答、剪贴板、分辨率等）。
    // M3 最小实现：仅解析，暂不处理。
    try {
      decodeControlMessage(data);
    } catch (_) {}
  }

  // ==================== 发送/工具 ====================

  void _sendControl(Uint8List data) {
    if (!_controlReady || _controlChannel == null) return;
    _controlChannel!.send(RTCDataChannelMessage.fromBinary(data));
  }

  void _sendLocalCandidate(RTCIceCandidate candidate) {
    session._sendToClient(
      clientId,
      _jingle.buildTransportInfo(IceCandidateInfo(
        candidate: candidate.candidate!,
        sdpMid: candidate.sdpMid ?? '',
        sdpMLineIndex: candidate.sdpMLineIndex ?? 0,
      )),
    );
  }

  void _flushLocalCandidates() {
    for (final c in _pendingLocalCandidates) {
      _sendLocalCandidate(c);
    }
    _pendingLocalCandidates.clear();
  }

  Future<void> _flushRemoteCandidates() async {
    for (final c in _pendingRemoteCandidates) {
      try {
        await _pc!.addCandidate(c);
      } catch (e) {
        session._log('peer[$clientId] flush candidate failed: $e');
      }
    }
    _pendingRemoteCandidates.clear();
  }

  /// HMAC-SHA256(auth_key, `"<type> " + NormalizedForSignature(sdp)`)
  String _signSdp(String sdp, String type) {
    final authKey = _auth?.authKey;
    if (authKey == null) return '';
    final normalized = sdp
        .split('\n')
        .map((l) => l.trim())
        .where((l) => l.isNotEmpty)
        .join('\n');
    final message = utf8.encode('$type $normalized\n');
    final mac = crypto.Hmac(crypto.sha256, authKey).convert(message);
    return base64Encode(Uint8List.fromList(mac.bytes));
  }

  bool _verifySignature(String sdp, String type, String? signature) {
    if (signature == null || signature.isEmpty) return false;
    return _signSdp(sdp, type) == signature;
  }

  Future<void> _fail(String reason) async {
    session._log('peer[$clientId] failed: $reason');
    state = HostPeerState.failed;
    await close(sendTerminate: true);
    session._onPeerClosed(clientId);
  }

  Future<void> close({required bool sendTerminate}) async {
    if (state == HostPeerState.closed) return;
    if (sendTerminate && _jingle.sessionId != null) {
      try {
        session._sendToClient(clientId, _jingle.buildSessionTerminate('success'));
      } catch (_) {}
    }
    state = HostPeerState.closed;
    _queue.clear();
    _processing = false;
    try {
      await _controlChannel?.close();
      await _eventChannel?.close();
    } catch (_) {}
    if (_pc != null) {
      await _pc!.close();
      _pc = null;
    }
  }
}
