/// protobuf_messages.dart - 手动 Protobuf 编解码器
///
/// 1:1 对照 WebClient/js/protocol/protobuf-messages.js，
/// 实现 Chromium Remoting 的 protobuf 消息编解码
/// （src/remoting/proto/event.proto, control.proto, internal.proto）。
///
/// Wire format:
///   tag = (field_number << 3) | wire_type
///   wire_type: 0=varint, 1=64bit, 2=length-delimited, 5=32bit
library;

import 'dart:convert';
import 'dart:typed_data';

// ==================== 编码工具 ====================

Uint8List _encodeVarint(int value) {
  final bytes = <int>[];
  if (value < 0) {
    // 负数按 64 位 two's complement varint（最长 10 字节）。
    // Dart int 的 >> 是算术移位，用 BigInt.toUnsigned 保证逻辑移位语义。
    var big = BigInt.from(value).toUnsigned(64);
    while (big > BigInt.from(0x7F)) {
      bytes.add(((big & BigInt.from(0x7F)).toInt()) | 0x80);
      big >>= 7;
    }
    bytes.add(big.toInt());
    return Uint8List.fromList(bytes);
  }
  var v = value;
  while (v > 0x7F) {
    bytes.add((v & 0x7F) | 0x80);
    v >>= 7;
  }
  bytes.add(v);
  return Uint8List.fromList(bytes);
}

Uint8List _encodeTag(int fieldNumber, int wireType) =>
    _encodeVarint((fieldNumber << 3) | wireType);

/// varint 字段 (int32/uint32/bool/enum/int64)
Uint8List _varintField(int fieldNumber, Object? value) {
  if (value == null) return Uint8List(0);
  final int v = value is bool ? (value ? 1 : 0) : value as int;
  return _concat([_encodeTag(fieldNumber, 0), _encodeVarint(v)]);
}

/// float 字段 (wire type 5, fixed32, little-endian)
Uint8List _floatField(int fieldNumber, double? value) {
  if (value == null) return Uint8List(0);
  final buf = ByteData(4)..setFloat32(0, value, Endian.little);
  return _concat([_encodeTag(fieldNumber, 5), buf.buffer.asUint8List()]);
}

/// length-delimited 字段 (bytes/string/embedded message)
Uint8List _lengthDelimitedField(int fieldNumber, Object? data) {
  if (data == null) return Uint8List(0);
  final Uint8List bytes;
  if (data is String) {
    bytes = Uint8List.fromList(utf8.encode(data));
  } else if (data is Uint8List) {
    bytes = data;
  } else {
    bytes = Uint8List.fromList(data as List<int>);
  }
  return _concat([_encodeTag(fieldNumber, 2), _encodeVarint(bytes.length), bytes]);
}

Uint8List _concat(List<Uint8List> arrays) {
  final filtered = arrays.where((a) => a.isNotEmpty).toList();
  final total = filtered.fold<int>(0, (s, a) => s + a.length);
  final result = Uint8List(total);
  var offset = 0;
  for (final a in filtered) {
    result.setRange(offset, offset + a.length, a);
    offset += a.length;
  }
  return result;
}

// ==================== 解码工具 ====================

class ProtobufReader {
  final Uint8List data;
  int offset = 0;

  ProtobufReader(List<int> input)
      : data = input is Uint8List ? input : Uint8List.fromList(input);

  bool get hasMore => offset < data.length;

  /// 读取 varint（返回 Dart int，64 位安全）
  int readVarint() {
    var result = 0;
    var shift = 0;
    while (offset < data.length) {
      final byte = data[offset++];
      result |= (byte & 0x7F) << shift;
      if ((byte & 0x80) == 0) {
        return result;
      }
      shift += 7;
      if (shift >= 64) {
        // 防御超长 varint
        while (offset < data.length && (data[offset] & 0x80) != 0) {
          offset++;
        }
        if (offset < data.length) offset++;
        return result;
      }
    }
    return result;
  }

  /// 读取按 int32 语义解释的 varint（负数还原）
  int readSignedVarint() {
    final v = readVarint();
    // 截断到 32 位有符号（对照 JS 的 `| 0`）
    final truncated = v & 0xFFFFFFFF;
    return truncated >= 0x80000000 ? truncated - 0x100000000 : truncated;
  }

  ({int fieldNumber, int wireType}) readTag() {
    final varint = readVarint();
    return (fieldNumber: varint >> 3, wireType: varint & 0x07);
  }

