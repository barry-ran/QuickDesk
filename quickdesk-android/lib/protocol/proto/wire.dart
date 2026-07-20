/// wire.dart - Protobuf wire format 基础读写
///
/// 供 protobuf_messages.dart（control/event 消息）与 file_transfer.dart
/// （file_transfer.proto）共用的手写编解码原语。
///
/// Wire format:
///   tag = (field_number << 3) | wire_type
///   wire_type: 0=varint, 1=64bit, 2=length-delimited, 5=32bit
library;

import 'dart:convert';
import 'dart:typed_data';

Uint8List encodeVarint(int value) {
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

Uint8List encodeTag(int fieldNumber, int wireType) =>
    encodeVarint((fieldNumber << 3) | wireType);

/// varint 字段 (int32/uint32/bool/enum/int64)
Uint8List varintField(int fieldNumber, Object? value) {
  if (value == null) return Uint8List(0);
  final int v = value is bool ? (value ? 1 : 0) : value as int;
  return concatBytes([encodeTag(fieldNumber, 0), encodeVarint(v)]);
}

/// float 字段 (wire type 5, fixed32, little-endian)
Uint8List floatField(int fieldNumber, double? value) {
  if (value == null) return Uint8List(0);
  final buf = ByteData(4)..setFloat32(0, value, Endian.little);
  return concatBytes([encodeTag(fieldNumber, 5), buf.buffer.asUint8List()]);
}

/// length-delimited 字段 (bytes/string/embedded message)
Uint8List lengthDelimitedField(int fieldNumber, Object? data) {
  if (data == null) return Uint8List(0);
  final Uint8List bytes;
  if (data is String) {
    bytes = Uint8List.fromList(utf8.encode(data));
  } else if (data is Uint8List) {
    bytes = data;
  } else {
    bytes = Uint8List.fromList(data as List<int>);
  }
  return concatBytes(
      [encodeTag(fieldNumber, 2), encodeVarint(bytes.length), bytes]);
}

Uint8List concatBytes(List<Uint8List> arrays) {
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
