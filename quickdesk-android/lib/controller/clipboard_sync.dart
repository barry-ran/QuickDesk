/// clipboard_sync.dart - 主控端剪贴板双向同步
///
/// 对照 WebClient/js/input/clipboard-handler.js，但针对移动端做了取舍：
///   - 收到被控端剪贴板事件 → 写入本地（host→client，实时）
///   - 本地 → 被控端：不做高频轮询（Android 12+ 每次读剪贴板都会弹系统提示，
///     轮询会刷屏）。改为在 App 回到前台时读一次 + 提供手动 syncNow()
///     （远程页工具条按钮），既能覆盖"切到别处复制再回来"的常见场景，
///     又不打扰用户。
library;

import 'dart:async';
import 'dart:convert';

import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';

import '../protocol/datachannel_handler.dart';
import '../protocol/proto/protobuf_messages.dart';

class ClipboardSync with WidgetsBindingObserver {
  final DataChannelHandler dcHandler;

  StreamSubscription<ClipboardEventMsg>? _sub;
  String _lastText = '';
  bool _enabled = false;

  ClipboardSync(this.dcHandler);

  void enable() {
    if (_enabled) return;
    _enabled = true;
    WidgetsBinding.instance.addObserver(this);
    _sub = dcHandler.onClipboard.listen(_onRemoteClipboard);
  }

  void disable() {
    if (!_enabled) return;
    _enabled = false;
    WidgetsBinding.instance.removeObserver(this);
    _sub?.cancel();
    _sub = null;
  }

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    if (state == AppLifecycleState.resumed) {
      _pushLocalIfChanged();
    }
  }

  /// 手动触发：把本地剪贴板内容推给被控端（供工具条按钮调用）
  Future<void> syncNow() => _pushLocalIfChanged(force: true);

  Future<void> _pushLocalIfChanged({bool force = false}) async {
    if (!_enabled) return;
    try {
      final data = await Clipboard.getData(Clipboard.kTextPlain);
      final text = data?.text;
      if (text != null && text.isNotEmpty && (force || text != _lastText)) {
        _lastText = text;
        dcHandler.sendClipboard('text/plain', utf8.encode(text));
      }
    } catch (_) {
      // 剪贴板不可读（无焦点/权限）时静默
    }
  }

  void _onRemoteClipboard(ClipboardEventMsg event) {
    if (!event.mimeType.startsWith('text/')) return;
    final text = utf8.decode(event.data, allowMalformed: true);
    _lastText = text; // 防回环
    Clipboard.setData(ClipboardData(text: text));
  }

  void dispose() {
    disable();
  }
}
