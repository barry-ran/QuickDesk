/// spake2.dart - SPAKE2 Curve25519 认证协议（Dart 移植）
///
/// 1:1 对照 WebClient/js/auth/spake2.js（其严格参照 BoringSSL spake25519.cc
/// 与 Chromium spake2_authenticator.cc），作为 Android 客户端协议栈 PoC。
///
/// 协议流程:
/// 1. 双方各生成 SPAKE2 消息 (generateMessage)
/// 2. 交换消息后各自计算共享密钥 (processMessage)
/// 3. 使用 verification-hash 验证双方得到相同密钥
library;

import 'dart:convert';
import 'dart:math';
import 'dart:typed_data';

import 'package:crypto/crypto.dart' as crypto;

import 'edwards25519.dart';

// ==================== 常量 ====================

/// BoringSSL 的 SPAKE2 M/N 点 (与 RFC 9382 不同!)
/// M: SHA-256 迭代哈希 'edwards25519 point generation seed (M)' 生成
/// N: SHA-256 迭代哈希 'edwards25519 point generation seed (N)' 生成
const String mHex = '5ada7e4bf6ddd9adb6626d32131c6b5c51a1e347a3478f53cfcf441b88eed12e';
const String nHex = '10e3df0ae37d8e7a99b5fe74b44672103dbddcbd06af680d71329a11693bc778';

/// 角色定义（与 BoringSSL spake2_role_alice/bob 对应）
const int spake2RoleAlice = 0; // Client
const int spake2RoleBob = 1; // Host

// ==================== 工具函数 ====================

Uint8List hexToBytes(String hex) {
  final bytes = Uint8List(hex.length ~/ 2);
  for (var i = 0; i < bytes.length; i++) {
    bytes[i] = int.parse(hex.substring(i * 2, i * 2 + 2), radix: 16);
  }
  return bytes;
}

String bytesToHex(List<int> bytes) =>
    bytes.map((b) => b.toRadixString(16).padLeft(2, '0')).join();

/// 字节数组(LE)转 BigInt
BigInt bytesToNumberLE(List<int> bytes) {
  var result = BigInt.zero;
  for (var i = bytes.length - 1; i >= 0; i--) {
    result = (result << 8) | BigInt.from(bytes[i]);
  }
  return result;
}

Uint8List concatBytes(List<List<int>> arrays) {
  final total = arrays.fold<int>(0, (sum, a) => sum + a.length);
  final result = Uint8List(total);
  var offset = 0;
  for (final a in arrays) {
    result.setRange(offset, offset + a.length, a);
    offset += a.length;
  }
  return result;
}

/// uint64 转 8 字节小端（对照 auth-util.js uint64ToLittleEndian）
Uint8List uint64ToLittleEndian(int value) {
  final b = ByteData(8);
  b.setUint64(0, value, Endian.little);
  return b.buffer.asUint8List();
}

Uint8List sha256Bytes(List<int> data) =>
    Uint8List.fromList(crypto.sha256.convert(data).bytes);

Uint8List sha512Bytes(List<int> data) =>
    Uint8List.fromList(crypto.sha512.convert(data).bytes);

Uint8List hmacSha256(List<int> key, List<int> data) =>
    Uint8List.fromList(crypto.Hmac(crypto.sha256, key).convert(data).bytes);

/// 共享密钥哈希：HMAC_SHA256(key=tag, data=sharedSecret)
/// 对照 auth-util.js getSharedSecretHash（tag=device_id, secret=device_id+access_code）
Uint8List getSharedSecretHash(String tag, String sharedSecret) =>
    hmacSha256(utf8.encode(tag), utf8.encode(sharedSecret));

/// sc_reduce: 将 64 字节 (512-bit LE) 归约为 mod L
BigInt scReduce(List<int> bytes64) => bytesToNumberLE(bytes64) % curveOrder;

/// 点乘以余因子 8: 计算 (8*scalar) * point = 8 * (scalar * point)
/// 与 BoringSSL 的 left_shift_3 + ge_scalarmult 数学等价（对照 spake2.js）
EdwardsPoint multiplyWithCofactor(EdwardsPoint point, BigInt reducedScalar) {
  var result = point.multiply(reducedScalar);
  result = result.add(result); // 2x
  result = result.add(result); // 4x
  result = result.add(result); // 8x
  return result;
}

/// 密码 → 标量: SHA512(password) → sc_reduce
/// (BoringSSL 的 password scalar hack 可安全省略，见 spake2.js 注释)
({BigInt scalar, Uint8List hash}) passwordToScalar(List<int> passwordBytes) {
  final passwordHash = sha512Bytes(passwordBytes);
  final scalar = scReduce(passwordHash);
  return (scalar: scalar == BigInt.zero ? BigInt.one : scalar, hash: passwordHash);
}

/// 生成随机私钥: random(64 bytes) → sc_reduce（余因子 ×8 在点运算时应用）
BigInt generateReducedPrivateKey([Random? rng]) {
  final r = rng ?? Random.secure();
  final randomData = Uint8List(64);
  for (var i = 0; i < 64; i++) {
    randomData[i] = r.nextInt(256);
  }
  final reduced = scReduce(randomData);
  return reduced == BigInt.zero ? BigInt.one : reduced;
}

