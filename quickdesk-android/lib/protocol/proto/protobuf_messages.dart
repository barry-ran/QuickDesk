/// protobuf_messages.dart - Chromium Remoting control/event 消息编解码
///
/// 1:1 对照 WebClient/js/protocol/protobuf-messages.js，
/// 实现 Chromium Remoting 的 protobuf 消息编解码
/// （src/remoting/proto/event.proto, control.proto, internal.proto）。
///
/// 仅保留 Android 端实际收发的字段：
///   发送（client→host）：EventMessage(mouse/key/text)、clipboard、
///     capabilities、audio-control
///   发送（host→client）：capabilities、VideoLayout
///   接收：EventMessage、clipboard、capabilities、audio-control、VideoLayout
/// 其余字段按 wire type 跳过，协议兼容性不受影响。
library;

import 'dart:typed_data';

import 'wire.dart';

// ==================== 消息模型 ====================

enum MouseButton {
  undefined(0),
  left(1),
  middle(2),
  right(3),
  back(4),
  forward(5);

  final int value;
  const MouseButton(this.value);
}

class MouseEventMsg {
  int? x, y;
  int? button;
  bool? buttonDown;
  double? wheelDeltaX, wheelDeltaY, wheelTicksX, wheelTicksY;

  MouseEventMsg({
    this.x,
    this.y,
    this.button,
    this.buttonDown,
    this.wheelDeltaX,
    this.wheelDeltaY,
    this.wheelTicksX,
    this.wheelTicksY,
  });
}

class KeyEventMsg {
  bool pressed;
  int usbKeycode;

  KeyEventMsg({required this.pressed, required this.usbKeycode});
}

class ClipboardEventMsg {
  String mimeType;
  Uint8List data;

  ClipboardEventMsg({required this.mimeType, required this.data});
}

class VideoTrackLayout {
  String? mediaStreamId;
  int? positionX, positionY, width, height, xDpi, yDpi, screenId;
  String? displayName;
}

class VideoLayoutMsg {
  final List<VideoTrackLayout> videoTracks = [];
  bool? supportsFullDesktopCapture;
  int? primaryScreenId;
}

/// 解码后的 EventMessage（internal.proto）
class EventMessage {
  int? timestamp;
  KeyEventMsg? keyEvent;
  MouseEventMsg? mouseEvent;
  String? textEventText;
}

/// 解码后的 ControlMessage（internal.proto，仅保留消费的字段）
class ControlMessage {
  ClipboardEventMsg? clipboardEvent;
  bool? audioControlEnable;
  String? capabilities;
  VideoLayoutMsg? videoLayout;
}

// ==================== 消息编码 ====================

Uint8List encodeMouseEvent(MouseEventMsg e) => concatBytes([
      if (e.x != null) varintField(1, e.x),
      if (e.y != null) varintField(2, e.y),
      if (e.button != null) varintField(5, e.button),
      if (e.buttonDown != null) varintField(6, e.buttonDown),
      if (e.wheelDeltaX != null) floatField(7, e.wheelDeltaX),
      if (e.wheelDeltaY != null) floatField(8, e.wheelDeltaY),
      if (e.wheelTicksX != null) floatField(9, e.wheelTicksX),
      if (e.wheelTicksY != null) floatField(10, e.wheelTicksY),
    ]);

Uint8List encodeKeyEvent(KeyEventMsg e) => concatBytes([
      varintField(2, e.pressed),
      varintField(3, e.usbKeycode),
    ]);

Uint8List encodeClipboardEvent(ClipboardEventMsg e) => concatBytes([
      lengthDelimitedField(1, e.mimeType),
      lengthDelimitedField(2, e.data),
    ]);

