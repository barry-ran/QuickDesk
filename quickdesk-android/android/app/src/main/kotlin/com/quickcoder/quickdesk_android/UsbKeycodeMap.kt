package com.quickcoder.quickdesk_android

import android.view.KeyEvent

/**
 * USB HID Keyboard/Keypad Page (0x07) usage → Android KeyEvent.KEYCODE_*。
 *
 * 主控端下发的 usbKeycode 为 Chromium 风格的 0x0700xx，这里取低字节 usage id
 * 映射到 Android 键码，供 Shizuku 增强档注入真实按键。
 */
object UsbKeycodeMap {

    fun toAndroid(usbKeycode: Int): Int {
        val usage = usbKeycode and 0xFF
        return MAP[usage] ?: KeyEvent.KEYCODE_UNKNOWN
    }

    private val MAP: Map<Int, Int> = buildMap {
        // a-z (0x04..0x1D)
        for (i in 0..25) put(0x04 + i, KeyEvent.KEYCODE_A + i)
        // 1-9 (0x1E..0x26)
        for (i in 0..8) put(0x1E + i, KeyEvent.KEYCODE_1 + i)
        put(0x27, KeyEvent.KEYCODE_0)

        put(0x28, KeyEvent.KEYCODE_ENTER)
        put(0x29, KeyEvent.KEYCODE_ESCAPE)
        put(0x2A, KeyEvent.KEYCODE_DEL)          // Backspace
        put(0x2B, KeyEvent.KEYCODE_TAB)
        put(0x2C, KeyEvent.KEYCODE_SPACE)
        put(0x2D, KeyEvent.KEYCODE_MINUS)
        put(0x2E, KeyEvent.KEYCODE_EQUALS)
        put(0x2F, KeyEvent.KEYCODE_LEFT_BRACKET)
        put(0x30, KeyEvent.KEYCODE_RIGHT_BRACKET)
        put(0x31, KeyEvent.KEYCODE_BACKSLASH)
        put(0x33, KeyEvent.KEYCODE_SEMICOLON)
        put(0x34, KeyEvent.KEYCODE_APOSTROPHE)
        put(0x35, KeyEvent.KEYCODE_GRAVE)
        put(0x36, KeyEvent.KEYCODE_COMMA)
        put(0x37, KeyEvent.KEYCODE_PERIOD)
        put(0x38, KeyEvent.KEYCODE_SLASH)
        put(0x39, KeyEvent.KEYCODE_CAPS_LOCK)

        // F1-F12 (0x3A..0x45)
        for (i in 0..11) put(0x3A + i, KeyEvent.KEYCODE_F1 + i)

        put(0x46, KeyEvent.KEYCODE_SYSRQ)        // PrintScreen
        put(0x48, KeyEvent.KEYCODE_BREAK)        // Pause
        put(0x49, KeyEvent.KEYCODE_INSERT)
        put(0x4A, KeyEvent.KEYCODE_MOVE_HOME)
        put(0x4B, KeyEvent.KEYCODE_PAGE_UP)
        put(0x4C, KeyEvent.KEYCODE_FORWARD_DEL)  // Delete
        put(0x4D, KeyEvent.KEYCODE_MOVE_END)
        put(0x4E, KeyEvent.KEYCODE_PAGE_DOWN)
        put(0x4F, KeyEvent.KEYCODE_DPAD_RIGHT)
        put(0x50, KeyEvent.KEYCODE_DPAD_LEFT)
        put(0x51, KeyEvent.KEYCODE_DPAD_DOWN)
        put(0x52, KeyEvent.KEYCODE_DPAD_UP)

        put(0xE0, KeyEvent.KEYCODE_CTRL_LEFT)
        put(0xE1, KeyEvent.KEYCODE_SHIFT_LEFT)
        put(0xE2, KeyEvent.KEYCODE_ALT_LEFT)
        put(0xE3, KeyEvent.KEYCODE_META_LEFT)
        put(0xE4, KeyEvent.KEYCODE_CTRL_RIGHT)
        put(0xE5, KeyEvent.KEYCODE_SHIFT_RIGHT)
        put(0xE6, KeyEvent.KEYCODE_ALT_RIGHT)
        put(0xE7, KeyEvent.KEYCODE_META_RIGHT)
    }
}
