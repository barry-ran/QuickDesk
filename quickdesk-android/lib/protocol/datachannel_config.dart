import 'package:flutter_webrtc/flutter_webrtc.dart';

/// 创建与 Chromium Remoting 一致的 in-band DataChannel 配置。
///
/// `flutter_webrtc` 的 Dart 接口默认会显式传入 `id=0` 和 `protocol="sctp"`，
/// 但原生 libwebrtc 的默认值分别是 `id=-1`（按 DTLS 角色自动分配 SID）和
/// 空应用子协议。Remoting 的两端会分别创建通道，必须让 libwebrtc 按奇偶规则
/// 分配 SID，避免 `control` 与 `event` 同时占用 stream 0。
RTCDataChannelInit createRemotingDataChannelInit() {
  return RTCDataChannelInit()
    ..ordered = true
    ..negotiated = false
    ..id = -1
    ..protocol = '';
}
