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

import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:shared_preferences/shared_preferences.dart';

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
      deviceId != null && deviceId!.isNotEmpty && deviceSecret != null && deviceSecret!.isNotEmpty;
}

class HostCredentialStore {
  static const _kUuid = 'host_device_uuid';
  static const _kDeviceId = 'host_device_id';
  static const _kDeviceSecret = 'host_device_secret';
  static const _kAccessCode = 'host_access_code';

  // flutter_secure_storage v10：Android 默认走 Keystore 加密（自定义 cipher），
  // 无需再显式开启 encryptedSharedPreferences（已废弃）。
  static const _secure = FlutterSecureStorage();

  Future<HostCredentials> load() async {
    final prefs = await SharedPreferences.getInstance();
    var uuid = prefs.getString(_kUuid);
    if (uuid == null || uuid.isEmpty) {
      uuid = generateUuidV4();
      await prefs.setString(_kUuid, uuid);
    }

    // 安全存储在个别设备（Keystore 失效等）可能抛异常，读失败按未注册处理。
    String? secret;
    try {
      secret = await _secure.read(key: _kDeviceSecret);
    } catch (_) {
      secret = null;
    }

    return HostCredentials(
      deviceUuid: uuid,
      deviceId: prefs.getString(_kDeviceId),
      deviceSecret: secret,
      accessCode: prefs.getString(_kAccessCode),
    );
  }

  Future<void> saveProvision({required String deviceId, required String deviceSecret}) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_kDeviceId, deviceId);
    await _secure.write(key: _kDeviceSecret, value: deviceSecret);
  }

  Future<void> saveAccessCode(String accessCode) async {
    final prefs = await SharedPreferences.getInstance();
    await prefs.setString(_kAccessCode, accessCode);
  }

  /// 生成一个随机 6 位数字访问码
  static String generateAccessCode() => randomDigits(6);
}
