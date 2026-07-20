/// host_session.dart - 被控端会话状态机（Host / responder 角色）
///
/// 与 client_session.dart 互为镜像，负责被桌面端/其它客户端控制时的协议处理。
/// 与 Chromium Remoting Host 一致，认证与 WebRTC 传输分阶段进行：
///
///   Chromium 标准流程：
///   1. client → session-initiate（仅 auth supported-methods + 空 WebRTC transport）
///   2. Host → session-accept（仅 method + 自己的 SPAKE2 消息 + 空 transport）
///   3. session-info 往返完成 SPAKE2
///   4. 认证完成后 Host 才创建 PeerConnection，加屏幕轨和 'control' DataChannel
///      → createOffer → 用 auth_key 签名 → transport-info(offer)
///   5. client → transport-info(answer, 带签名)，Host 校验后 setRemote
///   6. 双向 transport-info 交换 ICE candidate
///   7. client 创建 'event' DataChannel，Host 收到后把输入事件交给注入层
///
/// 为兼容旧版 Android/Web 客户端，如果 session-initiate 仍携带初始 offer，
/// Host 会先返回基础 answer，再在认证后按上述步骤发起媒体重协商。
///
/// 一个 HostSession 复用一条 host 信令 WS，可服务多个并发客户端
/// （按 signaling client_id 区分），共享同一路屏幕采集 MediaStream。
library;

import 'dart:async';
import 'dart:convert';
import 'dart:typed_data';

import 'package:flutter_webrtc/flutter_webrtc.dart';

