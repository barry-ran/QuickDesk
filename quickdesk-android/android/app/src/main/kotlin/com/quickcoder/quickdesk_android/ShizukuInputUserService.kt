package com.quickcoder.quickdesk_android

import android.os.SystemClock
import android.util.Log
import android.view.InputDevice
import android.view.InputEvent
import android.view.KeyCharacterMap
import android.view.KeyEvent
import android.view.MotionEvent
import java.lang.reflect.Method

/**
 * Shizuku User Service 实现，运行在 shell(uid 2000) 进程内。
 *
 * shell 拥有 INJECT_EVENTS 权限，可通过隐藏 API 注入真实的触摸/按键事件
 * （相比无障碍手势，支持连续拖拽、精确点按、任意物理键，适合游戏等场景）。
 *
 * 注入通道用反射适配不同 Android 版本：
 *   - API < 34：android.hardware.input.InputManager#getInstance()#injectInputEvent
 *   - API >= 34：android.hardware.input.InputManagerGlobal#getInstance()#injectInputEvent
 *
 * Shizuku 文档明确：User Service 内不受非 SDK 接口限制，反射可用。
 */
class ShizukuInputUserService : IShizukuInputService.Stub {

    // 默认构造器（Shizuku 老版本使用）
    constructor() : super() {
        Log.i(TAG, "ShizukuInputUserService created")
    }

    // 带 Context 的构造器（Shizuku v13+ 优先尝试）
    @Suppress("UNUSED_PARAMETER")
    constructor(context: android.content.Context) : super() {
        Log.i(TAG, "ShizukuInputUserService created (with context)")
    }

    private var injectTarget: Any? = null
    private var injectMethod: Method? = null

    override fun destroy() {
        Log.i(TAG, "destroy")
        System.exit(0)
    }

    override fun exit() {
        destroy()
    }

    override fun injectMotion(action: Int, x: Float, y: Float, downTime: Long) {
        val now = SystemClock.uptimeMillis()
        val event = MotionEvent.obtain(downTime, now, action, x, y, 0)
        event.source = InputDevice.SOURCE_TOUCHSCREEN
        try {
            inject(event)
        } finally {
            event.recycle()
        }
    }

    override fun injectKey(androidKeyCode: Int, down: Boolean) {
        val now = SystemClock.uptimeMillis()
        val action = if (down) KeyEvent.ACTION_DOWN else KeyEvent.ACTION_UP
        val event = KeyEvent(
            now, now, action, androidKeyCode, 0, 0,
            KeyCharacterMap.VIRTUAL_KEYBOARD, 0, 0, InputDevice.SOURCE_KEYBOARD,
        )
        inject(event)
    }

    override fun injectText(text: String) {
        val kcm = KeyCharacterMap.load(KeyCharacterMap.VIRTUAL_KEYBOARD)
        val events = kcm.getEvents(text.toCharArray()) ?: return
        for (e in events) {
            inject(e)
        }
    }

    private fun inject(event: InputEvent) {
        try {
            resolveInjector()
            injectMethod?.invoke(injectTarget, event, INJECT_INPUT_EVENT_MODE_ASYNC)
        } catch (t: Throwable) {
            Log.e(TAG, "injectInputEvent failed", t)
        }
    }

    private fun resolveInjector() {
        if (injectMethod != null) return
        // API 34+：InputManagerGlobal
        try {
            val cls = Class.forName("android.hardware.input.InputManagerGlobal")
            val instance = cls.getMethod("getInstance").invoke(null)
            val m = cls.getMethod("injectInputEvent", InputEvent::class.java, Int::class.javaPrimitiveType)
            injectTarget = instance
            injectMethod = m
            return
        } catch (_: Throwable) {
            // 回退到旧路径
        }
        val cls = Class.forName("android.hardware.input.InputManager")
        val instance = cls.getMethod("getInstance").invoke(null)
        val m = cls.getMethod("injectInputEvent", InputEvent::class.java, Int::class.javaPrimitiveType)
        injectTarget = instance
        injectMethod = m
    }

    companion object {
        private const val TAG = "QuickDeskShizukuSvc"
        // InputManager.INJECT_INPUT_EVENT_MODE_ASYNC
        private const val INJECT_INPUT_EVENT_MODE_ASYNC = 0
    }
}
