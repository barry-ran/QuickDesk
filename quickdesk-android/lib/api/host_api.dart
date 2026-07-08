/// host_api.dart - 被控端（device/host 角色）REST 客户端
///
/// 对照 SignalingServer/internal/handler/host_handler.go 的设备侧接口：
///   POST /v1/devices:provision                (X-API-Key)        → device_id + device_secret
///   POST /v1/devices/:id/heartbeat            (device_secret)    → server_time / turn_config_version
///   POST /v1/devices/:id/signal-tokens        (device_secret)    → 一次性 host signal_token
///   PUT  /v1/devices/:id/access-code          (device_secret)    → 设置访问码
///   GET  /v1/ice-config                       (device_secret)    → STUN/TURN
///
/// device_secret 通过 `Authorization: Bearer <device_secret>` 提交，仅在 provision
/// 时返回一次，需本地持久化（见 HostCredentialStore）。
library;

import 'dart:async';
import 'dart:convert';

import 'package:http/http.dart' as http;

import 'signaling_api.dart' show IceServerEntry, SignalingApiException;

class HostProvisionResult {
  final String deviceId;
  final String deviceSecret;
  final bool isNew;

  HostProvisionResult({
    required this.deviceId,
    required this.deviceSecret,
    required this.isNew,
  });
}

class HostApi {
  /// 信令服务器 ws:// 或 wss:// 地址（与设置页一致），内部转 http(s)
  final String signalingWsUrl;

  /// provision 需要的 X-API-Key（自建服务器开启保护时）
  final String? apiKey;

  late final String _httpBase = _wsToHttp(signalingWsUrl);

  HostApi(this.signalingWsUrl, {this.apiKey});

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

  Map<String, String> _headers({String? deviceSecret}) => {
        'Content-Type': 'application/json',
        if (apiKey != null && apiKey!.isNotEmpty) 'X-API-Key': apiKey!,
        if (deviceSecret != null && deviceSecret.isNotEmpty)
          'Authorization': 'Bearer $deviceSecret',
      };

  /// 首次注册设备，换取 device_id + device_secret（device_secret 只此一次可见）
  Future<HostProvisionResult> provision({
    required String deviceUuid,
    String machineFingerprint = '',
    String os = 'Android',
    String osVersion = '',
    String appVersion = '',
  }) async {
    final resp = await http
        .post(
          Uri.parse('$_httpBase/v1/devices:provision'),
          headers: _headers(),
          body: jsonEncode({
            'device_uuid': deviceUuid,
            'machine_fingerprint': machineFingerprint,
            'os': os,
            'os_version': osVersion,
            'app_version': appVersion,
          }),
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
    return HostProvisionResult(
      deviceId: (json['device_id'] ?? '').toString(),
      deviceSecret: (json['device_secret'] ?? '').toString(),
      isNew: json['is_new'] == true,
    );
  }

  /// 心跳（刷新在线状态）；返回 turn_config_version 供判断 ICE 是否需重取
  Future<int> heartbeat({
    required String deviceId,
    required String deviceSecret,
    String os = 'Android',
    String osVersion = '',
    String appVersion = '',
  }) async {
    final resp = await http
        .post(
          Uri.parse('$_httpBase/v1/devices/$deviceId/heartbeat'),
          headers: _headers(deviceSecret: deviceSecret),
          body: jsonEncode({
            'os': os,
            'os_version': osVersion,
            'app_version': appVersion,
          }),
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
    return (json['turn_config_version'] as num?)?.toInt() ?? 0;
  }

  /// 申请一次性 host signal_token（用于 WS 首帧 auth，role=host）
  Future<String> issueSignalToken({
    required String deviceId,
    required String deviceSecret,
  }) async {
    final resp = await http
        .post(
          Uri.parse('$_httpBase/v1/devices/$deviceId/signal-tokens'),
          headers: _headers(deviceSecret: deviceSecret),
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

  /// 设置/更新本设备访问码
  Future<void> setAccessCode({
    required String deviceId,
    required String deviceSecret,
    required String accessCode,
  }) async {
    final resp = await http
        .put(
          Uri.parse('$_httpBase/v1/devices/$deviceId/access-code'),
          headers: _headers(deviceSecret: deviceSecret),
          body: jsonEncode({'access_code': accessCode}),
        )
        .timeout(const Duration(seconds: 10));

    if (resp.statusCode != 200) {
      final json = _decodeBody(resp);
      throw SignalingApiException(
        resp.statusCode,
        (json['code'] ?? 'UNKNOWN').toString(),
        (json['message'] ?? resp.body).toString(),
      );
    }
  }

  /// 获取 ICE 配置（device_secret 鉴权）；失败返回空列表
  Future<List<IceServerEntry>> getIceServers({
    required String deviceSecret,
  }) async {
    try {
      final resp = await http
          .get(
            Uri.parse('$_httpBase/v1/ice-config'),
            headers: _headers(deviceSecret: deviceSecret),
          )
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
