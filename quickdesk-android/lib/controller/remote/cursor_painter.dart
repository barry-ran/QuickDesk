/// cursor_painter.dart - 虚拟光标绘制
library;

import 'package:flutter/material.dart';

/// 标准箭头光标（对照 touch-handler.js 的 SVG path）
class CursorPainter extends CustomPainter {
  const CursorPainter();

  @override
  void paint(Canvas canvas, Size size) {
    final path = Path()
      ..moveTo(3, 2)
      ..lineTo(3, 17)
      ..lineTo(7.5, 12.5)
      ..lineTo(11, 19)
      ..lineTo(13.5, 18)
      ..lineTo(10, 11.5)
      ..lineTo(16, 11.5)
      ..close();

    canvas.drawPath(path, Paint()..color = Colors.white);
    canvas.drawPath(
      path,
      Paint()
        ..color = Colors.black
        ..style = PaintingStyle.stroke
        ..strokeWidth = 1.2
        ..strokeJoin = StrokeJoin.round,
    );
  }

  @override
  bool shouldRepaint(covariant CustomPainter oldDelegate) => false;
}