import '../api/signaling_api.dart' show IceServerEntry;
import 'auth/spake2_authenticator.dart';
import 'datachannel_config.dart';
import 'datachannel_handler.dart';
import 'host_input.dart';
import 'peer/candidates.dart';
import 'peer/peer_connection.dart';
import 'peer/serial_queue.dart';
import 'proto/protobuf_messages.dart';
import 'signaling/jingle.dart';
import 'signaling/sdp_signature.dart';
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

  /// Chromium Client 要求 Host 第一条 SPAKE2 消息携带 DER X.509 证书（base64）。
  final String hostCertificate;

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

  int get peerCount =>
      _peers.values.where((p) => p.state == HostPeerState.connected).length;

  HostSession({
    required this.signalingUrl,
    required this.deviceId,
    required this.sharedSecretHash,
    required this.hostCertificate,
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

  /// control（本端创建）/ event（对端创建）双通道
  late final DataChannelHandler _channels = DataChannelHandler(
    onControlOpen: _onControlOpen,
    onEventOpen: _notifyIfChannelsReady,
    onChannelClosed: _onChannelClosed,
    onEventMessage: _dispatchInput,
    onLog: (m) => session._log('peer[$clientId] $m'),
  );

  /// 远端 candidate：answer 未 setRemote 前缓冲（认证前信令就可能带来 ICE）
  late final RemoteCandidateSink _remoteCandidates = RemoteCandidateSink(
    () => _pc!,
    (m) => session._log('peer[$clientId] $m'),
  );

  /// 本端 candidate：签名 offer 发出且 setLocal 完成前缓冲。
  /// Chromium 会把认证后的 transport-info 直接交给 WebrtcTransport，
  /// 必须保证签名 offer 先到，candidate 不能抢在 offer 前面。
  late final LocalCandidateGate _localCandidates =
      LocalCandidateGate(_sendLocalCandidate);

  // 逐条串行处理信令
  late final SerialTaskQueue<String> _queue = SerialTaskQueue(
    _process,
    onError: (e) => session._log('peer[$clientId] error: $e'),
  );

  bool _initialControlMessagesSent = false;
  bool _closing = false;

  HostPeerState state = HostPeerState.negotiating;
  bool _mediaNegotiationStarted = false;
  bool _authenticated = false;

  _HostPeer({required this.session, required this.clientId});

  void enqueue(String xml) => _queue.add(xml);

  Future<void> _process(String xml) async {
    final parsed = _parser.parse(xml);
    if (parsed != null) await _handle(parsed);
  }

  Future<void> _handle(JingleMessage message) async {
    // 过滤不属于本会话的消息
    if (message.sid.isNotEmpty &&
        _jingle.sessionId != null &&
        message.sid != _jingle.sessionId) {
      if (message.iqType == 'set' &&
          message.iqId.isNotEmpty &&
          message.from.isNotEmpty) {
        session._sendToClient(
            clientId, _jingle.buildIqResult(message.iqId, message.from));
      }
      return;
    }

    // Chromium 要求先回 IQ result，再发送由该请求触发的 session-accept、
    // session-info 或 transport-info；否则桌面端的 IQ 请求队列可能将后续消息
    // 判为异常时序。这里与 JingleSessionManager::SendReply 保持一致。
    if (message.iqType == 'set' &&
        message.iqId.isNotEmpty &&
        message.from.isNotEmpty) {
      session._sendToClient(
          clientId, _jingle.buildIqResult(message.iqId, message.from));
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
        session._log('peer[$clientId] terminated by client: '
            '${message.terminateInfo?.describe() ?? 'unknown'}');
        await close(sendTerminate: false);
        session._onPeerClosed(clientId);
        break;
      case '_iq_response':
        break;
      default:
        session._log('peer[$clientId] unhandled action: ${message.action}');
    }
  }

  Future<void> _handleSessionInitiate(JingleMessage message) async {
    session._log('peer[$clientId] session-initiate (sid=${message.sid})');

    // Chromium 将 SPAKE2 的身份绑定到实际发送 IQ 的 `from` 地址。
    // `initiator` 只是 Jingle 元数据，不能覆盖信令层地址；否则动态 client UUID
    // 不一致时双方会派生不同的 auth_key。
    final peerJid = message.from;
    if (peerJid.isEmpty) {
      await _fail('session-initiate missing sender JID');
      return;
    }
    if (message.initiator.isNotEmpty && message.initiator != peerJid) {
      session._log(
        'peer[$clientId] initiator/from mismatch: '
        '${message.initiator} != $peerJid; binding auth to from',
      );
    }

    _jingle.sessionId = message.sid;
    _jingle.localJid = session._hostJid;
    _jingle.remoteJid = peerJid;

    // SPAKE2 Bob：local=hostJid, remote=clientJid
    _auth = Spake2HostAuthenticator(
      _jingle.localJid,
      _jingle.remoteJid,
      session.sharedSecretHash,
      certificate: session.hostCertificate,
    );

    // Chromium NegotiatingHostAuthenticator 同时支持客户端直接指定 method，
    // 或通过 supported-methods 让 Host 选择共同方法。
    final clientAuth = message.authMessage;
    if (clientAuth == null ||
        (clientAuth.supportedMethods == null && clientAuth.method == null)) {
      session._log('peer[$clientId] session-initiate missing auth method');
      await _fail('no auth method in session-initiate');
      return;
    }
    _auth!.processMessage(clientAuth);
    if (_auth!.state == AuthState.rejected) {
      await _fail('auth method rejected: ${_auth!.rejectionReason}');
      return;
    }

    state = HostPeerState.authenticating;
    final authMsg = _auth!.getNextMessage();
    final initialSdp = message.sdp;

    if (initialSdp == null || initialSdp.sdp.isEmpty) {
      // Chromium 标准流程：session-initiate/session-accept 只协商认证，
      // WebRTC 在认证完成后才启动。transport 元素仍需存在，但内容为空。
      session._sendToClient(
        clientId,
        _jingle.buildSessionAccept(null, authMsg),
      );
      session._log('peer[$clientId] sent auth-only session-accept');
      return;
    }

    if (initialSdp.type != 'offer') {
      await _fail('session-initiate SDP is not an offer');
      return;
    }

    // 兼容旧版 Android/Web 客户端：它们在 session-initiate 中就携带
    // 基础 offer。先应答该 offer，认证后再由 Host 发起带视频的重协商。
    await _createPeerConnection();
    try {
      await _pc!.setRemoteDescription(
        RTCSessionDescription(initialSdp.sdp, 'offer'),
      );
      final answer = await _pc!.createAnswer();
      await _pc!.setLocalDescription(answer);

      session._sendToClient(
        clientId,
        _jingle.buildSessionAccept(answer.sdp, authMsg),
      );
      session
          ._log('peer[$clientId] sent legacy session-accept (answer + auth)');

      await _remoteCandidates.markReady();
    } catch (e) {
      await _fail('base negotiation failed: $e');
    }
  }

  Future<void> _handleSessionInfo(JingleMessage message) async {
    if (message.authMessage == null || _auth == null) return;

    final st = driveAuthExchange(
      _auth!,
      message.authMessage!,
      (next) =>
          session._sendToClient(clientId, _jingle.buildSessionInfo(next)),
    );

    if (st == AuthState.rejected) {
      await _fail('auth rejected: ${_auth!.rejectionReason}');
      return;
    }

    if (st == AuthState.accepted && !_authenticated) {
      session._log('peer[$clientId] authenticated');
      _authenticated = true;
      // 认证完成 → host 作为 offerer 发起媒体协商
      await _startMediaNegotiation();
    }
  }

  Future<void> _handleTransportInfo(JingleMessage message) async {
    if (_pc == null) {
      // Chromium 标准流程中，PeerConnection 应在认证成功时由 Host 创建。
      // 认证完成前到达的 ICE 先缓存；异常提前到达的 SDP 则拒绝，避免空引用
      // 被外层队列吞掉后只表现为模糊的“通道错误”。
      for (final info in message.iceCandidates) {
        await _remoteCandidates.add(info);
      }
      if (message.sdp != null) {
        await _fail('transport SDP arrived before authentication');
      }
      return;
    }

    // client 的 answer（对 host offer 的应答）
    if (message.sdp != null && message.sdp!.type == 'answer') {
      if (!_verifySignature(
          message.sdp!.sdp, 'answer', message.sdp!.signature)) {
        await _fail('answer signature mismatch');
        return;
      }
      try {
        await _pc!.setRemoteDescription(
          RTCSessionDescription(message.sdp!.sdp, 'answer'),
        );
        session._log('peer[$clientId] remote answer set');
        await _remoteCandidates.markReady();
      } catch (e) {
        await _fail('setRemoteDescription failed: $e');
        return;
      }
    } else if (message.sdp != null) {
      await _fail('unexpected transport SDP type: ${message.sdp!.type}');
      return;
    }

    for (final info in message.iceCandidates) {
      await _remoteCandidates.add(info);
    }
  }

  Future<void> _createPeerConnection() async {
    if (_pc != null) return;

    _pc = await createRemotingPeerConnection(session.iceServers);

    _pc!.onIceCandidate = _localCandidates.add;

    _pc!.onIceConnectionState = (s) {
      session._log('peer[$clientId] ICE: $s');
      if (s == RTCIceConnectionState.RTCIceConnectionStateConnected ||
          s == RTCIceConnectionState.RTCIceConnectionStateCompleted) {
        session._log(
          'peer[$clientId] ICE connected; waiting for control/event channels',
        );
      } else if (s == RTCIceConnectionState.RTCIceConnectionStateFailed) {
        unawaited(_fail('ICE failed'));
      }
    };

    _pc!.onConnectionState = (s) {
      session._log('peer[$clientId] connection: $s');
    };

    _pc!.onIceGatheringState = (s) {
      session._log('peer[$clientId] ICE gathering: $s');
    };

    _pc!.onSignalingState = (s) {
      session._log('peer[$clientId] signaling: $s');
    };

    // client 会创建 'event' DataChannel（输入事件）
    _pc!.onDataChannel = (channel) {
      session._log(
        'peer[$clientId] incoming datachannel: ${channel.label} '
        '(sid=${channel.id}, state=${channel.state})',
      );
      if (channel.label == 'event') {
        _channels.handleDataChannel(channel);
      }
    };
  }

  /// 认证完成后：创建连接、加视频轨和 control 通道，再生成签名 offer。
  Future<void> _startMediaNegotiation() async {
    if (_mediaNegotiationStarted) return;
    _mediaNegotiationStarted = true;

    try {
      await _createPeerConnection();

      // 加入屏幕视频轨（sendonly）
      final stream = session.screenStreamProvider();
      for (final track in stream.getVideoTracks()) {
        session._log(
          'peer[$clientId] adding screen track: id=${track.id}, '
          'enabled=${track.enabled}, muted=${track.muted}',
        );
        track.onMute = () {
          session._log('peer[$clientId] screen track muted: ${track.id}');
        };
        track.onUnMute = () {
          session._log('peer[$clientId] screen track unmuted: ${track.id}');
        };
        track.onEnded = () {
          session._log('peer[$clientId] screen track ended: ${track.id}');
        };
        await _pc!.addTrack(track, stream);
      }

      // host 创建 'control' 通道（下发 cursor/clipboard/capabilities/VideoLayout）
      final controlChannel = await _pc!.createDataChannel(
        'control',
        createRemotingDataChannelInit(),
      );
      session._log(
        'peer[$clientId] created control datachannel: '
        'sid=${controlChannel.id}',
      );
      // handleDataChannel 绑定回调后会立即检查当前状态，覆盖出站通道在
      // 极快网络下已完成打开的情况。
      _channels.handleDataChannel(controlChannel);

      final offer = await _pc!.createOffer();
      final sdp = offer.sdp;
      if (sdp == null || sdp.isEmpty) {
        throw StateError('createOffer returned empty SDP');
      }
      final signature = signSdp(_auth!.authKey!, sdp, 'offer');
      if (signature.isEmpty) {
        throw StateError('authentication key unavailable for SDP signature');
      }

      // Chromium 先通过 transport-info 发送已签名 SDP，再调用
      // SetLocalDescription。保持相同时序，确保 ICE candidate 永远排在 offer 后。
      session._sendToClient(
        clientId,
        _jingle.buildTransportInfoSdp(sdp, 'offer', signature: signature),
      );
      session._log('peer[$clientId] sent signed offer (transport-info)');

      await _pc!.setLocalDescription(offer);
      _localCandidates.open();
    } catch (e) {
      await _fail('media negotiation failed: $e');
    }
  }

  // ==================== 通道回调 ====================

  void _onControlOpen() {
    if (!_initialControlMessagesSent) {
      _initialControlMessagesSent = true;
      if (session.capabilities.isNotEmpty) {
        _channels.sendCapabilities(session.capabilities);
      }
      sendVideoLayout(session.screenWidth, session.screenHeight);
    }
    _notifyIfChannelsReady();
  }

  void _onChannelClosed(String label) {
    if (_closing) return;
    unawaited(_fail('$label datachannel closed'));
  }

  void _notifyIfChannelsReady() {
    if (!_channels.controlReady || !_channels.eventReady || _closing) return;
    if (state == HostPeerState.connected) return;

    state = HostPeerState.connected;
    session._log(
      'peer[$clientId] connected: control/event datachannels are open',
    );
    session._onPeerConnected();
  }

  void sendVideoLayout(int width, int height) {
    if (!_channels.controlReady) return;
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
    _channels.sendVideoLayout(layout);
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

  // ==================== 发送/工具 ====================

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

  bool _verifySignature(String sdp, String type, String? signature) {
    final authKey = _auth?.authKey;
    if (authKey == null) return false;
    return verifySdpSignature(authKey, sdp, type, signature);
  }

  Future<void> _fail(String reason) async {
    if (_closing || state == HostPeerState.closed) return;
    session._log('peer[$clientId] failed: $reason');
    state = HostPeerState.failed;
    // 失败用 general-error，客户端（含 Chromium 端）才会按错误而非正常断开处理
    await close(sendTerminate: true, terminateReason: 'general-error');
    session._onPeerClosed(clientId);
  }

  Future<void> close({
    required bool sendTerminate,
    String terminateReason = 'success',
  }) async {
    if (_closing || state == HostPeerState.closed) return;
    _closing = true;
    if (sendTerminate && _jingle.sessionId != null) {
      try {
        session._sendToClient(
            clientId, _jingle.buildSessionTerminate(terminateReason));
      } catch (_) {}
    }
    _queue.clear();
    await _channels.dispose();
    if (_pc != null) {
      await _pc!.close();
      _pc = null;
    }
    _remoteCandidates.reset();
    _localCandidates.reset();
    state = HostPeerState.closed;
  }
}