/// EventMessage 包装（internal.proto: timestamp=1, key=3, mouse=4, text=5）
Uint8List encodeEventMessage({
  int? timestamp,
  KeyEventMsg? keyEvent,
  MouseEventMsg? mouseEvent,
  String? textEventText,
}) =>
    concatBytes([
      if (timestamp != null) varintField(1, timestamp),
      if (keyEvent != null) lengthDelimitedField(3, encodeKeyEvent(keyEvent)),
      if (mouseEvent != null)
        lengthDelimitedField(4, encodeMouseEvent(mouseEvent)),
      if (textEventText != null)
        lengthDelimitedField(5, lengthDelimitedField(1, textEventText)),
    ]);

Uint8List encodeVideoTrackLayout(VideoTrackLayout t) => concatBytes([
      if (t.mediaStreamId != null) lengthDelimitedField(1, t.mediaStreamId),
      if (t.positionX != null) varintField(2, t.positionX),
      if (t.positionY != null) varintField(3, t.positionY),
      if (t.width != null) varintField(4, t.width),
      if (t.height != null) varintField(5, t.height),
      if (t.xDpi != null) varintField(6, t.xDpi),
      if (t.yDpi != null) varintField(7, t.yDpi),
      if (t.screenId != null) varintField(8, t.screenId),
      if (t.displayName != null) lengthDelimitedField(9, t.displayName),
    ]);

Uint8List encodeVideoLayout(VideoLayoutMsg layout) => concatBytes([
      for (final t in layout.videoTracks)
        lengthDelimitedField(1, encodeVideoTrackLayout(t)),
      if (layout.supportsFullDesktopCapture != null)
        varintField(2, layout.supportsFullDesktopCapture),
      if (layout.primaryScreenId != null)
        varintField(3, layout.primaryScreenId),
    ]);

/// ControlMessage 包装（internal.proto 字段号对照 JS 版本）
Uint8List encodeControlMessage({
  ClipboardEventMsg? clipboardEvent,
  bool? audioControlEnable,
  String? capabilities,
  VideoLayoutMsg? videoLayout,
}) =>
    concatBytes([
      if (clipboardEvent != null)
        lengthDelimitedField(1, encodeClipboardEvent(clipboardEvent)),
      if (audioControlEnable != null)
        lengthDelimitedField(5, varintField(1, audioControlEnable)),
      if (capabilities != null)
        lengthDelimitedField(6, lengthDelimitedField(1, capabilities)),
      if (videoLayout != null)
        lengthDelimitedField(10, encodeVideoLayout(videoLayout)),
    ]);

// ==================== 消息解码 ====================

ClipboardEventMsg decodeClipboardEvent(Uint8List data) {
  final reader = ProtobufReader(data);
  var mimeType = '';
  var payload = Uint8List(0);
  while (reader.hasMore) {
    final tag = reader.readTag();
    switch (tag.fieldNumber) {
      case 1:
        mimeType = reader.readString();
        break;
      case 2:
        payload = reader.readBytes();
        break;
      default:
        reader.skipField(tag.wireType);
    }
  }
  return ClipboardEventMsg(mimeType: mimeType, data: payload);
}

VideoTrackLayout decodeVideoTrackLayout(Uint8List data) {
  final reader = ProtobufReader(data);
  final result = VideoTrackLayout();
  while (reader.hasMore) {
    final tag = reader.readTag();
    switch (tag.fieldNumber) {
      case 1:
        result.mediaStreamId = reader.readString();
        break;
      case 2:
        result.positionX = reader.readSignedVarint();
        break;
      case 3:
        result.positionY = reader.readSignedVarint();
        break;
      case 4:
        result.width = reader.readSignedVarint();
        break;
      case 5:
        result.height = reader.readSignedVarint();
        break;
      case 6:
        result.xDpi = reader.readSignedVarint();
        break;
      case 7:
        result.yDpi = reader.readSignedVarint();
        break;
      case 8:
        result.screenId = reader.readVarint();
        break;
      case 9:
        result.displayName = reader.readString();
        break;
      default:
        reader.skipField(tag.wireType);
    }
  }
  return result;
}

