package com.lextures.android.features.parent

import android.app.AlarmManager
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import androidx.core.content.getSystemService
import com.lextures.android.R
import com.lextures.android.core.lms.ConferenceSlot
import com.lextures.android.core.lms.DueReminderLeadTime
import com.lextures.android.core.lms.LmsDates
import com.lextures.android.features.planner.DueReminderScheduler
import com.lextures.android.features.planner.PlannerReminderReceiver
import java.time.Instant

/** Schedules local reminders for booked parent–teacher conferences (M10.2 / M0.1). */
object ConferenceReminderScheduler {
    fun scheduleReminder(
        context: Context,
        slot: ConferenceSlot,
        teacherName: String,
        childName: String,
    ) {
        val lead = DueReminderScheduler.selectedLeadTime(context)
        cancelReminder(context, slot.id)
        if (lead == DueReminderLeadTime.None) return
        val start = LmsDates.parse(slot.startAt) ?: return
        val fireAt = start.minusSeconds(lead.minutes.toLong() * 60)
        if (fireAt.isBefore(Instant.now())) return

        val title = context.getString(R.string.mobile_parent_conferences_reminder_title)
        val body = context.getString(
            R.string.mobile_parent_conferences_reminder_body,
            teacherName,
            childName,
        )
        val intent = Intent(context, PlannerReminderReceiver::class.java).apply {
            putExtra(PlannerReminderReceiver.EXTRA_TITLE, title)
            putExtra(PlannerReminderReceiver.EXTRA_BODY, body)
        }
        val pending = PendingIntent.getBroadcast(
            context,
            reminderRequestCode(slot.id),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        context.getSystemService<AlarmManager>()?.setAndAllowWhileIdle(
            AlarmManager.RTC_WAKEUP,
            fireAt.toEpochMilli(),
            pending,
        )
    }

    fun cancelReminder(context: Context, slotId: String) {
        val intent = Intent(context, PlannerReminderReceiver::class.java)
        val pending = PendingIntent.getBroadcast(
            context,
            reminderRequestCode(slotId),
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_IMMUTABLE,
        )
        context.getSystemService<AlarmManager>()?.cancel(pending)
    }

    private fun reminderRequestCode(slotId: String): Int = "conference:$slotId".hashCode()
}
