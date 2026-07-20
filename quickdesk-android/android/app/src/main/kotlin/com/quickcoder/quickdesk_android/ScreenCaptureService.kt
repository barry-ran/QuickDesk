package com.quickcoder.quickdesk_android

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.Service
import android.content.Context
import android.content.Intent
import android.content.pm.ServiceInfo
import android.os.Build
import android.os.Bundle
import android.os.Handler
import android.os.IBinder
import android.os.Looper
import android.os.ResultReceiver
import android.util.Log

/**
 * 被控端屏幕采集前台服务。
 *
 * Android 10+（API 29）要求 MediaProjection 采集期间运行
 * foregroundServiceType=mediaProjection 的前台服务。Android 14+ 还要求先由用户
 * 授予本次屏幕捕获权限，再启动该类型服务，之后才能创建 MediaProjection。
 *
 * 本服务只负责持有前台通知以满足系统约束，实际采集由 flutter_webrtc 完成。
 */
class ScreenCaptureService : Service() {

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        val receiver = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU) {
            intent?.getParcelableExtra(EXTRA_RESULT_RECEIVER, ResultReceiver::class.java)
        } else {
            @Suppress("DEPRECATION")
            intent?.getParcelableExtra(EXTRA_RESULT_RECEIVER)
        }

        try {
            startAsForeground()
            receiver?.send(RESULT_STARTED, Bundle())
        } catch (error: Throwable) {
            Log.e(TAG, "Unable to start media projection foreground service", error)
            receiver?.send(
                RESULT_FAILED,
                Bundle().apply {
                    putString(EXTRA_ERROR_MESSAGE, error.message ?: error.javaClass.simpleName)
                },
            )
            stopSelf(startId)
        }
        return START_NOT_STICKY
    }

    private fun startAsForeground() {
        val notification = buildNotification()
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            startForeground(
                NOTIFICATION_ID,
                notification,
                ServiceInfo.FOREGROUND_SERVICE_TYPE_MEDIA_PROJECTION,
            )
        } else {
            startForeground(NOTIFICATION_ID, notification)
        }
    }

    private fun buildNotification(): Notification {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val manager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            val channel = NotificationChannel(
                CHANNEL_ID,
                "屏幕共享",
                NotificationManager.IMPORTANCE_LOW,
            ).apply {
                description = "QuickDesk 正在被远程控制/共享屏幕"
                setShowBadge(false)
            }
            manager.createNotificationChannel(channel)
        }

        val builder = if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            Notification.Builder(this, CHANNEL_ID)
        } else {
            @Suppress("DEPRECATION")
            Notification.Builder(this)
        }

        return builder
            .setContentTitle("QuickDesk 屏幕共享中")
            .setContentText("您的屏幕正在被远程访问")
            .setSmallIcon(android.R.drawable.ic_menu_share)
            .setOngoing(true)
            .build()
    }

    override fun onDestroy() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.N) {
            stopForeground(STOP_FOREGROUND_REMOVE)
        } else {
            @Suppress("DEPRECATION")
            stopForeground(true)
        }
        super.onDestroy()
    }

    companion object {
        private const val TAG = "ScreenCaptureService"
        private const val CHANNEL_ID = "quickdesk_screen_capture"
        private const val NOTIFICATION_ID = 1001
        private const val EXTRA_RESULT_RECEIVER = "result_receiver"
        private const val EXTRA_ERROR_MESSAGE = "error_message"
        private const val RESULT_STARTED = 1
        private const val RESULT_FAILED = 2

        fun start(context: Context, callback: (String?) -> Unit) {
            val receiver = object : ResultReceiver(Handler(Looper.getMainLooper())) {
                override fun onReceiveResult(resultCode: Int, resultData: Bundle?) {
                    when (resultCode) {
                        RESULT_STARTED -> callback(null)
                        RESULT_FAILED -> callback(
                            resultData?.getString(EXTRA_ERROR_MESSAGE)
                                ?: "Unable to start screen capture service",
                        )
                    }
                }
            }
            val intent = Intent(context, ScreenCaptureService::class.java).apply {
                putExtra(EXTRA_RESULT_RECEIVER, receiver)
            }
            try {
                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                    context.startForegroundService(intent)
                } else {
                    context.startService(intent)
                }
            } catch (error: Throwable) {
                Log.e(TAG, "Unable to launch media projection foreground service", error)
                callback(error.message ?: error.javaClass.simpleName)
            }
        }

        fun stop(context: Context) {
            context.stopService(Intent(context, ScreenCaptureService::class.java))
        }
    }
}