/// PrefixWithLength - 4 字节大端长度前缀（对照 spake2_authenticator.cc）
Uint8List prefixWithLength(String str) {
  final strBytes = utf8.encode(str);
  final len = ByteData(4)..setUint32(0, strBytes.length, Endian.big);
  return concatBytes([len.buffer.asUint8List(), strBytes]);
}

// ==================== SPAKE2 上下文 ====================

class Spake2Context {
  final int role;
  final String myName;
  final String theirName;

  late final EdwardsPoint pointM;
  late final EdwardsPoint pointN;

  BigInt? passwordScalar;
  Uint8List? passwordHash;
  BigInt? reducedPrivateKey;
  Uint8List? myMessage;
  Uint8List? authKey;

  Spake2Context(this.role, this.myName, this.theirName) {
    pointM = EdwardsPoint.fromBytes(hexToBytes(mHex));
    pointN = EdwardsPoint.fromBytes(hexToBytes(nHex));
  }

  /// 生成 SPAKE2 消息（32 字节压缩点）
  ///
  /// my_msg = (8 * reducedPrivateKey) * G + passwordScalar * (M or N)
  Uint8List generateMessage(List<int> password, {BigInt? privateKeyOverride}) {
    final pw = passwordToScalar(password);
    passwordScalar = pw.scalar;
    passwordHash = pw.hash;

    reducedPrivateKey = privateKeyOverride ?? generateReducedPrivateKey();

    final blindingPoint = role == spake2RoleAlice ? pointM : pointN;

    final pubPoint = multiplyWithCofactor(EdwardsPoint.base, reducedPrivateKey!);
    final blindedPoint = blindingPoint.multiply(passwordScalar!);
    final myMsgPoint = pubPoint.add(blindedPoint);

    myMessage = Uint8List.fromList(myMsgPoint.toBytes());
    return myMessage!;
  }

  /// 处理对方消息并派生密钥（64 字节 = SHA512(transcript)）
  ///
  /// Q = their_msg - password_scalar * (N or M)
  /// K = (8 * reducedPrivateKey) * Q
  Uint8List processMessage(List<int> theirMessage) {
    if (reducedPrivateKey == null || myMessage == null) {
      throw StateError('Must call generateMessage() first');
    }

    final theirPoint = EdwardsPoint.fromBytes(theirMessage);

    final unblindingPoint = role == spake2RoleAlice ? pointN : pointM;

    final blindedPoint = unblindingPoint.multiply(passwordScalar!);
    final q = theirPoint.add(blindedPoint.negate());

    final sharedPoint = multiplyWithCofactor(q, reducedPrivateKey!);
    final dhShared = Uint8List.fromList(sharedPoint.toBytes());

    // 转录: 每个字段带 8 字节 LE 长度前缀（对照 BoringSSL update_with_length_prefix）
    final String aliceName;
    final String bobName;
    final List<int> tMsg;
    final List<int> sMsg;
    if (role == spake2RoleAlice) {
      aliceName = myName;
      bobName = theirName;
      tMsg = myMessage!;
      sMsg = theirMessage;
    } else {
      aliceName = theirName;
      bobName = myName;
      tMsg = theirMessage;
      sMsg = myMessage!;
    }

    final aliceNameBytes = utf8.encode(aliceName);
    final bobNameBytes = utf8.encode(bobName);

    final transcript = concatBytes([
      uint64ToLittleEndian(aliceNameBytes.length),
      aliceNameBytes,
      uint64ToLittleEndian(bobNameBytes.length),
      bobNameBytes,
      uint64ToLittleEndian(tMsg.length),
      tMsg,
      uint64ToLittleEndian(sMsg.length),
      sMsg,
      uint64ToLittleEndian(dhShared.length),
      dhShared,
      uint64ToLittleEndian(passwordHash!.length),
      passwordHash!,
    ]);

    authKey = sha512Bytes(transcript);
    return authKey!;
  }

  /// 验证哈希:
  /// HMAC_SHA256(auth_key, ("host"|"client") + PrefixWithLength(local) + PrefixWithLength(remote))
  Uint8List calculateVerificationHash(bool fromHost, String localId, String remoteId) {
    if (authKey == null) {
      throw StateError('Must call processMessage() first');
    }
    final message = concatBytes([
      utf8.encode(fromHost ? 'host' : 'client'),
      prefixWithLength(localId),
      prefixWithLength(remoteId),
    ]);
    return hmacSha256(authKey!, message);
  }

  /// 本端发出的验证哈希
  Uint8List getOutgoingVerificationHash() {
    final isHost = role == spake2RoleBob;
    return calculateVerificationHash(isHost, myName, theirName);
  }

  /// 期望对端发来的验证哈希
  Uint8List getExpectedVerificationHash() {
    final isHost = role == spake2RoleBob;
    return calculateVerificationHash(!isHost, theirName, myName);
  }

  bool verifyHash(List<int> theirHash) {
    final expected = getExpectedVerificationHash();
    if (theirHash.length != expected.length) return false;
    var diff = 0;
    for (var i = 0; i < theirHash.length; i++) {
      diff |= theirHash[i] ^ expected[i];
    }
    return diff == 0;
  }
}
