/// host_controller.dart - 被控端总控
///
/// 串起：设备注册(provision) → 访问码同步 → 屏幕采集 → 申请 host signal_token
/// → 连接信令监听(HostSession) → 心跳保活；并把收到的输入事件转交注入层。
///
/// 状态机对上层 UI 暴露 HostStatus 流。
library;

import 'dart:async';

import 'package:flutter/services.dart';

import '../api/host_api.dart';
import '../api/signaling_api.dart' show SignalingApiException;
import '../l10n/app_strings.dart';
import '../protocol/auth/spake2.dart' as spake2;
import '../protocol/host_session.dart';
import 'host_credential_store.dart';
import 'input_injector.dart';
import 'screen_capture.dart';

enum HostStage {
  idle,
  provisioning,
  needAccessibility, // 无障碍未开启，等待用户授权
  requestingCapture, // 等待屏幕采集授权
  connecting, // 连接信令
  online, // 已上线待命/被控中
  error,
}

class HostStatus {
  final HostStage stage;
  final String? deviceId;
  final String? accessCode;
  final int peerCount;
  final String? message;

  HostStatus({
    required this.stage,
    this.deviceId,
    this.accessCode,
    this.peerCount = 0,
    this.message,
  });
}

class HostController {
  static const _identityChannel = MethodChannel('quickdesk/host_identity');

  final String signalingUrl;
  final String? apiKey;

  final HostCredentialStore? _providedStore;
  final HostApi? _providedApi;
  late final HostCredentialStore _store =
      _providedStore ??
      HostCredentialStore(signalingUrl: signalingUrl, apiKey: apiKey);
  final ScreenCapture _capture;
  final InputInjector _injector;
  late final HostApi _api =
      _providedApi ?? HostApi(signalingUrl, apiKey: apiKey);

  Future<void>? _starting;
  Future<void>? _stopping;
  int _generation = 0;
  bool _disposed = false;

  HostCredentials? _creds;
  HostSession? _session;
  Timer? _heartbeatTimer;
  final List<StreamSubscription> _subs = [];

  final _statusCtrl = StreamController<HostStatus>.broadcast();
  Stream<HostStatus> get onStatus => _statusCtrl.stream;

  HostStage _stage = HostStage.idle;
  int _peerCount = 0;

  HostController({
    required this.signalingUrl,
    this.apiKey,
    HostCredentialStore? credentialStore,
    HostApi? api,
    ScreenCapture? capture,
    InputInjector? injector,
  }) : _providedStore = credentialStore,
       _providedApi = api,
       _capture = capture ?? ScreenCapture(),
       _injector = injector ?? InputInjector();

  String? get deviceId => _creds?.deviceId;
  String? get accessCode => _creds?.accessCode;

  Future<HostCredentials> loadCredentials() async {
    _creds = await _store.load();
    return _creds!;
  }

  /// 启动被控：完整流程。调用前应确保无障碍已开启（否则会停在 needAccessibility）。
  Future<void> start() async {
    if (_disposed || _starting != null || _stopping != null || _session != null)
      return;
    final generation = ++_generation;
    final work = _start(generation);
    _starting = work;
    try {
      await work;
    } finally {
      _starting = null;
    }
  }

  void _check(int generation) {
    if (_disposed || generation != _generation) throw const _HostCancelled();
  }

