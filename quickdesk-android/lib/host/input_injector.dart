/// input_injector.dart - 输入注入桥接（被控端）
///
/// 把 HostSession 收到的 HostInputEvent 通过平台通道下发到原生无障碍服务
/// （InputAccessibilityService）执行。同时封装无障碍开关状态查询与引导。
library;

import 'package:flutter/services.dart';

import '../protocol/host_input.dart';

class InputInjector {
  static const _channel = MethodChannel('quickdesk/input');

  /// 无障碍服务是否已启用（被控输入的前提）
  Future<bool> isAccessibilityEnabled() async {
    try {
      return await _channel.invokeMethod<bool>('isAccessibilityEnabled') ?? false;
    } catch (_) {
      return false;
    }
  }

  /// 打开系统无障碍设置页，引导用户启用
  Future<void> openAccessibilitySettings() async {
    try {
      await _channel.invokeMethod('openAccessibilitySettings');
    } catch (_) {}
  }

  /// Shizuku 是否安装且服务在运行
  Future<bool> isShizukuRunning() async {
    try {
      return await _channel.invokeMethod<bool>('isShizukuRunning') ?? false;
    } catch (_) {
      return false;
    }
  }

  /// 是否已获得 Shizuku 授权
  Future<bool> hasShizukuPermission() async {
    try {
      return await _channel.invokeMethod<bool>('hasShizukuPermission') ?? false;
    } catch (_) {
      return false;
    }
  }

  /// Shizuku 增强档是否可用（已授权且 UserService 已绑定）
  Future<bool> isShizukuAvailable() async {
    try {
      return await _channel.invokeMethod<bool>('isShizukuAvailable') ?? false;
    } catch (_) {
      return false;
    }
  }

  /// 发起 Shizuku 权限申请（结果通过原生监听器异步回调）
  Future<void> requestShizukuPermission() async {
    try {
      await _channel.invokeMethod('requestShizukuPermission');
    } catch (_) {}
  }

  /// 绑定 Shizuku UserService（已授权后调用）
  Future<void> bindShizuku() async {
    try {
      await _channel.invokeMethod('bindShizuku');
    } catch (_) {}
  }

  /// 注册原生回调：Shizuku 权限申请结果
  void setShizukuPermissionResultHandler(void Function(bool granted) handler) {
    _channel.setMethodCallHandler((call) async {
      if (call.method == 'shizukuPermissionResult') {
        handler(call.arguments == true);
      }
      return null;
    });
  }

  /// 分发一条输入事件到原生注入层
  Future<void> inject(HostInputEvent event) async {
    try {
      switch (event.type) {
        case HostInputType.mouse:
          await _channel.invokeMethod('injectMouse', {
            'x': event.x ?? 0,
            'y': event.y ?? 0,
            'button': event.button ?? 0,
            // buttonDown: 1=down 0=up -1=仅移动
            'buttonDown': event.buttonDown == null ? -1 : (event.buttonDown! ? 1 : 0),
            'wheelDeltaY': event.wheelDeltaY ?? 0.0,
          });
          break;
        case HostInputType.key:
          await _channel.invokeMethod('injectKey', {
            'usbKeycode': event.usbKeycode ?? 0,
            'pressed': event.pressed ?? false,
          });
          break;
        case HostInputType.text:
          await _channel.invokeMethod('injectText', {'text': event.text ?? ''});
          break;
      }
    } catch (_) {
      // 单条注入失败不应中断会话
    }
  }
}
