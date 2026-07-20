/// client_session.dart - 主控端会话状态机
///
/// 对照 WebClient/js/protocol/session.js：
/// WebSocket 信令 + Jingle 编解码 + SPAKE2 认证 + RTCPeerConnection。
library;

import 'dart:async';
import 'dart:convert';

import 'package:flutter_webrtc/flutter_webrtc.dart';

import '../api/signaling_api.dart';
import '../core/rand_id.dart';
import 'auth/spake2.dart' as spake2;
import 'auth/spake2_authenticator.dart';
import 'datachannel_config.dart';
import 'datachannel_handler.dart';
import 'peer/candidates.dart';
import 'peer/peer_connection.dart';
import 'peer/serial_queue.dart';
import 'signaling/jingle.dart';
import 'signaling/sdp_signature.dart';
import 'signaling/websocket_transport.dart';

enum SessionState {
  idle,
  connecting,
  initiating,
  accepting,
  authenticating,
  connected,
  closed,
  failed,
}

class ClientSession {
  final String signalingUrl;
  final List<IceServerEntry> iceServers;

  SessionState state = SessionState.idle;
  String? failureReason;

  String? deviceId;
  late String _clientId;

  WebSocketTransport? _transport;
  final JingleBuilder _jingleBuilder = JingleBuilder();
  final JingleParser _jingleParser = JingleParser();
  Spake2ClientAuthenticator? _authenticator;
  RTCPeerConnection? _pc;
  late final DataChannelHandler dcHandler = DataChannelHandler(onLog: _log);

  /// 远端 candidate：remote description 就绪前缓冲
  late final RemoteCandidateSink _remoteCandidates =
      RemoteCandidateSink(() => _pc!, _log);

  /// 本端 candidate：认证完成前缓冲（签名 SDP 必须先于 candidate 到达对端）
  late final LocalCandidateGate _localCandidates =
      LocalCandidateGate(_sendCandidateNow);

  RTCDataChannel? _eventChannel;

  // 消息串行处理队列（防止 async 处理交叉）
  late final SerialTaskQueue<String> _signalQueue = SerialTaskQueue(
    _processSignalingMessage,
    onError: (e) => _log('error processing signaling message: $e'),
  );

  /// 接收到的远端媒体流（按 stream id 索引，支持桌面端多显示器多流）
  final Map<String, MediaStream> remoteStreams = {};

  final _stateCtrl = StreamController<SessionState>.broadcast();
  final _trackCtrl = StreamController<MediaStreamTrack>.broadcast();
  final _streamCtrl = StreamController<void>.broadcast();
  final _logCtrl = StreamController<String>.broadcast();

  Stream<SessionState> get onStateChange => _stateCtrl.stream;
  Stream<MediaStreamTrack> get onTrack => _trackCtrl.stream;

  /// 远端流集合发生变化（新流到达）时触发
  Stream<void> get onRemoteStreamsChanged => _streamCtrl.stream;
  Stream<String> get onLog => _logCtrl.stream;

  ClientSession({required this.signalingUrl, this.iceServers = const []});

  /// 供文件传输创建独立 DataChannel 使用（连接后才非空）
  RTCPeerConnection? get peerConnection => _pc;

