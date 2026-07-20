/// signaling_api.dart - 信令服务器 REST 客户端（主控端所需的最小集合）
///
/// 对照 SignalingServer/docs/user-api-docs.md 与 WebClient 的调用方式：
///   POST /v1/devices/:device_id/access-code:verify → signal_token
///   GET  /v1/ice-config                            → STUN/TURN 列表
library;

import 'signaling_http.dart';

export 'signaling_http.dart' show IceServerEntry, SignalingApiException;

class SignalingApi extends SignalingHttpBase {
  SignalingApi(super.signalingWsUrl, {super.apiKey});

  /// 校验访问码并换取一次性 signal_token
  Future<String> verifyAccessCode(String deviceId, String accessCode) async {
    final json = await requestJson(
      'POST',
      '/v1/devices/$deviceId/access-code:verify',
      body: {'code': accessCode},
    );
    final token = json['signal_token'] as String?;
    if (token == null || token.isEmpty) {
      throw SignalingApiException(200, 'NO_TOKEN', 'missing signal_token');
    }
    return token;
  }

  /// 获取 ICE 配置（STUN/TURN），失败返回空列表
  Future<List<IceServerEntry>> getIceServers() => fetchIceServers();
}
