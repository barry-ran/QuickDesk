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

import 'signaling_http.dart';

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

class HostApi extends SignalingHttpBase {
  HostApi(super.signalingWsUrl, {super.apiKey});

  /// 首次注册设备，换取 device_id + device_secret（device_secret 只此一次可见）
  Future<HostProvisionResult> provision({
    required String deviceUuid,
    String machineFingerprint = '',
    String os = 'Android',
    String osVersion = '',
    String appVersion = '',
  }) async {
    final json = await requestJson('POST', '/v1/devices:provision', body: {
      'device_uuid': deviceUuid,
      'machine_fingerprint': machineFingerprint,
      'os': os,
      'os_version': osVersion,
      'app_version': appVersion,
    });
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
    final json = await requestJson(
      'POST',
      '/v1/devices/$deviceId/heartbeat',
      bearer: deviceSecret,
      body: {
        'os': os,
        'os_version': osVersion,
        'app_version': appVersion,
      },
    );
    return (json['turn_config_version'] as num?)?.toInt() ?? 0;
  }

  /// 申请一次性 host signal_token（用于 WS 首帧 auth，role=host）
  Future<String> issueSignalToken({
    required String deviceId,
    required String deviceSecret,
  }) async {
    final json = await requestJson(
      'POST',
      '/v1/devices/$deviceId/signal-tokens',
      bearer: deviceSecret,
    );
    final token = json['signal_token'] as String?;
    if (token == null || token.isEmpty) {
      throw SignalingApiException(200, 'NO_TOKEN', 'missing signal_token');
    }
    return token;
  }

  /// 设置/更新本设备访问码
  Future<void> setAccessCode({
    required String deviceId,
    required String deviceSecret,
    required String accessCode,
  }) async {
    await requestJson(
      'PUT',
      '/v1/devices/$deviceId/access-code',
      bearer: deviceSecret,
      body: {'access_code': accessCode},
    );
  }

  /// 获取 ICE 配置（device_secret 鉴权）；失败返回空列表
  Future<List<IceServerEntry>> getIceServers({
    required String deviceSecret,
  }) =>
      fetchIceServers(bearer: deviceSecret);
}
