package com.quickcoder.quickdesk_android

import android.accessibilityservice.AccessibilityService
import android.accessibilityservice.GestureDescription
import android.graphics.Path
import android.os.Build
import android.os.Bundle
import android.util.Log
import android.view.accessibility.AccessibilityEvent
import android.view.accessibility.AccessibilityNodeInfo
import kotlin.math.abs
import kotlin.math.hypot

/**
 * 被控端输入注入（标准档）。
 *
 * 依赖无障碍服务的能力：
 *   - dispatchGesture：点按 / 长按 / 拖拽 / 滚动
 *   - performGlobalAction：返回 / 主屏 / 最近任务 等系统级动作
 *   - ACTION_SET_TEXT：向当前聚焦的可编辑控件写文本（覆盖 IME 文本注入）
 *
 * 无障碍无法注入任意物理按键（需 INJECT_EVENTS 签名权限），完整键盘由
 * 增强档（Shizuku，见 ShizukuInputInjector）补足。
 *
 * 主控端下发的鼠标坐标是被控屏幕的绝对像素坐标（client 已按 VideoLayout
 * 映射），这里直接用于手势坐标系。
 */
class InputAccessibilityService : AccessibilityService() {

    // 鼠标状态（用于把 down/move/up 合成为 tap / long-press / drag）
    private var cursorX = 0f
    private var cursorY = 0f
    private var downX = 0f
    private var downY = 0f
    private var downTime = 0L
    private var buttonDown = false

    override fun onServiceConnected() {
        super.onServiceConnected()
        instance = this
        Log.i(TAG, "InputAccessibilityService connected")
    }

    override fun onUnbind(intent: android.content.Intent?): Boolean {
        if (instance === this) instance = null
        return super.onUnbind(intent)
    }

    override fun onDestroy() {
        if (instance === this) instance = null
        super.onDestroy()
    }

    override fun onAccessibilityEvent(event: AccessibilityEvent?) { /* 只做注入，不消费事件 */ }

    override fun onInterrupt() {}

    // ==================== 鼠标 ====================

    /**
     * @param button MouseButton.value：1=left 2=middle 3=right（0=仅移动）
     * @param buttonDown true=按下 false=抬起（移动事件两者均为 null → 传 -1 表示无按键变化）
     */
    fun injectMouse(x: Int, y: Int, button: Int, buttonDown: Int, wheelDeltaY: Float) {
        cursorX = x.toFloat()
        cursorY = y.toFloat()

        // 滚轮 → 滚动手势
        if (wheelDeltaY != 0f) {
            dispatchScroll(cursorX, cursorY, wheelDeltaY)
            return
        }

        when (buttonDown) {
            1 -> onButtonDown(button, x.toFloat(), y.toFloat())
            0 -> onButtonUp(button, x.toFloat(), y.toFloat())
            else -> { /* 纯移动：仅更新光标位置，无障碍档不做实时移动 */ }
        }
    }

    private fun onButtonDown(button: Int, x: Float, y: Float) {
        this.buttonDown = true
        downX = x
        downY = y
        downTime = System.currentTimeMillis()
    }

    private fun onButtonUp(button: Int, x: Float, y: Float) {
        if (!buttonDown) {
            // 没有配对的 down（可能是移动后直接抬起），按点按处理
            dispatchTap(x, y)
            return
        }
        buttonDown = false
        val dt = System.currentTimeMillis() - downTime
        val dist = hypot((x - downX).toDouble(), (y - downY).toDouble())

        when {
            button == 3 -> dispatchLongPress(x, y) // 右键 → 长按（上下文菜单）
            dist > MOVE_THRESHOLD -> dispatchDrag(downX, downY, x, y, dt)
            dt >= LONG_PRESS_MS -> dispatchLongPress(x, y)
            else -> dispatchTap(x, y)
        }
    }

    private fun dispatchTap(x: Float, y: Float) {
        val path = Path().apply { moveTo(clampX(x), clampY(y)) }
        val stroke = GestureDescription.StrokeDescription(path, 0, TAP_DURATION_MS)
        dispatch(stroke)
    }

