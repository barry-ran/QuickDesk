package com.quickcoder.quickdesk_android

import android.content.Context
import android.content.Intent
import android.hardware.display.DisplayManager
import android.provider.Settings
import android.text.TextUtils
import android.view.Display
import io.flutter.embedding.android.FlutterActivity
import io.flutter.embedding.engine.FlutterEngine
import io.flutter.plugin.common.MethodChannel
import rikka.shizuku.Shizuku

/**
 * 注册两个平台通道：
 *   quickdesk/screen_capture —— 屏幕采集前台服务的启停
 *   quickdesk/input          —— 被控端输入注入（无障碍/Shizuku 双档）+ 权限引导
 *
 * 输入注入后端选择：Shizuku 就绪则优先（真实事件、流畅拖拽、全键盘），
 * 否则回退无障碍服务（手势合成）。
 */
class MainActivity : FlutterActivity() {

    private var inputChannel: MethodChannel? = null
    private var screenChannel: MethodChannel? = null
    private var displayManager: DisplayManager? = null

    // 屏幕旋转/分辨率变化 → 通知 Dart 重发 VideoLayout（被控端坐标系随之更新）
    private val displayListener = object : DisplayManager.DisplayListener {
        override fun onDisplayAdded(displayId: Int) {}
        override fun onDisplayRemoved(displayId: Int) {}
        override fun onDisplayChanged(displayId: Int) {
            if (displayId != Display.DEFAULT_DISPLAY) return
            val (w, h) = realScreenSize()
            screenChannel?.invokeMethod("displayChanged", mapOf("width" to w, "height" to h))
        }
    }

    private val permissionListener =
        Shizuku.OnRequestPermissionResultListener { _, grantResult ->
            if (grantResult == android.content.pm.PackageManager.PERMISSION_GRANTED) {
                ShizukuInputInjector.bind()
            }
            inputChannel?.invokeMethod("shizukuPermissionResult", grantResult == 0)
        }

    override fun configureFlutterEngine(flutterEngine: FlutterEngine) {
        super.configureFlutterEngine(flutterEngine)
        val messenger = flutterEngine.dartExecutor.binaryMessenger

        ShizukuInputInjector.init(applicationContext)
        try {
            Shizuku.addRequestPermissionResultListener(permissionListener)
        } catch (_: Throwable) {
        }

        val screenCh = MethodChannel(messenger, CHANNEL_SCREEN)
        screenChannel = screenCh
        displayManager = getSystemService(Context.DISPLAY_SERVICE) as? DisplayManager
        displayManager?.registerDisplayListener(displayListener, null)
        screenCh.setMethodCallHandler { call, result ->
            when (call.method) {
                "startService" -> {
                    ScreenCaptureService.start(applicationContext)
                    result.success(true)
                }
                "stopService" -> {
                    ScreenCaptureService.stop(applicationContext)
                    result.success(true)
                }
                "getScreenSize" -> {
                    val (w, h) = realScreenSize()
                    result.success(mapOf("width" to w, "height" to h))
                }
                else -> result.notImplemented()
            }
        }

        val inputCh = MethodChannel(messenger, CHANNEL_INPUT)
        inputChannel = inputCh
        inputCh.setMethodCallHandler { call, result ->
            when (call.method) {
                // ---- 无障碍 ----
                "isAccessibilityEnabled" -> result.success(isAccessibilityEnabled())
                "openAccessibilitySettings" -> {
                    startActivity(
                        Intent(Settings.ACTION_ACCESSIBILITY_SETTINGS)
                            .addFlags(Intent.FLAG_ACTIVITY_NEW_TASK),
                    )
                    result.success(true)
                }
                // ---- Shizuku ----
                "isShizukuRunning" -> result.success(ShizukuInputInjector.isShizukuRunning())
                "hasShizukuPermission" -> result.success(ShizukuInputInjector.hasPermission())
                "isShizukuAvailable" -> result.success(ShizukuInputInjector.isAvailable())
                "requestShizukuPermission" -> {
                    ShizukuInputInjector.requestPermission()
                    result.success(true)
                }
                "bindShizuku" -> {
                    ShizukuInputInjector.bind()
                    result.success(true)
                }
                // ---- 注入（自动选后端）----
                "injectMouse" -> {
                    routeMouse(
                        call.argument<Int>("x") ?: 0,
                        call.argument<Int>("y") ?: 0,
                        call.argument<Int>("button") ?: 0,
                        call.argument<Int>("buttonDown") ?: -1,
                        (call.argument<Double>("wheelDeltaY") ?: 0.0).toFloat(),
                        result,
                    )
                }
                "injectKey" -> {
                    routeKey(
                        call.argument<Int>("usbKeycode") ?: 0,
                        call.argument<Boolean>("pressed") ?: false,
                        result,
                    )
                }
                "injectText" -> {
                    routeText(call.argument<String>("text") ?: "", result)
                }
                "globalAction" -> {
                    val svc = InputAccessibilityService.instance
                    if (svc == null) {
                        result.error("NO_A11Y", "Accessibility service not running", null)
                    } else {
                        when (call.argument<String>("action")) {
                            "back" -> svc.performBack()
                            "home" -> svc.performHome()
                            "recents" -> svc.performRecents()
                        }
                        result.success(true)
                    }
                }
                else -> result.notImplemented()
            }
        }
    }

    private fun routeMouse(
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

    private fun routeKey(usbKeycode: Int, pressed: Boolean, result: MethodChannel.Result) {
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

    private fun routeText(text: String, result: MethodChannel.Result) {
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

    /** 真实屏幕像素尺寸（含旋转），供 flutter_webrtc 采集尺寸缺失时兜底。 */
    private fun realScreenSize(): Pair<Int, Int> {
        return try {
            if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.R) {
                val bounds = windowManager.currentWindowMetrics.bounds
                Pair(bounds.width(), bounds.height())
            } else {
                val metrics = android.util.DisplayMetrics()
                @Suppress("DEPRECATION")
                windowManager.defaultDisplay.getRealMetrics(metrics)
                Pair(metrics.widthPixels, metrics.heightPixels)
            }
        } catch (_: Throwable) {
            Pair(0, 0)
        }
    }

    /** 检查本应用的无障碍服务是否已被用户启用。 */
    private fun isAccessibilityEnabled(): Boolean {
        val expected = "$packageName/$packageName.InputAccessibilityService"
        val enabled = try {
            Settings.Secure.getInt(contentResolver, Settings.Secure.ACCESSIBILITY_ENABLED)
        } catch (e: Settings.SettingNotFoundException) {
            0
        }
        if (enabled != 1) return false
        val services = Settings.Secure.getString(
            contentResolver,
            Settings.Secure.ENABLED_ACCESSIBILITY_SERVICES,
        ) ?: return false
        val splitter = TextUtils.SimpleStringSplitter(':')
        splitter.setString(services)
        for (component in splitter) {
            if (component.equals(expected, ignoreCase = true)) return true
        }
        return false
    }

    override fun onDestroy() {
        try {
            Shizuku.removeRequestPermissionResultListener(permissionListener)
        } catch (_: Throwable) {
        }
        try {
            displayManager?.unregisterDisplayListener(displayListener)
        } catch (_: Throwable) {
        }
        super.onDestroy()
    }

    companion object {
        private const val CHANNEL_SCREEN = "quickdesk/screen_capture"
        private const val CHANNEL_INPUT = "quickdesk/input"
    }
}
