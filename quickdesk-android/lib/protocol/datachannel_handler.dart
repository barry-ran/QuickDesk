/// datachannel_handler.dart - DataChannel 处理器
///
/// 对照 WebClient/js/protocol/datachannel-handler.js（M1 范围：
/// control/event 双通道 + 光标/剪贴板/能力/VideoLayout 事件；
/// 文件传输与 actions 通道留待 M2）。
library;

import 'dart:async';
import 'dart:typed_data';

import 'package:flutter_webrtc/flutter_webrtc.dart';

import 'proto/protobuf_messages.dart';

class DataChannelHandler {
  RTCDataChannel? _controlChannel;
  RTCDataChannel? _eventChannel;
  bool _controlReady = false;
  bool _eventReady = false;

  String hostCapabilities = '';

  final _controlReadyCtrl = StreamController<void>.broadcast();
  final _eventReadyCtrl = StreamController<void>.broadcast();
  final _cursorShapeCtrl = StreamController<CursorShapeInfo>.broadcast();
  final _clipboardCtrl = StreamController<ClipboardEventMsg>.broadcast();
  final _capabilitiesCtrl = StreamController<String>.broadcast();
  final _videoLayoutCtrl = StreamController<VideoLayoutMsg>.broadcast();

  Stream<void> get onControlReady => _controlReadyCtrl.stream;
  Stream<void> get onEventReady => _eventReadyCtrl.stream;
  Stream<CursorShapeInfo> get onCursorShape => _cursorShapeCtrl.stream;
  Stream<ClipboardEventMsg> get onClipboard => _clipboardCtrl.stream;
  Stream<String> get onCapabilities => _capabilitiesCtrl.stream;
  Stream<VideoLayoutMsg> get onVideoLayout => _videoLayoutCtrl.stream;

  bool get controlReady => _controlReady;
  bool get eventReady => _eventReady;

  /// 接管一个新的 DataChannel（Host 创建的入站通道，或本端创建的出站通道）
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

    channel.onDataChannelState = (state) {
      if (state == RTCDataChannelState.RTCDataChannelOpen) {
        _controlReady = true;
        _controlReadyCtrl.add(null);
      } else if (state == RTCDataChannelState.RTCDataChannelClosed) {
        _controlReady = false;
      }
    };

    channel.onMessage = (msg) {
      if (!msg.isBinary) return;
      try {
        final message = decodeControlMessage(msg.binary);
        _handleControlMessage(message);
      } catch (_) {
        // 忽略无法解析的消息
      }
    };
  }

  void _setupEventChannel(RTCDataChannel channel) {
    _eventChannel = channel;

    channel.onDataChannelState = (state) {
      if (state == RTCDataChannelState.RTCDataChannelOpen) {
        _eventReady = true;
        _eventReadyCtrl.add(null);
      } else if (state == RTCDataChannelState.RTCDataChannelClosed) {
        _eventReady = false;
      }
    };

    // Client 角色下 host 基本不会往 event 通道发消息
    channel.onMessage = (_) {};
  }

  void _handleControlMessage(ControlMessage message) {
    if (message.cursorShape != null) {
      _cursorShapeCtrl.add(message.cursorShape!);
    }
    if (message.clipboardEvent != null) {
      _clipboardCtrl.add(message.clipboardEvent!);
    }
    if (message.capabilities != null) {
      hostCapabilities = message.capabilities!.capabilities;
      _capabilitiesCtrl.add(hostCapabilities);
    }
    if (message.videoLayout != null) {
      _videoLayoutCtrl.add(message.videoLayout!);
    }
  }

  // ==================== 发送 ====================

  void _sendEvent(Uint8List data) {
    if (!_eventReady || _eventChannel == null) return;
    _eventChannel!.send(RTCDataChannelMessage.fromBinary(data));
  }

  void _sendControl(Uint8List data) {
    if (!_controlReady || _controlChannel == null) return;
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

  void sendVideoControl(VideoControlMsg control) {
    _sendControl(encodeControlMessage(videoControl: control));
  }

  void sendClientResolution(ClientResolutionMsg resolution) {
    _sendControl(encodeControlMessage(clientResolution: resolution));
  }

  void dispose() {
    _controlChannel?.close();
    _eventChannel?.close();
    _controlReadyCtrl.close();
    _eventReadyCtrl.close();
    _cursorShapeCtrl.close();
    _clipboardCtrl.close();
    _capabilitiesCtrl.close();
    _videoLayoutCtrl.close();
  }
}
