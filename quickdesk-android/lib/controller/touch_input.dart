/// touch_input.dart - 触控输入映射（触控板 + 画布缩放）
///
/// 对齐 RustDesk 手机「鼠标模式」+ WebClient：
///   单指拖动     → 始终移动光标（放大时自动平移画布跟随光标）
///   单击         → 左键点击
///   双击         → 左键按下保持（再点松开），便于拖窗口
///   长按         → 右键
///   双指捏合     → 绕捏合中心缩放（1x–5x），不跳动
///   双指滑动     → 未放大时滚轮
library;

import 'dart:async';

import 'package:flutter/gestures.dart';

import '../core/geometry.dart';
import '../protocol/datachannel_handler.dart';
import '../protocol/proto/protobuf_messages.dart';

const _tapTimeout = Duration(milliseconds: 200);
const _longPressTimeout = Duration(milliseconds: 500);
const _doubleTapGap = Duration(milliseconds: 300);
const _moveThreshold = 8.0;
const _cursorSpeed = 2.5;
const _minScale = 1.0;
const _maxScale = 5.0;

class TouchInputController {
  final DataChannelHandler dcHandler;

  int remoteWidth = 0;
  int remoteHeight = 0;

  double cursorX = 0;
  double cursorY = 0;

  /// 相对视口中心的平移；画面映射：
  /// screen = center + translate + (local - center) * scale
  double scale = 1;
  double translateX = 0;
  double translateY = 0;

  double viewportWidth = 0;
  double viewportHeight = 0;

  bool leftButtonHeld = false;

  void Function()? onCursorMoved;
  void Function()? onTransformChanged;

  DateTime _touchStartTime = DateTime.now();
  Offset _touchStartPos = Offset.zero;
  Offset _lastSinglePos = Offset.zero;
  DateTime? _lastTapTime;
  Timer? _longPressTimer;
  Timer? _pendingClickTimer;
  bool _moved = false;
  int _activeTouches = 0;
  Offset _lastTwoFingerCenter = Offset.zero;

  double _pinchStartDist = 0;
  double _pinchStartScale = 1;
  Offset _pinchStartCenter = Offset.zero;
  Offset _pinchStartTranslate = Offset.zero;
  bool _pinchActive = false;

  TouchInputController(this.dcHandler);

  bool get isZoomed => scale > 1.05;

  Offset get _center => Offset(viewportWidth / 2, viewportHeight / 2);

  void setRemoteResolution(int w, int h) {
    remoteWidth = w;
    remoteHeight = h;
    if (cursorX == 0 && cursorY == 0 && w > 0 && h > 0) {
      cursorX = w / 2;
      cursorY = h / 2;
      onCursorMoved?.call();
    }
  }

  void setViewportSize(double w, double h) {
    if (viewportWidth == w && viewportHeight == h) return;
    viewportWidth = w;
    viewportHeight = h;
    _clampTranslate();
  }

  void resetZoom() {
    scale = 1;
    translateX = 0;
    translateY = 0;
    onTransformChanged?.call();
  }

  /// 构建与手势模型一致的变换矩阵（绕视口中心缩放 + 平移）
  Matrix4 buildTransformMatrix() {
    final cx = viewportWidth / 2;
    final cy = viewportHeight / 2;
    // translateByDouble/scaleByDouble 是 vector_math 对已废弃 dynamic 版
    // translate/scale 的类型化替代；w 分量传 1 与旧行为一致。
    return Matrix4.identity()
      ..translateByDouble(cx + translateX, cy + translateY, 0, 1)
      ..scaleByDouble(scale, scale, 1, 1)
      ..translateByDouble(-cx, -cy, 0, 1);
  }

  // ==================== 手势入口 ====================

  void onPointerDown(PointerDownEvent event, int pointerCount) {
    _activeTouches = pointerCount;
    if (pointerCount == 1) {
      _touchStartTime = DateTime.now();
      _touchStartPos = event.position;
      _lastSinglePos = event.position;
      _moved = false;
      _pinchActive = false;
      if (!leftButtonHeld) {
        _startLongPress();
      }
    } else if (pointerCount == 2) {
      _cancelLongPress();
      _moved = true;
      _pinchActive = false;
    }
  }

