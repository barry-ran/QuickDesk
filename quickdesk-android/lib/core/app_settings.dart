/// app_settings.dart - 全局设置（信令服务器地址 / API Key / 最近设备）
///
/// 主控端连接页与被控端 HostController 共用同一份服务器配置。
library;

import 'package:flutter/foundation.dart';

import 'package:shared_preferences/shared_preferences.dart';

const kDefaultSignalingUrl = 'ws://qdsignaling.quickcoder.cc:8060';
const _kLegacyDefaultSignalingUrl = 'wss://qd.quickcoder.cn';
const kBuildTimeApiKey = String.fromEnvironment('QUICKDESK_API_KEY');

class AppSettings {
  static final changes = ValueNotifier<int>(0);
  static const _kSignalingUrl = 'signaling_url';
  static const _kApiKey = 'api_key';
  static const _kLastDeviceId = 'last_device_id';

  final String signalingUrl;
  final String apiKey;
  final String lastDeviceId;

  AppSettings({
    required this.signalingUrl,
    required this.apiKey,
    required this.lastDeviceId,
  });

  /// API Key 为空时返回 null（API 客户端以 null 表示"未启用"）
  String? get apiKeyOrNull => apiKey.isEmpty ? null : apiKey;

  static Future<AppSettings> load() async {
    final prefs = await SharedPreferences.getInstance();
    var signalingUrl = prefs.getString(_kSignalingUrl) ?? kDefaultSignalingUrl;
    if (signalingUrl == _kLegacyDefaultSignalingUrl) {
      signalingUrl = kDefaultSignalingUrl;
      await prefs.setString(_kSignalingUrl, signalingUrl);
    }
    return AppSettings(
      signalingUrl: signalingUrl,
      apiKey: prefs.getString(_kApiKey) ?? kBuildTimeApiKey,
      lastDeviceId: prefs.getString(_kLastDeviceId) ?? '',
    );
  }

  static Future<void> save({
    required String signalingUrl,
    required String apiKey,
    String? lastDeviceId,
  }) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_kSignalingUrl, signalingUrl.trim());
    await prefs.setString(_kApiKey, apiKey.trim());
    changes.value++;
    if (lastDeviceId != null) {
      await prefs.setString(_kLastDeviceId, lastDeviceId.trim());
    }
  }
}
