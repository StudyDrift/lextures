package com.lextures.android.core.push

import android.Manifest
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.NotificationCompat
import androidx.core.app.NotificationManagerCompat
import androidx.core.content.ContextCompat
import com.google.firebase.messaging.FirebaseMessaging
import com.lextures.android.R
import com.lextures.android.app.MainActivity
import com.lextures.android.core.lms.LmsApi
import com.lextures.android.core.routing.DeepLinkDestination
import com.lextures.android.core.routing.DeepLinkRouter
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import kotlinx.coroutines.tasks.await

/** FCM registration, permission priming, and token sync with the backend. */
class PushManager private constructor(private val appContext: Context) {
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)
    private var accessTokenProvider: (() -> String?)? = null

    private val _pendingDeepLink = MutableStateFlow<DeepLinkDestination?>(null)
    val pendingDeepLink: StateFlow<DeepLinkDestination?> = _pendingDeepLink.asStateFlow()

    var registeredTokenId: String? = null
        private set

    var lastFcmToken: String? = null
        private set

    fun updateFcmToken(token: String) {
        lastFcmToken = token
        syncTokenWithBackend()
    }

    fun configure(accessToken: () -> String?) {
        accessTokenProvider = accessToken
        ensureNotificationChannel()
    }

    /** Request POST_NOTIFICATIONS in context (Android 13+), not at cold launch. */
    fun hasNotificationPermission(): Boolean {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.TIRAMISU) return true
        return ContextCompat.checkSelfPermission(
            appContext,
            Manifest.permission.POST_NOTIFICATIONS,
        ) == PackageManager.PERMISSION_GRANTED
    }

    fun requestTokenSync() {
        scope.launch {
            runCatching {
                lastFcmToken = FirebaseMessaging.getInstance().token.await()
                syncTokenWithBackend()
            }
        }
    }

    fun syncTokenWithBackend() {
        scope.launch {
            val token = lastFcmToken ?: return@launch
            val accessToken = accessTokenProvider?.invoke()?.takeIf { it.isNotBlank() } ?: return@launch
            runCatching {
                registeredTokenId = LmsApi.registerDeviceToken(token, "fcm", accessToken).id
            }
        }
    }

    fun deregisterFromBackend(explicitAccessToken: String? = null) {
        val tokenId = registeredTokenId ?: return
        val accessToken = explicitAccessToken?.takeIf { it.isNotBlank() }
            ?: accessTokenProvider?.invoke()?.takeIf { it.isNotBlank() }
            ?: return
        scope.launch {
            runCatching { LmsApi.deregisterDeviceToken(tokenId, accessToken) }
            registeredTokenId = null
        }
    }

    fun onDeepLinkFromPayload(actionUrl: String?) {
        _pendingDeepLink.value = DeepLinkRouter.resolve(actionUrl)
    }

    fun consumePendingDeepLink(): DeepLinkDestination? {
        val current = _pendingDeepLink.value
        _pendingDeepLink.value = null
        return current
    }

    fun showLocalNotification(title: String, body: String, actionUrl: String?) {
        if (!hasNotificationPermission()) return
        ensureNotificationChannel()
        val intent = Intent(appContext, MainActivity::class.java).apply {
            flags = Intent.FLAG_ACTIVITY_NEW_TASK or Intent.FLAG_ACTIVITY_CLEAR_TOP
            putExtra(EXTRA_ACTION_URL, actionUrl)
        }
        val pending = PendingIntent.getActivity(
            appContext,
            actionUrl.hashCode(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        val notification = NotificationCompat.Builder(appContext, CHANNEL_ID)
            .setSmallIcon(R.mipmap.ic_launcher)
            .setContentTitle(title)
            .setContentText(body)
            .setAutoCancel(true)
            .setContentIntent(pending)
            .setPriority(NotificationCompat.PRIORITY_HIGH)
            .build()
        NotificationManagerCompat.from(appContext).notify(actionUrl.hashCode(), notification)
    }

    private fun ensureNotificationChannel() {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) return
        val manager = appContext.getSystemService(NotificationManager::class.java) ?: return
        val channel = NotificationChannel(
            CHANNEL_ID,
            "Lextures",
            NotificationManager.IMPORTANCE_HIGH,
        )
        manager.createNotificationChannel(channel)
    }

    companion object {
        const val CHANNEL_ID = "lextures_push"
        const val EXTRA_ACTION_URL = "action_url"

        @Volatile
        private var instance: PushManager? = null

        fun getInstance(context: Context): PushManager =
            instance ?: synchronized(this) {
                instance ?: PushManager(context.applicationContext).also { instance = it }
            }
    }
}
