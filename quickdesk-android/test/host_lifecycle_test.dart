import 'dart:async';

import 'package:flutter_test/flutter_test.dart';
import 'package:quickdesk_android/api/host_api.dart';
import 'package:quickdesk_android/host/host_controller.dart';
import 'package:quickdesk_android/host/host_credential_store.dart';
import 'package:quickdesk_android/host/input_injector.dart';
import 'package:quickdesk_android/host/screen_capture.dart';

class LifecycleStore extends HostCredentialStore {
  LifecycleStore() : super(signalingUrl: 'https://a.example');
  HostCredentials credentials = HostCredentials(deviceUuid: 'uuid');
  int saved = 0;
  Completer<void>? loadGate;
  @override
  Future<HostCredentials> load() async {
    await loadGate?.future;
    return credentials;
  }

  @override
  Future<void> saveProvision({
    required String deviceId,
    required String deviceSecret,
  }) async {
    saved++;
    credentials = HostCredentials(
      deviceUuid: 'uuid',
      deviceId: deviceId,
      deviceSecret: deviceSecret,
    );
  }
}

class LifecycleInput extends InputInjector {
  final gate = Completer<bool>();
  int checks = 0;
  @override
  Future<bool> isAccessibilityEnabled() {
    checks++;
    return gate.future;
  }
}

class LifecycleApi extends HostApi {
  LifecycleApi() : super('https://a.example');
  final provisionGate = Completer<HostProvisionResult>();
  final provisionStarted = Completer<void>();
  int calls = 0;
  @override
  Future<HostProvisionResult> provision({
    required String deviceUuid,
    String machineFingerprint = '',
    String os = 'Android',
    String osVersion = '',
    String appVersion = '',
  }) {
    calls++;
    provisionStarted.complete();
    return provisionGate.future;
  }
}

class LifecycleCapture extends ScreenCapture {
  int starts = 0;
  int stops = 0;
  @override
  Future<ScreenCaptureResult> start() async {
    starts++;
    throw StateError('Must not capture');
  }

  @override
  Future<void> stop() async {
    stops++;
  }
}

void main() {
  TestWidgetsFlutterBinding.ensureInitialized();
  test(
    'stop during credential load cannot progress to permission or capture',
    () async {
      final store = LifecycleStore()..loadGate = Completer<void>();
      final input = LifecycleInput();
      final capture = LifecycleCapture();
      final api = LifecycleApi();
      final controller = HostController(
        signalingUrl: 'https://a.example',
        credentialStore: store,
        injector: input,
        capture: capture,
        api: api,
      );
      final start = controller.start();
      final stop = controller.stop();
      store.loadGate!.complete();
      await start;
      await stop;
      expect(input.checks, 0);
      expect(api.calls, 0);
      expect(capture.starts, 0);
      controller.dispose();
    },
  );
  test('repeated start is coalesced and stop invalidates pending permission result', () async {
    final store = LifecycleStore();
    final input = LifecycleInput();
    final capture = LifecycleCapture();
    final api = LifecycleApi();
    final controller = HostController(
      signalingUrl: 'https://a.example',
      credentialStore: store,
      injector: input,
      capture: capture,
      api: api,
    );
    final start = controller.start();
    await Future<void>.delayed(Duration.zero);
    await controller.start();
    final stop = controller.stop();
    input.gate.complete(true);
    await start;
    await stop;
    expect(input.checks, 1);
    expect(api.calls, 0);
    expect(capture.starts, 0);
    controller.dispose();
  });
  test(
    'late provision secret is saved in original scope without restarting host',
    () async {
      final store = LifecycleStore();
      final input = LifecycleInput()..gate.complete(true);
      final capture = LifecycleCapture();
      final api = LifecycleApi();
      final controller = HostController(
        signalingUrl: 'https://a.example',
        credentialStore: store,
        injector: input,
        capture: capture,
        api: api,
      );
      final stages = <HostStage>[];
      final subscription = controller.onStatus.listen(
        (s) => stages.add(s.stage),
      );
      final start = controller.start();
      await api.provisionStarted.future;
      final stop = controller.stop();
      api.provisionGate.complete(
        HostProvisionResult(
          deviceId: '123',
          deviceSecret: 'secret',
          isNew: true,
        ),
      );
      await start;
      await stop;
      expect(store.saved, 1);
      expect(store.credentials.deviceSecret, 'secret');
      expect(capture.starts, 0);
      expect(stages, isNot(contains(HostStage.online)));
      await subscription.cancel();
      controller.dispose();
    },
  );
}
