package com.lextures.android.features.planner

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import com.lextures.android.core.push.PushManager

class PlannerReminderReceiver : BroadcastReceiver() {
    override fun onReceive(context: Context, intent: Intent) {
        val title = intent.getStringExtra(EXTRA_TITLE) ?: return
        val body = intent.getStringExtra(EXTRA_BODY) ?: return
        PushManager.getInstance(context).showLocalNotification(title, body, null)
    }

    companion object {
        const val EXTRA_TITLE = "planner_reminder_title"
        const val EXTRA_BODY = "planner_reminder_body"
    }
}
