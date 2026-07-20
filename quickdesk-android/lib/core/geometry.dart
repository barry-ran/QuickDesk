/// geometry.dart - 视频画面布局计算
library;

import 'dart:ui';

/// 按 contain 规则把 [contentW]×[contentH] 的内容适配进 [boxW]×[boxH] 的容器：
/// 保持宽高比完整显示、居中、必要时留黑边。返回内容在容器内的绘制区域。
///
/// 视频渲染（letterbox）与触控坐标映射（光标位置换算）必须使用同一份结果，
/// 否则光标会相对画面漂移。
Rect fitContain({
  required double contentW,
  required double contentH,
  required double boxW,
  required double boxH,
}) {
  if (contentW <= 0 || contentH <= 0 || boxW <= 0 || boxH <= 0) {
    return Rect.fromLTWH(0, 0, boxW > 0 ? boxW : 0, boxH > 0 ? boxH : 0);
  }
  final contentAspect = contentW / contentH;
  final boxAspect = boxW / boxH;
  late final double drawW, drawH;
  if (boxAspect > contentAspect) {
    drawH = boxH;
    drawW = drawH * contentAspect;
  } else {
    drawW = boxW;
    drawH = drawW / contentAspect;
  }
  return Rect.fromLTWH((boxW - drawW) / 2, (boxH - drawH) / 2, drawW, drawH);
}
