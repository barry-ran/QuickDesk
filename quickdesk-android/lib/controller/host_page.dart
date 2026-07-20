/// host_page.dart - 被控端页面：展示设备 ID/访问码、权限引导、上线开关
library;

import 'dart:async';

import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:wakelock_plus/wakelock_plus.dart';

import '../core/app_settings.dart';
import '../host/host_controller.dart';
import '../host/input_injector.dart';
import '../l10n/app_strings.dart';

class HostPage extends StatefulWidget {
  const HostPage({super.key});

  @override
  State<HostPage> createState() => _HostPageState();
}

class _HostPageState extends State<HostPage> {
  HostController? _controller;
  HostStatus _status = HostStatus(stage: HostStage.idle);
  bool _busy = false;
  final List<StreamSubscription<HostStatus>> _subs = [];

  final InputInjector _injector = InputInjector();
  bool _shizukuRunning = false;
  bool _shizukuAvailable = false;

  @override
  void initState() {
    super.initState();
    _init();
    _injector.setShizukuPermissionResultHandler((granted) async {
      if (granted) await _injector.bindShizuku();
      await _refreshShizuku();
    });
    _refreshShizuku();
  }

  Future<void> _refreshShizuku() async {
    final running = await _injector.isShizukuRunning();
    final available = await _injector.isShizukuAvailable();
    if (!mounted) return;
    setState(() {
      _shizukuRunning = running;
      _shizukuAvailable = available;
    });
  }

  Future<void> _enableShizuku() async {
    final hasPerm = await _injector.hasShizukuPermission();
    if (hasPerm) {
      await _injector.bindShizuku();
      await _refreshShizuku();
    } else {
      await _injector.requestShizukuPermission();
      // 结果经 setShizukuPermissionResultHandler 异步回调刷新
    }
  }

  Future<void> _init() async {
    final settings = await AppSettings.load();
    final controller = HostController(
      signalingUrl: settings.signalingUrl,
      apiKey: settings.apiKeyOrNull,
    );
    _subs.add(controller.onStatus.listen((s) {
      if (!mounted) return;
      setState(() => _status = s);
      if (s.stage == HostStage.online) {
        WakelockPlus.enable();
      } else if (s.stage == HostStage.idle || s.stage == HostStage.error) {
        WakelockPlus.disable();
      }
    }));
    final creds = await controller.loadCredentials();
    if (!mounted) return;
    setState(() {
      _controller = controller;
      _status = HostStatus(
        stage: HostStage.idle,
        deviceId: creds.deviceId,
        accessCode: creds.accessCode,
      );
    });
  }

  Future<void> _start() async {
    if (_controller == null) return;
    setState(() => _busy = true);
    await _controller!.start();
    if (mounted) setState(() => _busy = false);
  }

  Future<void> _stop() async {
    if (_controller == null) return;
    setState(() => _busy = true);
    await _controller!.stop();
    if (mounted) setState(() => _busy = false);
  }

  bool get _online => _status.stage == HostStage.online;

  @override
  Widget build(BuildContext context) {
    final scheme = Theme.of(context).colorScheme;

    return SingleChildScrollView(
      padding: const EdgeInsets.all(24),
      child: Column(
        crossAxisAlignment: CrossAxisAlignment.stretch,
        children: [
          const SizedBox(height: 8),
          _buildStatusCard(scheme),
          const SizedBox(height: 24),

          if (_status.deviceId != null) _buildCredentialCard(scheme),

          const SizedBox(height: 16),
          _buildBackendCard(scheme),

          if (_status.stage == HostStage.needAccessibility) ...[
            const SizedBox(height: 16),
            _buildAccessibilityPrompt(scheme),
          ],

          const SizedBox(height: 24),
          FilledButton.icon(
            onPressed: _busy
                ? null
                : (_online ? _stop : _start),
            style: FilledButton.styleFrom(padding: const EdgeInsets.symmetric(vertical: 16)),
            icon: _busy
                ? const SizedBox(width: 18, height: 18, child: CircularProgressIndicator(strokeWidth: 2))
                : Icon(_online ? Icons.stop : Icons.play_arrow),
            label: Text(_online ? L10n.t('host.stop') : L10n.t('host.start')),
          ),

          if (_status.message != null) ...[
            const SizedBox(height: 12),
            Text(
              _status.message!,
              textAlign: TextAlign.center,
              style: TextStyle(
                color: _status.stage == HostStage.error ? scheme.error : scheme.outline,
              ),
            ),
          ],

          const SizedBox(height: 24),
          Text(
            L10n.t('host.hint'),
            style: TextStyle(fontSize: 12, color: scheme.outline),
          ),
        ],
      ),
    );
  }

