/// candidates.dart - ICE candidate 缓冲（client / host 共用）
///
/// Chromium remoting 的时序约束：
///   - 远端 candidate 必须在 setRemoteDescription 成功后才能 addCandidate，
///     之前到达的先缓冲（认证前信令就可能带来 candidate）；
///   - 本端 candidate 必须排在签名 SDP 之后发出（认证完成、offer/answer
///     已发出前产生的先缓冲），否则对端会视为异常时序。
library;

import 'package:flutter_webrtc/flutter_webrtc.dart';

import '../signaling/jingle.dart' show IceCandidateInfo;

/// 远端 candidate 接收器：remote description 就绪前缓冲，就绪后直通。
class RemoteCandidateSink {
  final RTCPeerConnection Function() _pc;
  final void Function(String message) _log;

  final List<RTCIceCandidate> _pending = [];
  bool _ready = false;

  RemoteCandidateSink(this._pc, this._log);

  Future<void> add(IceCandidateInfo info) async {
    final candidate =
        RTCIceCandidate(info.candidate, info.sdpMid, info.sdpMLineIndex);
    if (!_ready) {
      _pending.add(candidate);
      return;
    }
    try {
      await _pc().addCandidate(candidate);
    } catch (e) {
      _log('addCandidate failed: $e');
    }
  }

  /// setRemoteDescription 成功后调用：冲刷积压并进入直通模式。可重复调用
  /// （重协商再次 setRemote 时积压为空，无副作用）。
  Future<void> markReady() async {
    _ready = true;
    for (final c in _pending) {
      try {
        await _pc().addCandidate(c);
      } catch (e) {
        _log('flush candidate failed: $e');
      }
    }
    _pending.clear();
  }

  void reset() {
    _ready = false;
    _pending.clear();
  }
}

/// 本端 candidate 发送门：开门前缓冲，开门时冲刷积压，之后直通。
class LocalCandidateGate {
  final void Function(RTCIceCandidate candidate) _send;

  final List<RTCIceCandidate> _pending = [];
  bool _open = false;

  LocalCandidateGate(this._send);

  void add(RTCIceCandidate candidate) {
    if (candidate.candidate == null || candidate.candidate!.isEmpty) return;
    if (_open) {
      _send(candidate);
    } else {
      _pending.add(candidate);
    }
  }

  /// 签名 SDP 发出、setLocalDescription 完成后调用；返回冲刷的积压条数。
  int open() {
    _open = true;
    final flushed = _pending.length;
    for (final c in _pending) {
      _send(c);
    }
    _pending.clear();
    return flushed;
  }

  void reset() {
    _open = false;
    _pending.clear();
  }
}
