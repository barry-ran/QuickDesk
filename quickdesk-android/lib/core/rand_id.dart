/// rand_id.dart - 随机标识生成（UUID v4 / 十六进制串 / 数字串）
library;

import 'dart:math';

final _rand = Random.secure();

/// RFC 4122 v4 UUID（小写连字符格式）
String generateUuidV4() {
  final b = List<int>.generate(16, (_) => _rand.nextInt(256));
  b[6] = (b[6] & 0x0f) | 0x40;
  b[8] = (b[8] & 0x3f) | 0x80;
  final h = b.map((x) => x.toRadixString(16).padLeft(2, '0')).join();
  return '${h.substring(0, 8)}-${h.substring(8, 12)}-${h.substring(12, 16)}-'
      '${h.substring(16, 20)}-${h.substring(20)}';
}

/// [len] 位小写十六进制随机串
String randomHex(int len) {
  const chars = '0123456789abcdef';
  return List.generate(len, (_) => chars[_rand.nextInt(16)]).join();
}

/// [len] 位数字随机串（访问码等）
String randomDigits(int len) =>
    List.generate(len, (_) => _rand.nextInt(10)).join();
