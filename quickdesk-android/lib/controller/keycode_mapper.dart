/// keycode_mapper.dart - Flutter LogicalKeyboardKey → USB HID keycode
///
/// 对照 WebClient/js/input/keyboard-handler.js 的映射表
/// （USB HID Usage Table Page 0x07，Chromium Remoting 的 usb_keycode 格式
///  为 0x07<<16 | usage）。
///
/// 手机场景下可打印字符经 TextEvent 注入（支持 IME/中文），
/// 这里只需覆盖控制键与常用组合键所需按键。
library;

import 'package:flutter/services.dart';

class KeycodeMapper {
  static const int _page = 0x070000;

  static final Map<LogicalKeyboardKey, int> _map = {
    // 控制键
    LogicalKeyboardKey.enter: _page | 0x28,
    LogicalKeyboardKey.escape: _page | 0x29,
    LogicalKeyboardKey.backspace: _page | 0x2a,
    LogicalKeyboardKey.tab: _page | 0x2b,
    LogicalKeyboardKey.space: _page | 0x2c,
    LogicalKeyboardKey.capsLock: _page | 0x39,
    LogicalKeyboardKey.delete: _page | 0x4c,
    LogicalKeyboardKey.insert: _page | 0x49,
    LogicalKeyboardKey.home: _page | 0x4a,
    LogicalKeyboardKey.end: _page | 0x4d,
    LogicalKeyboardKey.pageUp: _page | 0x4b,
    LogicalKeyboardKey.pageDown: _page | 0x4e,

    // 方向键
    LogicalKeyboardKey.arrowRight: _page | 0x4f,
    LogicalKeyboardKey.arrowLeft: _page | 0x50,
    LogicalKeyboardKey.arrowDown: _page | 0x51,
    LogicalKeyboardKey.arrowUp: _page | 0x52,

    // 修饰键
    LogicalKeyboardKey.controlLeft: _page | 0xe0,
    LogicalKeyboardKey.shiftLeft: _page | 0xe1,
    LogicalKeyboardKey.altLeft: _page | 0xe2,
    LogicalKeyboardKey.metaLeft: _page | 0xe3,
    LogicalKeyboardKey.controlRight: _page | 0xe4,
    LogicalKeyboardKey.shiftRight: _page | 0xe5,
    LogicalKeyboardKey.altRight: _page | 0xe6,
    LogicalKeyboardKey.metaRight: _page | 0xe7,

    // 功能键
    LogicalKeyboardKey.f1: _page | 0x3a,
    LogicalKeyboardKey.f2: _page | 0x3b,
    LogicalKeyboardKey.f3: _page | 0x3c,
    LogicalKeyboardKey.f4: _page | 0x3d,
    LogicalKeyboardKey.f5: _page | 0x3e,
    LogicalKeyboardKey.f6: _page | 0x3f,
    LogicalKeyboardKey.f7: _page | 0x40,
    LogicalKeyboardKey.f8: _page | 0x41,
    LogicalKeyboardKey.f9: _page | 0x42,
    LogicalKeyboardKey.f10: _page | 0x43,
    LogicalKeyboardKey.f11: _page | 0x44,
    LogicalKeyboardKey.f12: _page | 0x45,
  };

  /// 数字/字母键（软键盘的物理按键事件走文本通道，这里兜底外接键盘）
  static int? _alphaNumeric(LogicalKeyboardKey key) {
    final id = key.keyId;
    // a-z: LogicalKeyboardKey.keyA.keyId == 0x61 (unicode 'a')
    if (id >= 0x61 && id <= 0x7a) {
      return _page | (0x04 + (id - 0x61));
    }
    // 1-9: usage 0x1e-0x26, 0: usage 0x27
    if (id >= 0x31 && id <= 0x39) {
      return _page | (0x1e + (id - 0x31));
    }
    if (id == 0x30) {
      return _page | 0x27;
    }
    return null;
  }

  /// 返回 usb keycode；不认识的键返回 null（调用方忽略即可）
  static int? logicalToUsb(LogicalKeyboardKey key) {
    return _map[key] ?? _alphaNumeric(key);
  }
}