  /// 建立连接。
  /// [signalToken] 由 SignalingApi.verifyAccessCode 预先换取（一次性）。
  Future<void> connect(
      String deviceId, String accessCode, String signalToken) async {
    this.deviceId = deviceId;

    try {
      _setState(SessionState.connecting);

      // 1. 共享密钥哈希: HMAC(device_id, device_id + access_code)
      final sharedSecretHash =
          spake2.getSharedSecretHash(deviceId, deviceId + accessCode);

      // 2. JID：与 WebClient / Chromium Host 的 FTL resource 格式对齐
      final clientUuid = randomHex(12);
      _clientId = 'android_$clientUuid';
      final localJid = '$deviceId@quickdesk.local/chromoting_ftl_$_clientId';
      final remoteJid =
          '$deviceId@quickdesk.local/chromoting_ftl_quickdesk_host';

      _jingleBuilder.localJid = localJid;
      _jingleBuilder.remoteJid = remoteJid;

      // 3. SPAKE2 认证器（Alice）
      _authenticator =
          Spake2ClientAuthenticator(localJid, remoteJid, sharedSecretHash);

      // 4. 信令 WebSocket（首帧 auth）
      _transport = WebSocketTransport(
        signalingUrl: signalingUrl,
        onMessage: (msg, _) => _onSignalingMessage(msg),
        onClose: (code, reason) => _log('signaling closed: $code $reason'),
        onError: (e) => _log('signaling error: $e'),
      );
      await _transport!.connect(
        deviceId: deviceId,
        signalToken: signalToken,
        clientId: _clientId,
        role: 'client',
      );
      _log('signaling auth_ok');

      // 5. PeerConnection
      await _createPeerConnection();

      // 6. SDP Offer + session-initiate
      _setState(SessionState.initiating);
      await _createAndSendOffer();
      _setState(SessionState.accepting);
    } catch (e) {
      _log('connect failed: $e');
      _setState(SessionState.failed);
      rethrow;
    }
  }

  /// 拉取 WebRTC 统计（供性能面板使用）；未连接时返回空列表。
  Future<List<StatsReport>> getStats() async {
    final pc = _pc;
    if (pc == null) return const [];
    try {
      return await pc.getStats();
    } catch (_) {
      return const [];
    }
  }

  Future<void> disconnect() async {
    if (_transport != null &&
        _transport!.isConnected &&
        _jingleBuilder.sessionId != null) {
      try {
        _transport!.send(_jingleBuilder.buildSessionTerminate('success'));
      } catch (_) {}
    }
    await _cleanup();
    _setState(SessionState.closed);
  }

  // ==================== 内部 ====================

  Future<void> _createPeerConnection() async {
    _pc = await createRemotingPeerConnection(iceServers);
    _log('PeerConnection created');

    _pc!.onIceCandidate = _localCandidates.add;

    _pc!.onIceConnectionState = (iceState) {
      _log('ICE state: $iceState');
      if (iceState == RTCIceConnectionState.RTCIceConnectionStateConnected ||
          iceState == RTCIceConnectionState.RTCIceConnectionStateCompleted) {
        if (state != SessionState.connected) {
          _setState(SessionState.connected);
        }
      } else if (iceState ==
          RTCIceConnectionState.RTCIceConnectionStateFailed) {
        _setState(SessionState.failed);
      }
    };

    _pc!.onConnectionState = (connectionState) {
      _log('PeerConnection state: $connectionState');
    };

    _pc!.onIceGatheringState = (gatheringState) {
      _log('ICE gathering state: $gatheringState');
    };

    _pc!.onSignalingState = (signalingState) {
      _log('signaling state: $signalingState');
    };

    _pc!.onTrack = (event) async {
      _log(
          'track received: ${event.track.kind}, streams=${event.streams.length}');
      if (event.streams.isNotEmpty) {
        for (final stream in event.streams) {
          remoteStreams[stream.id] = stream;
        }
      } else if (event.track.kind == 'video') {
        // 兜底：track 未关联 stream 时自建一个
        final s = await createLocalMediaStream('remote_${event.track.id}');
        await s.addTrack(event.track);
        remoteStreams[s.id] = s;
      }
      _streamCtrl.add(null);
      _trackCtrl.add(event.track);
    };

    _pc!.onDataChannel = (channel) {
      _log(
        'datachannel received: ${channel.label}, '
        'sid=${channel.id}, state=${channel.state}',
      );
      dcHandler.handleDataChannel(channel);
    };
  }

