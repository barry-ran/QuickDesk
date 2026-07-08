// IShizukuInputService.aidl
package com.quickcoder.quickdesk_android;

/**
 * Shizuku User Service 接口：运行在 shell(uid 2000) 进程内，拥有 INJECT_EVENTS，
 * 可通过隐藏 API InputManager.injectInputEvent 注入真实触摸/按键事件。
 */
interface IShizukuInputService {

    // Shizuku 服务器约定的销毁方法，事务号固定为 16777114
    void destroy() = 16777114;

    void exit() = 1;

    /**
     * 注入一次触摸动作。
     * @param action    MotionEvent.ACTION_DOWN/MOVE/UP/CANCEL
     * @param x,y       屏幕绝对像素坐标
     * @param downTime  本次手势按下时间（SystemClock.uptimeMillis），MOVE/UP 需复用 DOWN 的值
     */
    void injectMotion(int action, float x, float y, long downTime) = 2;

    /**
     * 注入一个按键事件。
     * @param androidKeyCode KeyEvent.KEYCODE_*
     * @param down           true=按下 false=抬起
     */
    void injectKey(int androidKeyCode, boolean down) = 3;

    /** 注入一段文本（通过 KeyCharacterMap 转按键序列）。 */
    void injectText(String text) = 4;
}
