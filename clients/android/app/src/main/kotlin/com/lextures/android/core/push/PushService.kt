package com.lextures.android.core.push

import com.google.firebase.messaging.FirebaseMessagingService
import com.google.firebase.messaging.RemoteMessage

/** Receives FCM messages and routes deep links into the app shell. */
class PushService : FirebaseMessagingService() {
    private val pushManager by lazy { PushManager.getInstance(this) }

    override fun onNewToken(token: String) {
        pushManager.updateFcmToken(token)
    }

    override fun onMessageReceived(message: RemoteMessage) {
        val title = message.notification?.title
            ?: message.data["title"]
            ?: "Lextures"
        val body = message.notification?.body
            ?: message.data["body"]
            ?: ""
        val actionUrl = message.data["action_url"]
        val eventType = message.data["event_type"]
        pushManager.showLocalNotification(title, body, actionUrl, eventType)
    }
}