  void onTwoFingerStart(Offset center, double distance) {
    _lastTwoFingerCenter = center;
    _pinchStartDist = distance;
    _pinchStartScale = scale;
    _pinchStartCenter = center;
    _pinchStartTranslate = Offset(translateX, translateY);
    _pinchActive = distance > 12;
  }

  void onPointerMove(
    PointerMoveEvent event,
    int pointerCount, {
    Offset? twoFingerCenter,
    double? twoFingerDistance,
  }) {
    if (pointerCount == 1 && _activeTouches == 1) {
      final dx = event.position.dx - _lastSinglePos.dx;
      final dy = event.position.dy - _lastSinglePos.dy;
      _lastSinglePos = event.position;

      if ((event.position - _touchStartPos).distance > _moveThreshold) {
        _moved = true;
        _cancelLongPress();
      }
      if (!_moved) return;

      // RustDesk 鼠标模式：单指始终移动光标；放大时画布自动跟随
      _moveCursor(dx * _cursorSpeed, dy * _cursorSpeed);
      if (isZoomed) {
        _autoPanToFollow();
      }
    } else if (pointerCount == 2 &&
        twoFingerCenter != null &&
        twoFingerDistance != null) {
      final scrollDx = twoFingerCenter.dx - _lastTwoFingerCenter.dx;
      final scrollDy = twoFingerCenter.dy - _lastTwoFingerCenter.dy;
      _lastTwoFingerCenter = twoFingerCenter;

      if (_pinchActive && _pinchStartDist > 0) {
        _applyPinchZoom(twoFingerCenter, twoFingerDistance);
      } else if (!isZoomed && scrollDy.abs() > 2) {
        _sendScroll(scrollDx, scrollDy);
      }
    }
  }

  /// 绕捏合焦点缩放（对齐 RustDesk canvasModel.updateScale）
  /// screen = center + translate + (local - center) * scale
  /// 保持「捏合开始时焦点下的画面点」始终落在当前两指中心下。
  void _applyPinchZoom(Offset focal, double distance) {
    final newScale =
        (_pinchStartScale * (distance / _pinchStartDist)).clamp(_minScale, _maxScale);

    if (newScale <= _minScale + 0.001) {
      scale = 1;
      translateX = 0;
      translateY = 0;
      onTransformChanged?.call();
      return;
    }

    final c = _center;
    final t0 = _pinchStartTranslate;
    final s0 = _pinchStartScale;
    final f0 = _pinchStartCenter;

    // newT = focal - c - (f0 - c - t0) * (newScale / s0)
    final newT = focal - c - (f0 - c - t0) * (newScale / s0);

    scale = newScale;
    translateX = newT.dx;
    translateY = newT.dy;
    _clampTranslate();
    onTransformChanged?.call();
  }