    private fun dispatchLongPress(x: Float, y: Float) {
        val path = Path().apply { moveTo(clampX(x), clampY(y)) }
        val stroke = GestureDescription.StrokeDescription(path, 0, LONG_PRESS_MS)
        dispatch(stroke)
    }

    private fun dispatchDrag(x1: Float, y1: Float, x2: Float, y2: Float, durationMs: Long) {
        val path = Path().apply {
            moveTo(clampX(x1), clampY(y1))
            lineTo(clampX(x2), clampY(y2))
        }
        val dur = durationMs.coerceIn(MIN_DRAG_MS, MAX_DRAG_MS)
        val stroke = GestureDescription.StrokeDescription(path, 0, dur)
        dispatch(stroke)
    }

    private fun dispatchScroll(x: Float, y: Float, wheelDeltaY: Float) {
        // wheelDeltaY>0 内容向上滚（手指上滑）；反之向下
        val distance = (wheelDeltaY.coerceIn(-300f, 300f)) * SCROLL_GAIN
        val startY = clampY(y)
        val endY = clampY(y - distance)
        val path = Path().apply {
            moveTo(clampX(x), startY)
            lineTo(clampX(x), endY)
        }
        val stroke = GestureDescription.StrokeDescription(path, 0, SCROLL_DURATION_MS)
        dispatch(stroke)
    }

    private fun dispatch(stroke: GestureDescription.StrokeDescription) {
        val gesture = GestureDescription.Builder().addStroke(stroke).build()
        dispatchGesture(gesture, null, null)
    }

    // ==================== 键盘 ====================

    /** USB HID usage（0x07 页）→ 系统全局动作；返回 true 表示已处理。 */
    fun injectKey(usbKeycode: Int, pressed: Boolean): Boolean {
        if (!pressed) return true // 全局动作在抬起时不重复触发，按下即可
        val usage = usbKeycode and 0xFFFF
        val action = when (usage) {
            0x0029 -> GLOBAL_ACTION_BACK          // Escape → 返回
            0x004A -> GLOBAL_ACTION_HOME           // Home
            0x0076 -> GLOBAL_ACTION_RECENTS        // Menu → 最近任务
            else -> return false                    // 其它键交给增强档
        }
        return performGlobalAction(action)
    }

    fun performBack() = performGlobalAction(GLOBAL_ACTION_BACK)
    fun performHome() = performGlobalAction(GLOBAL_ACTION_HOME)
    fun performRecents() = performGlobalAction(GLOBAL_ACTION_RECENTS)

    // ==================== 文本 ====================

    /** 向当前聚焦的可编辑控件追加文本（IME 文本注入的兜底实现）。 */
    fun injectText(text: String): Boolean {
        val focused = findFocusedEditable() ?: return false
        val existing = focused.text?.toString() ?: ""
        val args = Bundle().apply {
            putCharSequence(
                AccessibilityNodeInfo.ACTION_ARGUMENT_SET_TEXT_CHARSEQUENCE,
                existing + text,
            )
        }
        return focused.performAction(AccessibilityNodeInfo.ACTION_SET_TEXT, args)
    }

    private fun findFocusedEditable(): AccessibilityNodeInfo? {
        val root = rootInActiveWindow ?: return null
        val focused = root.findFocus(AccessibilityNodeInfo.FOCUS_INPUT)
        return if (focused != null && focused.isEditable) focused else null
    }

    // ==================== 工具 ====================

    private fun clampX(x: Float): Float {
        val w = resources.displayMetrics.widthPixels.toFloat()
        return x.coerceIn(0f, w - 1)
    }

    private fun clampY(y: Float): Float {
        val h = resources.displayMetrics.heightPixels.toFloat()
        return y.coerceIn(0f, h - 1)
    }

    companion object {
        private const val TAG = "QuickDeskA11y"

        @Volatile
        var instance: InputAccessibilityService? = null
            private set

        private const val MOVE_THRESHOLD = 12.0    // px，超过视为拖拽
        private const val LONG_PRESS_MS = 500L
        private const val TAP_DURATION_MS = 40L
        private const val MIN_DRAG_MS = 80L
        private const val MAX_DRAG_MS = 2000L
        private const val SCROLL_GAIN = 1.2f
        private const val SCROLL_DURATION_MS = 150L

        fun isRunning(): Boolean = instance != null
    }
}
