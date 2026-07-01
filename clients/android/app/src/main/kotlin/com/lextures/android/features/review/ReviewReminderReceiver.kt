package com.lextures.android.features.review

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import com.lextures.android.core.push.PushManager

class ReviewReminderReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        val title = intent.getStringExtra(EXTRA_TITLE) ?: return
        val body = intent.getStringExtra(EXTRA_BODY) ?: return
        val actionUrl = intent.getStringExtra(EXTRA_ACTION_URL)
        PushManager.getInstance(context).showLocalNotification(title, body, actionUrl)
    }

    companion object {
        const val EXTRA_TITLE = "review_reminder_title"
        const val EXTRA_BODY = "review_reminder_body"
        const val EXTRA_ACTION_URL = "review_reminder_action_url"
    }
}