  void onPointerUp(PointerUpEvent event, int remainingPointers) {
    _cancelLongPress();

    if (_activeTouches == 1 && remainingPointers == 0) {
      final elapsed = DateTime.now().difference(_touchStartTime);
      if (!_moved && elapsed < _tapTimeout) {
        final now = DateTime.now();
        if (leftButtonHeld) {
          _pendingClickTimer?.cancel();
          _pendingClickTimer = null;
          _releaseLeftButton();
          _lastTapTime = null;
        } else if (_lastTapTime != null &&
            now.difference(_lastTapTime!) < _doubleTapGap) {
          _pendingClickTimer?.cancel();
          _pendingClickTimer = null;
          _pressLeftButton();
          _lastTapTime = null;
        } else {
          _lastTapTime = now;
          _pendingClickTimer?.cancel();
          _pendingClickTimer = Timer(_doubleTapGap, () {
            _pendingClickTimer = null;
            _lastTapTime = null;
            if (!leftButtonHeld) {
              _sendClick(MouseButton.left);
            }
          });
        }
      }
    }

    if (remainingPointers == 0) {
      _pinchActive = false;
      _pinchStartDist = 0;
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

  void _pressLeftButton() {
    leftButtonHeld = true;
    dcHandler.sendMouseEvent(MouseEventMsg(
      x: cursorX.round(),
      y: cursorY.round(),
      button: MouseButton.left.value,
      buttonDown: true,
    ));
    onCursorMoved?.call();
  }

  void _releaseLeftButton() {
    if (!leftButtonHeld) return;
    leftButtonHeld = false;
    dcHandler.sendMouseEvent(MouseEventMsg(
      x: cursorX.round(),
      y: cursorY.round(),
      button: MouseButton.left.value,
      buttonDown: false,
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
      if (!_moved && !leftButtonHeld) {
        _sendClick(MouseButton.right);
        _moved = true;
      }
    });
  }

  void _cancelLongPress() {
    _longPressTimer?.cancel();
    _longPressTimer = null;
  }

  void _clampTranslate() {
    if (scale <= 1 || viewportWidth <= 0 || viewportHeight <= 0) {
      if (scale <= 1) {
        translateX = 0;
        translateY = 0;
      }
      return;
    }
    final maxX = (viewportWidth * (scale - 1)) / 2;
    final maxY = (viewportHeight * (scale - 1)) / 2;
    translateX = translateX.clamp(-maxX, maxX);
    translateY = translateY.clamp(-maxY, maxY);
  }

  /// 光标靠近屏幕边缘时自动平移画布（RustDesk / WebClient 跟镜）
  void _autoPanToFollow() {
    if (!isZoomed || viewportWidth <= 0 || viewportHeight <= 0) return;
    if (remoteWidth <= 0 || remoteHeight <= 0) return;

    final screen = _cursorScreenPosition();
    if (screen == null) return;

    const edge = 56.0;
    var panX = 0.0;
    var panY = 0.0;
    if (screen.dx < edge) panX = edge - screen.dx;
    if (screen.dx > viewportWidth - edge) {
      panX = viewportWidth - edge - screen.dx;
    }
    if (screen.dy < edge) panY = edge - screen.dy;
    if (screen.dy > viewportHeight - edge) {
      panY = viewportHeight - edge - screen.dy;
    }
    if (panX != 0 || panY != 0) {
      translateX += panX;
      translateY += panY;
      _clampTranslate();
      onTransformChanged?.call();
    }
  }

  /// 光标在视口中的屏幕坐标（含缩放平移）。
  /// letterbox 区域必须与视频渲染层（remote_page）用同一份 fitContain 结果。
  Offset? _cursorScreenPosition() {
    if (remoteWidth <= 0 || remoteHeight <= 0) return null;
    if (viewportWidth <= 0 || viewportHeight <= 0) return null;

    final rect = fitContain(
      contentW: remoteWidth.toDouble(),
      contentH: remoteHeight.toDouble(),
      boxW: viewportWidth,
      boxH: viewportHeight,
    );

    final localX = rect.left + (cursorX / remoteWidth) * rect.width;
    final localY = rect.top + (cursorY / remoteHeight) * rect.height;
    final c = _center;
    return Offset(
      c.dx + translateX + (localX - c.dx) * scale,
      c.dy + translateY + (localY - c.dy) * scale,
    );
  }

  /// 供 UI 画虚拟光标用
  Offset? cursorScreenPosition() => _cursorScreenPosition();

  void dispose() {
    _cancelLongPress();
    _pendingClickTimer?.cancel();
    _pendingClickTimer = null;
    if (leftButtonHeld) {
      _releaseLeftButton();
    }
  }
}

double twoFingerDistance(Offset a, Offset b) => (a - b).distance;

Offset twoFingerCenterOf(Offset a, Offset b) =>
    Offset((a.dx + b.dx) / 2, (a.dy + b.dy) / 2);
