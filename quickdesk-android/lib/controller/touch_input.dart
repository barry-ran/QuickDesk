/// touch_input.dart - 触控输入映射（触控板模式）
///
/// 对照 WebClient/js/input/touch-handler.js 的交互模型：
///   单指拖动    → 移动鼠标光标（相对移动，触控板模式）
///   单击        → 左键点击
///   双击        → 双击
///   长按(500ms) → 右键点击
///   双指滑动    → 滚轮
/// 画面缩放/平移由 Flutter InteractiveViewer 承担，这里只管输入注入。
library;

import 'dart:async';

import 'package:flutter/gestures.dart';

import '../protocol/datachannel_handler.dart';
import '../protocol/proto/protobuf_messages.dart';

const _tapTimeout = Duration(milliseconds: 200);
const _longPressTimeout = Duration(milliseconds: 500);
const _doubleTapGap = Duration(milliseconds: 300);
const _moveThreshold = 8.0;
const _cursorSpeed = 2.5;

class TouchInputController {
  final DataChannelHandler dcHandler;

  /// 远程桌面分辨率（来自视频帧尺寸或 VideoLayout）
  int remoteWidth = 0;
  int remoteHeight = 0;

  /// 虚拟光标位置（远程坐标系）
  double cursorX = 0;
  double cursorY = 0;

  /// 光标位置变化回调（驱动 UI 上的虚拟光标）
  void Function()? onCursorMoved;

  DateTime _touchStartTime = DateTime.now();
  Offset _touchStartPos = Offset.zero;
  Offset _lastSinglePos = Offset.zero;
  DateTime? _lastTapTime;
  Timer? _longPressTimer;
  bool _moved = false;
  int _activeTouches = 0;
  Offset _lastTwoFingerCenter = Offset.zero;

  TouchInputController(this.dcHandler);

  void setRemoteResolution(int w, int h) {
    remoteWidth = w;
    remoteHeight = h;
    if (cursorX == 0 && cursorY == 0 && w > 0 && h > 0) {
      cursorX = w / 2;
      cursorY = h / 2;
      onCursorMoved?.call();
    }
  }

  // ==================== 手势入口（由 RemotePage 的 Listener 调用） ====================

  void onPointerDown(PointerDownEvent event, int pointerCount) {
    _activeTouches = pointerCount;
    if (pointerCount == 1) {
      _touchStartTime = DateTime.now();
      _touchStartPos = event.position;
      _lastSinglePos = event.position;
      _moved = false;
      _startLongPress();
    } else if (pointerCount == 2) {
      _cancelLongPress();
      _moved = true;
    }
  }

  void onPointerMove(PointerMoveEvent event, int pointerCount, {Offset? twoFingerCenter}) {
    if (pointerCount == 1 && _activeTouches == 1) {
      final dx = event.position.dx - _lastSinglePos.dx;
      final dy = event.position.dy - _lastSinglePos.dy;
      _lastSinglePos = event.position;

      if ((event.position - _touchStartPos).distance > _moveThreshold) {
        _moved = true;
        _cancelLongPress();
      }

      if (_moved) {
        _moveCursor(dx * _cursorSpeed, dy * _cursorSpeed);
      }
    } else if (pointerCount == 2 && twoFingerCenter != null) {
      final scrollDx = twoFingerCenter.dx - _lastTwoFingerCenter.dx;
      final scrollDy = twoFingerCenter.dy - _lastTwoFingerCenter.dy;
      _lastTwoFingerCenter = twoFingerCenter;
      if (scrollDy.abs() > 2) {
        _sendScroll(scrollDx, scrollDy);
      }
    }
  }

  void onTwoFingerStart(Offset center) {
    _lastTwoFingerCenter = center;
  }

  void onPointerUp(PointerUpEvent event, int remainingPointers) {
    _cancelLongPress();

    if (_activeTouches == 1 && remainingPointers == 0) {
      final elapsed = DateTime.now().difference(_touchStartTime);
      if (!_moved && elapsed < _tapTimeout) {
        final now = DateTime.now();
        if (_lastTapTime != null && now.difference(_lastTapTime!) < _doubleTapGap) {
          _sendDoubleClick();
          _lastTapTime = null;
        } else {
          _lastTapTime = now;
          _sendClick(MouseButton.left);
        }
      }
    }
    _activeTouches = remainingPointers;
  }

  // ==================== 输入注入 ====================

  void _moveCursor(double dx, double dy) {
    if (remoteWidth <= 0 || remoteHeight <= 0) return;
    cursorX = (cursorX + dx).clamp(0, remoteWidth - 1.0);
    cursorY = (cursorY + dy).clamp(0, remoteHeight - 1.0);

    dcHandler.sendMouseEvent(MouseEventMsg(
      x: cursorX.round(),
      y: cursorY.round(),
    ));
    onCursorMoved?.call();
  }

  void _sendClick(MouseButton button) {
    final x = cursorX.round();
    final y = cursorY.round();
    dcHandler.sendMouseEvent(
        MouseEventMsg(x: x, y: y, button: button.value, buttonDown: true));
    dcHandler.sendMouseEvent(
        MouseEventMsg(x: x, y: y, button: button.value, buttonDown: false));
  }

  void _sendDoubleClick() {
    _sendClick(MouseButton.left);
    _sendClick(MouseButton.left);
  }

  void _sendScroll(double dx, double dy) {
    dcHandler.sendMouseEvent(MouseEventMsg(
      x: cursorX.round(),
      y: cursorY.round(),
      wheelDeltaX: dx * 3,
      wheelDeltaY: dy * 3,
      wheelTicksX: dx / 40,
      wheelTicksY: dy / 40,
    ));
  }

  void _startLongPress() {
    _cancelLongPress();
    _longPressTimer = Timer(_longPressTimeout, () {
      _longPressTimer = null;
      if (!_moved) {
        _sendClick(MouseButton.right);
        _moved = true;
      }
    });
  }

  void _cancelLongPress() {
    _longPressTimer?.cancel();
    _longPressTimer = null;
  }

  void dispose() {
    _cancelLongPress();
  }
}
