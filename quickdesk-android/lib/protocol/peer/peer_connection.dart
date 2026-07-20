/// peer_connection.dart - RTCPeerConnection 工厂
library;

import 'package:flutter_webrtc/flutter_webrtc.dart';

import '../../api/signaling_api.dart' show IceServerEntry;

/// 创建与 Chromium Remoting 对齐的 PeerConnection：
/// max-bundle（所有通道复用单一传输）+ unified-plan。
Future<RTCPeerConnection> createRemotingPeerConnection(
    List<IceServerEntry> iceServers) {
  return createPeerConnection({
    'iceServers': iceServers.map((e) => e.toRtcConfig()).toList(),
    'bundlePolicy': 'max-bundle',
    'sdpSemantics': 'unified-plan',
  });
}
