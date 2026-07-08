package com.quickcoder.quickdesk_android

import android.content.ComponentName
import android.content.Context
import android.content.ServiceConnection
import android.content.pm.PackageManager
import android.os.IBinder
import android.os.SystemClock
import android.util.Log
import android.view.KeyEvent
import android.view.MotionEvent
import rikka.shizuku.Shizuku

/**
 * 被控端输入注入（增强档，Shizuku）。
 *
 * 通过 Shizuku 绑定一个 shell(uid 2000) 权限的 UserService
 * （[ShizukuInputUserService]），用隐藏 API 注入真实触摸/按键事件：
 *   - 连续拖拽 / 精确点按（无障碍手势做不到的流畅度）
 *   - 任意物理按键（无障碍无法注入）
 *
 * 生命周期：init(context) → requestPermission()（用户授权）→ bind() → 就绪。
 * 注入前用 [isReady] 判断；未就绪时调用方应回退到无障碍档。
 */
object ShizukuInputInjector {

    private const val TAG = "QuickDeskShizuku"
    const val PERMISSION_REQUEST_CODE = 47001

    private var appContext: Context? = null
    private var service: IShizukuInputService? = null
    private var binding = false

    // 触摸手势状态（把 mouse down/move/up 转成 MotionEvent 流）
    private var touchDown = false
    private var touchDownTime = 0L

    private val userServiceArgs: Shizuku.UserServiceArgs?
        get() {
            val ctx = appContext ?: return null
            return Shizuku.UserServiceArgs(
                ComponentName(ctx.packageName, ShizukuInputUserService::class.java.name),
            )
                .daemon(false)
                .processNameSuffix("shizuku_input")
                .debuggable(false)
                .version(1)
        }

    private val connection = object : ServiceConnection {
        override fun onServiceConnected(name: ComponentName?, binder: IBinder?) {
            binding = false
            service = if (binder != null && binder.pingBinder()) {
                IShizukuInputService.Stub.asInterface(binder)
            } else {
                null
            }
            Log.i(TAG, "Shizuku user service connected: ${service != null}")
        }

        override fun onServiceDisconnected(name: ComponentName?) {
            service = null
            binding = false
            Log.i(TAG, "Shizuku user service disconnected")
        }
    }

    fun init(context: Context) {
        appContext = context.applicationContext
    }

    /** Shizuku 是否安装且服务在运行（binder 可 ping）。 */
    fun isShizukuRunning(): Boolean = try {
        Shizuku.pingBinder()
    } catch (_: Throwable) {
        false
    }

    /** 是否已获得 Shizuku 授权。 */
    fun hasPermission(): Boolean = try {
        isShizukuRunning() && !Shizuku.isPreV11() &&
            Shizuku.checkSelfPermission() == PackageManager.PERMISSION_GRANTED
    } catch (_: Throwable) {
        false
    }

    /** UserService 已绑定且可用。 */
    fun isReady(): Boolean = service != null

    /** 综合可用性：供 Dart 侧 isShizukuAvailable 查询。 */
    fun isAvailable(): Boolean = hasPermission() && isReady()

    /** 发起 Shizuku 权限申请（需在 Activity 上下文，结果由监听器回调）。 */
    fun requestPermission() {
        if (!isShizukuRunning()) return
        try {
            if (Shizuku.shouldShowRequestPermissionRationale()) return
            Shizuku.requestPermission(PERMISSION_REQUEST_CODE)
        } catch (t: Throwable) {
            Log.e(TAG, "requestPermission failed", t)
        }
    }

    /** 绑定 UserService（需已授权）。幂等。 */
    fun bind() {
        if (service != null || binding) return
        if (!hasPermission()) return
        val args = userServiceArgs ?: return
        binding = true
        try {
            Shizuku.bindUserService(args, connection)
        } catch (t: Throwable) {
            binding = false
            Log.e(TAG, "bindUserService failed", t)
        }
    }

    fun unbind() {
        val args = userServiceArgs ?: return
        try {
            Shizuku.unbindUserService(args, connection, true)
        } catch (_: Throwable) {
        }
        service = null
        binding = false
    }

    // ==================== 注入 ====================

    /** 与无障碍档同签名，便于路由层统一转发。 */
    fun injectMouse(x: Int, y: Int, button: Int, buttonDown: Int, wheelDeltaY: Float) {
        val svc = service ?: return
        val fx = x.toFloat()
        val fy = y.toFloat()

        if (wheelDeltaY != 0f) {
            dispatchScroll(svc, fx, fy, wheelDeltaY)
            return
        }

        when (buttonDown) {
            1 -> { // 按下
                touchDown = true
                touchDownTime = SystemClock.uptimeMillis()
                svc.injectMotion(MotionEvent.ACTION_DOWN, fx, fy, touchDownTime)
            }
            0 -> { // 抬起
                if (touchDown) {
                    svc.injectMotion(MotionEvent.ACTION_UP, fx, fy, touchDownTime)
                    touchDown = false
                } else {
                    // 无配对 down：补一个瞬时点按
                    val t = SystemClock.uptimeMillis()
                    svc.injectMotion(MotionEvent.ACTION_DOWN, fx, fy, t)
                    svc.injectMotion(MotionEvent.ACTION_UP, fx, fy, t)
                }
            }
            else -> { // 移动
                if (touchDown) {
                    svc.injectMotion(MotionEvent.ACTION_MOVE, fx, fy, touchDownTime)
                }
            }
        }
    }

    private fun dispatchScroll(svc: IShizukuInputService, x: Float, y: Float, wheelDeltaY: Float) {
        // 用一段快速滑动模拟滚轮：wheelDeltaY>0 → 内容上滚（手指上滑）
        val distance = wheelDeltaY.coerceIn(-300f, 300f) * 1.2f
        val t = SystemClock.uptimeMillis()
        svc.injectMotion(MotionEvent.ACTION_DOWN, x, y, t)
        val steps = 5
        for (i in 1..steps) {
            val yy = y - distance * i / steps
            svc.injectMotion(MotionEvent.ACTION_MOVE, x, yy, t)
        }
        svc.injectMotion(MotionEvent.ACTION_UP, x, y - distance, t)
    }

    fun injectKey(usbKeycode: Int, pressed: Boolean): Boolean {
        val svc = service ?: return false
        val code = UsbKeycodeMap.toAndroid(usbKeycode)
        if (code == KeyEvent.KEYCODE_UNKNOWN) return false
        svc.injectKey(code, pressed)
        return true
    }

    fun injectText(text: String): Boolean {
        val svc = service ?: return false
        svc.injectText(text)
        return true
    }
}
