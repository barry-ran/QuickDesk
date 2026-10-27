/// Coalesce native per-pointer updates before classifying two-finger intent.
library;

import 'package:flutter/gestures.dart';
import 'package:flutter/scheduler.dart';

import 'touch_input.dart';

class TouchPointerRouter {
  final TouchInputController controller;
  final Map<int, Offset> _positions = {};
  int? _frame;

  TouchPointerRouter(this.controller);

  void down(PointerDownEvent event) {
    _positions[event.pointer] = event.localPosition;
    controller.onPointerDown(
      PointerDownEvent(pointer: event.pointer, position: event.localPosition),
      _positions.length,
    );
    if (_positions.length == 2) {
      final p = _positions.values.toList();
      controller.onTwoFingerStart(
        twoFingerCenterOf(p[0], p[1]),
        twoFingerDistance(p[0], p[1]),
      );
    }
  }

  void move(PointerMoveEvent event) {
    if (!_positions.containsKey(event.pointer)) return;
    _positions[event.pointer] = event.localPosition;
    if (_positions.length == 2) {
      _frame ??= SchedulerBinding.instance.scheduleFrameCallback((_) {
        _frame = null;
        _sendPair();
      });
    } else {
      controller.onPointerMove(
        PointerMoveEvent(pointer: event.pointer, position: event.localPosition),
        _positions.length,
      );
    }
  }

  void _sendPair() {
    if (_positions.length != 2) return;
    final p = _positions.values.toList();
    controller.onPointerMove(
      PointerMoveEvent(position: p[0]),
      2,
      twoFingerCenter: twoFingerCenterOf(p[0], p[1]),
      twoFingerDistance: twoFingerDistance(p[0], p[1]),
    );
  }

  void up(PointerUpEvent event) {
    if (_frame != null) {
      SchedulerBinding.instance.cancelFrameCallbackWithId(_frame!);
      _frame = null;
      _sendPair();
    }
    _positions.remove(event.pointer);
    controller.onPointerUp(
      event,
      _positions.length,
      remainingPosition: _positions.length == 1
          ? _positions.values.first
          : null,
    );
  }

  void cancelPointer(PointerCancelEvent event) {
    if (_frame != null) {
      SchedulerBinding.instance.cancelFrameCallbackWithId(_frame!);
    }
    _frame = null;
    _positions.remove(event.pointer);
    controller.onPointerCancel(_positions.length);
  }

  void cancel() {
    if (_frame != null)
      SchedulerBinding.instance.cancelFrameCallbackWithId(_frame!);
    _frame = null;
    _positions.clear();
    controller.onPointerCancel(0);
  }
}