VideoLayoutMsg decodeVideoLayout(Uint8List data) {
  final reader = ProtobufReader(data);
  final result = VideoLayoutMsg();
  while (reader.hasMore) {
    final tag = reader.readTag();
    switch (tag.fieldNumber) {
      case 1:
        result.videoTracks.add(decodeVideoTrackLayout(reader.readBytes()));
        break;
      case 2:
        result.supportsFullDesktopCapture = reader.readVarint() != 0;
        break;
      case 3:
        result.primaryScreenId = reader.readVarint();
        break;
      default:
        reader.skipField(tag.wireType);
    }
  }
  return result;
}

ControlMessage decodeControlMessage(Uint8List data) {
  final reader = ProtobufReader(data);
  final result = ControlMessage();
  while (reader.hasMore) {
    final tag = reader.readTag();
    switch (tag.fieldNumber) {
      case 1:
        result.clipboardEvent = decodeClipboardEvent(reader.readBytes());
        break;
      case 5:
        final audioReader = ProtobufReader(reader.readBytes());
        while (audioReader.hasMore) {
          final at = audioReader.readTag();
          if (at.fieldNumber == 1) {
            result.audioControlEnable = audioReader.readVarint() != 0;
          } else {
            audioReader.skipField(at.wireType);
          }
        }
        break;
      case 6:
        final capsReader = ProtobufReader(reader.readBytes());
        while (capsReader.hasMore) {
          final ct = capsReader.readTag();
          if (ct.fieldNumber == 1) {
            result.capabilities = capsReader.readString();
          } else {
            capsReader.skipField(ct.wireType);
          }
        }
        break;
      case 10:
        result.videoLayout = decodeVideoLayout(reader.readBytes());
        break;
      default:
        reader.skipField(tag.wireType);
    }
  }
  return result;
}

EventMessage decodeEventMessage(Uint8List data) {
  final reader = ProtobufReader(data);
  final result = EventMessage();
  while (reader.hasMore) {
    final tag = reader.readTag();
    switch (tag.fieldNumber) {
      case 1:
        result.timestamp = reader.readVarint();
        break;
      case 3:
        final keyReader = ProtobufReader(reader.readBytes());
        var pressed = false;
        var usbKeycode = 0;
        while (keyReader.hasMore) {
          final kt = keyReader.readTag();
          switch (kt.fieldNumber) {
            case 2:
              pressed = keyReader.readVarint() != 0;
              break;
            case 3:
              usbKeycode = keyReader.readVarint();
              break;
            default:
              keyReader.skipField(kt.wireType);
          }
        }
        result.keyEvent = KeyEventMsg(pressed: pressed, usbKeycode: usbKeycode);
        break;
      case 4:
        final mouseReader = ProtobufReader(reader.readBytes());
        final mouse = MouseEventMsg();
        while (mouseReader.hasMore) {
          final mt = mouseReader.readTag();
          switch (mt.fieldNumber) {
            case 1:
              mouse.x = mouseReader.readSignedVarint();
              break;
            case 2:
              mouse.y = mouseReader.readSignedVarint();
              break;
            case 5:
              mouse.button = mouseReader.readVarint();
              break;
            case 6:
              mouse.buttonDown = mouseReader.readVarint() != 0;
              break;
            case 7:
              mouse.wheelDeltaX = mouseReader.readFloat();
              break;
            case 8:
              mouse.wheelDeltaY = mouseReader.readFloat();
              break;
            default:
              mouseReader.skipField(mt.wireType);
          }
        }
        result.mouseEvent = mouse;
        break;
      case 5:
        final textReader = ProtobufReader(reader.readBytes());
        while (textReader.hasMore) {
          final tt = textReader.readTag();
          if (tt.fieldNumber == 1) {
            result.textEventText = textReader.readString();
          } else {
            textReader.skipField(tt.wireType);
          }
        }
        break;
      default:
        reader.skipField(tag.wireType);
    }
  }
  return result;
}
