/// datachannel_handler.dart - control/event 双通道管理（client / host 共用）
///
/// 对照 WebClient/js/protocol/datachannel-handler.js 与 Chromium remoting
/// 的通道分工：
///   - client 角色：本端创建 'event'（键鼠输入），对端(host)创建 'control'
///   - host 角色：本端创建 'control'（能力/剪贴板/VideoLayout），对端创建 'event'
///
/// 两个角色都通过 [handleDataChannel] 接管通道（出站或入站均可）。绑定回调后
/// 会立即检查通道当前状态——入站通道对象可能已携带 OPEN 状态，出站通道在
/// 极快网络下也可能在回调绑定前完成打开。
library;

import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_webrtc/flutter_webrtc.dart';

import 'proto/protobuf_messages.dart';

class DataChannelHandler {
  /// control 通道每次进入打开状态（host 借此下发初始 capabilities/VideoLayout，
  /// 并检查双通道就绪）。同步回调，先于 [onControlReady] 流事件。
  final void Function()? onControlOpen;

  /// event 通道每次进入打开状态
  final void Function()? onEventOpen;

  /// 通道进入 Closing/Closed（label 为 'control'/'event'）。
  /// host 据此判定会话失效；不关心（client 角色）则不传。
  final void Function(String label)? onChannelClosed;

  /// 收到对端输入事件（host 角色消费）。不传时 event 通道入站消息被忽略。
  final void Function(EventMessage event)? onEventMessage;

  final void Function(String message)? onLog;

  DataChannelHandler({
    this.onControlOpen,
    this.onEventOpen,
    this.onChannelClosed,
    this.onEventMessage,
    this.onLog,
  });

  RTCDataChannel? _controlChannel;
  RTCDataChannel? _eventChannel;
  bool _controlReady = false;
  bool _eventReady = false;
  bool _disposed = false;

  bool get controlReady => _controlReady;
  bool get eventReady => _eventReady;

  final _controlReadyCtrl = StreamController<void>.broadcast();
  final _clipboardCtrl = StreamController<ClipboardEventMsg>.broadcast();
  final _capabilitiesCtrl = StreamController<String>.broadcast();
  final _videoLayoutCtrl = StreamController<VideoLayoutMsg>.broadcast();

  Stream<void> get onControlReady => _controlReadyCtrl.stream;
  Stream<ClipboardEventMsg> get onClipboard => _clipboardCtrl.stream;
  Stream<String> get onCapabilities => _capabilitiesCtrl.stream;
  Stream<VideoLayoutMsg> get onVideoLayout => _videoLayoutCtrl.stream;

  /// 接管一个 DataChannel（本端创建的出站通道，或对端创建的入站通道）
  void handleDataChannel(RTCDataChannel channel) {
    switch (channel.label) {
      case 'control':
        _setupControlChannel(channel);
        break;
      case 'event':
        _setupEventChannel(channel);
        break;
      default:
        break;
    }
  }

  void _setupControlChannel(RTCDataChannel channel) {
    _controlChannel = channel;
    channel.onDataChannelState = _handleControlState;
    channel.onMessage = (msg) {
      if (_disposed || !msg.isBinary) return;
      try {
        _handleControlMessage(decodeControlMessage(msg.binary));
      } catch (_) {
        // 忽略无法解析的消息
      }
    };
    _handleControlState(channel.state);
  }

  void _setupEventChannel(RTCDataChannel channel) {
    _eventChannel = channel;
    channel.onDataChannelState = _handleEventState;
    channel.onMessage = (msg) {
      if (_disposed || onEventMessage == null || !msg.isBinary) return;
      try {
        onEventMessage!.call(decodeEventMessage(msg.binary));
      } catch (e) {
        onLog?.call('bad event message: $e');
      }
    };
    _handleEventState(channel.state);
  }

  void _handleControlState(RTCDataChannelState? channelState) {
    if (_disposed) return;
    onLog?.call(
        'control datachannel: $channelState (sid=${_controlChannel?.id})');
    final wasReady = _controlReady;
    _controlReady = channelState == RTCDataChannelState.RTCDataChannelOpen;
    if (_controlReady && !wasReady) {
      onControlOpen?.call();
      _controlReadyCtrl.add(null);
    } else if (channelState == RTCDataChannelState.RTCDataChannelClosing ||
        channelState == RTCDataChannelState.RTCDataChannelClosed) {
      onChannelClosed?.call('control');
    }
  }

  void _handleEventState(RTCDataChannelState? channelState) {
    if (_disposed) return;
    onLog?.call('event datachannel: $channelState (sid=${_eventChannel?.id})');
    final wasReady = _eventReady;
    _eventReady = channelState == RTCDataChannelState.RTCDataChannelOpen;
    if (_eventReady && !wasReady) {
      onEventOpen?.call();
    } else if (channelState == RTCDataChannelState.RTCDataChannelClosing ||
        channelState == RTCDataChannelState.RTCDataChannelClosed) {
      onChannelClosed?.call('event');
    }
  }

  void _handleControlMessage(ControlMessage message) {
    if (message.clipboardEvent != null) {
      _clipboardCtrl.add(message.clipboardEvent!);
    }
    if (message.capabilities != null) {
      _capabilitiesCtrl.add(message.capabilities!);
    }
    if (message.videoLayout != null) {
      _videoLayoutCtrl.add(message.videoLayout!);
    }
  }

  // ==================== 发送 ====================

  void _sendEvent(Uint8List data) {
    if (_disposed || !_eventReady || _eventChannel == null) return;
    _eventChannel!.send(RTCDataChannelMessage.fromBinary(data));
  }

  void _sendControl(Uint8List data) {
    if (_disposed || !_controlReady || _controlChannel == null) return;
    _controlChannel!.send(RTCDataChannelMessage.fromBinary(data));
  }

  void sendMouseEvent(MouseEventMsg mouseEvent) {
    _sendEvent(encodeEventMessage(
      timestamp: DateTime.now().millisecondsSinceEpoch,
      mouseEvent: mouseEvent,
    ));
  }

  void sendKeyEvent(KeyEventMsg keyEvent) {
    _sendEvent(encodeEventMessage(
      timestamp: DateTime.now().millisecondsSinceEpoch,
      keyEvent: keyEvent,
    ));
  }

  /// 文本注入（IME 输入，如中文）
  void sendTextEvent(String text) {
    _sendEvent(encodeEventMessage(
      timestamp: DateTime.now().millisecondsSinceEpoch,
      textEventText: text,
    ));
  }

  void sendClipboard(String mimeType, Uint8List data) {
    _sendControl(encodeControlMessage(
      clipboardEvent: ClipboardEventMsg(mimeType: mimeType, data: data),
    ));
  }

  void sendCapabilities(String capabilities) {
    _sendControl(encodeControlMessage(capabilities: capabilities));
  }

  void sendAudioControl({required bool enable}) {
    _sendControl(encodeControlMessage(audioControlEnable: enable));
  }

  void sendVideoLayout(VideoLayoutMsg layout) {
    _sendControl(encodeControlMessage(videoLayout: layout));
  }

  Future<void> dispose() async {
    if (_disposed) return;
    _disposed = true;
    try {
      await _controlChannel?.close();
    } catch (_) {}
    try {
      await _eventChannel?.close();
    } catch (_) {}
    _controlChannel = null;
    _eventChannel = null;
    _controlReady = false;
    _eventReady = false;
    await _controlReadyCtrl.close();
    await _clipboardCtrl.close();
    await _capabilitiesCtrl.close();
    await _videoLayoutCtrl.close();
  }
}
