/// screen_capture.dart - 屏幕采集封装（被控端）
///
/// 通过平台通道启动前台服务（满足 Android 10+/14 的 MediaProjection 约束），
/// 再用 flutter_webrtc 的 getDisplayMedia 触发系统授权并拿到含视频轨的流。
///
/// 顺序很关键：必须**先** startService() 再 getDisplayMedia()，否则
/// Android 14 会因缺少 mediaProjection 前台服务而拒绝采集。
library;

import 'package:flutter/services.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart';

class ScreenCaptureResult {
  final MediaStream stream;
  final int width;
  final int height;

  ScreenCaptureResult({required this.stream, required this.width, required this.height});

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

  /// 启动采集：拉起前台服务 → 请求 MediaProjection → 返回视频流与尺寸。
  /// 失败（用户拒绝授权等）时抛异常，并回滚前台服务。
  Future<ScreenCaptureResult> start() async {
    await _channel.invokeMethod('startService');
    _serviceRunning = true;

    try {
      // Android 端 flutter_webrtc 采集整块默认屏幕，忽略细粒度约束，video:true 即可。
      final stream = await navigator.mediaDevices.getDisplayMedia({
        'video': true,
        'audio': false,
      });
      _stream = stream;

      final size = await _resolveSize(stream);
      return ScreenCaptureResult(stream: stream, width: size.$1, height: size.$2);
    } catch (e) {
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
      final size = await _channel.invokeMethod<Map<dynamic, dynamic>>('getScreenSize');
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
