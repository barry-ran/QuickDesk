/// Chromium Remoting WebRTC SDP 签名工具。
///
/// `SdpMessage` 会按换行拆分、逐行去除首尾空白、丢弃空行，随后使用
/// LF 重新连接并保留结尾换行。签名输入为 `<type> <normalized-sdp>`。
library;

import 'dart:convert';
import 'dart:typed_data';

import 'package:crypto/crypto.dart' as crypto;

String normalizeSdpForSignature(String sdp) {
  final lines = const LineSplitter()
      .convert(sdp)
      .map((line) => line.trim())
      .where((line) => line.isNotEmpty);
  return '${lines.join('\n')}\n';
}

String signSdp(Uint8List authKey, String sdp, String type) {
  final payload = utf8.encode('$type ${normalizeSdpForSignature(sdp)}');
  final mac = crypto.Hmac(crypto.sha256, authKey).convert(payload);
  return base64Encode(mac.bytes);
}

bool verifySdpSignature(
  Uint8List authKey,
  String sdp,
  String type,
  String? signature,
) {
  if (signature == null || signature.isEmpty) return false;

  Uint8List received;
  try {
    received = base64Decode(signature);
  } on FormatException {
    return false;
  }

  final expected = base64Decode(signSdp(authKey, sdp, type));
  if (received.length != expected.length) return false;

  var difference = 0;
  for (var i = 0; i < received.length; i++) {
    difference |= received[i] ^ expected[i];
  }
  return difference == 0;
}