  Uint8List readBytes() {
    final length = readVarint();
    final bytes = Uint8List.sublistView(data, offset, offset + length);
    offset += length;
    return Uint8List.fromList(bytes);
  }

  String readString() => utf8.decode(readBytes(), allowMalformed: true);

  double readFloat() {
    final v = ByteData.sublistView(data, offset, offset + 4)
        .getFloat32(0, Endian.little);
    offset += 4;
    return v;
  }

  void skipField(int wireType) {
    switch (wireType) {
      case 0:
        readVarint();
        break;
      case 1:
        offset += 8;
        break;
      case 2:
        final len = readVarint();
        offset += len;
        break;
      case 5:
        offset += 4;
        break;
      default:
        throw FormatException('Unknown wire type: $wireType');
    }
  }
}

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
  int? lockStates;

  KeyEventMsg({required this.pressed, required this.usbKeycode, this.lockStates});
}

class ClipboardEventMsg {
  String mimeType;
  Uint8List data;

  ClipboardEventMsg({required this.mimeType, required this.data});
}

class CursorShapeInfo {
  int? width, height, hotspotX, hotspotY;
  Uint8List? data;
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

class CapabilitiesMsg {
  String capabilities;
  CapabilitiesMsg(this.capabilities);
}

class ExtensionMessageMsg {
  String? type;
  String? data;
}

class VideoControlMsg {
  bool? enable;
  int? targetFramerate;
  ({bool? enabled, int? captureIntervalMs, int? boostDurationMs})? framerateBoost;
}

class ClientResolutionMsg {
  int? widthPixels, heightPixels, xDpi, yDpi, screenId;

