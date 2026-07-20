import 'package:flutter/material.dart';
import 'package:flutter/services.dart';
import 'package:flutter_webrtc/flutter_webrtc.dart' show WebRTC;

import 'controller/home_page.dart';
import 'l10n/app_strings.dart';
import 'l10n/locale_controller.dart';
import 'theme/app_theme.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await _initializeWebRtc();
  await LocaleController.instance.load();
  runApp(const QuickDeskApp());
}

/// 初始化 WebRTC 并注入被控端必需的 field trial。
///
/// Chromium remoting 客户端(桌面 quickdesk_client)在
/// WebrtcVideoRendererAdapter::OnFrame 中断言收到帧的渲染时间不晚于当前
/// 时刻(NOTREACHED,release 下同样致命),前提是 Host 把每帧的
/// playout-delay RTP 扩展强制为 {0,0}(桌面 host 在
/// webrtc_video_encoder_wrapper.cc 里 SetPlayoutDelay(Minimal))。
/// Android 被控端走标准 libwebrtc 编码器,没有逐帧 API 可设置该扩展,
/// 只能用发送端 field trial WebRTC-ForceSendPlayoutDelay 注入;缺了它,
/// 桌面端 jitter buffer 会给帧排出"未来"的渲染时间,第一帧之后立刻命中
/// 断言崩溃,表现为"出一帧画面马上断开"。
///
/// 顺序要求:flutter_webrtc 的原生初始化会以空字符串调用
/// PeerConnectionFactory.initialize 重置全局 field trials,因此必须先
/// await WebRTC.initialize(),再注入,否则会被覆盖。
Future<void> _initializeWebRtc() async {
  try {
    await WebRTC.initialize();
    await const MethodChannel('quickdesk/webrtc_config')
        .invokeMethod('applyFieldTrials');
  } catch (e) {
    // 注入失败不阻塞启动:主控角色不受影响,但被控角色对桌面端会复现
    // "首帧后断开",此日志用于定位。
    // ignore: avoid_print
    print('[main] WebRTC field trial injection failed: $e');
  }
}

class QuickDeskApp extends StatelessWidget {
  const QuickDeskApp({super.key});

  @override
  Widget build(BuildContext context) {
    return ValueListenableBuilder<AppLocale>(
      valueListenable: LocaleController.instance.locale,
      builder: (context, _, __) {
        return MaterialApp(
          title: 'QuickDesk',
          debugShowCheckedModeBanner: false,
          theme: AppTheme.light(),
          darkTheme: AppTheme.dark(),
          home: const HomePage(),
        );
      },
    );
  }
}