  Widget _buildStatusCard(ColorScheme scheme) {
    final (icon, color, text) = switch (_status.stage) {
      HostStage.online when _status.peerCount > 0 => (
          Icons.cast_connected,
          scheme.primary,
          L10n.t('host.controlling', {'n': _status.peerCount})
        ),
      HostStage.online => (Icons.wifi_tethering, scheme.primary, L10n.t('host.online')),
      HostStage.error => (Icons.error_outline, scheme.error, L10n.t('host.offline')),
      HostStage.idle => (Icons.cloud_off, scheme.outline, L10n.t('host.notStarted')),
      _ => (Icons.hourglass_top, scheme.tertiary, L10n.t('host.preparing')),
    };

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Row(
          children: [
            Icon(icon, size: 36, color: color),
            const SizedBox(width: 16),
            Expanded(
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.start,
                children: [
                  Text(L10n.t('host.statusTitle'), style: Theme.of(context).textTheme.labelMedium),
                  const SizedBox(height: 4),
                  Text(text, style: Theme.of(context).textTheme.titleMedium?.copyWith(color: color)),
                ],
              ),
            ),
          ],
        ),
      ),
    );
  }

  Widget _buildCredentialCard(ColorScheme scheme) {
    return Card(
      child: Padding(
        padding: const EdgeInsets.all(20),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            _credRow(L10n.t('host.deviceId'), _status.deviceId ?? '-', copyable: true),
            const Divider(height: 24),
            Row(
              children: [
                Expanded(
                    child: _credRow(L10n.t('host.accessCode'),
                        _status.accessCode ?? L10n.t('host.accessCodeUnset'),
                        copyable: true)),
                IconButton(
                  tooltip: L10n.t('host.regenCode'),
                  onPressed: _online || _busy || _controller == null
                      ? null
                      : () => _controller!.regenerateAccessCode(),
                  icon: const Icon(Icons.refresh),
                ),
              ],
            ),
          ],
        ),
      ),
    );
  }

  Widget _credRow(String label, String value, {bool copyable = false}) {
    return Column(
      crossAxisAlignment: CrossAxisAlignment.start,
      children: [
        Text(label, style: Theme.of(context).textTheme.labelMedium),
        const SizedBox(height: 4),
        Row(
          children: [
            Text(
              value,
              style: Theme.of(context).textTheme.headlineSmall?.copyWith(
                    fontFeatures: const [FontFeature.tabularFigures()],
                    letterSpacing: 2,
                  ),
            ),
            if (copyable && value.length > 1) ...[
              const SizedBox(width: 8),
              IconButton(
                iconSize: 18,
                visualDensity: VisualDensity.compact,
                onPressed: () {
                  Clipboard.setData(ClipboardData(text: value));
                  ScaffoldMessenger.of(context).showSnackBar(
                    SnackBar(
                        content: Text(L10n.t('common.copied', {'label': label})),
                        duration: const Duration(seconds: 1)),
                  );
                },
                icon: const Icon(Icons.copy),
              ),
            ],
          ],
        ),
      ],
    );
  }

  Widget _buildBackendCard(ColorScheme scheme) {
    final (label, desc) = _shizukuAvailable
        ? (L10n.t('host.backendShizuku'), L10n.t('host.backendShizukuDesc'))
        : (L10n.t('host.backendA11y'), L10n.t('host.backendA11yDesc'));

    return Card(
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(
                  _shizukuAvailable ? Icons.bolt : Icons.touch_app,
                  color: _shizukuAvailable ? scheme.primary : scheme.outline,
                ),
                const SizedBox(width: 8),
                Expanded(
                  child: Column(
                    crossAxisAlignment: CrossAxisAlignment.start,
                    children: [
                      Text(L10n.t('host.inputMethod', {'label': label}),
                          style: const TextStyle(fontWeight: FontWeight.bold)),
                      const SizedBox(height: 2),
                      Text(desc, style: TextStyle(fontSize: 12, color: scheme.outline)),
                    ],
                  ),
                ),
              ],
            ),
            if (!_shizukuAvailable) ...[
              const SizedBox(height: 8),
              if (_shizukuRunning)
                Align(
                  alignment: Alignment.centerLeft,
                  child: FilledButton.tonal(
                    onPressed: _enableShizuku,
                    child: Text(L10n.t('host.enableShizuku')),
                  ),
                )
              else
                Text(
                  L10n.t('host.shizukuHint'),
                  style: TextStyle(fontSize: 12, color: scheme.outline),
                ),
            ],
          ],
        ),
      ),
    );
  }

  Widget _buildAccessibilityPrompt(ColorScheme scheme) {
    return Card(
      color: scheme.errorContainer,
      child: Padding(
        padding: const EdgeInsets.all(16),
        child: Column(
          crossAxisAlignment: CrossAxisAlignment.start,
          children: [
            Row(
              children: [
                Icon(Icons.accessibility_new, color: scheme.onErrorContainer),
                const SizedBox(width: 8),
                Expanded(
                  child: Text(
                    L10n.t('host.a11yTitle'),
                    style: TextStyle(color: scheme.onErrorContainer, fontWeight: FontWeight.bold),
                  ),
                ),
              ],
            ),
            const SizedBox(height: 8),
            Text(
              L10n.t('host.a11yDesc'),
              style: TextStyle(color: scheme.onErrorContainer, fontSize: 13),
            ),
            const SizedBox(height: 12),
            FilledButton.tonal(
              onPressed: () => _controller?.openAccessibilitySettings(),
              child: Text(L10n.t('host.a11yGo')),
            ),
          ],
        ),
      ),
    );
  }

  @override
  void dispose() {
    for (final s in _subs) {
      s.cancel();
    }
    _controller?.dispose();
    WakelockPlus.disable();
    super.dispose();
  }
}