  ClientResolutionMsg({this.widthPixels, this.heightPixels, this.xDpi, this.yDpi, this.screenId});
}

/// 解码后的 EventMessage（internal.proto）
class EventMessage {
  int? timestamp;
  KeyEventMsg? keyEvent;
  MouseEventMsg? mouseEvent;
  String? textEventText;
}

/// 解码后的 ControlMessage（internal.proto）
class ControlMessage {
  ClipboardEventMsg? clipboardEvent;
  ClientResolutionMsg? clientResolution;
  VideoControlMsg? videoControl;
  CursorShapeInfo? cursorShape;
  bool? audioControlEnable;
  CapabilitiesMsg? capabilities;
  ExtensionMessageMsg? extensionMessage;
  VideoLayoutMsg? videoLayout;
  String? transportInfoProtocol;
}

// ==================== 消息编码 ====================

Uint8List encodeMouseEvent(MouseEventMsg e) => _concat([
      if (e.x != null) _varintField(1, e.x),
      if (e.y != null) _varintField(2, e.y),
      if (e.button != null) _varintField(5, e.button),
      if (e.buttonDown != null) _varintField(6, e.buttonDown),
      if (e.wheelDeltaX != null) _floatField(7, e.wheelDeltaX),
      if (e.wheelDeltaY != null) _floatField(8, e.wheelDeltaY),
      if (e.wheelTicksX != null) _floatField(9, e.wheelTicksX),
      if (e.wheelTicksY != null) _floatField(10, e.wheelTicksY),
    ]);

Uint8List encodeKeyEvent(KeyEventMsg e) => _concat([
      _varintField(2, e.pressed),
      _varintField(3, e.usbKeycode),
      if (e.lockStates != null) _varintField(4, e.lockStates),
    ]);

Uint8List encodeClipboardEvent(ClipboardEventMsg e) => _concat([
      _lengthDelimitedField(1, e.mimeType),
      _lengthDelimitedField(2, e.data),
    ]);

/// EventMessage 包装（internal.proto: timestamp=1, key=3, mouse=4, text=5）
Uint8List encodeEventMessage({
  int? timestamp,
  KeyEventMsg? keyEvent,
  MouseEventMsg? mouseEvent,
  String? textEventText,
}) =>
    _concat([
      if (timestamp != null) _varintField(1, timestamp),
      if (keyEvent != null) _lengthDelimitedField(3, encodeKeyEvent(keyEvent)),
      if (mouseEvent != null) _lengthDelimitedField(4, encodeMouseEvent(mouseEvent)),
      if (textEventText != null)
        _lengthDelimitedField(5, _lengthDelimitedField(1, textEventText)),
    ]);

Uint8List encodeClientResolution(ClientResolutionMsg r) => _concat([
      if (r.widthPixels != null) _varintField(1, r.widthPixels),
      if (r.heightPixels != null) _varintField(2, r.heightPixels),
      if (r.xDpi != null) _varintField(5, r.xDpi),
      if (r.yDpi != null) _varintField(6, r.yDpi),
      if (r.screenId != null) _varintField(7, r.screenId),
    ]);

Uint8List encodeVideoControl(VideoControlMsg c) => _concat([
      if (c.enable != null) _varintField(1, c.enable),
      if (c.targetFramerate != null) _varintField(5, c.targetFramerate),
      if (c.framerateBoost != null)
        _lengthDelimitedField(
            4,
            _concat([
              if (c.framerateBoost!.enabled != null) _varintField(1, c.framerateBoost!.enabled),
              if (c.framerateBoost!.captureIntervalMs != null)
                _varintField(2, c.framerateBoost!.captureIntervalMs),
              if (c.framerateBoost!.boostDurationMs != null)
                _varintField(3, c.framerateBoost!.boostDurationMs),
            ])),
    ]);

Uint8List encodeCursorShapeInfo(CursorShapeInfo info) => _concat([
      if (info.width != null) _varintField(1, info.width),
      if (info.height != null) _varintField(2, info.height),
      if (info.hotspotX != null) _varintField(3, info.hotspotX),
      if (info.hotspotY != null) _varintField(4, info.hotspotY),
      if (info.data != null) _lengthDelimitedField(5, info.data),
    ]);

Uint8List encodeVideoTrackLayout(VideoTrackLayout t) => _concat([
      if (t.mediaStreamId != null) _lengthDelimitedField(1, t.mediaStreamId),
      if (t.positionX != null) _varintField(2, t.positionX),
      if (t.positionY != null) _varintField(3, t.positionY),
      if (t.width != null) _varintField(4, t.width),
      if (t.height != null) _varintField(5, t.height),
      if (t.xDpi != null) _varintField(6, t.xDpi),
      if (t.yDpi != null) _varintField(7, t.yDpi),
      if (t.screenId != null) _varintField(8, t.screenId),
      if (t.displayName != null) _lengthDelimitedField(9, t.displayName),
    ]);

Uint8List encodeVideoLayout(VideoLayoutMsg layout) => _concat([
      for (final t in layout.videoTracks) _lengthDelimitedField(1, encodeVideoTrackLayout(t)),
      if (layout.supportsFullDesktopCapture != null)
        _varintField(2, layout.supportsFullDesktopCapture),
      if (layout.primaryScreenId != null) _varintField(3, layout.primaryScreenId),
    ]);

/// ControlMessage 包装（internal.proto 字段号对照 JS 版本）
Uint8List encodeControlMessage({
  ClipboardEventMsg? clipboardEvent,
  ClientResolutionMsg? clientResolution,
  VideoControlMsg? videoControl,
  CursorShapeInfo? cursorShape,
  bool? audioControlEnable,
  String? capabilities,
  ExtensionMessageMsg? extensionMessage,
  VideoLayoutMsg? videoLayout,
  ({int? preferredMinBitrateBps, int? preferredMaxBitrateBps, bool? requestIceRestart, bool? requestSdpRestart})?
      peerConnectionParameters,
}) =>
    _concat([
      if (clipboardEvent != null) _lengthDelimitedField(1, encodeClipboardEvent(clipboardEvent)),
      if (clientResolution != null)
        _lengthDelimitedField(2, encodeClientResolution(clientResolution)),
      if (videoControl != null) _lengthDelimitedField(3, encodeVideoControl(videoControl)),
      if (cursorShape != null) _lengthDelimitedField(4, encodeCursorShapeInfo(cursorShape)),
      if (audioControlEnable != null)
        _lengthDelimitedField(5, _varintField(1, audioControlEnable)),
      if (capabilities != null)
        _lengthDelimitedField(6, _lengthDelimitedField(1, capabilities)),
      if (extensionMessage != null)
        _lengthDelimitedField(
            9,
            _concat([
              if (extensionMessage.type != null) _lengthDelimitedField(1, extensionMessage.type),
              if (extensionMessage.data != null) _lengthDelimitedField(2, extensionMessage.data),
            ])),
      if (videoLayout != null) _lengthDelimitedField(10, encodeVideoLayout(videoLayout)),
      if (peerConnectionParameters != null)
        _lengthDelimitedField(
            14,
            _concat([
              if (peerConnectionParameters.preferredMinBitrateBps != null)
                _varintField(1, peerConnectionParameters.preferredMinBitrateBps),
              if (peerConnectionParameters.preferredMaxBitrateBps != null)
                _varintField(2, peerConnectionParameters.preferredMaxBitrateBps),
              if (peerConnectionParameters.requestIceRestart != null)
                _varintField(3, peerConnectionParameters.requestIceRestart),
              if (peerConnectionParameters.requestSdpRestart != null)
                _varintField(4, peerConnectionParameters.requestSdpRestart),
            ])),
    ]);

// ==================== 消息解码 ====================

CursorShapeInfo decodeCursorShapeInfo(Uint8List data) {
  final reader = ProtobufReader(data);
  final result = CursorShapeInfo();
  while (reader.hasMore) {
    final tag = reader.readTag();
    switch (tag.fieldNumber) {
      case 1:
        result.width = reader.readSignedVarint();
        break;
      case 2:
        result.height = reader.readSignedVarint();
        break;
      case 3:
        result.hotspotX = reader.readSignedVarint();
        break;
      case 4:
        result.hotspotY = reader.readSignedVarint();
        break;
      case 5:
        result.data = reader.readBytes();
        break;
      default:
        reader.skipField(tag.wireType);
    }
  }
  return result;
}

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

VideoControlMsg decodeVideoControl(Uint8List data) {
  final reader = ProtobufReader(data);
  final result = VideoControlMsg();
  while (reader.hasMore) {
    final tag = reader.readTag();
    switch (tag.fieldNumber) {
      case 1:
        result.enable = reader.readVarint() != 0;
        break;
      case 5:
        result.targetFramerate = reader.readVarint();
        break;
      case 4:
        final boostReader = ProtobufReader(reader.readBytes());
        bool? enabled;
        int? captureIntervalMs;
        int? boostDurationMs;
        while (boostReader.hasMore) {
          final bt = boostReader.readTag();
          switch (bt.fieldNumber) {
            case 1:
              enabled = boostReader.readVarint() != 0;
              break;
            case 2:
              captureIntervalMs = boostReader.readSignedVarint();
              break;
            case 3:
              boostDurationMs = boostReader.readSignedVarint();
              break;
            default:
              boostReader.skipField(bt.wireType);
          }
        }
        result.framerateBoost = (
          enabled: enabled,
          captureIntervalMs: captureIntervalMs,
          boostDurationMs: boostDurationMs
        );
        break;
      default:
        reader.skipField(tag.wireType);
    }
  }
  return result;
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

ExtensionMessageMsg decodeExtensionMessage(Uint8List data) {
  final reader = ProtobufReader(data);
  final result = ExtensionMessageMsg();
  while (reader.hasMore) {
    final tag = reader.readTag();
    switch (tag.fieldNumber) {
      case 1:
        result.type = reader.readString();
        break;
      case 2:
        result.data = reader.readString();
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
      case 2:
        final resReader = ProtobufReader(reader.readBytes());
        final res = ClientResolutionMsg();
        while (resReader.hasMore) {
          final rt = resReader.readTag();
          switch (rt.fieldNumber) {
            case 1:
              res.widthPixels = resReader.readSignedVarint();
              break;
            case 2:
              res.heightPixels = resReader.readSignedVarint();
              break;
            case 5:
              res.xDpi = resReader.readSignedVarint();
              break;
            case 6:
              res.yDpi = resReader.readSignedVarint();
              break;
            case 7:
              res.screenId = resReader.readVarint();
              break;
            default:
              resReader.skipField(rt.wireType);
          }
        }
        result.clientResolution = res;
        break;
      case 3:
        result.videoControl = decodeVideoControl(reader.readBytes());
        break;
      case 4:
        result.cursorShape = decodeCursorShapeInfo(reader.readBytes());
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
        var caps = '';
        while (capsReader.hasMore) {
          final ct = capsReader.readTag();
          if (ct.fieldNumber == 1) {
            caps = capsReader.readString();
          } else {
            capsReader.skipField(ct.wireType);
          }
        }
        result.capabilities = CapabilitiesMsg(caps);
        break;
      case 9:
        result.extensionMessage = decodeExtensionMessage(reader.readBytes());
        break;
      case 10:
        result.videoLayout = decodeVideoLayout(reader.readBytes());
        break;
      case 13:
        final tiReader = ProtobufReader(reader.readBytes());
        while (tiReader.hasMore) {
          final tt = tiReader.readTag();
          if (tt.fieldNumber == 1) {
            result.transportInfoProtocol = tiReader.readString();
          } else {
            tiReader.skipField(tt.wireType);
          }
        }
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