  Future<void> _createAndSendOffer() async {
    await _pc!.addTransceiver(
      kind: RTCRtpMediaType.RTCRtpMediaTypeVideo,
      init: RTCRtpTransceiverInit(direction: TransceiverDirection.RecvOnly),
    );
    await _pc!.addTransceiver(
      kind: RTCRtpMediaType.RTCRtpMediaTypeAudio,
      init: RTCRtpTransceiverInit(direction: TransceiverDirection.RecvOnly),
    );

    final offer = await _pc!.createOffer();
    await _pc!.setLocalDescription(offer);

    final authMessage = _authenticator!.getFirstNegotiationMessage();
    final xml = _jingleBuilder.buildSessionInitiate(offer.sdp!, authMessage);
    _log('sending session-initiate');
    _transport!.send(xml);
  }

  void _sendIqResult(String iqId, String toJid) {
    if (_transport == null || !_transport!.isConnected) return;
    _transport!.send(_jingleBuilder.buildIqResult(iqId, toJid));
  }

  void _sendCandidateNow(RTCIceCandidate candidate) {
    if (_transport == null || !_transport!.isConnected) return;
    _transport!.send(_jingleBuilder.buildTransportInfo(IceCandidateInfo(
      candidate: candidate.candidate!,
      sdpMid: candidate.sdpMid ?? '',
      sdpMLineIndex: candidate.sdpMLineIndex ?? 0,
    )));
  }

  void _onSignalingMessage(String message) => _signalQueue.add(message);

  Future<void> _processSignalingMessage(String message) async {
    final trimmed = message.trim();

    if (trimmed.startsWith('{')) {
      try {
        final json = jsonDecode(trimmed) as Map<String, dynamic>;
        _handleJsonMessage(json);
        return;
      } catch (_) {
        // 不是 JSON，继续按 XML 处理
      }
    }

    if (trimmed.startsWith('<')) {
      final parsed = _jingleParser.parse(trimmed);
      if (parsed != null) {
        await _handleJingleMessage(parsed);
      }
    }
  }

  void _handleJsonMessage(Map<String, dynamic> json) {
    if (json['type'] != 'error') return;
    final code = (json['code'] ??
            (json['data'] is Map ? json['data']['code'] : '') ??
            '')
        .toString();
    _log('signaling error: $code');
    if (code == 'HOST_OFFLINE' ||
        code == 'PEER_DISCONNECTED' ||
        code == 'TOKEN_INVALID' ||
        code == 'AUTH_INVALID') {
      failureReason = code;
      _setState(SessionState.failed);
    }
  }

  Future<void> _handleJingleMessage(JingleMessage message) async {
    // 过滤不属于本会话的消息（多客户端并发场景）
    if (message.sid.isNotEmpty &&
        _jingleBuilder.sessionId != null &&
        message.sid != _jingleBuilder.sessionId) {
      if (message.iqType == 'set' &&
          message.iqId.isNotEmpty &&
          message.from.isNotEmpty) {
        _sendIqResult(message.iqId, message.from);
      }
      return;
    }

    // Chromium 的 Jingle 实现先确认当前 IQ，再发送认证或 transport 应答。
    if (message.iqType == 'set' &&
        message.iqId.isNotEmpty &&
        message.from.isNotEmpty) {
      _sendIqResult(message.iqId, message.from);
    }

    switch (message.action) {
      case 'session-accept':
        await _handleSessionAccept(message);
        break;
      case 'transport-info':
        await _handleTransportInfo(message);
        break;
      case 'session-info':
        await _handleSessionInfo(message);
        break;
      case 'session-terminate':
        _log('session terminated by host: '
            '${message.terminateInfo?.describe() ?? 'unknown'}');
        await _cleanup();
        _setState(SessionState.closed);
        break;
      case '_iq_response':
        break;
      default:
        _log('unhandled jingle action: ${message.action}');
    }
  }

