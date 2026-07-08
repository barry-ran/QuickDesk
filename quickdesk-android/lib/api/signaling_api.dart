/// signaling_api.dart - 信令服务器 REST 客户端（M1 主控端所需的最小集合）
///
/// 对照 SignalingServer/docs/user-api-docs.md 与 WebClient 的调用方式：
///   POST /v1/devices/:device_id/access-code:verify → signal_token
///   GET  /v1/ice-config                            → STUN/TURN 列表
library;

import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

class SignalingApiException implements Exception {
  final int statusCode;
  final String code;
  final String message;

  SignalingApiException(this.statusCode, this.code, this.message);

  @override
  String toString() => 'SignalingApiException($statusCode, $code): $message';
}

class IceServerEntry {
  final List<String> urls;
  final String? username;
  final String? credential;

  IceServerEntry({required this.urls, this.username, this.credential});

  Map<String, dynamic> toRtcConfig() => {
        'urls': urls,
        if (username != null) 'username': username,
        if (credential != null) 'credential': credential,
      };
}

class SignalingApi {
  /// 信令服务器 ws:// 或 wss:// 地址（与设置页一致），内部转 http(s)
  final String signalingWsUrl;

  /// 可选 X-API-Key（自建服务器开启保护时）
  final String? apiKey;

  late final String _httpBase = _wsToHttp(signalingWsUrl);

  SignalingApi(this.signalingWsUrl, {this.apiKey});

  static String _wsToHttp(String wsUrl) {
    var url = wsUrl.trim();
    if (url.startsWith('wss://')) {
      url = 'https://${url.substring(6)}';
    } else if (url.startsWith('ws://')) {
      url = 'http://${url.substring(5)}';
    }
    while (url.endsWith('/')) {
      url = url.substring(0, url.length - 1);
    }
    return url;
  }

  Map<String, String> get _headers => {
        'Content-Type': 'application/json',
        if (apiKey != null && apiKey!.isNotEmpty) 'X-API-Key': apiKey!,
      };

  /// 校验访问码并换取一次性 signal_token
  Future<String> verifyAccessCode(String deviceId, String accessCode) async {
    final resp = await http
        .post(
          Uri.parse('$_httpBase/v1/devices/$deviceId/access-code:verify'),
          headers: _headers,
          body: jsonEncode({'code': accessCode}),
        )
        .timeout(const Duration(seconds: 10));

    final json = _decodeBody(resp);
    if (resp.statusCode != 200) {
      throw SignalingApiException(
        resp.statusCode,
        (json['code'] ?? 'UNKNOWN').toString(),
        (json['message'] ?? resp.body).toString(),
      );
    }
    final token = json['signal_token'] as String?;
    if (token == null || token.isEmpty) {
      throw SignalingApiException(resp.statusCode, 'NO_TOKEN', 'missing signal_token');
    }
    return token;
  }

  /// 获取 ICE 配置（STUN/TURN），失败返回空列表（P2P 仍可尝试主机候选）
  Future<List<IceServerEntry>> getIceServers() async {
    try {
      final resp = await http
          .get(Uri.parse('$_httpBase/v1/ice-config'), headers: _headers)
          .timeout(const Duration(seconds: 3));
      if (resp.statusCode != 200) return [];

      final json = _decodeBody(resp);
      final list = (json['ice_servers'] ?? json['iceServers']) as List<dynamic>? ?? [];
      return list.map((server) {
        final map = server as Map<String, dynamic>;
        final urlsRaw = map['urls'];
        final urls = urlsRaw is String
            ? [urlsRaw]
            : (urlsRaw as List<dynamic>).map((u) => u.toString()).toList();
        return IceServerEntry(
          urls: urls,
          username: map['username'] as String?,
          credential: map['credential'] as String?,
        );
      }).toList();
    } catch (_) {
      return [];
    }
  }

  Map<String, dynamic> _decodeBody(http.Response resp) {
    try {
      final decoded = jsonDecode(utf8.decode(resp.bodyBytes));
      return decoded is Map<String, dynamic> ? decoded : {};
    } catch (_) {
      return {};
    }
  }
}
