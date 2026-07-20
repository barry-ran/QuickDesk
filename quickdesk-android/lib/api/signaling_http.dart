/// signaling_http.dart - 信令服务器 REST 公共层
///
/// SignalingApi（主控端）与 HostApi（被控端）共用：ws→http 地址转换、
/// 请求头（X-API-Key / Bearer device_secret）、JSON 请求模板与错误抛出、
/// ICE 配置解析。
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

/// REST 客户端基类。
abstract class SignalingHttpBase {
  /// 信令服务器 ws:// 或 wss:// 地址（与设置页一致），内部转 http(s)
  final String signalingWsUrl;

  /// 可选 X-API-Key（自建服务器开启保护时）
  final String? apiKey;

  late final String httpBase = _wsToHttp(signalingWsUrl);

  SignalingHttpBase(this.signalingWsUrl, {this.apiKey});

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

  Map<String, String> headers({String? bearer}) => {
        'Content-Type': 'application/json',
        if (apiKey != null && apiKey!.isNotEmpty) 'X-API-Key': apiKey!,
        if (bearer != null && bearer.isNotEmpty)
          'Authorization': 'Bearer $bearer',
      };

  /// 发送 JSON 请求；非 200 时抛 [SignalingApiException]，成功返回解码后的 body。
  Future<Map<String, dynamic>> requestJson(
    String method,
    String path, {
    Object? body,
    String? bearer,
    Duration timeout = const Duration(seconds: 10),
  }) async {
    final uri = Uri.parse('$httpBase$path');
    final reqHeaders = headers(bearer: bearer);
    final encoded = body == null ? null : jsonEncode(body);

    final Future<http.Response> request = switch (method) {
      'GET' => http.get(uri, headers: reqHeaders),
      'PUT' => http.put(uri, headers: reqHeaders, body: encoded),
      _ => http.post(uri, headers: reqHeaders, body: encoded),
    };
    final resp = await request.timeout(timeout);

    final json = _decodeBody(resp);
    if (resp.statusCode != 200) {
      throw SignalingApiException(
        resp.statusCode,
        (json['code'] ?? 'UNKNOWN').toString(),
        (json['message'] ?? resp.body).toString(),
      );
    }
    return json;
  }

  /// 获取 ICE 配置（STUN/TURN），失败返回空列表（P2P 仍可尝试主机候选）
  Future<List<IceServerEntry>> fetchIceServers({String? bearer}) async {
    try {
      final json = await requestJson(
        'GET',
        '/v1/ice-config',
        bearer: bearer,
        timeout: const Duration(seconds: 3),
      );
      final list =
          (json['ice_servers'] ?? json['iceServers']) as List<dynamic>? ?? [];
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