  Future<void> _start(int generation) async {
    try {
      _creds ??= await _store.load();
      _check(generation);

      // 0. 无障碍开关检查（输入注入前提）
      _emit(HostStage.requestingCapture, message: L10n.t('host.msgCheckA11y'));
      final a11y = await _injector.isAccessibilityEnabled();
      _check(generation);
      if (!a11y) {
        _emit(HostStage.needAccessibility, message: L10n.t('host.msgNeedA11y'));
        return;
      }

      // 1. 设备注册（幂等：已注册则复用本地凭据）
      if (!_creds!.isProvisioned) {
        _emit(HostStage.provisioning, message: L10n.t('host.msgProvisioning'));
        final result = await _api.provision(deviceUuid: _creds!.deviceUuid);
        // Persist the one-time secret even if stop was requested in flight.
        await _store.saveProvision(
          deviceId: result.deviceId,
          deviceSecret: result.deviceSecret,
        );
        _check(generation);
        _creds = await _store.load();
        _check(generation);
      }

      // 2. 访问码（无则生成并上报）
      var accessCode = _creds!.accessCode;
      if (accessCode == null || accessCode.isEmpty) {
        accessCode = HostCredentialStore.generateAccessCode();
        await _api.setAccessCode(
          deviceId: _creds!.deviceId!,
          deviceSecret: _creds!.deviceSecret!,
          accessCode: accessCode,
        );
        await _store.saveAccessCode(accessCode);
        _creds = await _store.load();
        _check(generation);
      }

      // 3. 屏幕采集授权
      _emit(
        HostStage.requestingCapture,
        message: L10n.t('host.msgRequestCapture'),
      );
      // 旋转/分辨率变化 → 更新会话屏幕尺寸并向所有客户端重发 VideoLayout
      _capture.onSizeChanged = (w, h) => _session?.updateScreenSize(w, h);
      final capture = await _capture.start();
      _check(generation);

      // 4. ICE 配置 + host signal_token
      _emit(HostStage.connecting, message: L10n.t('host.msgConnecting'));
      final iceServers = await _api.getIceServers(
        deviceSecret: _creds!.deviceSecret!,
      );
      _check(generation);
      final signalToken = await _api.issueSignalToken(
        deviceId: _creds!.deviceId!,
        deviceSecret: _creds!.deviceSecret!,
      );

      _check(generation);

      // 5. Host 身份证书 + SPAKE2 共享密钥哈希。
      final hostCertificate = await _identityChannel.invokeMethod<String>(
        'getCertificate',
      );
      _check(generation);
      if (hostCertificate == null || hostCertificate.isEmpty) {
        throw StateError('Android host certificate is unavailable');
      }
      final deviceId = _creds!.deviceId!;
      final sharedSecretHash = spake2.getSharedSecretHash(
        deviceId,
        deviceId + accessCode,
      );

      // 6. 建立 HostSession
      final session = HostSession(
        signalingUrl: signalingUrl,
        deviceId: deviceId,
        sharedSecretHash: sharedSecretHash,
        hostCertificate: hostCertificate,
        screenStreamProvider: () => capture.stream,
        screenWidth: capture.width,
        screenHeight: capture.height,
        iceServers: iceServers,
        capabilities: '',
        onInput: (event) => _injector.inject(event),
      );
      _session = session;

      _subs.add(
        session.onPeerCountChange.listen((n) {
          _peerCount = n;
          _emit(HostStage.online);
        }),
      );
      _subs.add(
        session.onStateChange.listen((s) {
          if (s == HostSessionState.failed) {
            _emit(HostStage.error, message: L10n.t('host.msgSignalDropped'));
          }
        }),
      );

      await session.listen(signalToken);
      _check(generation);

      // 7. 心跳保活
      _startHeartbeat();
      _emit(HostStage.online, message: L10n.t('host.msgOnline'));
    } on _HostCancelled {
      // stop() waits for startup to settle, then releases capture/session.
      return;
    } on SignalingApiException catch (e) {
      if (generation != _generation || _disposed) return;
      _emit(
        HostStage.error,
        message: L10n.t('host.msgServerError', {'code': e.code}),
      );
      await _cleanup();
    } catch (e) {
      if (generation != _generation || _disposed) return;
      _emit(
        HostStage.error,
        message: L10n.t('host.msgStartFailed', {'e': '$e'}),
      );
      await _cleanup();
    }
  }

  /// 用户手动重设访问码
  Future<void> regenerateAccessCode() async {
    if (_creds == null || !_creds!.isProvisioned) return;
    final code = HostCredentialStore.generateAccessCode();
    await _api.setAccessCode(
      deviceId: _creds!.deviceId!,
      deviceSecret: _creds!.deviceSecret!,
      accessCode: code,
    );
    await _store.saveAccessCode(code);
    _creds = await _store.load();
    _emit(_stage);
  }

  void _startHeartbeat() {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = Timer.periodic(const Duration(seconds: 30), (_) async {
      if (_creds == null || !_creds!.isProvisioned) return;
      try {
        await _api.heartbeat(
          deviceId: _creds!.deviceId!,
          deviceSecret: _creds!.deviceSecret!,
        );
      } catch (_) {
        // 单次心跳失败忽略；连接状态由信令 WS 反映
      }
    });
  }

  Future<void> stop() {
    ++_generation;
    return _stopping ??= _stop().whenComplete(() => _stopping = null);
  }

  Future<void> _stop() async {
    await _starting;
    await _cleanup();
  }

  Future<void> _cleanup() async {
    _heartbeatTimer?.cancel();
    _heartbeatTimer = null;
    for (final s in _subs) {
      await s.cancel();
    }
    _subs.clear();
    await _session?.stop();
    _session?.dispose();
    _session = null;
    await _capture.stop();
    _peerCount = 0;
    if (_stage != HostStage.error) {
      _emit(HostStage.idle);
    }
  }

  Future<void> openAccessibilitySettings() =>
      _injector.openAccessibilitySettings();

  void _emit(HostStage stage, {String? message}) {
    _stage = stage;
    if (_disposed || _statusCtrl.isClosed) return;
    _statusCtrl.add(
      HostStatus(
        stage: stage,
        deviceId: _creds?.deviceId,
        accessCode: _creds?.accessCode,
        peerCount: _peerCount,
        message: message,
      ),
    );
  }

  void dispose() {
    _disposed = true;
    unawaited(stop().whenComplete(_statusCtrl.close));
  }
}

class _HostCancelled implements Exception {
  const _HostCancelled();
}
