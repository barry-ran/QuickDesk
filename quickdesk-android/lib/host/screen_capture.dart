/// screen_capture.dart - 屏幕采集封装（被控端）
///
/// 先通过 flutter_webrtc 请求 MediaProjection 用户授权，再启动
/// mediaProjection 类型前台服务，最后用缓存的授权结果创建屏幕流。
///
/// Android 14+ 会在授权前启动该类型前台服务时抛出 SecurityException；
/// 同时必须等服务真正调用 startForeground 后，才能调用 getMediaProjection。
library;

import 'package:flutter/services.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';

class ScreenCaptureResult {
  final MediaStream stream;
  final int width;
  final int height;

  ScreenCaptureResult(
      {required this.stream, required this.width, required this.height});

  /// 采集流的真实 id（用于 VideoLayout.mediaStreamId，与 SDP msid 对齐）
  String get streamId => stream.id;
}

class ScreenCapture {
  static const _channel = MethodChannel('quickdesk/screen_capture');

  MediaStream? _stream;
  bool _serviceRunning = false;

  /// 屏幕旋转/分辨率变化回调（原生 DisplayListener 触发）
  void Function(int width, int height)? onSizeChanged;

  ScreenCapture() {
    _channel.setMethodCallHandler((call) async {
      if (call.method == 'displayChanged') {
        final args = call.arguments as Map?;
        final w = _asInt(args?['width']);
        final h = _asInt(args?['height']);
        if (w > 0 && h > 0) onSizeChanged?.call(w, h);
      }
      return null;
    });
  }

  MediaStream? get stream => _stream;

  /// 启动采集：请求 MediaProjection 授权 → 启动并确认前台服务 → 创建屏幕流。
  /// 失败（用户拒绝授权等）时抛异常，并回滚前台服务。
  Future<ScreenCaptureResult> start() async {
    try {
      // Android 14+ 必须先取得本次捕获授权，才能启动 mediaProjection 类型服务。
      // fullScreenOnly 避免用户选中单个应用窗口，远控需要完整设备画面。
      final granted =
          await Helper.requestCapturePermission(fullScreenOnly: true);
      if (!granted) {
        throw PlatformException(
          code: 'SCREEN_CAPTURE_PERMISSION_DENIED',
          message: 'Screen capture permission was denied',
        );
      }

      // 等原生服务实际完成 startForeground 后再创建 MediaProjection，不能只等待
      // startForegroundService() 返回，否则服务启动与 getMediaProjection() 存在竞态。
      await _channel.invokeMethod<bool>('startService');
      _serviceRunning = true;

      // requestCapturePermission 已把授权结果缓存到 flutter_webrtc，调用时不会再次弹窗。
      final stream = await navigator.mediaDevices.getDisplayMedia({
        'video': true,
        'audio': false,
      });
      _stream = stream;

      final size = await _resolveSize(stream);
      return ScreenCaptureResult(
          stream: stream, width: size.$1, height: size.$2);
    } catch (_) {
      await stop();
      rethrow;
    }
  }

  /// 读取采集分辨率。flutter_webrtc 的屏幕轨不一定暴露尺寸，
  /// 优先取 track settings，缺失则回退到原生真实显示尺寸。
  Future<(int, int)> _resolveSize(MediaStream stream) async {
    final tracks = stream.getVideoTracks();
    if (tracks.isNotEmpty) {
      try {
        final settings = tracks.first.getSettings();
        final w = _asInt(settings['width']);
        final h = _asInt(settings['height']);
        if (w > 0 && h > 0) return (w, h);
      } catch (_) {}
    }
    // 回退：原生 WindowManager 真实屏幕尺寸
    try {
      final size =
          await _channel.invokeMethod<Map<dynamic, dynamic>>('getScreenSize');
      if (size != null) {
        final w = _asInt(size['width']);
        final h = _asInt(size['height']);
        if (w > 0 && h > 0) return (w, h);
      }
    } catch (_) {}
    return (0, 0);
  }

  int _asInt(Object? v) {
    if (v is int) return v;
    if (v is double) return v.round();
    if (v is String) return int.tryParse(v) ?? 0;
    return 0;
  }

  Future<void> stop() async {
    try {
      final tracks = _stream?.getTracks() ?? [];
      for (final t in tracks) {
        await t.stop();
      }
      await _stream?.dispose();
    } catch (_) {}
    _stream = null;

    if (_serviceRunning) {
      try {
        await _channel.invokeMethod('stopService');
      } catch (_) {}
      _serviceRunning = false;
    }
  }
}
