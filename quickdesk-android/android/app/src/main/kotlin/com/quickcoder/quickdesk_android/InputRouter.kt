package com.quickcoder.quickdesk_android

import io.flutter.plugin.common.MethodChannel

/**
 * 被控端输入路由：Shizuku 增强档可用则优先（真实事件、流畅拖拽、全键盘），
 * 否则回退无障碍标准档（手势合成）。
 *
 * 所有方法与 Dart 侧 quickdesk/input 通道的注入调用一一对应，
 * result 返回实际使用的后端标识（"shizuku" / "a11y"）。
 */
object InputRouter {

    fun routeMouse(
        x: Int, y: Int, button: Int, buttonDown: Int, wheelDeltaY: Float,
        result: MethodChannel.Result,
    ) {
        if (ShizukuInputInjector.isAvailable()) {
            ShizukuInputInjector.injectMouse(x, y, button, buttonDown, wheelDeltaY)
            result.success("shizuku")
            return
        }
        val svc = InputAccessibilityService.instance
        if (svc == null) {
            result.error("NO_BACKEND", "No input backend available", null)
        } else {
            svc.injectMouse(x, y, button, buttonDown, wheelDeltaY)
            result.success("a11y")
        }
    }

    fun routeKey(usbKeycode: Int, pressed: Boolean, result: MethodChannel.Result) {
        if (ShizukuInputInjector.isAvailable() && ShizukuInputInjector.injectKey(usbKeycode, pressed)) {
            result.success("shizuku")
            return
        }
        val svc = InputAccessibilityService.instance
        if (svc == null) {
            result.error("NO_BACKEND", "No input backend available", null)
        } else {
            result.success(if (svc.injectKey(usbKeycode, pressed)) "a11y" else "ignored")
        }
    }

    fun routeText(text: String, result: MethodChannel.Result) {
        if (ShizukuInputInjector.isAvailable() && ShizukuInputInjector.injectText(text)) {
            result.success("shizuku")
            return
        }
        val svc = InputAccessibilityService.instance
        if (svc == null) {
            result.error("NO_BACKEND", "No input backend available", null)
        } else {
            result.success(if (svc.injectText(text)) "a11y" else "failed")
        }
    }
}
