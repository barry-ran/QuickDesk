/// host_credential_store.dart - 被控端设备凭据本地持久化
///
/// device_secret 仅在 provision 时返回一次，必须落盘复用；device_id 展示给
/// 主控方输入；access_code 由用户设置并同步到服务器。
///
/// 敏感度分级：
///   - device_secret：高敏感 → flutter_secure_storage（Android 走 Keystore 加密）
///   - device_uuid / device_id / access_code：低敏感（access_code 本就在 UI 明示）
///     → shared_preferences
library;

import 'dart:convert';

import 'package:crypto/crypto.dart';

import '../account/account_service.dart';

import 'package:shared_preferences/shared_preferences.dart';
import 'package:flutter/foundation.dart';

import '../core/rand_id.dart';

class HostCredentials {
  final String deviceUuid;
  final String? deviceId;
  final String? deviceSecret;
  final String? accessCode;

  HostCredentials({
    required this.deviceUuid,
    this.deviceId,
    this.deviceSecret,
    this.accessCode,
  });

  bool get isProvisioned =>
      deviceId != null &&
      deviceId!.isNotEmpty &&
      deviceSecret != null &&
      deviceSecret!.isNotEmpty;
}

class HostCredentialStore {
  final AccountTokenStore secure;
  final String scope;
  static final changes = ValueNotifier<int>(0);

  HostCredentialStore({
    required String signalingUrl,
    String? apiKey,
    AccountTokenStore? secure,
  }) : secure = secure ?? const SecureAccountTokenStore(),
       scope = scopeFor(signalingUrl, apiKey);

  /// No migration: legacy host credentials have no trustworthy server origin.
  /// A fresh UUID prevents provisioning the old device on another server.
  static String scopeFor(String url, String? apiKey) {
    final uri = Uri.parse(url.trim());
    final base = uri
        .replace(
          scheme: uri.scheme == 'wss'
              ? 'https'
              : uri.scheme == 'ws'
              ? 'http'
              : uri.scheme,
        )
        .toString()
        .replaceFirst(RegExp(r'/+$'), '');
    return 'host_v2_${sha256.convert(utf8.encode('$base\n${apiKey ?? ''}'))}';
  }

  String get _kUuid => '${scope}_uuid';
  String get _kDeviceId => '${scope}_id';
  String get _kDeviceSecret => '${scope}_secret';
  String get _kAccessCode => '${scope}_access_code';

  Future<HostCredentials> load() async {
    final prefs = await SharedPreferences.getInstance();
    var uuid = prefs.getString(_kUuid);
    if (uuid == null || uuid.isEmpty) {
      uuid = generateUuidV4();
      await prefs.setString(_kUuid, uuid);
    }

    // Fail closed: a Keystore error must not silently reprovision an existing host.
    String? secret;
    try {
      secret = await secure.read(_kDeviceSecret);
    } catch (_) {
      throw const AccountException(
        0,
        'SECURE_STORAGE',
        'Unable to read host credentials securely.',
      );
    }

    return HostCredentials(
      deviceUuid: uuid,
      deviceId: prefs.getString(_kDeviceId),
      deviceSecret: secret,
      accessCode: prefs.getString(_kAccessCode),
    );
  }

  Future<void> saveProvision({
    required String deviceId,
    required String deviceSecret,
  }) async {
    final prefs = await SharedPreferences.getInstance();
    await secure.write(_kDeviceSecret, deviceSecret);
    await prefs.setString(_kDeviceId, deviceId);
    changes.value++;
  }

  Future<void> saveAccessCode(String accessCode) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_kAccessCode, accessCode);
    changes.value++;
  }

  /// 生成一个随机 6 位数字访问码
  static String generateAccessCode() => randomDigits(6);
}
