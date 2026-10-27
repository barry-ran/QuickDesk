import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:quickdesk_android/api/host_api.dart';
import 'package:quickdesk_android/host/host_controller.dart';
import 'package:quickdesk_android/host/host_credential_store.dart';

import 'host_lifecycle_test.dart' show LifecycleCapture, LifecycleInput;

class RotationStore extends HostCredentialStore {
  RotationStore() : super(signalingUrl: 'https://rotation.example');
  String code = '123456';
  int saves = 0;
  @override
  Future<HostCredentials> load() async => HostCredentials(
    deviceUuid: 'uuid',
    deviceId: 'device',
    deviceSecret: 'secret',
    accessCode: code,
  );
  @override
  Future<void> saveAccessCode(String value) async {
    code = value;
    saves++;
  }
}

class RotationApi extends HostApi {
  RotationApi() : super('https://rotation.example');
  Completer<void> gate = Completer<void>();
  int changes = 0;
  String? requestedCode;
  @override
  Future<void> setAccessCode({
    required String deviceId,
    required String deviceSecret,
    required String accessCode,
  }) {
    changes++;
    requestedCode = accessCode;
    return gate.future;
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();

  test(
    'rotation blocks repeated rotation and startup until server commits code',
    () async {
      final store = RotationStore();
      final api = RotationApi();
      final input = LifecycleInput();
      final controller = HostController(
        signalingUrl: 'https://rotation.example',
        credentialStore: store,
        api: api,
        injector: input,
        capture: LifecycleCapture(),
      );
      await controller.loadCredentials();
      final rotation = controller.regenerateAccessCode();
      await controller.regenerateAccessCode();
      await controller.start();
      expect(api.changes, 1);
      expect(input.checks, 0);
      expect(store.code, '123456');
      expect(store.saves, 0);
      api.gate.complete();
      await rotation;
      expect(store.saves, 1);
      expect(store.code, api.requestedCode);
      controller.dispose();
    },
  );

  test(
    'failed rotation retains previous code and clears busy guard for retry',
    () async {
      final store = RotationStore();
      final api = RotationApi();
      final controller = HostController(
        signalingUrl: 'https://rotation.example',
        credentialStore: store,
        api: api,
        injector: LifecycleInput(),
        capture: LifecycleCapture(),
      );
      await controller.loadCredentials();
      final rotation = controller.regenerateAccessCode();
      final failure = expectLater(rotation, throwsStateError);
      api.gate.completeError(StateError('offline'));
      await failure;
      expect(store.code, '123456');
      expect(store.saves, 0);
      api.gate = Completer<void>()..complete();
      await controller.regenerateAccessCode();
      expect(api.changes, 2);
      expect(store.saves, 1);
      controller.dispose();
    },
  );
}
