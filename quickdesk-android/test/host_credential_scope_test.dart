import 'package:flutter_test/flutter_test.dart';
import 'package:shared_preferences/shared_preferences.dart';
import 'package:quickdesk_android/account/account_service.dart';
import 'package:quickdesk_android/host/host_credential_store.dart';

class MemorySecrets implements AccountTokenStore {
  final values = <String, String>{};
  bool fail = false;
  @override
  Future<String?> read(String key) async {
    if (fail) throw StateError('Keystore unavailable');
    return values[key];
  }

  @override
  Future<void> write(String key, String value) async {
    if (fail) throw StateError('Keystore unavailable');
    values[key] = value;
  }

  @override
  Future<void> delete(String key) async {
    values.remove(key);
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  setUp(() => SharedPreferences.setMockInitialValues({}));
  test(
    'host identity/secret/code are isolated by canonical server and API key',
    () async {
      final secrets = MemorySecrets();
      HostCredentialStore store(String url, [String? key]) =>
          HostCredentialStore(signalingUrl: url, apiKey: key, secure: secrets);
      final a = store('wss://a.example/');
      final uuid = (await a.load()).deviceUuid;
      await a.saveProvision(deviceId: '123', deviceSecret: 'secret-A');
      await a.saveAccessCode('654321');
      final restored = await store('https://a.example').load();
      expect(restored.deviceUuid, uuid);
      expect(restored.deviceSecret, 'secret-A');
      expect(restored.accessCode, '654321');
      for (final other in [
        store('wss://b.example'),
        store('wss://a.example', 'other-key'),
      ]) {
        final credentials = await other.load();
        expect(credentials.deviceUuid, isNot(uuid));
        expect(credentials.isProvisioned, isFalse);
        expect(credentials.deviceSecret, isNull);
        expect(credentials.accessCode, isNull);
      }
    },
  );
  test('legacy unscoped identity and secret are never adopted', () async {
    SharedPreferences.setMockInitialValues({
      'host_device_uuid': 'legacy-uuid',
      'host_device_id': 'legacy-id',
      'host_access_code': 'legacy-code',
    });
    final secrets = MemorySecrets()
      ..values['host_device_secret'] = 'legacy-secret';
    final credentials = await HostCredentialStore(
      signalingUrl: 'https://a.example',
      secure: secrets,
    ).load();
    expect(credentials.deviceUuid, isNot('legacy-uuid'));
    expect(credentials.deviceId, isNull);
    expect(credentials.deviceSecret, isNull);
    expect(credentials.accessCode, isNull);
    expect(secrets.values, {'host_device_secret': 'legacy-secret'});
  });
  test(
    'secure storage read failure fails closed rather than reprovisioning',
    () async {
      final secrets = MemorySecrets()..fail = true;
      final store = HostCredentialStore(
        signalingUrl: 'https://a.example',
        secure: secrets,
      );
      await expectLater(store.load(), throwsA(isA<AccountException>()));
    },
  );
  test('failed secret write never publishes device ID', () async {
    final secrets = MemorySecrets()..fail = true;
    final store = HostCredentialStore(
      signalingUrl: 'https://a.example',
      secure: secrets,
    );
    await expectLater(
      store.saveProvision(deviceId: '123', deviceSecret: 'secret'),
      throwsStateError,
    );
    secrets.fail = false;
    expect((await store.load()).deviceId, isNull);
  });
}
