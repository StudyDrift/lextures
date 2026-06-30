package com.lextures.android.features.planner

import android.app.AlarmManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import androidx.core.content.getSystemService
import com.lextures.android.core.lms.DueReminderLeadTime
import com.lextures.android.core.lms.StudentTodoItem
import java.time.Instant

object DueReminderScheduler {
    private const val PREFS = "planner_reminders"
    private const val KEY_LEAD = "lead_minutes"

    fun selectedLeadTime(context: Context): DueReminderLeadTime {
        val raw = context.getSharedPreferences(PREFS, Context.MODE_PRIVATE).getInt(KEY_LEAD, 0)
        return DueReminderLeadTime.entries.firstOrNull { it.minutes == raw } ?: DueReminderLeadTime.None
    }

    fun setSelectedLeadTime(context: Context, lead: DueReminderLeadTime) {
        context.getSharedPreferences(PREFS, Context.MODE_PRIVATE)
            .edit()
            .putInt(KEY_LEAD, lead.minutes)
            .apply()
    }

    fun scheduleReminder(context: Context, item: StudentTodoItem) {
        val lead = selectedLeadTime(context)
        cancelReminder(context, item.key)
        val due = item.dueAt ?: return
        if (lead == DueReminderLeadTime.None) return
        val fireAt = due.minusSeconds(lead.minutes.toLong() * 60)
        if (fireAt.isBefore(Instant.now())) return

        val intent = Intent(context, PlannerReminderReceiver::class.java).apply {
            putExtra(PlannerReminderReceiver.EXTRA_TITLE, "Due soon")
            putExtra(PlannerReminderReceiver.EXTRA_BODY, "${item.title} · ${item.courseTitle}")
        }
        val pending = PendingIntent.getBroadcast(
            context,
            item.key.hashCode(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        context.getSystemService<AlarmManager>()?.setAndAllowWhileIdle(
            AlarmManager.RTC_WAKEUP,
            fireAt.toEpochMilli(),
            pending,
        )
    }

    fun cancelReminder(context: Context, itemKey: String) {
        val intent = Intent(context, PlannerReminderReceiver::class.java)
        val pending = PendingIntent.getBroadcast(
            context,
            itemKey.hashCode(),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        context.getSystemService<AlarmManager>()?.cancel(pending)
    }
}