  Future<void> _handleSessionAccept(JingleMessage message) async {
    _log('processing session-accept');

    if (message.authMessage != null) {
      _setState(SessionState.authenticating);
      final st = driveAuthExchange(_authenticator!, message.authMessage!,
          (next) {
        _transport!.send(_jingleBuilder.buildSessionInfo(next));
        _log('sent auth message (session-info)');
      });
      if (st == AuthState.rejected) {
        _log('auth rejected: ${_authenticator!.rejectionReason}');
        _setState(SessionState.failed);
        return;
      }
    }

    if (message.sdp != null) {
      try {
        await _pc!.setRemoteDescription(
            RTCSessionDescription(message.sdp!.sdp, message.sdp!.type));
        _log('remote SDP set');
        await _remoteCandidates.markReady();
      } catch (e) {
        _log('failed to set remote SDP: $e');
        _setState(SessionState.failed);
      }
    }
  }

  Future<void> _handleTransportInfo(JingleMessage message) async {
    if (message.sdp != null) {
      final sdpType = message.sdp!.type;
      try {
        if (sdpType == 'offer') {
          // 重协商：host 会重新发 offer（如开始推视频流）
          final signalingState = await _pc!.getSignalingState();
          if (signalingState ==
              RTCSignalingState.RTCSignalingStateHaveLocalOffer) {
            await _pc!
                .setLocalDescription(RTCSessionDescription('', 'rollback'));
          }
          await _pc!.setRemoteDescription(
              RTCSessionDescription(message.sdp!.sdp, 'offer'));

          final answer = await _pc!.createAnswer();
          await _pc!.setLocalDescription(answer);

          final authKey = _authenticator?.authKey;
          if (authKey == null) {
            throw StateError(
                'authentication key unavailable for SDP signature');
          }
          final signature = signSdp(authKey, answer.sdp!, 'answer');
          _transport!.send(_jingleBuilder.buildTransportInfoSdp(
              answer.sdp!, 'answer',
              signature: signature));
          _log('sent SDP answer via transport-info');

          // Host 要求 client 创建 "event" DataChannel（键鼠输入）
          if (_eventChannel == null) {
            _eventChannel = await _pc!.createDataChannel(
              'event',
              createRemotingDataChannelInit(),
            );
            dcHandler.handleDataChannel(_eventChannel!);
            _log(
              'created outgoing "event" DataChannel: '
              'sid=${_eventChannel!.id}',
            );
          }
        } else {
          await _pc!.setRemoteDescription(
              RTCSessionDescription(message.sdp!.sdp, sdpType));
        }
        await _remoteCandidates.markReady();
      } catch (e) {
        _log('failed to handle SDP from transport-info: $e');
      }
    }

    for (final info in message.iceCandidates) {
      await _remoteCandidates.add(info);
    }
  }

  Future<void> _handleSessionInfo(JingleMessage message) async {
    if (message.authMessage == null) return;

    final st = driveAuthExchange(_authenticator!, message.authMessage!,
        (next) => _transport!.send(_jingleBuilder.buildSessionInfo(next)));

    if (st == AuthState.rejected) {
      _log('auth rejected: ${_authenticator!.rejectionReason}');
      _setState(SessionState.failed);
      return;
    }

    if (st == AuthState.accepted) {
      _log('authentication successful');
      final flushed = _localCandidates.open();
      _log('flushed $flushed buffered candidates');
    }
  }

  Future<void> _cleanup() async {
    _signalQueue.clear();
    _eventChannel = null;
    if (_pc != null) {
      await _pc!.close();
      _pc = null;
    }
    _transport?.disconnect();
    _transport = null;
    _remoteCandidates.reset();
    _localCandidates.reset();
  }

  void _setState(SessionState newState) {
    state = newState;
    _stateCtrl.add(newState);
  }

  void _log(String message) {
    // ignore: avoid_print
    print('[ClientSession] $message');
    _logCtrl.add(message);
  }

  void dispose() {
    _cleanup();
    dcHandler.dispose();
    _stateCtrl.close();
    _trackCtrl.close();
    _streamCtrl.close();
    _logCtrl.close();
  }
}
